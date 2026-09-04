//go:build darwin

package darwin

/*
#include <CoreFoundation/CoreFoundation.h>
*/
import "C"

import (
	"context"

	"github.com/gostafa/inputstats/internal/domain"
)

// Adapter monitors keyboard and mouse via a listen-only CGEventTap.
type Adapter struct{}

type eventSink struct {
	ctx    context.Context
	events chan<- domain.Event
}

type eventTap struct {
	tap    C.CFMachPortRef
	source C.CFRunLoopSourceRef
	loop   C.CFRunLoopRef
}
