//go:build darwin

package darwin

import (
	"errors"
)

var (
	errTapCreate = errors.New("inputstats: CGEventTapCreate failed")
	errTapSource = errors.New("inputstats: CFMachPortCreateRunLoopSource failed")
)
