//go:build linux

package linux

import (
	"errors"
	"os"
	"time"

	"github.com/holoplot/go-evdev"
)

type readStep struct {
	event *evdev.InputEvent
	err   error
}

type fakeDev struct {
	name     string
	nameErr  error
	props    []evdev.EvProp
	keys     []evdev.EvCode
	rels     []evdev.EvCode
	steps    []readStep
	step     int
	nonBlock error
	closeErr error
	closed   int
}

func (f *fakeDev) Name() (string, error) {
	return f.name, f.nameErr
}

func (f *fakeDev) Properties() []evdev.EvProp {
	return f.props
}

func (f *fakeDev) CapableEvents(evType evdev.EvType) []evdev.EvCode {
	switch evType {
	case evdev.EV_KEY:
		return f.keys
	case evdev.EV_REL:
		return f.rels
	default:
		return nil
	}
}

func (f *fakeDev) NonBlock() error {
	return f.nonBlock
}

func (f *fakeDev) Close() error {
	f.closed++

	return f.closeErr
}

func (f *fakeDev) ReadOne() (*evdev.InputEvent, error) {
	if f.step >= len(f.steps) {
		return nil, errors.New("eof")
	}

	cur := f.steps[f.step]
	f.step++

	return cur.event, cur.err
}

func testHookSet() *hookSet {
	return defaultHookSet()
}

func setImmediateTimers(deps *hookSet) {
	deps.afterFunc = func(_ time.Duration, fn func()) *time.Timer {
		return time.AfterFunc(0, fn)
	}
	deps.afterDelay = func(_ time.Duration) <-chan time.Time {
		ch := make(chan time.Time, 1)
		ch <- time.Now()

		return ch
	}
}

func kbdFake() *fakeDev {
	return &fakeDev{name: "Keyboard", keys: []evdev.EvCode{evdev.KEY_A}}
}

func mouseFake() *fakeDev {
	return &fakeDev{
		name: "Mouse",
		keys: []evdev.EvCode{evdev.BTN_LEFT},
		rels: []evdev.EvCode{evdev.REL_X, evdev.REL_Y},
	}
}

func comboFake() *fakeDev {
	return &fakeDev{
		name: "Combo",
		keys: []evdev.EvCode{evdev.KEY_SPACE, evdev.BTN_LEFT},
		rels: []evdev.EvCode{evdev.REL_X},
	}
}

func stubOpen(deps *hookSet, devs map[string]*fakeDev) {
	deps.openDevice = func(path string) (evdevDevice, error) {
		dev, ok := devs[path]
		if !ok {
			return nil, os.ErrNotExist
		}

		return dev, nil
	}
}

func stubGlob(deps *hookSet, paths []string, err error) {
	deps.globEventPaths = func(string) ([]string, error) {
		return paths, err
	}
}

func ev(typ, code uint16, value int32) *evdev.InputEvent {
	return &evdev.InputEvent{Type: evdev.EvType(typ), Code: evdev.EvCode(code), Value: value}
}
