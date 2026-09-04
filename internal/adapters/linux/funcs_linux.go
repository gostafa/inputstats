//go:build linux

package linux

import (
	"context"
	"fmt"
	"os"

	"github.com/gostafa/inputstats/internal/domain"
)

// New probes for usable input devices and returns an Adapter.
func New() (*Adapter, error) {
	port, err := newAdapter(defaultHookSet())
	if err != nil {
		return nil, fmt.Errorf(errFmtWrap, err)
	}

	return port, nil
}

func newAdapter(deps *hookSet) (*Adapter, error) {
	err := applyTestGrant()
	if err != nil {
		return nil, fmt.Errorf(errFmtWrap, err)
	}

	if testGrantAllow() {
		return &Adapter{}, nil
	}

	err = probeDevices(deps)
	if err != nil {
		return nil, fmt.Errorf(errFmtLinux, err)
	}

	return &Adapter{}, nil
}

func applyTestGrant() error {
	if os.Getenv(grantEnvName) == grantEnvDeny {
		return domain.ErrPermissionDenied
	}

	return nil
}

func testGrantAllow() bool {
	return os.Getenv(grantEnvName) == grantEnvAllow
}

// Run discovers devices, reads them concurrently, and watches for hot-plug.
func (*Adapter) Run(ctx context.Context, events chan<- domain.Event) error {
	return fmt.Errorf(errFmtLinux, runSession(ctx, newSession(events, defaultHookSet())))
}

func doneErr(ctx context.Context) error {
	err := ctx.Err()
	if err != nil {
		return fmt.Errorf(errFmtLinux, err)
	}

	return nil
}

func newSession(events chan<- domain.Event, deps *hookSet) *session {
	return &session{
		reg:    &registry{opened: make(map[string]evdevHandle)},
		events: events,
		deps:   deps,
	}
}

func openExisting(ctx context.Context, sess *session) error {
	paths, err := listEventPaths(sess.deps)
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

	return fmt.Errorf(errFmtWrap, waitCancel(ctx, sess))
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

	return fmt.Errorf(errFmtWrap, doneErr(ctx))
}
