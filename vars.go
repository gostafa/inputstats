// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package inputstats

import (
	"github.com/gostafa/inputstats/internal/domain"
)

var (
	// ErrPermissionDenied is returned when the OS denies input monitoring access.
	ErrPermissionDenied = domain.ErrPermissionDenied

	// ErrNoInputDevices is returned when no usable keyboard or mouse devices are found.
	ErrNoInputDevices = domain.ErrNoInputDevices

	// ErrUnsupportedPlatform is returned when the current OS is not supported.
	ErrUnsupportedPlatform = domain.ErrUnsupportedPlatform

	// ErrInvalidInterval is returned when the aggregation interval is not positive.
	ErrInvalidInterval = domain.ErrInvalidInterval

	// ErrNilContext is returned when Start is called with a nil context.
	ErrNilContext = domain.ErrNilContext
)
