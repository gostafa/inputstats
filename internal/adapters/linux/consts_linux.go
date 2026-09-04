//go:build linux

package linux

import (
	"time"
)

const (
	inputDir = "/dev/input"

	againDelay        = 2 * time.Millisecond
	hotplugPollDelay  = 50 * time.Millisecond
	hotplugRetryDelay = 100 * time.Millisecond

	inotifyBufSlots = 8
	inotifyLenOff   = 12
	inotifyLenSize  = 4
	inotifyBufCap   = (unixSizeofInotifyEvent + unixNameMax + 1) * inotifyBufSlots

	eventNamePrefix = "event"
	eventNameGlob   = "event*"
	emptyName       = ""
	nullByte        = "\x00"

	errFmtLinux = "linux: %w"
	errFmtWrap  = "%w"

	// unixSizeofInotifyEvent and unixNameMax mirror golang.org/x/sys/unix
	// constants so buffer capacity stays a compile-time const.
	unixSizeofInotifyEvent = 16
	unixNameMax            = 255

	inotifyStop = -1
)
