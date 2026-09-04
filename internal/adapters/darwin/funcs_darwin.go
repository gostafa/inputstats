//go:build darwin

package darwin

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"runtime/cgo"

	"github.com/gostafa/inputstats/internal/domain"
	"github.com/gostafa/inputstats/internal/ports"
)

// New returns a Darwin CGEventTap adapter. Returns ErrPermissionDenied when
// IOHIDCheckAccess reports listen access denied.
func New() (*Adapter, error) {
	port, err := pickAdapter(grantCode())
	if err != nil {
		return nil, fmt.Errorf(errFmtWrap, err)
	}

	return port, nil
}

// Run installs a session CGEventTap on a locked OS thread and pumps CFRunLoop
// until ctx is canceled. Returns ErrPermissionDenied if the tap stays disabled.
func (*Adapter) Run(ctx context.Context, events chan<- domain.Event) error {
	runtime.LockOSThread()

	defer runtime.UnlockOSThread()

	err := wrapRun(runEventTap(ctx, events))
	if err != nil {
		return fmt.Errorf(errFmtWrap, err)
	}

	return nil
}

func acceptTap(tap, source uintptr) (*eventTap, error) {
	if tap == cfRefNull {
		return nil, errTapCreate
	}

	if source == cfRefNull {
		return nil, errTapSource
	}

	return newEventTap(tap, source), nil
}

func bindState(
	done <-chan struct{},
	events chan<- domain.Event,
) (handle cgo.Handle, state *tapState) {
	state = &tapState{done: done, events: events}
	handle = cgo.NewHandle(state)

	return handle, state
}

func classifyCGEvent(typ uint32) (domain.EventType, bool) {
	if kind, ok := classifyClick(typ); ok {
		return kind, true
	}

	if kind, ok := classifyKey(typ); ok {
		return kind, true
	}

	return classifyMove(typ)
}

func classifyClick(typ uint32) (domain.EventType, bool) {
	switch typ {
	case cgEventLeftMouseDown:
		return domain.EventLeftClick, true
	case cgEventRightMouseDown:
		return domain.EventRightClick, true
	default:
		return noKind()
	}
}

func classifyKey(typ uint32) (domain.EventType, bool) {
	if typ != cgEventKeyDown {
		return noKind()
	}

	return domain.EventKeyboardClick, true
}

func classifyMove(typ uint32) (domain.EventType, bool) {
	switch typ {
	case cgEventMouseMoved,
		cgEventLeftMouseDragged,
		cgEventRightMouseDragged,
		cgEventOtherMouseDragged:
		return domain.EventMouseMove, true
	default:
		return noKind()
	}
}

func createEventTap(handle uintptr) (*eventTap, error) {
	out, err := openEventTap(handle)
	if err != nil {
		return nil, fmt.Errorf(errFmtWrap, err)
	}

	return out, nil
}

func doneErr(ctx context.Context) error {
	err := ctx.Err()
	if err != nil {
		return fmt.Errorf("darwin run loop: %w", err)
	}

	return nil
}

func dropTap(tap, source uintptr) {
	cReleasePort(tap)
	cReleaseSource(source)
}

func emitCGEvent(typ uint32, handle uintptr) {
	state := lookupState(handle)
	if state == nil {
		return
	}

	pushEvent(state, typ)
}

func finishTap(ctx context.Context, loop runLoop) error {
	serveUntilDone(ctx, loop)

	err := wrapRun(doneErr(ctx))
	if err != nil {
		return fmt.Errorf(errFmtWrap, err)
	}

	return nil
}

func grantCode() int {
	forced := os.Getenv(grantEnvName)
	if forced == grantEnvDeny {
		return int(cfRefNull)
	}

	if forced == grantEnvAllow {
		return int(cgEventLeftMouseDown)
	}

	return truthInt(hidFlag())
}

func hidListenGranted() bool {
	return hidFlag() != cfRefNull
}

func installTap(switcher tapSwitch) error {
	switcher.enable()

	if !switcher.isEnabled() {
		return domain.ErrPermissionDenied
	}

	return nil
}

func lookupState(handle uintptr) *tapState {
	if handle == cfRefNull {
		return nil
	}

	return stateFrom(cgo.Handle(handle))
}

func makeTap(handle uintptr) (tap, source uintptr) {
	if handle == stubHandle {
		fake := uintptr(cgEventLeftMouseDown)

		return fake, fake
	}

	if handle == failHandle {
		return cfRefNull, cfRefNull
	}

	tap = cCreateTapPtr(handle)
	source = cCreateSourcePtr(tap)

	return tap, source
}

func noKind() (domain.EventType, bool) {
	var kind domain.EventType

	return kind, false
}

func openCreated(ctx context.Context, state *tapState, result tapResult) error {
	if result.err != nil {
		return domain.ErrPermissionDenied
	}

	err := serveTap(ctx, result.tap, state)
	if err != nil {
		return fmt.Errorf(errFmtWrap, err)
	}

	return nil
}

func openEventTap(handle uintptr) (*eventTap, error) {
	tap, source := makeTap(handle)

	out, err := acceptTap(tap, source)
	if err != nil {
		dropTap(tap, source)

		return nil, errTapCreate
	}

	return out, nil
}

func pickAdapter(status int) (*Adapter, error) {
	if status == cfRefNull {
		return nil, domain.ErrPermissionDenied
	}

	return &Adapter{}, nil
}

func pushEvent(state *tapState, typ uint32) {
	kind, ok := classifyCGEvent(typ)
	if !ok {
		return
	}

	ports.Send(state.done, state.events, domain.Event{Type: kind})
}

func reenableTap(handle uintptr) {
	state := lookupState(handle)
	if state == nil || state.tap == nil {
		return
	}

	state.tap.enable()
}

func runEventTap(ctx context.Context, events chan<- domain.Event) error {
	handle, state := bindState(ctx.Done(), events)
	defer handle.Delete()

	err := withTap(ctx, handle, state)
	if err != nil {
		return fmt.Errorf(errFmtWrap, err)
	}

	return nil
}

func runInstalledTap(ctx context.Context, switcher tapSwitch, loop runLoop) error {
	err := installTap(switcher)
	if err != nil {
		return fmt.Errorf("darwin install: %w", err)
	}

	err = finishTap(ctx, loop)
	if err != nil {
		return fmt.Errorf(errFmtWrap, err)
	}

	return nil
}

func serveTap(ctx context.Context, tap *eventTap, state *tapState) error {
	defer tap.release()

	state.tap = tap

	err := runInstalledTap(ctx, tap, tap)
	if err != nil {
		return fmt.Errorf(errFmtWrap, err)
	}

	return nil
}

func serveUntilDone(ctx context.Context, loop runLoop) {
	ready := make(chan struct{})
	stopCh := make(chan struct{})

	go waitAndStop(ctx, struct {
		ready  <-chan struct{}
		stopCh <-chan struct{}
	}{ready: ready, stopCh: stopCh}, loop)

	loop.attach(ready)
	loop.run()
	close(stopCh)
}

func stateFrom(handle cgo.Handle) *tapState {
	state, ok := handle.Value().(*tapState)
	if !ok {
		return nil
	}

	return state
}

func tapCreate(ctx context.Context) tapFactory {
	create, ok := ctx.Value(createKey{}).(tapFactory)
	if ok {
		return create
	}

	return createEventTap
}

func truthInt(flag int) int {
	if flag == cfRefNull {
		return int(cfRefNull)
	}

	return flag
}

func waitAndStop(ctx context.Context, gate struct {
	ready  <-chan struct{}
	stopCh <-chan struct{}
}, loop runLoop,
) {
	<-gate.ready

	select {
	case <-ctx.Done():
	case <-gate.stopCh:
		return
	}

	loop.disable()
	loop.stop()
}

func withTap(ctx context.Context, handle cgo.Handle, state *tapState) error {
	tap, err := tapCreate(ctx)(uintptr(handle))

	openErr := openCreated(ctx, state, tapResult{err: err, tap: tap})
	if openErr != nil {
		return fmt.Errorf(errFmtWrap, openErr)
	}

	return nil
}

func wrapRun(err error) error {
	if err != nil {
		return fmt.Errorf(errFmtDarwin, err)
	}

	return nil
}
