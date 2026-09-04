// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package app

import (
	"time"

	"github.com/gostafa/inputstats/internal/domain"
)

type (
	aggPipe = struct {
		events <-chan domain.Event
		out    chan<- domain.Stats
		from   *time.Time
		counts *[eventKindCount]uint64
	}
)
