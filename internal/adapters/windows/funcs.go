package windows

import "github.com/gostafa/inputstats/internal/domain"

// parseKeyboardFlags reports a keyboard click when the key is not a break (release).
// Make/VKey are intentionally unused — counts only.
func parseKeyboardFlags(flags uint16) (kind domain.EventType, ok bool) {
	if flags&riKeyBreak != 0 {
		return 0, false
	}
	return domain.EventKeyboardClick, true
}

// parseMouseButtons returns left/right click kinds for button-down flags only.
func parseMouseButtons(buttonFlags uint16) []domain.EventType {
	var out []domain.EventType
	if buttonFlags&riMouseLeftButtonDown != 0 {
		out = append(out, domain.EventLeftClick)
	}
	if buttonFlags&riMouseRightButtonDown != 0 {
		out = append(out, domain.EventRightClick)
	}
	return out
}

// parseMouseMove reports one move when either axis delta is non-zero.
func parseMouseMove(lastX, lastY int32) (kind domain.EventType, ok bool) {
	if lastX|lastY == 0 {
		return 0, false
	}
	return domain.EventMouseMove, true
}
