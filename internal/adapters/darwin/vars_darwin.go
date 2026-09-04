//go:build darwin

package darwin

import (
	"errors"
	"sync/atomic"
)

var (
	activeSink atomic.Pointer[eventSink]
	activeTap  atomic.Pointer[eventTap]
)

var (
	errTapCreate = errors.New("inputstats: CGEventTapCreate failed")
	errTapSource = errors.New("inputstats: CFMachPortCreateRunLoopSource failed")
)
