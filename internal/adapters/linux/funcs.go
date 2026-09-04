// Gostafa 2026.
// SPDX-License-Identifier: Apache-2.0.

package linux

import (
	"github.com/gostafa/inputstats/internal/domain"
)

// isMouseButton reports codes handled exclusively by the mouse parser.
func isMouseButton(code uint16) bool {
	switch code {
	case btnLeft, btnRight, btnMiddle, btnSide, btnExtra, btnForward, btnBack, btnTask:
		return true
	default:
		return false
	}
}

// isKeyPress reports EV_KEY down (1) and repeat (2); up (0) is ignored.
func isKeyPress(value int32) bool {
	return value == int32(evKey) || value == int32(evRel)
}

// noKind is the failed-parse result (ok=false).
func noKind() (domain.EventType, bool) {
	var kind domain.EventType

	return kind, false
}

// noteRel marks pending mouse motion for REL_X / REL_Y.
func noteRel(code uint16, state *mouseState) {
	if code == evSyn || code == evKey {
		state.pendingMove = true
	}
}

// parseEvKey prefers mouse buttons, else keyboard clicks.
func parseEvKey(code uint16, value int32) (domain.EventType, bool) {
	if kind, ok := parseMouseButton(code, value); ok {
		return kind, true
	}

	return parseKey(code, value)
}

// parseEvdev applies one input event against mouse coalesce state and returns
// a counted activity kind when appropriate.
func parseEvdev(sample evdevSample, state *mouseState) (domain.EventType, bool) {
	switch sample.typ {
	case evKey:
		return parseEvKey(sample.code, sample.value)
	case evRel:
		noteRel(sample.code, state)

		return noKind()
	case evSyn:
		return flushSyn(sample.code, state)
	default:
		return noKind()
	}
}

// parseKey maps an EV_KEY event to a keyboard click.
// Values 1 (down) and 2 (repeat) count; 0 (up) is ignored.
// Mouse buttons are excluded so the mouse parser owns them.
func parseKey(code uint16, value int32) (domain.EventType, bool) {
	if isMouseButton(code) {
		return noKind()
	}

	if !isKeyPress(value) {
		return noKind()
	}

	return domain.EventKeyboardClick, true
}

// parseMouseButton maps BTN_LEFT/BTN_RIGHT down (value 1) to click kinds.
func parseMouseButton(code uint16, value int32) (domain.EventType, bool) {
	if value != int32(evKey) {
		return noKind()
	}

	switch code {
	case btnLeft:
		return domain.EventLeftClick, true
	case btnRight:
		return domain.EventRightClick, true
	default:
		return noKind()
	}
}

// flushSyn emits one mouse move on SYN_REPORT when motion was pending.
func flushSyn(code uint16, state *mouseState) (domain.EventType, bool) {
	if code != evSyn || !state.pendingMove {
		return noKind()
	}

	state.pendingMove = false

	return domain.EventMouseMove, true
}
