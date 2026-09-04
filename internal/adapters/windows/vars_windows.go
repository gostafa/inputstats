//go:build windows

package windows

import (
	"sync"
	"syscall"

	"golang.org/x/sys/windows"
)

var (
	classOnce sync.Once
	className *uint16
	classErr  error
	wndProcCB = syscall.NewCallback(wndProc)
)

var (
	modUser32   = windows.NewLazySystemDLL("user32.dll")
	modKernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procRegisterClassW          = modUser32.NewProc("RegisterClassW")
	procCreateWindowExW         = modUser32.NewProc("CreateWindowExW")
	procDestroyWindow           = modUser32.NewProc("DestroyWindow")
	procDefWindowProcW          = modUser32.NewProc("DefWindowProcW")
	procGetMessageW             = modUser32.NewProc("GetMessageW")
	procTranslateMessage        = modUser32.NewProc("TranslateMessage")
	procDispatchMessageW        = modUser32.NewProc("DispatchMessageW")
	procPostThreadMessageW      = modUser32.NewProc("PostThreadMessageW")
	procPostQuitMessage         = modUser32.NewProc("PostQuitMessage")
	procSetWindowLongPtrW       = modUser32.NewProc("SetWindowLongPtrW")
	procGetWindowLongPtrW       = modUser32.NewProc("GetWindowLongPtrW")
	procRegisterRawInputDevices = modUser32.NewProc("RegisterRawInputDevices")
	procGetRawInputData         = modUser32.NewProc("GetRawInputData")

	procGetModuleHandleW   = modKernel32.NewProc("GetModuleHandleW")
	procGetCurrentThreadId = modKernel32.NewProc("GetCurrentThreadId")
)
