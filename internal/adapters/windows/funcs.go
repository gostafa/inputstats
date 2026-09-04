// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package windows

import (
	"github.com/gostafa/inputstats/internal/domain"
)

// noKind is the failed-parse result (ok=false).
func noKind() (domain.EventType, bool) {
	var kind domain.EventType

	return kind, false
}

// parseKeyboardFlags reports a keyboard click when the key is not a break (release).
// Make/VKey are intentionally unused — counts only.
func parseKeyboardFlags(flags uint16) (domain.EventType, bool) {
	if flags&riKeyBreak != flagClear {
		return noKind()
	}

	return domain.EventKeyboardClick, true
}

// parseMouseButtons returns left/right click kinds for button-down flags only.
func parseMouseButtons(buttonFlags uint16) []domain.EventType {
	var out []domain.EventType

	if buttonFlags&riKeyBreak != flagClear {
		out = append(out, domain.EventLeftClick)
	}

	if buttonFlags&riMouseRightButtonDown != flagClear {
		out = append(out, domain.EventRightClick)
	}

	return out
}

// parseMouseMove reports one move when either axis delta is non-zero.
func parseMouseMove(lastX, lastY int32) (domain.EventType, bool) {
	if lastX|lastY == flagClear {
		return noKind()
	}

	return domain.EventMouseMove, true
}
