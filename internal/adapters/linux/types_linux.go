//go:build linux

package linux

import (
	"sync"
	"time"

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
		opened map[string]evdevHandle
		wg     sync.WaitGroup
		mu     sync.Mutex
	}

	session = struct {
		reg    *registry
		events chan<- domain.Event
		deps   *hookSet
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

	evdevDevice interface {
		evdevHandle
		evdevInfo
	}

	readJob = struct {
		sess  *session
		dev   evdevHandle
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
		dev evdevHandle
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

	classified = struct {
		dev   evdevHandle
		class deviceClass
	}

	hookSet = struct {
		openDevice      func(string) (evdevDevice, error)
		globEventPaths  func(string) ([]string, error)
		inotifyInit1    func(int) (int, error)
		inotifyAddWatch func(int, string, uint32) (int, error)
		unixClose       func(int) error
		unixRead        func(int, []byte) (int, error)
		afterFunc       func(time.Duration, func()) *time.Timer
		afterDelay      func(time.Duration) <-chan time.Time
	}
)
