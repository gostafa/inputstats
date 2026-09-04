package ports

import (
	"context"

	"github.com/gostafa/inputstats/internal/domain"
)

// Deliver sends e to events. Mouse moves are non-blocking (may drop under
// backlog); keyboard and click events block until sent or ctx is done.
func Deliver(ctx context.Context, events chan<- domain.Event, e domain.Event) {
	if e.Type == domain.EventMouseMove {
		select {
		case events <- e:
		default:
		}
		return
	}
	select {
	case events <- e:
	case <-ctx.Done():
	}
}
