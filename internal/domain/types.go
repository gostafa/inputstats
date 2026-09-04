// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package domain

import (
	"time"
)

type (
	// EventType classifies a counted input activity.
	EventType = uint8

	// Event is a single counted input activity (no key identity or coordinates).
	Event = struct {
		Type EventType
	}

	// Stats holds activity counts for one aggregation interval [From, To).
	Stats = struct {
		From, To                                            time.Time
		KeyboardClicks, MouseMoves, LeftClicks, RightClicks uint64
	}
)
