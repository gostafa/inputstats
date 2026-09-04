//go:build windows

package windows

import (
	"context"
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/gostafa/inputstats/internal/domain"
	"github.com/gostafa/inputstats/internal/ports"
	"golang.org/x/sys/windows"
)

// New returns a Windows Raw Input adapter.
func New() *Adapter {
	return &Adapter{}
}

// Run locks the OS thread, creates a message-only window, registers Raw Input,
// and pumps messages until ctx is cancelled.
func (a *Adapter) Run(ctx context.Context, events chan<- domain.Event) error {
	if ctx == nil {
		return fmt.Errorf("windows: nil context")
	}
	if events == nil {
		return fmt.Errorf("windows: nil events")
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	s := &session{
		ctx:    ctx,
		events: events,
		buf:    make([]byte, 64),
	}

	hwnd, err := createMessageWindow(s)
	if err != nil {
		return err
	}
	s.hwnd = uintptr(hwnd)

	if err := registerInputDevices(hwnd); err != nil {
		_ = destroyWindow(hwnd)
		return err
	}

	// Cancel posts WM_QUIT to this thread; then unregister + DestroyWindow (no leaks).
	tid := getCurrentThreadID()
	cancelDone := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = postThreadMessage(tid, wmQuit, 0, 0)
		case <-cancelDone:
		}
	}()

	loopErr := runMessageLoop()
	close(cancelDone)

	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()

	_ = unregisterInputDevices(hwnd)
	_ = destroyWindow(hwnd)

	if ctx.Err() != nil {
		return ctx.Err()
	}
	return loopErr
}

func (s *session) deliver(kind domain.EventType) {
	s.mu.Lock()
	closed := s.closed
	s.mu.Unlock()
	if closed {
		return
	}
	ports.Deliver(s.ctx, s.events, domain.Event{Type: kind})
}

func registerInputDevices(hwnd windows.HWND) error {
	devices := []rawInputDevice{
		{
			usagePage: hidUsagePageGeneric,
			usage:     hidUsageGenericKeyboard,
			flags:     ridevInputSink | ridevDevNotify,
			target:    hwnd,
		},
		{
			usagePage: hidUsagePageGeneric,
			usage:     hidUsageGenericMouse,
			flags:     ridevInputSink | ridevDevNotify,
			target:    hwnd,
		},
	}
	if err := registerRawInputDevices(devices); err != nil {
		return fmt.Errorf("windows: RegisterRawInputDevices: %w", err)
	}
	return nil
}

func unregisterInputDevices(hwnd windows.HWND) error {
	devices := []rawInputDevice{
		{
			usagePage: hidUsagePageGeneric,
			usage:     hidUsageGenericKeyboard,
			flags:     ridevRemove,
			target:    hwnd,
		},
		{
			usagePage: hidUsagePageGeneric,
			usage:     hidUsageGenericMouse,
			flags:     ridevRemove,
			target:    hwnd,
		},
	}
	return registerRawInputDevices(devices)
}

func (s *session) handleRawInput(lParam uintptr) {
	ri, ok := s.readRawInput(lParam)
	if !ok {
		return
	}
	switch ri.header.dwType {
	case rimTypeKeyboard:
		handleKeyboard(s, ri.keyboardFlags())
	case rimTypeMouse:
		handleMouse(s, ri.mouse.buttonFlags, ri.mouse.lastX, ri.mouse.lastY)
	}
}

func (s *session) readRawInput(lParam uintptr) (*rawInputView, bool) {
	headerSize := uint32(unsafe.Sizeof(rawInputHeader{}))
	for {
		size := uint32(len(s.buf))
		if size == 0 {
			s.buf = make([]byte, 64)
			size = uint32(len(s.buf))
		}
		n, err := getRawInputData(lParam, ridInput, &s.buf[0], &size, headerSize)
		if n == ^uint32(0) {
			if size > uint32(len(s.buf)) {
				s.buf = make([]byte, size)
				continue
			}
			_ = err
			return nil, false
		}
		if n < headerSize {
			return nil, false
		}
		return (*rawInputView)(unsafe.Pointer(&s.buf[0])), true
	}
}

func (r *rawInputView) keyboardFlags() uint16 {
	// RAWKEYBOARD overlays the same union as RAWMOUSE; Flags is at offset 2.
	kb := (*rawKeyboard)(unsafe.Pointer(&r.mouse))
	return kb.flags
}

func ensureWindowClass() error {
	classOnce.Do(func() {
		name, err := windows.UTF16PtrFromString("inputstats.rawinput.v1")
		if err != nil {
			classErr = err
			return
		}
		className = name

		inst, err := getModuleHandle()
		if err != nil {
			classErr = err
			return
		}

		wc := wndClass{
			wndProc:   wndProcCB,
			instance:  inst,
			className: className,
		}
		if _, err := registerClass(&wc); err != nil {
			// ERROR_CLASS_ALREADY_EXISTS (1410) is fine across repeated New().
			if errno, ok := err.(syscall.Errno); ok && errno == 1410 {
				return
			}
			classErr = err
		}
	})
	return classErr
}

func createMessageWindow(s *session) (windows.HWND, error) {
	if err := ensureWindowClass(); err != nil {
		return 0, fmt.Errorf("windows: register class: %w", err)
	}
	inst, err := getModuleHandle()
	if err != nil {
		return 0, fmt.Errorf("windows: module handle: %w", err)
	}

	hwnd, err := createWindowEx(
		0,
		className,
		nil,
		0,
		0, 0, 0, 0,
		hwndMessage,
		0,
		inst,
		0,
	)
	if err != nil {
		return 0, fmt.Errorf("windows: create HWND_MESSAGE: %w", err)
	}

	if _, err := setWindowLongPtr(hwnd, gwlpUserData, uintptr(unsafe.Pointer(s))); err != nil {
		_ = destroyWindow(hwnd)
		return 0, fmt.Errorf("windows: set userdata: %w", err)
	}
	return hwnd, nil
}

func runMessageLoop() error {
	var m msg
	for {
		ret, err := getMessage(&m, 0, 0, 0)
		if ret == 0 {
			return nil // WM_QUIT
		}
		if ret == -1 {
			return fmt.Errorf("windows: GetMessage: %w", err)
		}
		translateMessage(&m)
		dispatchMessage(&m)
	}
}

func wndProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	h := windows.HWND(hwnd)
	switch message {
	case wmInput:
		if s := sessionFromHWND(h); s != nil {
			s.handleRawInput(lParam)
		}
		return 0
	case wmDestroy:
		// DestroyWindow is owned by Run after unregister; still quit if destroyed externally.
		postQuitMessage(0)
		return 0
	}
	return defWindowProc(h, message, wParam, lParam)
}

func sessionFromHWND(hwnd windows.HWND) *session {
	p := getWindowLongPtr(hwnd, gwlpUserData)
	if p == 0 {
		return nil
	}
	return (*session)(unsafe.Pointer(p))
}

func handleKeyboard(s *session, flags uint16) {
	kind, ok := parseKeyboardFlags(flags)
	if !ok {
		return
	}
	s.deliver(kind)
}

func handleMouse(s *session, buttonFlags uint16, lastX, lastY int32) {
	for _, kind := range parseMouseButtons(buttonFlags) {
		s.deliver(kind)
	}
	if kind, ok := parseMouseMove(lastX, lastY); ok {
		s.deliver(kind)
	}
}

func getModuleHandle() (windows.Handle, error) {
	r, _, e := procGetModuleHandleW.Call(0)
	if r == 0 {
		return 0, e
	}
	return windows.Handle(r), nil
}

func getCurrentThreadID() uint32 {
	r, _, _ := procGetCurrentThreadId.Call()
	return uint32(r)
}

func registerClass(wc *wndClass) (uint16, error) {
	r, _, e := procRegisterClassW.Call(uintptr(unsafe.Pointer(wc)))
	if r == 0 {
		return 0, e
	}
	return uint16(r), nil
}

func createWindowEx(exStyle uint32, className, windowName *uint16, style uint32,
	x, y, w, h int32, parent windows.HWND, menu, instance windows.Handle, param uintptr,
) (windows.HWND, error) {
	r, _, e := procCreateWindowExW.Call(
		uintptr(exStyle),
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowName)),
		uintptr(style),
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		uintptr(parent),
		uintptr(menu),
		uintptr(instance),
		param,
	)
	if r == 0 {
		return 0, e
	}
	return windows.HWND(r), nil
}

func destroyWindow(hwnd windows.HWND) error {
	r, _, e := procDestroyWindow.Call(uintptr(hwnd))
	if r == 0 {
		return e
	}
	return nil
}

func defWindowProc(hwnd windows.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	r, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return r
}

func getMessage(m *msg, hwnd windows.HWND, min, max uint32) (int32, error) {
	r, _, e := procGetMessageW.Call(
		uintptr(unsafe.Pointer(m)),
		uintptr(hwnd),
		uintptr(min),
		uintptr(max),
	)
	// GetMessage returns -1 on error, 0 on WM_QUIT, >0 otherwise.
	if int32(r) == -1 {
		return -1, e
	}
	return int32(r), nil
}

func translateMessage(m *msg) {
	procTranslateMessage.Call(uintptr(unsafe.Pointer(m)))
}

func dispatchMessage(m *msg) {
	procDispatchMessageW.Call(uintptr(unsafe.Pointer(m)))
}

func postThreadMessage(threadID uint32, msg uint32, wParam, lParam uintptr) error {
	r, _, e := procPostThreadMessageW.Call(uintptr(threadID), uintptr(msg), wParam, lParam)
	if r == 0 {
		return e
	}
	return nil
}

func postQuitMessage(code int32) {
	procPostQuitMessage.Call(uintptr(code))
}

func setWindowLongPtr(hwnd windows.HWND, index int32, value uintptr) (uintptr, error) {
	r, _, e := procSetWindowLongPtrW.Call(uintptr(hwnd), uintptr(index), value)
	if r == 0 && e != syscall.Errno(0) {
		return 0, e
	}
	return r, nil
}

func getWindowLongPtr(hwnd windows.HWND, index int32) uintptr {
	r, _, _ := procGetWindowLongPtrW.Call(uintptr(hwnd), uintptr(index))
	return r
}

func registerRawInputDevices(devices []rawInputDevice) error {
	if len(devices) == 0 {
		return nil
	}
	r, _, e := procRegisterRawInputDevices.Call(
		uintptr(unsafe.Pointer(&devices[0])),
		uintptr(uint32(len(devices))),
		uintptr(unsafe.Sizeof(devices[0])),
	)
	if r == 0 {
		return e
	}
	return nil
}

func getRawInputData(
	hRaw uintptr,
	command uint32,
	data *byte,
	dataSize *uint32,
	headerSize uint32,
) (uint32, error) {
	var pData uintptr
	if data != nil {
		pData = uintptr(unsafe.Pointer(data))
	}
	r, _, e := procGetRawInputData.Call(
		hRaw,
		uintptr(command),
		pData,
		uintptr(unsafe.Pointer(dataSize)),
		uintptr(headerSize),
	)
	if r == ^uintptr(0) {
		return ^uint32(0), e
	}
	return uint32(r), nil
}
