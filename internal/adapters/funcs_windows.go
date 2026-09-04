//go:build windows

package adapters

import (
	"github.com/gostafa/inputstats/internal/adapters/windows"
)

// New returns the Windows Raw Input adapter.
func New() (*windows.Adapter, error) {
	return windows.New(), nil
}
