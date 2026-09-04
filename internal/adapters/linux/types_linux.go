//go:build linux

package linux

import (
	"sync"

	"github.com/holoplot/go-evdev"
)

// Adapter monitors Linux evdev devices and emits privacy-preserving activity kinds.
type Adapter struct {
	mu     sync.Mutex
	opened map[string]*evdev.InputDevice
	wg     sync.WaitGroup
}

// deviceClass describes which parsers a node should feed.
type deviceClass struct {
	keyboard bool
	mouse    bool
}
