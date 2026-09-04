package domain

import "time"

// EventType classifies a counted input activity.
type EventType uint8

// Event is a single counted input activity (no key identity or coordinates).
type Event struct {
	Type EventType
}

// Stats holds activity counts for one aggregation interval [From, To).
type Stats struct {
	From, To                                            time.Time
	KeyboardClicks, MouseMoves, LeftClicks, RightClicks uint64
}
