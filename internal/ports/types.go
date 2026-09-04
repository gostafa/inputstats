package ports

import (
	"context"

	"github.com/gostafa/inputstats/internal/domain"
)

// InputPort is the OS-specific input monitoring adapter.
type InputPort interface {
	Run(ctx context.Context, events chan<- domain.Event) error
}
