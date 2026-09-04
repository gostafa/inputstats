package inputstats

import (
	"context"
	"errors"
	"testing"
	"time"
)

const testInterval = 20 * time.Millisecond

func TestStartNilContext(t *testing.T) {
	ch, err := Start(nil, time.Second)
	if !errors.Is(err, ErrNilContext) {
		t.Fatalf("err=%v, want ErrNilContext", err)
	}
	if ch != nil {
		t.Fatal("expected nil channel")
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

func TestStartCancelClosesChannel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	out, err := Start(ctx, testInterval)
	if errors.Is(err, ErrPermissionDenied) || errors.Is(err, ErrNoInputDevices) {
		t.Skip("no accessible input devices")
	}
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Adapter may emit nothing; may still get zero stats before cancel.
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
