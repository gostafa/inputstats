//go:build darwin

package darwin

/*
#cgo LDFLAGS: -framework CoreGraphics -framework ApplicationServices -framework IOKit -framework CoreFoundation

#include <ApplicationServices/ApplicationServices.h>
#include <CoreFoundation/CoreFoundation.h>
#include <IOKit/hidsystem/IOHIDLib.h>
#include <stdint.h>

extern void goHandleCGEvent(uint32_t type);
extern void goReenableTap(void);

static CGEventRef inputstatsTapCallback(
	CGEventTapProxy proxy,
	CGEventType type,
	CGEventRef event,
	void *refcon)
{
	(void)proxy;
	(void)refcon;
	if (type == kCGEventTapDisabledByTimeout ||
		type == kCGEventTapDisabledByUserInput) {
		goReenableTap();
		return event;
	}
	goHandleCGEvent((uint32_t)type);
	return event;
}

static CFMachPortRef inputstatsCreateTap(void) {
	CGEventMask mask =
		CGEventMaskBit(kCGEventKeyDown) |
		CGEventMaskBit(kCGEventMouseMoved) |
		CGEventMaskBit(kCGEventLeftMouseDragged) |
		CGEventMaskBit(kCGEventRightMouseDragged) |
		CGEventMaskBit(kCGEventOtherMouseDragged) |
		CGEventMaskBit(kCGEventLeftMouseDown) |
		CGEventMaskBit(kCGEventRightMouseDown);

	return CGEventTapCreate(
		kCGSessionEventTap,
		kCGHeadInsertEventTap,
		kCGEventTapOptionListenOnly,
		mask,
		inputstatsTapCallback,
		NULL);
}

static int inputstatsHIDListenGranted(void) {
	IOHIDAccessType access = IOHIDCheckAccess(kIOHIDRequestTypeListenEvent);
	return access != kIOHIDAccessTypeDenied;
}

static void inputstatsEnableTap(CFMachPortRef tap) {
	CGEventTapEnable(tap, true);
}

static int inputstatsTapEnabled(CFMachPortRef tap) {
	return CGEventTapIsEnabled(tap) ? 1 : 0;
}

static void inputstatsDisableTap(CFMachPortRef tap) {
	CGEventTapEnable(tap, false);
}

static CFRunLoopSourceRef inputstatsCreateSource(CFMachPortRef tap) {
	return CFMachPortCreateRunLoopSource(kCFAllocatorDefault, tap, 0);
}

static void inputstatsAddSource(CFRunLoopRef rl, CFRunLoopSourceRef src) {
	CFRunLoopAddSource(rl, src, kCFRunLoopDefaultMode);
}

static void inputstatsRemoveSource(CFRunLoopRef rl, CFRunLoopSourceRef src) {
	CFRunLoopRemoveSource(rl, src, kCFRunLoopDefaultMode);
}
*/
import "C"

import (
	"context"
	"runtime"

	"github.com/gostafa/inputstats/internal/domain"
	"github.com/gostafa/inputstats/internal/ports"
)

// New returns a Darwin CGEventTap adapter. Returns ErrPermissionDenied when
// IOHIDCheckAccess reports listen access denied.
func New() (*Adapter, error) {
	if !hidListenGranted() {
		return nil, domain.ErrPermissionDenied
	}
	return &Adapter{}, nil
}

// Run installs a session CGEventTap on a locked OS thread and pumps CFRunLoop
// until ctx is cancelled. Returns ErrPermissionDenied if the tap stays disabled.
func (a *Adapter) Run(ctx context.Context, events chan<- domain.Event) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	tap, err := createEventTap()
	if err != nil {
		return domain.ErrPermissionDenied
	}
	defer tap.release()

	sink := &eventSink{ctx: ctx, events: events}
	activeSink.Store(sink)
	activeTap.Store(tap)
	defer activeSink.Store(nil)
	defer activeTap.Store(nil)

	tap.enable()
	if !tap.isEnabled() {
		return domain.ErrPermissionDenied
	}

	ready := make(chan struct{})
	stopCh := make(chan struct{})
	go func() {
		<-ready
		select {
		case <-ctx.Done():
		case <-stopCh:
			return
		}
		tap.disable()
		tap.stop()
	}()

	tap.attach(ready)
	tap.run()
	close(stopCh)

	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func hidListenGranted() bool {
	return C.inputstatsHIDListenGranted() != 0
}

func createEventTap() (*eventTap, error) {
	tap := C.inputstatsCreateTap()
	if tap == 0 {
		return nil, errTapCreate
	}

	source := C.inputstatsCreateSource(tap)
	if source == 0 {
		C.CFRelease(C.CFTypeRef(tap))
		return nil, errTapSource
	}

	return &eventTap{tap: tap, source: source}, nil
}

func (t *eventTap) enable() {
	C.inputstatsEnableTap(t.tap)
}

func (t *eventTap) isEnabled() bool {
	return C.inputstatsTapEnabled(t.tap) != 0
}

func (t *eventTap) disable() {
	if t.tap != 0 {
		C.inputstatsDisableTap(t.tap)
	}
}

func (t *eventTap) attach(ready chan<- struct{}) {
	t.loop = C.CFRunLoopGetCurrent()
	C.CFRetain(C.CFTypeRef(t.loop))
	C.inputstatsAddSource(t.loop, t.source)
	close(ready)
}

func (t *eventTap) run() {
	C.CFRunLoopRun()
}

func (t *eventTap) stop() {
	if t.loop != 0 {
		C.CFRunLoopStop(t.loop)
	}
}

func (t *eventTap) release() {
	if t.loop != 0 && t.source != 0 {
		C.inputstatsRemoveSource(t.loop, t.source)
	}
	if t.source != 0 {
		C.CFRelease(C.CFTypeRef(t.source))
		t.source = 0
	}
	if t.tap != 0 {
		C.CFRelease(C.CFTypeRef(t.tap))
		t.tap = 0
	}
	if t.loop != 0 {
		C.CFRelease(C.CFTypeRef(t.loop))
		t.loop = 0
	}
}

//export goHandleCGEvent
func goHandleCGEvent(typ C.uint32_t) {
	sink := activeSink.Load()
	if sink == nil {
		return
	}
	kind, ok := classifyCGEvent(uint32(typ))
	if !ok {
		return
	}
	ports.Deliver(sink.ctx, sink.events, domain.Event{Type: kind})
}

//export goReenableTap
func goReenableTap() {
	if t := activeTap.Load(); t != nil {
		t.enable()
	}
}

// classifyCGEvent maps a CGEventType to a counted event kind.
// Pure Go — no cgo — so unit tests run without Input Monitoring permission.
func classifyCGEvent(t uint32) (kind domain.EventType, ok bool) {
	switch t {
	case cgEventKeyDown:
		return keyDownKind()
	case cgEventMouseMoved, cgEventLeftMouseDragged, cgEventRightMouseDragged, cgEventOtherMouseDragged:
		return mouseMoveKind()
	case cgEventLeftMouseDown:
		return leftClickKind()
	case cgEventRightMouseDown:
		return rightClickKind()
	default:
		return 0, false
	}
}

func keyDownKind() (domain.EventType, bool) {
	return domain.EventKeyboardClick, true
}

func mouseMoveKind() (domain.EventType, bool) {
	return domain.EventMouseMove, true
}

func leftClickKind() (domain.EventType, bool) {
	return domain.EventLeftClick, true
}

func rightClickKind() (domain.EventType, bool) {
	return domain.EventRightClick, true
}
