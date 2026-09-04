package linux

import "github.com/gostafa/inputstats/internal/domain"

// isMouseButton reports codes handled exclusively by the mouse parser.
func isMouseButton(code uint16) bool {
	switch code {
	case btnLeft, btnRight, btnMiddle, btnSide, btnExtra, btnForward, btnBack, btnTask:
		return true
	default:
		return false
	}
}

// parseKey maps an EV_KEY event to a keyboard click.
// Values 1 (down) and 2 (repeat) count; 0 (up) is ignored.
// Mouse buttons are excluded so the mouse parser owns them.
func parseKey(code uint16, value int32) (kind domain.EventType, ok bool) {
	if isMouseButton(code) {
		return 0, false
	}
	if value != 1 && value != 2 {
		return 0, false
	}
	return domain.EventKeyboardClick, true
}

// parseMouseButton maps BTN_LEFT/BTN_RIGHT down (value 1) to click kinds.
func parseMouseButton(code uint16, value int32) (kind domain.EventType, ok bool) {
	if value != 1 {
		return 0, false
	}
	switch code {
	case btnLeft:
		return domain.EventLeftClick, true
	case btnRight:
		return domain.EventRightClick, true
	default:
		return 0, false
	}
}

// noteRel marks pending mouse motion for REL_X / REL_Y.
func noteRel(code uint16, st *mouseState) {
	if code == relX || code == relY {
		st.pendingMove = true
	}
}

// flushSyn emits one mouse move on SYN_REPORT when motion was pending.
func flushSyn(code uint16, st *mouseState) (kind domain.EventType, ok bool) {
	if code != synReport || !st.pendingMove {
		return 0, false
	}
	st.pendingMove = false
	return domain.EventMouseMove, true
}

// parseEvdev applies one input event against mouse coalesce state and returns
// a counted activity kind when appropriate.
func parseEvdev(typ, code uint16, value int32, st *mouseState) (kind domain.EventType, ok bool) {
	switch typ {
	case evKey:
		if k, ok := parseMouseButton(code, value); ok {
			return k, true
		}
		return parseKey(code, value)
	case evRel:
		noteRel(code, st)
		return 0, false
	case evSyn:
		return flushSyn(code, st)
	default:
		return 0, false
	}
}
