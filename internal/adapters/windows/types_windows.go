//go:build windows

package windows

import (
	"context"
	"sync"

	"github.com/gostafa/inputstats/internal/domain"
	"golang.org/x/sys/windows"
)

// Adapter monitors keyboard and mouse via Win32 Raw Input.
//
// Raw Input device registration is process-wide (one HWND per device class);
// loading this package alongside another Raw Input consumer can conflict.
type Adapter struct{}

type session struct {
	ctx    context.Context
	events chan<- domain.Event
	hwnd   uintptr
	buf    []byte
	mu     sync.Mutex
	closed bool
}

// rawInputView overlays the GetRawInputData buffer.
type rawInputView struct {
	header rawInputHeader
	mouse  rawMouse
}

type wndClass struct {
	style      uint32
	wndProc    uintptr
	clsExtra   int32
	wndExtra   int32
	instance   windows.Handle
	icon       windows.Handle
	cursor     windows.Handle
	background windows.Handle
	menuName   *uint16
	className  *uint16
}

type msg struct {
	hwnd    windows.HWND
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      struct{ x, y int32 }
}

type rawInputDevice struct {
	usagePage uint16
	usage     uint16
	flags     uint32
	target    windows.HWND
}

type rawInputHeader struct {
	dwType  uint32
	dwSize  uint32
	hDevice uintptr
	wParam  uintptr
}

type rawKeyboard struct {
	makeCode         uint16
	flags            uint16
	reserved         uint16
	vKey             uint16
	message          uint32
	extraInformation uint32
}

// rawMouse matches MSVC layout (2-byte pad after usFlags before the buttons union).
type rawMouse struct {
	flags            uint16
	_pad             uint16
	buttonFlags      uint16
	buttonData       uint16
	rawButtons       uint32
	lastX            int32
	lastY            int32
	extraInformation uint32
}
