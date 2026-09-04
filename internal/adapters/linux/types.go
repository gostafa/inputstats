// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package linux

type (
	// mouseState tracks REL motion awaiting SYN_REPORT coalescing.
	mouseState struct {
		pendingMove bool
	}

	// evdevSample is one raw input event for pure parsing.
	evdevSample struct {
		typ, code uint16
		value     int32
	}
)
