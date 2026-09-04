//go:build linux

package linux

import (
	"context"
	"fmt"

	"github.com/gostafa/inputstats/internal/domain"
	"github.com/holoplot/go-evdev"
)

// New probes for usable input devices and returns an Adapter.
func New() (*Adapter, error) {
	err := probeDevices()
	if err != nil {
		return nil, fmt.Errorf(errFmtLinux, err)
	}

	return &Adapter{}, nil
}

// Run discovers devices, reads them concurrently, and watches for hot-plug.
func (*Adapter) Run(ctx context.Context, events chan<- domain.Event) error {
	err := wrapRun(runSession(ctx, newSession(events)))
	if err != nil {
		return fmt.Errorf(errFmtWrap, err)
	}

	return nil
}

func doneErr(ctx context.Context) error {
	err := ctx.Err()
	if err != nil {
		return fmt.Errorf(errFmtLinux, err)
	}

	return nil
}

func newSession(events chan<- domain.Event) *session {
	return &session{
		reg:    &registry{opened: make(map[string]*evdev.InputDevice)},
		events: events,
	}
}

func openExisting(ctx context.Context, sess *session) error {
	paths, err := listEventPaths()
	if err != nil {
		return fmt.Errorf(errFmtWrap, err)
	}

	openPaths(ctx, sess, paths)

	return nil
}

func openPaths(ctx context.Context, sess *session, paths []string) {
	for index := range paths {
		tryOpen(ctx, paths[index], sess)
	}
}

func runSession(ctx context.Context, sess *session) error {
	err := openExisting(ctx, sess)
	if err != nil {
		return fmt.Errorf(errFmtWrap, err)
	}

	err = waitCancel(ctx, sess)
	if err != nil {
		return fmt.Errorf(errFmtWrap, err)
	}

	return nil
}

func startHotplug(ctx context.Context, sess *session) <-chan struct{} {
	watchDone := make(chan struct{})

	go func() {
		defer close(watchDone)

		watchHotplug(ctx, sess)
	}()

	return watchDone
}

func waitCancel(ctx context.Context, sess *session) error {
	watchDone := startHotplug(ctx, sess)

	<-ctx.Done()
	closeAll(sess.reg)
	<-watchDone
	sess.reg.wg.Wait()

	err := doneErr(ctx)
	if err != nil {
		return fmt.Errorf(errFmtWrap, err)
	}

	return nil
}

func wrapRun(err error) error {
	if err != nil {
		return fmt.Errorf(errFmtLinux, err)
	}

	return nil
}
