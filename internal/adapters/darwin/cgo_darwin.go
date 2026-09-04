//go:build darwin

package darwin

/*
#cgo LDFLAGS: -framework CoreGraphics -framework ApplicationServices -framework IOKit -framework CoreFoundation

#include <ApplicationServices/ApplicationServices.h>
#include <CoreFoundation/CoreFoundation.h>
#include <IOKit/hidsystem/IOHIDLib.h>
#include <stdint.h>

extern void goHandleCGEvent(uint32_t type, uintptr_t handle);
extern void goReenableTap(uintptr_t handle);

static CGEventRef inputstatsTapCallback(
	CGEventTapProxy proxy,
	CGEventType type,
	CGEventRef event,
	void *refcon)
{
	(void)proxy;
	uintptr_t handle = (uintptr_t)refcon;
	if (type == kCGEventTapDisabledByTimeout ||
		type == kCGEventTapDisabledByUserInput) {
		goReenableTap(handle);

		return event;
	}
	goHandleCGEvent((uint32_t)type, handle);

	return event;
}

static CFMachPortRef inputstatsCreateTap(uintptr_t handle) {
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
		(void *)handle);
}

static int inputstatsHIDListenGranted(void) {
	IOHIDAccessType access = IOHIDCheckAccess(kIOHIDRequestTypeListenEvent);

	return access != kIOHIDAccessTypeDenied;
}

static void inputstatsEnableTap(CFMachPortRef tap) {
	if (tap != NULL) {
		CGEventTapEnable(tap, true);
	}
}

static int inputstatsTapEnabled(uintptr_t tap) {
	CFMachPortRef port = (CFMachPortRef)tap;
	if (port == NULL) {

		return 0;
	}

	return CGEventTapIsEnabled(port) ? 1 : 0;
}

static void inputstatsDisableTap(CFMachPortRef tap) {
	if (tap != NULL) {
		CGEventTapEnable(tap, false);
	}
}

static CFRunLoopSourceRef inputstatsCreateSource(uintptr_t tap) {
	CFMachPortRef port = (CFMachPortRef)tap;
	if (port == NULL) {

		return NULL;
	}

	return CFMachPortCreateRunLoopSource(kCFAllocatorDefault, port, 0);
}

static void inputstatsAddSource(CFRunLoopRef rl, CFRunLoopSourceRef src) {
	if (rl != NULL && src != NULL) {
		CFRunLoopAddSource(rl, src, kCFRunLoopDefaultMode);
	}
}

static void inputstatsRemoveSource(CFRunLoopRef rl, CFRunLoopSourceRef src) {
	if (rl != NULL && src != NULL) {
		CFRunLoopRemoveSource(rl, src, kCFRunLoopDefaultMode);
	}
}

static CFRunLoopRef inputstatsCurrentLoop(void) {

	return CFRunLoopGetCurrent();
}

static void inputstatsRunLoop(void) {
	CFRunLoopRun();
}

static void inputstatsStopLoop(CFRunLoopRef rl) {
	if (rl != NULL) {
		CFRunLoopStop(rl);
	}
}

static void inputstatsRetain(CFTypeRef ref) {
	if (ref != NULL) {
		CFRetain(ref);
	}
}

static void inputstatsRelease(CFTypeRef ref) {
	if (ref != NULL) {
		CFRelease(ref);
	}
}

static CFMachPortRef inputstatsPortFrom(uintptr_t ptr) {

	return (CFMachPortRef)ptr;
}

static CFRunLoopSourceRef inputstatsSourceFrom(uintptr_t ptr) {

	return (CFRunLoopSourceRef)ptr;
}
*/
import (
	"C"
)

type (
	eventTap struct {
		tap    C.CFMachPortRef
		source C.CFRunLoopSourceRef
		loop   C.CFRunLoopRef
		stub   bool
	}
)

func acceptSource(tap C.CFMachPortRef, source C.CFRunLoopSourceRef) (*eventTap, error) {
	if source == cfRefNull {
		C.inputstatsRelease(C.CFTypeRef(tap))

		return nil, errTapSource
	}

	return &eventTap{tap: tap, source: source}, nil
}

func cCreateSource(tap C.CFMachPortRef) C.CFRunLoopSourceRef {
	return C.inputstatsCreateSource(C.uintptr_t(tap))
}

func cCreateSourcePtr(tap uintptr) uintptr {
	port := C.inputstatsPortFrom(C.uintptr_t(tap))

	return cSourceAddr(cCreateSource(port))
}

func cCreateTap(handle uintptr) C.CFMachPortRef {
	return C.inputstatsCreateTap(C.uintptr_t(handle))
}

func cCreateTapPtr(handle uintptr) uintptr {
	return cPortAddr(cCreateTap(handle))
}

func cPortAddr(tap C.CFMachPortRef) uintptr {
	return uintptr(tap)
}

func cReleasePort(tap uintptr) {
	C.inputstatsRelease(C.CFTypeRef(C.inputstatsPortFrom(C.uintptr_t(tap))))
}

func cReleaseSource(source uintptr) {
	C.inputstatsRelease(C.CFTypeRef(C.inputstatsSourceFrom(C.uintptr_t(source))))
}

func cSourceAddr(source C.CFRunLoopSourceRef) uintptr {
	return uintptr(source)
}

func cTapFlag(tap C.CFMachPortRef) int {
	return int(C.inputstatsTapEnabled(C.uintptr_t(tap)))
}

func hidFlag() int {
	return int(C.inputstatsHIDListenGranted())
}

func newEventTap(tap, source uintptr) *eventTap {
	return &eventTap{
		tap:    C.inputstatsPortFrom(C.uintptr_t(tap)),
		source: C.inputstatsSourceFrom(C.uintptr_t(source)),
	}
}

//export goHandleCGEvent
func goHandleCGEvent(typ C.uint32_t, handle C.uintptr_t) {
	emitCGEvent(uint32(typ), uintptr(handle))
}

//export goReenableTap
func goReenableTap(handle C.uintptr_t) {
	reenableTap(uintptr(handle))
}

func (t *eventTap) attach(ready chan<- struct{}) {
	t.loop = C.inputstatsCurrentLoop()
	C.inputstatsRetain(C.CFTypeRef(t.loop))
	C.inputstatsAddSource(t.loop, t.source)
	close(ready)
}

func (t *eventTap) detachSource() {
	C.inputstatsRemoveSource(t.loop, t.source)
}

func (t *eventTap) disable() {
	C.inputstatsDisableTap(t.tap)
}

func (t *eventTap) enable() {
	C.inputstatsEnableTap(t.tap)
}

func (t *eventTap) isEnabled() bool {
	return t.stub || cTapFlag(t.tap) != cfRefNull
}

func (t *eventTap) release() {
	t.detachSource()
	t.releaseSource()
	t.releaseTapPort()
	t.releaseLoop()
}

func (t *eventTap) releaseLoop() {
	C.inputstatsRelease(C.CFTypeRef(t.loop))
	t.loop = cfRefNull
}

func (t *eventTap) releaseSource() {
	C.inputstatsRelease(C.CFTypeRef(t.source))
	t.source = cfRefNull
}

func (t *eventTap) releaseTapPort() {
	C.inputstatsRelease(C.CFTypeRef(t.tap))
	t.tap = cfRefNull
}

func (t *eventTap) run() {
	if t.stub {
		return
	}

	C.inputstatsRunLoop()
}

func (t *eventTap) stop() {
	C.inputstatsStopLoop(t.loop)
}
