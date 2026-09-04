//go:build windows

package windows

import "golang.org/x/sys/windows"

// Win32 constants for message-only Raw Input sink.
const (
	hwndMessage = ^windows.HWND(2) // HWND_MESSAGE = (HWND)-3

	wmDestroy = 0x0002
	wmQuit    = 0x0012
	wmInput   = 0x00FF

	gwlpUserData = -21

	ridInput = 0x10000003

	rimTypeMouse    = 0
	rimTypeKeyboard = 1

	ridevRemove    = 0x00000001
	ridevInputSink = 0x00000100
	ridevDevNotify = 0x00002000

	hidUsagePageGeneric     = 0x01
	hidUsageGenericMouse    = 0x02
	hidUsageGenericKeyboard = 0x06
)
