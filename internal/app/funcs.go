package app

import (
	"context"
	"time"

	"github.com/gostafa/inputstats/internal/domain"
	"github.com/gostafa/inputstats/internal/ports"
)

// aggregate reads input events, emits Stats on each ticker interval, and closes out on stop.
func aggregate(ctx context.Context, interval time.Duration, events <-chan domain.Event, out chan<- domain.Stats) {
	defer close(out)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	from := time.Now()
	var keyboard, moves, left, right uint64

	emit := func(to time.Time) bool {
		s := domain.Stats{
			From:           from,
			To:             to,
			KeyboardClicks: keyboard,
			MouseMoves:     moves,
			LeftClicks:     left,
			RightClicks:    right,
		}
		select {
		case out <- s:
			from = to
			keyboard, moves, left, right = 0, 0, 0, 0
			return true
		case <-ctx.Done():
			return false
		}
	}

	for {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-events:
			if !ok {
				return
			}
			switch e.Type {
			case domain.EventKeyboardClick:
				keyboard++
			case domain.EventMouseMove:
				moves++
			case domain.EventLeftClick:
				left++
			case domain.EventRightClick:
				right++
			}
		case t := <-ticker.C:
			if !emit(t) {
				return
			}
		}
	}
}

// Monitor validates inputs, aggregates events from port into interval Stats,
// and closes the returned channel when ctx is cancelled or the adapter exits.
func Monitor(ctx context.Context, interval time.Duration, port ports.InputPort) (<-chan domain.Stats, error) {
	if ctx == nil {
		return nil, domain.ErrNilContext
	}
	if interval <= 0 {
		return nil, domain.ErrInvalidInterval
	}

	events := make(chan domain.Event, 4096)
	out := make(chan domain.Stats, 1)

	go aggregate(ctx, interval, events, out)

	go func() {
		_ = port.Run(ctx, events)
		close(events)
	}()

	return out, nil
}
