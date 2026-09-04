//go:build darwin

package darwin

import (
	"context"
	"errors"
	"runtime"
	"runtime/cgo"
	"testing"
	"time"

	"github.com/gostafa/inputstats/internal/domain"
)

type fakeLoop struct {
	enabled  bool
	blockRun chan struct{}
	ran      int
	stopped  int
}

func (f *fakeLoop) attach(ready chan<- struct{}) {
	close(ready)
}

func (f *fakeLoop) disable() {}

func (f *fakeLoop) enable() {}

func (f *fakeLoop) isEnabled() bool {
	return f.enabled
}

func (f *fakeLoop) release() {}

func (f *fakeLoop) run() {
	if f.blockRun != nil {
		<-f.blockRun
	}

	f.ran++
}

func (f *fakeLoop) stop() {
	f.stopped++

	if f.blockRun != nil {
		close(f.blockRun)
	}
}

func TestClassifyCGEvent(t *testing.T) {
	tests := []struct {
		name string
		typ  uint32
		want domain.EventType
		ok   bool
	}{
		{"keyDown", cgEventKeyDown, domain.EventKeyboardClick, true},
		{"mouseMoved", cgEventMouseMoved, domain.EventMouseMove, true},
		{"leftDragged", cgEventLeftMouseDragged, domain.EventMouseMove, true},
		{"rightDragged", cgEventRightMouseDragged, domain.EventMouseMove, true},
		{"otherDragged", cgEventOtherMouseDragged, domain.EventMouseMove, true},
		{"leftDown", cgEventLeftMouseDown, domain.EventLeftClick, true},
		{"rightDown", cgEventRightMouseDown, domain.EventRightClick, true},
		{"keyUpIgnored", 11, 0, false},
		{"leftUpIgnored", 2, 0, false},
		{"nullIgnored", 0, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kind, ok := classifyCGEvent(tt.typ)
			if ok != tt.ok {
				t.Fatalf("ok=%v, want %v", ok, tt.ok)
			}

			if !ok {
				return
			}

			if kind != tt.want {
				t.Fatalf("kind=%v, want %v", kind, tt.want)
			}
		})
	}
}

func TestAcceptSourceNull(t *testing.T) {
	_, err := acceptSource(cfRefNull, cfRefNull)
	if !errors.Is(err, errTapSource) {
		t.Fatalf("err=%v", err)
	}
}

func TestAcceptSourceOK(t *testing.T) {
	tap, err := acceptSource(1, 1)
	if err != nil || tap == nil {
		t.Fatalf("tap=%v err=%v", tap, err)
	}
}

func TestAcceptTapNull(t *testing.T) {
	_, err := acceptTap(cfRefNull, cfRefNull)
	if !errors.Is(err, errTapCreate) {
		t.Fatalf("err=%v", err)
	}
}

func TestAcceptTapOK(t *testing.T) {
	tap, err := acceptTap(uintptr(cgEventLeftMouseDown), uintptr(cgEventLeftMouseDown))
	if err != nil || tap == nil {
		t.Fatalf("tap=%v err=%v", tap, err)
	}
}

func TestAcceptTapBadSource(t *testing.T) {
	_, err := acceptTap(uintptr(cgEventLeftMouseDown), cfRefNull)
	if !errors.Is(err, errTapSource) {
		t.Fatalf("err=%v", err)
	}
}

func TestBindStateLookup(t *testing.T) {
	events := make(chan domain.Event, 1)
	handle, state := bindState(nil, events)
	defer handle.Delete()

	got := lookupState(uintptr(handle))
	if got != state {
		t.Fatal("lookup mismatch")
	}

	if lookupState(cfRefNull) != nil {
		t.Fatal("empty handle")
	}
}

func TestCreateEventTapFail(t *testing.T) {
	_, err := createEventTap(failHandle)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateEventTapStub(t *testing.T) {
	tap, err := createEventTap(stubHandle)
	if err != nil || tap == nil {
		t.Fatalf("tap=%v err=%v", tap, err)
	}
}

func TestCreateEventTapNull(t *testing.T) {
	tap, err := createEventTap(cfRefNull)
	if err != nil {
		t.Log(err)

		return
	}

	tap.release()
}

func TestCreateAndReleaseTap(t *testing.T) {
	handle, state := bindState(nil, make(chan domain.Event, 1))
	defer handle.Delete()

	tap, err := createEventTap(uintptr(handle))
	if err != nil {
		t.Log(err)

		return
	}

	state.tap = tap
	tap.enable()
	_ = tap.isEnabled()
	tap.disable()
	tap.release()
}

func TestDropTap(t *testing.T) {
	dropTap(cfRefNull, cfRefNull)
}

func TestDoneErr(t *testing.T) {
	if err := doneErr(context.Background()); err != nil {
		t.Fatalf("background: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := doneErr(ctx); err == nil {
		t.Fatal("expected cancel error")
	}
}

func TestEmitCGEvent(t *testing.T) {
	events := make(chan domain.Event, 1)
	handle, _ := bindState(nil, events)
	defer handle.Delete()

	emitCGEvent(99, uintptr(handle))
	emitCGEvent(cgEventKeyDown, uintptr(handle))
	emitCGEvent(cgEventKeyDown, cfRefNull)

	select {
	case got := <-events:
		if got.Type != domain.EventKeyboardClick {
			t.Fatalf("type=%v", got.Type)
		}
	default:
		t.Fatal("expected keyboard event")
	}
}

func TestExportedCallbacks(t *testing.T) {
	goHandleCGEvent(0, 0)
	goReenableTap(0)
}

func TestFinishTapCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	loop := &fakeLoop{blockRun: make(chan struct{})}

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err := finishTap(ctx, loop)
	if err == nil {
		t.Fatal("expected done error")
	}
}

func TestFinishTapOK(t *testing.T) {
	err := finishTap(context.Background(), &fakeLoop{})
	if err != nil {
		t.Fatalf("finishTap: %v", err)
	}
}

func TestGrantCodeAllow(t *testing.T) {
	t.Setenv(grantEnvName, grantEnvAllow)

	if grantCode() == int(cfRefNull) {
		t.Fatal("expected allow")
	}
}

func TestGrantCodeDeny(t *testing.T) {
	t.Setenv(grantEnvName, grantEnvDeny)

	if grantCode() != int(cfRefNull) {
		t.Fatal("expected deny")
	}
}

func TestHidListenGranted(t *testing.T) {
	_ = hidListenGranted()
}

func TestInstallTapDenied(t *testing.T) {
	err := installTap(&fakeLoop{})
	if !errors.Is(err, domain.ErrPermissionDenied) {
		t.Fatalf("err=%v", err)
	}
}

func TestInstallTapOK(t *testing.T) {
	err := installTap(&fakeLoop{enabled: true})
	if err != nil {
		t.Fatalf("install: %v", err)
	}
}

func TestNew(t *testing.T) {
	_, err := New()
	if err != nil && !errors.Is(err, domain.ErrPermissionDenied) {
		t.Fatalf("New: %v", err)
	}
}

func TestNewAllow(t *testing.T) {
	t.Setenv(grantEnvName, grantEnvAllow)

	got, err := New()
	if err != nil || got == nil {
		t.Fatalf("got=%v err=%v", got, err)
	}
}

func TestNewDeny(t *testing.T) {
	t.Setenv(grantEnvName, grantEnvDeny)

	got, err := New()
	if got != nil || !errors.Is(err, domain.ErrPermissionDenied) {
		t.Fatalf("got=%v err=%v", got, err)
	}
}

func TestNewAdapterDenied(t *testing.T) {
	got, err := pickAdapter(cfRefNull)
	if got != nil || !errors.Is(err, domain.ErrPermissionDenied) {
		t.Fatalf("got=%v err=%v", got, err)
	}
}

func TestNewAdapterOK(t *testing.T) {
	got, err := pickAdapter(int(cgEventLeftMouseDown))
	if err != nil || got == nil {
		t.Fatalf("got=%v err=%v", got, err)
	}
}

func TestTruthInt(t *testing.T) {
	if truthInt(0) != int(cfRefNull) {
		t.Fatal("null")
	}

	if truthInt(1) == int(cfRefNull) {
		t.Fatal("nonzero")
	}
}

func TestOpenCreatedDenied(t *testing.T) {
	err := openCreated(context.Background(), &tapState{}, tapResult{err: errTapCreate})
	if !errors.Is(err, domain.ErrPermissionDenied) {
		t.Fatalf("err=%v", err)
	}
}

func TestOpenCreatedInstallDenied(t *testing.T) {
	err := openCreated(context.Background(), &tapState{}, tapResult{tap: &eventTap{}})
	if !errors.Is(err, domain.ErrPermissionDenied) {
		t.Fatalf("err=%v", err)
	}
}

func TestOpenCreatedOK(t *testing.T) {
	err := openCreated(context.Background(), &tapState{}, tapResult{tap: &eventTap{stub: true}})
	if err != nil {
		t.Fatalf("openCreated: %v", err)
	}
}

func TestPushMouseMove(t *testing.T) {
	events := make(chan domain.Event)
	state := &tapState{events: events}

	pushEvent(state, cgEventMouseMoved)
}

func TestReenableTap(t *testing.T) {
	reenableTap(cfRefNull)

	events := make(chan domain.Event, 1)
	handle, state := bindState(nil, events)
	defer handle.Delete()

	reenableTap(uintptr(handle))

	state.tap = &eventTap{}
	reenableTap(uintptr(handle))
}

func TestRunAdapter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	events := make(chan domain.Event, 1)

	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()

	err := (&Adapter{}).Run(ctx, events)
	if err != nil {
		t.Log(err)
	}
}

func TestRunAdapterOK(t *testing.T) {
	ctx := context.WithValue(
		context.Background(),
		createKey{},
		tapFactory(func(uintptr) (*eventTap, error) {
			return &eventTap{stub: true}, nil
		}),
	)
	err := (&Adapter{}).Run(ctx, make(chan domain.Event, 1))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestRunAdapterFail(t *testing.T) {
	ctx := context.WithValue(
		context.Background(),
		createKey{},
		tapFactory(func(uintptr) (*eventTap, error) {
			return nil, errTapCreate
		}),
	)
	err := (&Adapter{}).Run(ctx, make(chan domain.Event, 1))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunInstalledTapComplete(t *testing.T) {
	loop := &fakeLoop{enabled: true}

	err := runInstalledTap(context.Background(), loop, loop)
	if err != nil {
		t.Fatalf("runInstalledTap: %v", err)
	}
}

func TestRunInstalledTapDenied(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := runInstalledTap(ctx, &fakeLoop{}, &fakeLoop{})
	if !errors.Is(err, domain.ErrPermissionDenied) {
		t.Fatalf("err=%v", err)
	}
}

func TestRunInstalledTapOK(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	loop := &fakeLoop{enabled: true, blockRun: make(chan struct{})}

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err := runInstalledTap(ctx, loop, loop)
	if err == nil {
		t.Fatal("expected cancel")
	}
}

func TestServeUntilDoneImmediate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serveUntilDone(ctx, &fakeLoop{})
}

func TestServeUntilDoneRunLoop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	tap := &eventTap{}
	done := make(chan struct{})

	go func() {
		runtime.LockOSThread()
		serveUntilDone(ctx, tap)
		tap.release()
		close(done)
	}()

	time.Sleep(30 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("run loop did not stop")
	}
}

func TestStateFromWrongType(t *testing.T) {
	handle := cgo.NewHandle("nope")
	defer handle.Delete()

	if stateFrom(handle) != nil {
		t.Fatal("expected nil")
	}
}

func TestZeroTapMethods(t *testing.T) {
	tap := &eventTap{}
	tap.enable()
	_ = tap.isEnabled()
	tap.disable()
	tap.detachSource()
	tap.releaseSource()
	tap.releaseTapPort()
	tap.releaseLoop()
}

func TestRunEventTap(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	events := make(chan domain.Event, 1)

	go func() {
		time.Sleep(40 * time.Millisecond)
		cancel()
	}()

	err := runEventTap(ctx, events)
	if err != nil {
		t.Log(err)
	}
}

func TestRunEventTapFail(t *testing.T) {
	ctx := context.WithValue(
		context.Background(),
		createKey{},
		tapFactory(func(uintptr) (*eventTap, error) {
			return nil, errTapCreate
		}),
	)
	err := runEventTap(ctx, make(chan domain.Event, 1))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunEventTapOK(t *testing.T) {
	ctx := context.WithValue(
		context.Background(),
		createKey{},
		tapFactory(func(uintptr) (*eventTap, error) {
			return &eventTap{stub: true}, nil
		}),
	)
	err := runEventTap(ctx, make(chan domain.Event, 1))
	if err != nil {
		t.Fatalf("runEventTap: %v", err)
	}
}

func TestServeTapDenied(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := serveTap(ctx, &eventTap{}, &tapState{})
	if !errors.Is(err, domain.ErrPermissionDenied) {
		t.Fatalf("err=%v", err)
	}
}

func TestServeTapOK(t *testing.T) {
	err := serveTap(context.Background(), &eventTap{stub: true}, &tapState{})
	if err != nil {
		t.Fatalf("serveTap: %v", err)
	}
}

func TestWrapRun(t *testing.T) {
	if err := wrapRun(nil); err != nil {
		t.Fatalf("nil: %v", err)
	}

	if err := wrapRun(errors.New("x")); err == nil {
		t.Fatal("expected wrap")
	}
}

func TestWithTap(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handle, state := bindState(ctx.Done(), make(chan domain.Event, 1))
	defer handle.Delete()

	go func() {
		time.Sleep(40 * time.Millisecond)
		cancel()
	}()

	err := withTap(ctx, handle, state)
	if err != nil {
		t.Log(err)
	}
}

func TestWithTapFail(t *testing.T) {
	ctx := context.WithValue(
		context.Background(),
		createKey{},
		tapFactory(func(uintptr) (*eventTap, error) {
			return nil, errTapCreate
		}),
	)
	handle, state := bindState(ctx.Done(), make(chan domain.Event, 1))
	defer handle.Delete()

	err := withTap(ctx, handle, state)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWithTapOK(t *testing.T) {
	ctx := context.WithValue(
		context.Background(),
		createKey{},
		tapFactory(func(uintptr) (*eventTap, error) {
			return &eventTap{stub: true}, nil
		}),
	)
	handle, state := bindState(ctx.Done(), make(chan domain.Event, 1))
	defer handle.Delete()

	err := withTap(ctx, handle, state)
	if err != nil {
		t.Fatalf("withTap: %v", err)
	}
}
