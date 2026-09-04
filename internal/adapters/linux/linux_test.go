package linux

import (
	"testing"

	"github.com/gostafa/inputstats/internal/domain"
)

func TestParseKeyDownAndRepeat(t *testing.T) {
	for _, value := range []int32{1, 2} {
		kind, ok := parseKey(keyA, value)
		if !ok || kind != domain.EventKeyboardClick {
			t.Fatalf("value=%d: got ok=%v kind=%d, want keyboard", value, ok, kind)
		}
	}
}

func TestParseKeyUpIgnored(t *testing.T) {
	if _, ok := parseKey(keyA, 0); ok {
		t.Fatal("key up should be ignored")
	}
}

func TestParseKeyIgnoresMouseButtons(t *testing.T) {
	for _, code := range []uint16{btnLeft, btnRight, btnMiddle} {
		if _, ok := parseKey(code, 1); ok {
			t.Fatalf("mouse button %d should not count as keyboard", code)
		}
	}
}

func TestParseMouseButtonDownOnly(t *testing.T) {
	kind, ok := parseMouseButton(btnLeft, 1)
	if !ok || kind != domain.EventLeftClick {
		t.Fatalf("left down: ok=%v kind=%d", ok, kind)
	}
	kind, ok = parseMouseButton(btnRight, 1)
	if !ok || kind != domain.EventRightClick {
		t.Fatalf("right down: ok=%v kind=%d", ok, kind)
	}
	if _, ok := parseMouseButton(btnLeft, 0); ok {
		t.Fatal("button up should be ignored")
	}
	if _, ok := parseMouseButton(btnLeft, 2); ok {
		t.Fatal("button repeat should be ignored")
	}
}

func TestSYNCoalescesRelToOneMove(t *testing.T) {
	var st mouseState
	events := []struct {
		typ, code uint16
		value     int32
	}{
		{evRel, evSyn, 3},
		{evRel, evKey, -2},
		{evSyn, evSyn, 0},
	}

	var kinds []domain.EventType
	for _, e := range events {
		if kind, ok := parseEvdev(evdevSample{typ: e.typ, code: e.code, value: e.value}, &st); ok {
			kinds = append(kinds, kind)
		}
	}
	if len(kinds) != 1 || kinds[0] != domain.EventMouseMove {
		t.Fatalf("kinds=%v, want one mouse move", kinds)
	}
	if st.pendingMove {
		t.Fatal("pending move should clear after SYN_REPORT")
	}
}

func TestSYNWithoutRelEmitsNothing(t *testing.T) {
	var st mouseState
	if _, ok := parseEvdev(evdevSample{typ: evSyn, code: evSyn}, &st); ok {
		t.Fatal("SYN alone should not emit a move")
	}
}

func TestMultiplePacketsEachEmitOneMove(t *testing.T) {
	var st mouseState
	feed := func() (domain.EventType, bool) {
		parseEvdev(evdevSample{typ: evRel, code: evSyn, value: 1}, &st)
		return parseEvdev(evdevSample{typ: evSyn, code: evSyn}, &st)
	}
	for i := 0; i < 3; i++ {
		kind, ok := feed()
		if !ok || kind != domain.EventMouseMove {
			t.Fatalf("packet %d: ok=%v kind=%d", i, ok, kind)
		}
	}
}

func TestParseEvdevMixedKeyboardAndClick(t *testing.T) {
	var st mouseState
	kind, ok := parseEvdev(evdevSample{typ: evKey, code: keySpace, value: 1}, &st)
	if !ok || kind != domain.EventKeyboardClick {
		t.Fatalf("space: ok=%v kind=%d", ok, kind)
	}
	kind, ok = parseEvdev(evdevSample{typ: evKey, code: btnLeft, value: 1}, &st)
	if !ok || kind != domain.EventLeftClick {
		t.Fatalf("left: ok=%v kind=%d", ok, kind)
	}
}

func TestParseEvdevUnknownType(t *testing.T) {
	var st mouseState
	if _, ok := parseEvdev(evdevSample{typ: 0xffff, code: evSyn}, &st); ok {
		t.Fatal("unknown type should not emit")
	}
}
