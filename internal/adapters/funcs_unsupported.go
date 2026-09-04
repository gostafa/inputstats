//go:build !linux && !windows && !darwin

package adapters

import (
	"context"

	"github.com/gostafa/inputstats/internal/domain"
)

type (
	// Adapter is the placeholder port for unsupported operating systems.
	Adapter struct{}
)

// New reports that this OS has no inputstats adapter.
func New() (*Adapter, error) {
	return nil, domain.ErrUnsupportedPlatform
}

// Run reports that this OS has no inputstats adapter.
func (*Adapter) Run(context.Context, chan<- domain.Event) error {
	return domain.ErrUnsupportedPlatform
}
