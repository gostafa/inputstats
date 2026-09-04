package linux

// Evdev type/code constants (linux/input-event-codes.h) for pure parsers.
const (
	evSyn = 0x00
	evKey = 0x01
	evRel = 0x02

	synReport = 0

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

	relX = 0x00
	relY = 0x01
)
