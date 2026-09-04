//go:build linux

package adapters

import (
	"github.com/gostafa/inputstats/internal/adapters/linux"
	"github.com/gostafa/inputstats/internal/ports"
)

// New returns the Linux evdev input adapter.
func New() (ports.InputPort, error) {
	return linux.New()
}
