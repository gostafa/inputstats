//go:build linux

package linux

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/holoplot/go-evdev"
	"golang.org/x/sys/unix"

	"github.com/gostafa/inputstats/internal/domain"
	"github.com/gostafa/inputstats/internal/ports"
)

// New probes for usable input devices and returns an Adapter.
func New() (*Adapter, error) {
	if err := probeDevices(); err != nil {
		return nil, err
	}
	return &Adapter{
		opened: make(map[string]*evdev.InputDevice),
	}, nil
}

// Run discovers devices, reads them concurrently, and watches for hot-plug.
func (a *Adapter) Run(ctx context.Context, events chan<- domain.Event) error {
	paths, _ := listEventPaths()
	for _, path := range paths {
		a.tryOpen(ctx, path, events)
	}

	watchDone := make(chan struct{})
	go func() {
		defer close(watchDone)
		a.watchHotplug(ctx, events)
	}()

	<-ctx.Done()
	a.closeAll()
	<-watchDone
	a.wg.Wait()
	return ctx.Err()
}

func (a *Adapter) tryOpen(ctx context.Context, path string, events chan<- domain.Event) {
	a.mu.Lock()
	if _, exists := a.opened[path]; exists {
		a.mu.Unlock()
		return
	}
	a.mu.Unlock()

	dev, class, err := openAndClassify(path)
	if err != nil || !class.usable() {
		return
	}
	_ = dev.NonBlock()

	a.mu.Lock()
	if _, exists := a.opened[path]; exists {
		a.mu.Unlock()
		_ = dev.Close()
		return
	}
	if ctx.Err() != nil {
		a.mu.Unlock()
		_ = dev.Close()
		return
	}
	a.opened[path] = dev
	a.mu.Unlock()

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		a.readDevice(ctx, path, dev, class, events)
	}()
}

func (a *Adapter) readDevice(ctx context.Context, path string, dev *evdev.InputDevice, class deviceClass, events chan<- domain.Event) {
	defer func() {
		a.mu.Lock()
		if cur, ok := a.opened[path]; ok && cur == dev {
			delete(a.opened, path)
		}
		a.mu.Unlock()
		_ = dev.Close()
	}()

	var st mouseState
	for {
		if ctx.Err() != nil {
			return
		}
		ev, err := dev.ReadOne()
		if err != nil {
			if isAgain(err) {
				select {
				case <-ctx.Done():
					return
				case <-time.After(2 * time.Millisecond):
				}
				continue
			}
			// Device disappeared, was closed on cancel, or became unreadable.
			return
		}
		a.dispatch(ctx, ev, class, &st, events)
	}
}

func (a *Adapter) dispatch(ctx context.Context, ev *evdev.InputEvent, class deviceClass, st *mouseState, events chan<- domain.Event) {
	typ := uint16(ev.Type)
	code := uint16(ev.Code)
	switch {
	case class.keyboard && class.mouse:
		if kind, ok := parseEvdev(typ, code, ev.Value, st); ok {
			ports.Deliver(ctx, events, domain.Event{Type: kind})
		}
	case class.keyboard:
		if ev.Type == evdev.EV_KEY {
			handleKeyEvent(ctx, events, code, ev.Value)
		}
	case class.mouse:
		handleMouseEvent(ctx, events, typ, code, ev.Value, st)
	}
}

func (a *Adapter) closeAll() {
	a.mu.Lock()
	devs := make([]*evdev.InputDevice, 0, len(a.opened))
	for _, d := range a.opened {
		devs = append(devs, d)
	}
	a.mu.Unlock()
	for _, d := range devs {
		_ = d.Close()
	}
}

func (a *Adapter) watchHotplug(ctx context.Context, events chan<- domain.Event) {
	fd, err := unix.InotifyInit1(unix.IN_CLOEXEC | unix.IN_NONBLOCK)
	if err != nil {
		<-ctx.Done()
		return
	}
	defer unix.Close(fd)

	if _, err := unix.InotifyAddWatch(fd, inputDir, unix.IN_CREATE|unix.IN_ATTRIB|unix.IN_MOVED_TO); err != nil {
		<-ctx.Done()
		return
	}

	buf := make([]byte, (unix.SizeofInotifyEvent+unix.NAME_MAX+1)*8)
	for {
		if ctx.Err() != nil {
			return
		}
		n, err := unix.Read(fd, buf)
		if err != nil {
			if isAgain(err) {
				select {
				case <-ctx.Done():
					return
				case <-time.After(50 * time.Millisecond):
				}
				continue
			}
			return
		}
		for _, name := range parseInotifyNames(buf[:n]) {
			if !strings.HasPrefix(name, "event") {
				continue
			}
			path := filepath.Join(inputDir, name)
			// Permissions / node readiness can lag CREATE; brief retry.
			a.tryOpen(ctx, path, events)
			if ctx.Err() == nil {
				time.AfterFunc(100*time.Millisecond, func() {
					a.tryOpen(ctx, path, events)
				})
			}
		}
	}
}

func parseInotifyNames(buf []byte) []string {
	var names []string
	offset := 0
	for offset+unix.SizeofInotifyEvent <= len(buf) {
		raw := (*unix.InotifyEvent)(unsafe.Pointer(&buf[offset]))
		nameLen := int(raw.Len)
		offset += unix.SizeofInotifyEvent
		if nameLen <= 0 || offset+nameLen > len(buf) {
			break
		}
		name := strings.TrimRight(string(buf[offset:offset+nameLen]), "\x00")
		offset += nameLen
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

func isAgain(err error) bool {
	return errors.Is(err, syscall.EAGAIN) || errors.Is(err, syscall.EWOULDBLOCK) ||
		errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EWOULDBLOCK)
}

func (c deviceClass) usable() bool {
	return c.keyboard || c.mouse
}

// listEventPaths returns /dev/input/event* paths.
func listEventPaths() ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(inputDir, "event*"))
	if err != nil {
		return nil, err
	}
	return matches, nil
}

// openAndClassify opens path read-only and classifies keyboard/mouse capability.
// Caller owns the returned device when err is nil and class is usable; on
// unusable devices the handle is closed before return.
func openAndClassify(path string) (*evdev.InputDevice, deviceClass, error) {
	dev, err := evdev.OpenWithFlags(path, os.O_RDONLY)
	if err != nil {
		return nil, deviceClass{}, err
	}
	class := classifyDevice(dev)
	if !class.usable() {
		_ = dev.Close()
		return nil, class, nil
	}
	return dev, class, nil
}

// classifyDevice inspects capabilities; skips power/lid/accel-only nodes.
func classifyDevice(dev *evdev.InputDevice) deviceClass {
	if hasAccelProp(dev) {
		return deviceClass{}
	}

	var class deviceClass
	keyCodes := codeSet(dev.CapableEvents(evdev.EV_KEY))
	relCodes := codeSet(dev.CapableEvents(evdev.EV_REL))

	if hasKeyboardKeys(keyCodes) {
		class.keyboard = true
	}
	if hasMouseAxes(relCodes) || keyCodes[evdev.BTN_LEFT] {
		class.mouse = true
	}

	// Power/lid/switch-only (or ABS-only) devices have neither class.
	return class
}

func hasAccelProp(dev *evdev.InputDevice) bool {
	for _, p := range dev.Properties() {
		if p == evdev.INPUT_PROP_ACCELEROMETER {
			return true
		}
	}
	name, err := dev.Name()
	if err != nil {
		return false
	}
	lower := strings.ToLower(name)
	if strings.Contains(lower, "accelerometer") ||
		(strings.Contains(lower, "lid") && !strings.Contains(lower, "keyboard")) ||
		strings.Contains(lower, "power button") ||
		strings.Contains(lower, "sleep button") {
		return true
	}
	return false
}

func codeSet(codes []evdev.EvCode) map[evdev.EvCode]bool {
	m := make(map[evdev.EvCode]bool, len(codes))
	for _, c := range codes {
		m[c] = true
	}
	return m
}

func hasKeyboardKeys(keys map[evdev.EvCode]bool) bool {
	if keys[evdev.KEY_SPACE] {
		return true
	}
	for c := evdev.EvCode(evdev.KEY_A); c <= evdev.KEY_Z; c++ {
		if keys[c] {
			return true
		}
	}
	return false
}

func hasMouseAxes(rel map[evdev.EvCode]bool) bool {
	return rel[evdev.REL_X] || rel[evdev.REL_Y]
}

// probeDevices verifies at least one usable event device can be opened.
func probeDevices() error {
	paths, err := listEventPaths()
	if err != nil {
		if errors.Is(err, os.ErrPermission) || os.IsPermission(err) {
			return domain.ErrPermissionDenied
		}
		if errors.Is(err, os.ErrNotExist) {
			return domain.ErrNoInputDevices
		}
		return err
	}
	if len(paths) == 0 {
		return domain.ErrNoInputDevices
	}

	var sawPermission bool
	var usable int
	for _, path := range paths {
		dev, class, err := openAndClassify(path)
		if err != nil {
			if errors.Is(err, os.ErrPermission) || os.IsPermission(err) {
				sawPermission = true
			}
			continue
		}
		if class.usable() {
			usable++
			_ = dev.Close()
		}
	}
	if usable > 0 {
		return nil
	}
	if sawPermission {
		return domain.ErrPermissionDenied
	}
	return domain.ErrNoInputDevices
}

// handleKeyEvent processes an EV_KEY event for keyboard counting.
func handleKeyEvent(ctx context.Context, events chan<- domain.Event, code uint16, value int32) {
	if kind, ok := parseKey(code, value); ok {
		ports.Deliver(ctx, events, domain.Event{Type: kind})
	}
}

// handleMouseEvent processes mouse-related EV_KEY / EV_REL / EV_SYN events.
func handleMouseEvent(ctx context.Context, events chan<- domain.Event, typ, code uint16, value int32, st *mouseState) {
	switch typ {
	case evKey:
		if kind, ok := parseMouseButton(code, value); ok {
			ports.Deliver(ctx, events, domain.Event{Type: kind})
		}
	case evRel:
		noteRel(code, st)
	case evSyn:
		if kind, ok := flushSyn(code, st); ok {
			ports.Deliver(ctx, events, domain.Event{Type: kind})
		}
	}
}
