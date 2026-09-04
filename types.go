// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package inputstats

import (
	"github.com/gostafa/inputstats/internal/domain"
	"github.com/gostafa/inputstats/internal/ports"
)

type (
	// Stats holds activity counts for one aggregation interval [From, To).
	Stats = domain.Stats

	boot = struct {
		err  error
		port ports.InputPort
	}
)
