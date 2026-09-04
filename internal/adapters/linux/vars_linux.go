//go:build linux

package linux

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/holoplot/go-evdev"
	"golang.org/x/sys/unix"
)

var errDeviceSkipped = errors.New("inputstats: linux device skipped")

func defaultHookSet() *hookSet {
	return &hookSet{
		openDevice: func(path string) (evdevDevice, error) {
			return evdev.OpenWithFlags(path, os.O_RDONLY)
		},
		globEventPaths:  filepath.Glob,
		inotifyInit1:    unix.InotifyInit1,
		inotifyAddWatch: unix.InotifyAddWatch,
		unixClose:       unix.Close,
		unixRead:        unix.Read,
		afterFunc:       time.AfterFunc,
		afterDelay:      time.After,
	}
}
