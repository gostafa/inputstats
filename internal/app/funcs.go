// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package app

import (
	"context"
	"fmt"
	"time"

	"github.com/gostafa/inputstats/internal/domain"
	"github.com/gostafa/inputstats/internal/ports"
)

// Monitor validates inputs, aggregates events from port into interval Stats,
// and closes the returned channel when ctx is canceled or the adapter exits.
func Monitor(
	ctx context.Context,
	interval time.Duration,
	port ports.InputPort,
) (<-chan domain.Stats, error) {
	err := checkMonitorArgs(ctx, interval)
	if err != nil {
		return nil, fmt.Errorf("monitor: %w", err)
	}

	return spawnMonitor(ctx, interval, port), nil
}

// aggregate reads input events, emits Stats on each ticker interval, and closes out on stop.
func aggregate(
	ctx context.Context,
	interval time.Duration,
	events <-chan domain.Event,
) <-chan domain.Stats {
	out := make(chan domain.Stats, statsBufSize)
	from := time.Now()

	var counts [eventKindCount]uint64

	go drive(ctx, interval, &aggPipe{
		events: events,
		out:    out,
		from:   &from,
		counts: &counts,
	})

	return out
}

func bumpCount(counts *[eventKindCount]uint64, kind domain.EventType) {
	idx := int(kind)
	if idx >= len(counts) {
		return
	}

	counts[idx]++
}

func checkMonitorArgs(ctx context.Context, interval time.Duration) error {
	if ctx == nil {
		return domain.ErrNilContext
	}

	if interval <= zeroDuration {
		return domain.ErrInvalidInterval
	}

	return nil
}

func drainPort(ctx context.Context, port ports.InputPort, events chan domain.Event) {
	err := port.Run(ctx, events)
	close(events)

	if err != nil {
		return
	}
}

func drive(ctx context.Context, interval time.Duration, pipe *aggPipe) {
	defer close(pipe.out)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	loop(ctx, ticker, pipe)
}

func flushWindow(ctx context.Context, pipe *aggPipe, end time.Time) bool {
	stats := snapshot(pipe, end)
	select {
	case pipe.out <- stats:
		*pipe.from = end
		clear(pipe.counts[:])

		return true
	case <-ctx.Done():
		return false
	}
}

func loop(ctx context.Context, ticker *time.Ticker, pipe *aggPipe) {
	for {
		if !step(ctx, ticker, pipe) {
			return
		}
	}
}

func snapshot(pipe *aggPipe, end time.Time) domain.Stats {
	return domain.Stats{
		From:           *pipe.from,
		To:             end,
		KeyboardClicks: pipe.counts[domain.EventKeyboardClick],
		MouseMoves:     pipe.counts[domain.EventMouseMove],
		LeftClicks:     pipe.counts[domain.EventLeftClick],
		RightClicks:    pipe.counts[domain.EventRightClick],
	}
}

func spawnMonitor(
	ctx context.Context,
	interval time.Duration,
	port ports.InputPort,
) <-chan domain.Stats {
	events := make(chan domain.Event, eventBufSize)
	out := aggregate(ctx, interval, events)

	go drainPort(ctx, port, events)

	return out
}

func step(ctx context.Context, ticker *time.Ticker, pipe *aggPipe) bool {
	select {
	case <-ctx.Done():
		return false
	case event, opened := <-pipe.events:
		if !opened {
			return false
		}

		bumpCount(pipe.counts, event.Type)

		return true
	case tick := <-ticker.C:
		return flushWindow(ctx, pipe, tick)
	}
}
