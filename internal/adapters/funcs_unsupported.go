//go:build !linux && !windows && !darwin

package adapters

import (
	"github.com/gostafa/inputstats/internal/domain"
	"github.com/gostafa/inputstats/internal/ports"
)

// New reports that this OS has no inputstats adapter.
func New() (ports.InputPort, error) {
	return nil, domain.ErrUnsupportedPlatform
}
