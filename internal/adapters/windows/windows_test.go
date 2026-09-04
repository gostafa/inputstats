package windows

import (
	"testing"

	"github.com/gostafa/inputstats/internal/domain"
)

func TestParseKeyboardFlags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		flags uint16
		ok    bool
	}{
		{name: "make", flags: 0, ok: true},
		{name: "make_e0", flags: 0x02, ok: true},
		{name: "break", flags: riKeyBreak, ok: false},
		{name: "break_e0", flags: riKeyBreak | 0x02, ok: false},
		{name: "repeat_make", flags: 0, ok: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			kind, ok := parseKeyboardFlags(tt.flags)
			if ok != tt.ok {
				t.Fatalf("ok=%v want %v", ok, tt.ok)
			}
			if ok && kind != domain.EventKeyboardClick {
				t.Fatalf("kind=%d want %d", kind, domain.EventKeyboardClick)
			}
		})
	}
}

func TestParseMouseButtons(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		flags uint16
		want  []domain.EventType
	}{
		{name: "none", flags: 0, want: nil},
		{
			name:  "left_down",
			flags: riKeyBreak,
			want:  []domain.EventType{domain.EventLeftClick},
		},
		{
			name:  "right_down",
			flags: riMouseRightButtonDown,
			want:  []domain.EventType{domain.EventRightClick},
		},
		{
			name:  "both_down",
			flags: riKeyBreak | riMouseRightButtonDown,
			want:  []domain.EventType{domain.EventLeftClick, domain.EventRightClick},
		},
		{name: "left_up_ignored", flags: 0x0002, want: nil},
		{name: "right_up_ignored", flags: 0x0008, want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parseMouseButtons(tt.flags)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("got %v want %v", got, tt.want)
				}
			}
		})
	}
}

func TestParseMouseMove(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		x, y int32
		ok   bool
	}{
		{name: "still", x: 0, y: 0, ok: false},
		{name: "dx", x: 1, y: 0, ok: true},
		{name: "dy", x: 0, y: -3, ok: true},
		{name: "both", x: 2, y: 2, ok: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			kind, ok := parseMouseMove(tt.x, tt.y)
			if ok != tt.ok {
				t.Fatalf("ok=%v want %v", ok, tt.ok)
			}
			if ok && kind != domain.EventMouseMove {
				t.Fatalf("kind=%d want %d", kind, domain.EventMouseMove)
			}
		})
	}
}
