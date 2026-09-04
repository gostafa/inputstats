//go:build windows

package adapters

import (
	"github.com/gostafa/inputstats/internal/adapters/windows"
	"github.com/gostafa/inputstats/internal/ports"
)

// New returns the Windows Raw Input adapter.
func New() (ports.InputPort, error) {
	return windows.New(), nil
}
