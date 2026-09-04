//go:build darwin

package adapters

import (
	"fmt"

	"github.com/gostafa/inputstats/internal/adapters/darwin"
)

const (
	errFmtNew = "adapters: %w"
)

// New returns the Darwin CGEventTap input adapter.
func New() (*darwin.Adapter, error) {
	port, err := wrapDarwin(darwin.New())
	if err != nil {
		return nil, fmt.Errorf(errFmtNew, err)
	}

	return port, nil
}

func wrapDarwin(port *darwin.Adapter, err error) (*darwin.Adapter, error) {
	if err != nil {
		return nil, fmt.Errorf(errFmtNew, err)
	}

	return port, nil
}
