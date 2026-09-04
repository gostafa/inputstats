//go:build darwin

package darwin

import (
	"github.com/gostafa/inputstats/internal/domain"
)

type (
	// Adapter monitors keyboard and mouse via a listen-only CGEventTap.
	Adapter struct{}

	createKey struct{}

	tapState = struct {
		done   <-chan struct{}
		events chan<- domain.Event
		tap    *eventTap
	}

	tapResult = struct {
		err error
		tap *eventTap
	}

	tapFactory = func(uintptr) (*eventTap, error)

	runLoop interface {
		attach(ready chan<- struct{})
		disable()
		run()
		stop()
	}

	tapSwitch interface {
		enable()
		isEnabled() bool
	}
)
