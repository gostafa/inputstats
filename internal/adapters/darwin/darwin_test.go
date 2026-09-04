//go:build darwin

package darwin

import (
	"testing"

	"github.com/gostafa/inputstats/internal/domain"
)

func TestClassifyCGEvent(t *testing.T) {
	tests := []struct {
		name string
		typ  uint32
		want domain.EventType
		ok   bool
	}{
		{"keyDown", cgEventKeyDown, domain.EventKeyboardClick, true},
		{"mouseMoved", cgEventMouseMoved, domain.EventMouseMove, true},
		{"leftDragged", cgEventLeftMouseDragged, domain.EventMouseMove, true},
		{"rightDragged", cgEventRightMouseDragged, domain.EventMouseMove, true},
		{"otherDragged", cgEventOtherMouseDragged, domain.EventMouseMove, true},
		{"leftDown", cgEventLeftMouseDown, domain.EventLeftClick, true},
		{"rightDown", cgEventRightMouseDown, domain.EventRightClick, true},
		{"keyUpIgnored", 11, 0, false},
		{"leftUpIgnored", 2, 0, false},
		{"nullIgnored", 0, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kind, ok := classifyCGEvent(tt.typ)
			if ok != tt.ok {
				t.Fatalf("ok=%v, want %v", ok, tt.ok)
			}
			if !ok {
				return
			}
			if kind != tt.want {
				t.Fatalf("kind=%v, want %v", kind, tt.want)
			}
		})
	}
}
