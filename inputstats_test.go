package inputstats

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gostafa/inputstats/internal/domain"
)

const testInterval = 20 * time.Millisecond

type fakePort struct {
	err error
}

func (f *fakePort) Run(ctx context.Context, _ chan<- domain.Event) error {
	if f.err != nil {
		return f.err
	}

	<-ctx.Done()

	return ctx.Err()
}

func TestMonitorInvalidInterval(t *testing.T) {
	ch, err := monitor(context.Background(), 0, &fakePort{})
	if !errors.Is(err, ErrInvalidInterval) {
		t.Fatalf("err=%v, want ErrInvalidInterval", err)
	}

	if ch != nil {
		t.Fatal("expected nil channel")
	}
}

func TestMonitorOK(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := monitor(ctx, testInterval, &fakePort{})
	if err != nil {
		t.Fatalf("monitor: %v", err)
	}

	cancel()

	for range ch {
	}
}

func TestStartCancelClosesChannel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	out, err := Start(ctx, testInterval)
	if errors.Is(err, ErrPermissionDenied) || errors.Is(err, ErrNoInputDevices) {
		t.Skip("no accessible input devices")
	}

	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	cancel()

	deadline := time.After(time.Second)
	for {
		select {
		case _, ok := <-out:
			if !ok {
				return
			}
		case <-deadline:
			t.Fatal("timed out waiting for Start channel close")
		}
	}
}

func TestStartInvalidInterval(t *testing.T) {
	for _, interval := range []time.Duration{0, -time.Millisecond} {
		ch, err := Start(context.Background(), interval)
		if !errors.Is(err, ErrInvalidInterval) {
			t.Fatalf("interval=%v err=%v, want ErrInvalidInterval", interval, err)
		}

		if ch != nil {
			t.Fatal("expected nil channel")
		}
	}
}

func TestStartNilContext(t *testing.T) {
	ch, err := Start(nil, time.Second)
	if !errors.Is(err, ErrNilContext) {
		t.Fatalf("err=%v, want ErrNilContext", err)
	}

	if ch != nil {
		t.Fatal("expected nil channel")
	}
}

func TestStartWithError(t *testing.T) {
	ch, err := startWith(context.Background(), time.Second, &boot{err: errors.New("denied")})
	if err == nil || ch != nil {
		t.Fatal("expected wrapped error")
	}
}

func TestStartWithMonitorError(t *testing.T) {
	ch, err := startWith(context.Background(), 0, &boot{port: &fakePort{}})
	if err == nil || ch != nil {
		t.Fatal("expected wrapped error")
	}
}

func TestStartWithOK(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := startWith(ctx, testInterval, &boot{port: &fakePort{}})
	if err != nil {
		t.Fatalf("startWith: %v", err)
	}

	cancel()

	for range ch {
	}
}

func TestStartOK(t *testing.T) {
	t.Setenv("INPUTSTATS_TEST_GRANT", "allow")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := Start(ctx, testInterval)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	cancel()

	for range ch {
	}
}

func TestStartAdapterError(t *testing.T) {
	t.Setenv("INPUTSTATS_TEST_GRANT", "deny")

	ch, err := Start(context.Background(), testInterval)
	if err == nil || ch != nil {
		t.Fatal("expected adapter error")
	}
}
