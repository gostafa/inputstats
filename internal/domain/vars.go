// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package domain

import (
	"errors"
)

// Sentinel errors returned by adapters and Monitor.
var (
	ErrPermissionDenied    = errors.New("inputstats: permission denied")
	ErrNoInputDevices      = errors.New("inputstats: no input devices")
	ErrUnsupportedPlatform = errors.New("inputstats: unsupported platform")
	ErrInvalidInterval     = errors.New("inputstats: interval must be positive")
	ErrNilContext          = errors.New("inputstats: nil context")
)
