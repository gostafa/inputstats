// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package linux

// Evdev type/code constants (linux/input-event-codes.h) for pure parsers.
// SYN_REPORT, REL_X, and key-up share 0 with EV_SYN; REL_Y and key-down share
// 1 with EV_KEY; key-repeat shares 2 with EV_REL — reuse those names at call sites.
const (
	evSyn = 0x00
	evKey = 0x01
	evRel = 0x02

	keyA     = 30
	keyZ     = 44
	keySpace = 57

	btnLeft    = 0x110
	btnRight   = 0x111
	btnMiddle  = 0x112
	btnSide    = 0x113
	btnExtra   = 0x114
	btnForward = 0x115
	btnBack    = 0x116
	btnTask    = 0x117
)
