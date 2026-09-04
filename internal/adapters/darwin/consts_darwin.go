//go:build darwin

package darwin

// CGEventType values from CoreGraphics (CGEventTypes.h).
const (
	cgEventLeftMouseDown     uint32 = 1
	cgEventRightMouseDown    uint32 = 3
	cgEventMouseMoved        uint32 = 5
	cgEventLeftMouseDragged  uint32 = 6
	cgEventRightMouseDragged uint32 = 7
	cgEventKeyDown           uint32 = 10
	cgEventOtherMouseDragged uint32 = 27

	cfRefNull = 0

	errFmtDarwin = "darwin: %w"
	errFmtWrap   = "%w"

	grantEnvAllow = "allow"
	grantEnvDeny  = "deny"
	grantEnvName  = "INPUTSTATS_TEST_GRANT"

	failHandle = ^uintptr(1)
	stubHandle = ^uintptr(0)
)
