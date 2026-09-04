package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gostafa/inputstats/internal/domain"
)

const testInterval = 20 * time.Millisecond

func recvStats(t *testing.T, out <-chan domain.Stats, timeout time.Duration) domain.Stats {
	t.Helper()
	select {
	case s, ok := <-out:
		if !ok {
			t.Fatal("stats channel closed unexpectedly")
		}
		return s
	case <-time.After(timeout):
		t.Fatal("timed out waiting for stats")
		return domain.Stats{}
	}
}

func recvStatsWhere(t *testing.T, out <-chan domain.Stats, timeout time.Duration, match func(domain.Stats) bool) domain.Stats {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			t.Fatal("timed out waiting for matching stats")
		}
		s := recvStats(t, out, remaining)
		if match(s) {
			return s
		}
	}
}

func startAggregate(t *testing.T, interval time.Duration, initial ...domain.EventType) (context.CancelFunc, chan domain.Event, <-chan domain.Stats) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	events := make(chan domain.Event, 64)
	for _, typ := range initial {
		events <- domain.Event{Type: typ}
	}
	out := make(chan domain.Stats, 1)
	go aggregate(ctx, interval, events, out)
	t.Cleanup(func() {
		cancel()
		for range out {
		}
	})
	return cancel, events, out
}

func TestAggregateIntervalTotals(t *testing.T) {
	_, _, out := startAggregate(t, testInterval,
		domain.EventKeyboardClick, domain.EventKeyboardClick,
		domain.EventMouseMove,
		domain.EventLeftClick,
		domain.EventRightClick,
	)

	s := recvStatsWhere(t, out, time.Second, func(s domain.Stats) bool {
		return s.KeyboardClicks == 2 && s.MouseMoves == 1 && s.LeftClicks == 1 && s.RightClicks == 1
	})
	if s.From.IsZero() || s.To.IsZero() || s.To.Before(s.From) {
		t.Fatalf("invalid window From=%v To=%v", s.From, s.To)
	}
}

func TestAggregateKeyboardRepeats(t *testing.T) {
	_, _, out := startAggregate(t, testInterval,
		domain.EventKeyboardClick,
		domain.EventKeyboardClick,
		domain.EventKeyboardClick,
	)

	_ = recvStatsWhere(t, out, time.Second, func(s domain.Stats) bool {
		return s.KeyboardClicks == 3
	})
}

func TestAggregateZeroActivityEmits(t *testing.T) {
	_, _, out := startAggregate(t, testInterval)

	s := recvStats(t, out, time.Second)
	if s.KeyboardClicks != 0 || s.MouseMoves != 0 || s.LeftClicks != 0 || s.RightClicks != 0 {
		t.Fatalf("expected zero stats, got %+v", s)
	}
}

func TestAggregateCancelClosesChannel(t *testing.T) {
	cancel, _, out := startAggregate(t, testInterval)
	cancel()

	deadline := time.After(time.Second)
	for {
		select {
		case _, ok := <-out:
			if !ok {
				return
			}
		case <-deadline:
			t.Fatal("timed out waiting for channel close after cancel")
		}
	}
}

func TestAggregateMultipleIntervalsReset(t *testing.T) {
	_, events, out := startAggregate(t, testInterval, domain.EventKeyboardClick, domain.EventLeftClick)

	_ = recvStatsWhere(t, out, time.Second, func(s domain.Stats) bool {
		return s.KeyboardClicks == 1 && s.LeftClicks == 1
	})

	fakeFeeder{events: events}.send(domain.EventMouseMove, domain.EventMouseMove, domain.EventRightClick)

	_ = recvStatsWhere(t, out, time.Second, func(s domain.Stats) bool {
		return s.KeyboardClicks == 0 && s.MouseMoves == 2 && s.LeftClicks == 0 && s.RightClicks == 1
	})
}

func TestAggregateEventsClosedStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := make(chan domain.Event, 8)
	out := make(chan domain.Stats, 1)
	go aggregate(ctx, testInterval, events, out)

	close(events)

	deadline := time.After(time.Second)
	for {
		select {
		case _, ok := <-out:
			if !ok {
				return
			}
		case <-deadline:
			t.Fatal("timed out waiting for close after events closed")
		}
	}
}

// fakeFeeder sends scripted events into the aggregator's event channel.
type fakeFeeder struct {
	events chan<- domain.Event
}

func (f fakeFeeder) send(types ...domain.EventType) {
	for _, typ := range types {
		f.events <- domain.Event{Type: typ}
	}
}

// fakePort is an InputPort that feeds scripted events then waits for cancel.
type fakePort struct {
	toSend []domain.Event
	err    error
}

func (f *fakePort) Run(ctx context.Context, events chan<- domain.Event) error {
	for _, e := range f.toSend {
		select {
		case events <- e:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if f.err != nil {
		return f.err
	}
	<-ctx.Done()
	return ctx.Err()
}

func TestMonitorNilContext(t *testing.T) {
	ch, err := Monitor(nil, time.Second, &fakePort{})
	if !errors.Is(err, domain.ErrNilContext) {
		t.Fatalf("err=%v, want ErrNilContext", err)
	}
	if ch != nil {
		t.Fatal("expected nil channel")
	}
}

func TestMonitorInvalidInterval(t *testing.T) {
	for _, interval := range []time.Duration{0, -time.Millisecond} {
		ch, err := Monitor(context.Background(), interval, &fakePort{})
		if !errors.Is(err, domain.ErrInvalidInterval) {
			t.Fatalf("interval=%v err=%v, want ErrInvalidInterval", interval, err)
		}
		if ch != nil {
			t.Fatal("expected nil channel")
		}
	}
}

func TestMonitorFakePortAggregates(t *testing.T) {
	port := &fakePort{toSend: []domain.Event{
		{Type: domain.EventKeyboardClick},
		{Type: domain.EventKeyboardClick},
		{Type: domain.EventMouseMove},
		{Type: domain.EventLeftClick},
		{Type: domain.EventRightClick},
	}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	out, err := Monitor(ctx, testInterval, port)
	if err != nil {
		t.Fatalf("Monitor: %v", err)
	}

	s := recvStatsWhere(t, out, time.Second, func(s domain.Stats) bool {
		return s.KeyboardClicks == 2 && s.MouseMoves == 1 && s.LeftClicks == 1 && s.RightClicks == 1
	})
	if s.From.IsZero() || s.To.IsZero() || s.To.Before(s.From) {
		t.Fatalf("invalid window From=%v To=%v", s.From, s.To)
	}
}

func TestMonitorCancelClosesChannel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	out, err := Monitor(ctx, testInterval, &fakePort{})
	if err != nil {
		t.Fatalf("Monitor: %v", err)
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
			t.Fatal("timed out waiting for Monitor channel close")
		}
	}
}

func TestMonitorPortExitClosesChannel(t *testing.T) {
	port := &fakePort{err: errors.New("adapter exit")}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	out, err := Monitor(ctx, testInterval, port)
	if err != nil {
		t.Fatalf("Monitor: %v", err)
	}

	deadline := time.After(time.Second)
	for {
		select {
		case _, ok := <-out:
			if !ok {
				return
			}
		case <-deadline:
			t.Fatal("timed out waiting for close after port exit")
		}
	}
}
