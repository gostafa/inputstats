//go:build linux

package adapters

import (
	"fmt"

	"github.com/gostafa/inputstats/internal/adapters/linux"
)

const (
	errFmtNew = "adapters: %w"
)

// New returns the Linux evdev input adapter.
func New() (*linux.Adapter, error) {
	port, err := wrapLinux(linux.New())
	if err != nil {
		return nil, fmt.Errorf(errFmtNew, err)
	}

	return port, nil
}

func wrapLinux(port *linux.Adapter, err error) (*linux.Adapter, error) {
	if err != nil {
		return nil, fmt.Errorf(errFmtNew, err)
	}

	return port, nil
}
