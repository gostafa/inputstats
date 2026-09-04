// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package inputstats

import (
	"context"
	"fmt"
	"time"

	"github.com/gostafa/inputstats/internal/adapters"
	"github.com/gostafa/inputstats/internal/app"
	"github.com/gostafa/inputstats/internal/ports"
)

// Start begins input monitoring and returns a channel of interval Stats.
// The channel is closed when ctx is canceled or the adapter exits.
// Adapter construction is synchronous; goroutines start only after it succeeds.
func Start(ctx context.Context, interval time.Duration) (<-chan Stats, error) {
	// Validate before opening OS adapters so arg errors are not masked by
	// permission/device failures (e.g. Linux CI without /dev/input access).
	if err := checkStartArgs(ctx, interval); err != nil {
		return nil, fmt.Errorf(errFmtStart, err)
	}

	port, openErr := adapters.New()

	out, startErr := startWith(ctx, interval, &boot{err: openErr, port: port})
	if startErr != nil {
		return nil, fmt.Errorf(errFmtWrap, startErr)
	}

	return out, nil
}

func checkStartArgs(ctx context.Context, interval time.Duration) error {
	if ctx == nil {
		return ErrNilContext
	}

	if interval <= 0 {
		return ErrInvalidInterval
	}

	return nil
}

func monitor(
	ctx context.Context,
	interval time.Duration,
	port ports.InputPort,
) (<-chan Stats, error) {
	out, err := app.Monitor(ctx, interval, port)
	if err != nil {
		return nil, fmt.Errorf(errFmtStart, err)
	}

	return out, nil
}

func startWith(ctx context.Context, interval time.Duration, job *boot) (<-chan Stats, error) {
	if job.err != nil {
		return nil, fmt.Errorf(errFmtStart, job.err)
	}

	out, err := monitor(ctx, interval, job.port)
	if err != nil {
		return nil, fmt.Errorf(errFmtWrap, err)
	}

	return out, nil
}
