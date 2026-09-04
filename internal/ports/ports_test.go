// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package ports

import (
	"context"
	"testing"
	"time"

	"github.com/gostafa/inputstats/internal/domain"
)

func TestDeliverKeyboardBlocksUntilSent(t *testing.T) {
	ctx := context.Background()
	events := make(chan domain.Event, 1)
	Deliver(ctx, events, domain.Event{Type: domain.EventKeyboardClick})

	select {
	case got := <-events:
		if got.Type != domain.EventKeyboardClick {
			t.Fatalf("type=%v", got.Type)
		}
	default:
		t.Fatal("expected delivered keyboard event")
	}
}

func TestDeliverMouseMoveDropsWhenFull(t *testing.T) {
	ctx := context.Background()
	events := make(chan domain.Event)
	Deliver(ctx, events, domain.Event{Type: domain.EventMouseMove})
}

func TestDeliverKeyboardStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	events := make(chan domain.Event)
	done := make(chan struct{})
	go func() {
		Deliver(ctx, events, domain.Event{Type: domain.EventLeftClick})
		close(done)
	}()
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Deliver did not return after cancel")
	}
}
