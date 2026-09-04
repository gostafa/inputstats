//go:build linux

package linux

import (
	"sync"

	"github.com/gostafa/inputstats/internal/domain"
	"github.com/holoplot/go-evdev"
)

type (
	// Adapter monitors Linux evdev devices and emits privacy-preserving activity kinds.
	Adapter struct{}

	deviceClass struct {
		keyboard bool
		mouse    bool
	}

	registry = struct {
		opened map[string]*evdev.InputDevice
		wg     sync.WaitGroup
		mu     sync.Mutex
	}

	session = struct {
		reg    *registry
		events chan<- domain.Event
	}

	evdevHandle interface {
		ReadOne() (*evdev.InputEvent, error)
		Close() error
		NonBlock() error
	}

	evdevInfo interface {
		Name() (string, error)
		Properties() []evdev.EvProp
		CapableEvents(evType evdev.EvType) []evdev.EvCode
	}

	readJob = struct {
		sess  *session
		dev   *evdev.InputDevice
		path  string
		class deviceClass
	}

	dispatchJob = struct {
		event  *evdev.InputEvent
		state  *mouseState
		events chan<- domain.Event
		class  deviceClass
	}

	mouseSample = struct {
		state     *mouseState
		typ, code uint16
		value     int32
	}

	keySample = struct {
		code  uint16
		value int32
	}

	deviceClaim = struct {
		reg *registry
		dev *evdev.InputDevice
	}

	probeStats = struct {
		usable        int
		sawPermission bool
	}

	hotplugRead = struct {
		sess    *session
		buf     []byte
		watchFD int
	}
)
