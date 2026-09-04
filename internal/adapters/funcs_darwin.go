//go:build darwin

package adapters

import (
	"github.com/gostafa/inputstats/internal/adapters/darwin"
	"github.com/gostafa/inputstats/internal/ports"
)

// New returns the Darwin CGEventTap input adapter.
func New() (ports.InputPort, error) {
	return darwin.New()
}
