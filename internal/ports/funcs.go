// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package ports

import (
	"context"

	"github.com/gostafa/inputstats/internal/domain"
)

// Deliver sends e to events. Mouse moves are non-blocking (may drop under
// backlog); keyboard and click events block until sent or ctx is done.
func Deliver(ctx context.Context, events chan<- domain.Event, event domain.Event) {
	Send(ctx.Done(), events, event)
}

// Send is Deliver using a done channel instead of a context.
func Send(done <-chan struct{}, events chan<- domain.Event, event domain.Event) {
	if event.Type == domain.EventMouseMove {
		tryDeliver(events, event)

		return
	}

	waitDeliver(done, events, event)
}

func tryDeliver(events chan<- domain.Event, event domain.Event) {
	select {
	case events <- event:
	default:
	}
}

func waitDeliver(done <-chan struct{}, events chan<- domain.Event, event domain.Event) {
	select {
	case events <- event:
	case <-done:
	}
}
