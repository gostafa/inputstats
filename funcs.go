package inputstats

import (
	"context"
	"time"

	"github.com/gostafa/inputstats/internal/adapters"
	"github.com/gostafa/inputstats/internal/app"
)

// Start begins input monitoring and returns a channel of interval Stats.
// The channel is closed when ctx is cancelled or the adapter exits.
// Adapter construction is synchronous; goroutines start only after it succeeds.
func Start(ctx context.Context, interval time.Duration) (<-chan Stats, error) {
	port, err := adapters.New()
	if err != nil {
		return nil, err
	}
	return app.Monitor(ctx, interval, port)
}
