//go:build linux

package linux

import (
	"context"
	"encoding/binary"
	"errors"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/gostafa/inputstats/internal/domain"
	"github.com/holoplot/go-evdev"
	"golang.org/x/sys/unix"
)

func TestClassifyDeviceVariants(t *testing.T) {
	cases := []struct {
		name string
		dev  *fakeDev
		want deviceClass
	}{
		{
			"accelProp",
			&fakeDev{props: []evdev.EvProp{evdev.INPUT_PROP_ACCELEROMETER}},
			deviceClass{},
		},
		{"accelName", &fakeDev{name: "Laptop Accelerometer"}, deviceClass{}},
		{"lid", &fakeDev{name: "Lid Switch"}, deviceClass{}},
		{
			"lidKbd",
			&fakeDev{name: "Lid Keyboard", keys: []evdev.EvCode{evdev.KEY_A}},
			deviceClass{keyboard: true},
		},
		{"power", &fakeDev{name: "Power Button"}, deviceClass{}},
		{"sleep", &fakeDev{name: "Sleep Button"}, deviceClass{}},
		{
			"nameErr",
			&fakeDev{nameErr: errors.New("x"), keys: []evdev.EvCode{evdev.KEY_SPACE}},
			deviceClass{keyboard: true},
		},
		{"space", &fakeDev{keys: []evdev.EvCode{evdev.KEY_SPACE}}, deviceClass{keyboard: true}},
		{"relOnly", &fakeDev{rels: []evdev.EvCode{evdev.REL_Y}}, deviceClass{mouse: true}},
		{"btnOnly", &fakeDev{keys: []evdev.EvCode{evdev.BTN_LEFT}}, deviceClass{mouse: true}},
		{"empty", &fakeDev{}, deviceClass{}},
	}

	for _, tc := range cases {
		got := classifyDevice(tc.dev)
		if got != tc.want {
			t.Fatalf("%s: got %+v want %+v", tc.name, got, tc.want)
		}
	}
}

func TestIsAgain(t *testing.T) {
	for _, err := range []error{syscall.EAGAIN, syscall.EWOULDBLOCK, unix.EAGAIN, unix.EWOULDBLOCK} {
		if !isAgain(err) {
			t.Fatalf("expected again for %v", err)
		}
	}

	if isAgain(errors.New("other")) {
		t.Fatal("unexpected again")
	}
}

func TestMapListErr(t *testing.T) {
	if !errors.Is(mapListErr(os.ErrPermission), domain.ErrPermissionDenied) {
		t.Fatal("permission")
	}

	if !errors.Is(mapListErr(os.ErrNotExist), domain.ErrNoInputDevices) {
		t.Fatal("not exist")
	}

	if errors.Is(mapListErr(errors.New("x")), domain.ErrNoInputDevices) {
		t.Fatal("other")
	}
}

func TestProbeDevicesPaths(t *testing.T) {
	deps := testHookSet()
	stubGlob(deps, nil, errors.New("glob"))
	if probeDevices(deps) == nil {
		t.Fatal("glob err")
	}

	stubGlob(deps, nil, nil)
	if !errors.Is(probeDevices(deps), domain.ErrNoInputDevices) {
		t.Fatal("empty")
	}

	stubGlob(deps, []string{"/dev/input/event0"}, nil)
	stubOpen(deps, map[string]*fakeDev{"/dev/input/event0": {}})
	if !errors.Is(probeDevices(deps), domain.ErrNoInputDevices) {
		t.Fatal("unusable")
	}

	stubOpen(deps, map[string]*fakeDev{})
	deps.openDevice = func(string) (evdevDevice, error) { return nil, os.ErrPermission }
	if !errors.Is(probeDevices(deps), domain.ErrPermissionDenied) {
		t.Fatal("permission")
	}

	stubOpen(deps, map[string]*fakeDev{"/dev/input/event0": kbdFake()})
	if probeDevices(deps) != nil {
		t.Fatal("usable")
	}
}

func TestProbeOneSkipped(t *testing.T) {
	deps := testHookSet()
	var stats probeStats

	stubOpen(deps, map[string]*fakeDev{"p": {name: "Power Button"}})
	probeOne(deps, "p", &stats)
	deps.openDevice = func(string) (evdevDevice, error) { return nil, errDeviceSkipped }
	probeOne(deps, "q", &stats)

	if stats.usable != 0 || stats.sawPermission {
		t.Fatalf("stats=%+v", stats)
	}
}

func TestNoteProbeErr(t *testing.T) {
	var stats probeStats

	noteProbeErr(errDeviceSkipped, &stats)
	noteProbeErr(os.ErrPermission, &stats)
	noteProbeErr(errors.New("x"), &stats)

	if !stats.sawPermission || stats.usable != 0 {
		t.Fatalf("stats=%+v", stats)
	}
}

func TestNewOKAndFail(t *testing.T) {
	deps := testHookSet()
	stubGlob(deps, nil, nil)
	if _, err := newAdapter(deps); err == nil {
		t.Fatal("expected fail")
	}

	stubGlob(deps, []string{"/dev/input/event0"}, nil)
	stubOpen(deps, map[string]*fakeDev{"/dev/input/event0": kbdFake()})

	port, err := newAdapter(deps)
	if err != nil || port == nil {
		t.Fatalf("port=%v err=%v", port, err)
	}
}

func TestTestGrant(t *testing.T) {
	t.Setenv(grantEnvName, grantEnvDeny)
	if _, err := New(); err == nil {
		t.Fatal("deny")
	}

	t.Setenv(grantEnvName, grantEnvAllow)
	if port, err := New(); err != nil || port == nil {
		t.Fatal("allow")
	}

	t.Setenv(grantEnvName, "")
	if applyTestGrant() != nil || testGrantAllow() {
		t.Fatal("default")
	}
}

func TestDoneErr(t *testing.T) {
	if err := doneErr(context.Background()); err != nil {
		t.Fatalf("background: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := doneErr(ctx); err == nil {
		t.Fatal("expected cancel")
	}
}

func TestRunCancel(t *testing.T) {
	deps := testHookSet()
	setImmediateTimers(deps)

	stubGlob(deps, []string{"/dev/input/event0"}, nil)
	dev := kbdFake()
	dev.steps = []readStep{{err: syscall.EAGAIN}}
	stubOpen(deps, map[string]*fakeDev{"/dev/input/event0": dev})

	deps.inotifyInit1 = func(int) (int, error) { return 0, errors.New("no inotify") }

	ctx, cancel := context.WithCancel(context.Background())
	events := make(chan domain.Event, 4)

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err := runSession(ctx, newSession(events, deps))
	if err == nil {
		t.Fatal("expected cancel err")
	}
}

func TestOpenExistingListErr(t *testing.T) {
	deps := testHookSet()
	stubGlob(deps, nil, errors.New("glob"))

	err := openExisting(context.Background(), newSession(make(chan domain.Event), deps))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunSessionListErr(t *testing.T) {
	deps := testHookSet()
	stubGlob(deps, nil, errors.New("glob"))

	err := runSession(context.Background(), newSession(make(chan domain.Event), deps))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDefaultOpenDevice(t *testing.T) {
	deps := defaultHookSet()
	_, err := deps.openDevice("/inputstats-missing-device")
	if err == nil {
		t.Fatal("expected open error")
	}
}

func TestTryOpenBranches(t *testing.T) {
	deps := testHookSet()
	setImmediateTimers(deps)

	sess := newSession(make(chan domain.Event, 4), deps)
	path := "/dev/input/event9"
	dev := kbdFake()
	stubOpen(deps, map[string]*fakeDev{path: dev})
	sess.deps = deps

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tryOpen(ctx, path, sess)
	tryOpen(ctx, path, sess)

	fail := kbdFake()
	fail.nonBlock = errors.New("nb")
	stubOpen(deps, map[string]*fakeDev{"/dev/input/event8": fail})
	sess.deps = deps
	tryOpen(ctx, "/dev/input/event8", sess)

	stubOpen(deps, map[string]*fakeDev{})
	sess.deps = deps
	tryOpen(ctx, "/missing", sess)

	dup := kbdFake()
	stubOpen(deps, map[string]*fakeDev{"/dev/input/event7": dup})
	sess.deps = deps
	sess.reg.opened["/dev/input/event7"] = (&fakeDev{})
	tryOpen(ctx, "/dev/input/event7", sess)

	canceled, stop := context.WithCancel(context.Background())
	stop()

	late := kbdFake()
	stubOpen(deps, map[string]*fakeDev{"/dev/input/event6": late})
	sess.deps = deps
	tryOpen(canceled, "/dev/input/event6", sess)

	cancel()
	sess.reg.wg.Wait()
}

func TestClaimPathDuplicate(t *testing.T) {
	reg := &registry{opened: map[string]evdevHandle{}}
	dev := (&fakeDev{})
	claim := &deviceClaim{reg: reg, dev: dev}

	if !claimPath("p", claim) {
		t.Fatal("first claim")
	}

	if claimPath("p", &deviceClaim{reg: reg, dev: (&fakeDev{})}) {
		t.Fatal("duplicate")
	}

	unclaimPath("p", claim)
}

func TestRegisterDeviceDuplicate(t *testing.T) {
	reg := &registry{opened: map[string]evdevHandle{"p": (&fakeDev{})}}
	dev := (&fakeDev{})

	ok := registerDevice(context.Background(), "p", &deviceClaim{reg: reg, dev: dev})
	if ok || dev.closed == 0 {
		t.Fatal("expected close on duplicate")
	}
}

func TestCloseHandleError(t *testing.T) {
	closeHandle((&fakeDev{closeErr: errors.New("c")}))
}

func TestUnregisterMismatch(t *testing.T) {
	a, b := (&fakeDev{}), (&fakeDev{})
	reg := &registry{opened: map[string]evdevHandle{"p": a}}

	unregister(reg, "p", b)
	if reg.opened["p"] != a {
		t.Fatal("should keep")
	}

	unregister(reg, "p", a)
	if reg.opened["p"] != nil {
		t.Fatal("should delete")
	}
}

func TestWaitAgainCancel(t *testing.T) {
	deps := testHookSet()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	deps.afterDelay = func(time.Duration) <-chan time.Time {
		return make(chan time.Time)
	}

	if waitAgain(ctx, deps, time.Millisecond) {
		t.Fatal("expected false")
	}
}

func TestReadDeviceDispatch(t *testing.T) {
	deps := testHookSet()
	setImmediateTimers(deps)

	events := make(chan domain.Event, 16)
	sess := newSession(events, deps)
	ctx, cancel := context.WithCancel(context.Background())

	kbd := kbdFake()
	kbd.steps = []readStep{
		{event: ev(evKey, keyA, 1)},
		{err: syscall.EAGAIN},
		{err: errors.New("stop")},
	}

	path := "/dev/input/event0"
	sess.reg.opened[path] = kbd
	startReader(ctx, &readJob{sess: sess, dev: kbd, path: path, class: deviceClass{keyboard: true}})

	mouse := mouseFake()
	mouse.steps = []readStep{
		{event: ev(evRel, evSyn, 1)},
		{event: ev(evSyn, evSyn, 0)},
		{event: ev(evKey, btnLeft, 1)},
		{event: ev(evKey, btnRight, 1)},
		{event: ev(0xff, 0, 0)},
		{err: errors.New("stop")},
	}

	mpath := "/dev/input/event1"
	sess.reg.opened[mpath] = mouse
	startReader(ctx, &readJob{sess: sess, dev: mouse, path: mpath, class: deviceClass{mouse: true}})

	combo := comboFake()
	combo.steps = []readStep{
		{event: ev(evKey, keySpace, 1)},
		{event: ev(evKey, btnLeft, 1)},
		{event: ev(evRel, evSyn, 1)},
		{event: ev(evSyn, evSyn, 0)},
		{err: errors.New("stop")},
	}

	cpath := "/dev/input/event2"
	sess.reg.opened[cpath] = combo
	startReader(ctx, &readJob{
		sess: sess, dev: combo, path: cpath,
		class: deviceClass{keyboard: true, mouse: true},
	})

	deadline := time.After(time.Second)
	got := 0

	for got < 6 {
		select {
		case <-events:
			got++
		case <-deadline:
			cancel()
			sess.reg.wg.Wait()
			t.Fatalf("events=%d", got)
		}
	}

	cancel()
	sess.reg.wg.Wait()
}

func TestDeliverSingleEmpty(t *testing.T) {
	deliverSingle(context.Background(), &dispatchJob{
		event:  ev(evKey, keyA, 1),
		state:  &mouseState{},
		events: make(chan domain.Event, 1),
		class:  deviceClass{},
	})
}

func TestDeliverSingleKeyboard(t *testing.T) {
	events := make(chan domain.Event, 1)

	deliverSingle(context.Background(), &dispatchJob{
		event:  ev(evKey, keyA, 1),
		state:  &mouseState{},
		events: events,
		class:  deviceClass{keyboard: true},
	})

	select {
	case got := <-events:
		if got.Type != domain.EventKeyboardClick {
			t.Fatalf("kind=%d", got.Type)
		}
	default:
		t.Fatal("expected keyboard event")
	}
}

func TestHandleKeyIgnored(t *testing.T) {
	handleKeyEvent(
		context.Background(),
		make(chan domain.Event, 1),
		keySample{code: btnLeft, value: 1},
	)
}

func TestEncodeInotifyHelpers(t *testing.T) {
	if parseInotifyNames(nil) != nil {
		t.Fatal("empty")
	}

	buf := inotifyBuf("event0")
	names := parseInotifyNames(buf)
	if len(names) != 1 || names[0] != "event0" {
		t.Fatalf("names=%v", names)
	}

	empty := inotifyBuf("\x00\x00\x00\x00")
	if parseInotifyNames(empty) != nil {
		t.Fatal("null name")
	}

	short := make([]byte, 8)
	if parseInotifyNames(short) != nil {
		t.Fatal("short")
	}

	badLen := make([]byte, 20)
	binary.LittleEndian.PutUint32(badLen[12:16], 100)
	if parseInotifyNames(badLen) != nil {
		t.Fatal("oversize")
	}
}

func inotifyBuf(name string) []byte {
	raw := append([]byte(name), 0)
	for len(raw)%4 != 0 {
		raw = append(raw, 0)
	}

	buf := make([]byte, unixSizeofInotifyEvent+len(raw))
	binary.LittleEndian.PutUint32(buf[inotifyLenOff:inotifyLenOff+inotifyLenSize], uint32(len(raw)))
	copy(buf[unixSizeofInotifyEvent:], raw)

	return buf
}

func TestHotplugWatchFail(t *testing.T) {
	deps := testHookSet()
	deps.inotifyInit1 = func(int) (int, error) { return 0, errors.New("fail") }

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		watchHotplug(ctx, newSession(make(chan domain.Event), deps))
		close(done)
	}()

	cancel()
	<-done
}

func TestHotplugInitAddFail(t *testing.T) {
	deps := testHookSet()
	deps.inotifyInit1 = func(int) (int, error) { return 3, nil }
	deps.inotifyAddWatch = func(int, string, uint32) (int, error) {
		return 0, errors.New("add")
	}

	closed := false
	deps.unixClose = func(int) error {
		closed = true

		return errors.New("c")
	}

	if _, err := initInotify(deps); err == nil || !closed {
		t.Fatal("expected add fail")
	}

	deps.inotifyAddWatch = func(int, string, uint32) (int, error) { return 0, nil }
	if _, err := initInotify(deps); err == nil {
		t.Fatal("watch desc")
	}

	deps.inotifyAddWatch = func(int, string, uint32) (int, error) { return 1, nil }
	deps.unixClose = func(int) error { return nil }

	fd, err := initInotify(deps)
	if err != nil || fd != 3 {
		t.Fatalf("fd=%d err=%v", fd, err)
	}
}

func TestHotplugPump(t *testing.T) {
	deps := testHookSet()
	setImmediateTimers(deps)

	opened := make(chan string, 2)
	stubOpen(deps, map[string]*fakeDev{})
	deps.openDevice = func(path string) (evdevDevice, error) {
		opened <- path

		return kbdFake(), nil
	}

	reads := 0
	deps.unixRead = func(_ int, buf []byte) (int, error) {
		reads++
		if reads == 1 {
			raw := inotifyBuf("event3")
			copy(buf, raw)

			return len(raw), nil
		}

		if reads == 2 {
			raw := inotifyBuf("mouse0")
			copy(buf, raw)

			return len(raw), nil
		}

		return 0, errors.New("stop")
	}

	deps.unixClose = func(int) error { return nil }
	deps.inotifyInit1 = func(int) (int, error) { return 5, nil }
	deps.inotifyAddWatch = func(int, string, uint32) (int, error) { return 1, nil }

	ctx, cancel := context.WithCancel(context.Background())
	sess := newSession(make(chan domain.Event, 4), deps)

	done := make(chan struct{})
	go func() {
		watchHotplug(ctx, sess)
		close(done)
	}()

	select {
	case <-opened:
	case <-time.After(time.Second):
		t.Fatal("open timeout")
	}

	cancel()
	<-done
	sess.reg.wg.Wait()
}

func TestHandleHotplugErr(t *testing.T) {
	deps := testHookSet()
	setImmediateTimers(deps)

	if handleHotplugErr(context.Background(), deps, errors.New("x")) {
		t.Fatal("non-again")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if !handleHotplugErr(ctx, deps, syscall.EAGAIN) {
		t.Fatal("again")
	}
}

func TestScheduleRetryCanceled(t *testing.T) {
	deps := testHookSet()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	called := false
	deps.afterFunc = func(time.Duration, func()) *time.Timer {
		called = true

		return time.NewTimer(time.Hour)
	}

	scheduleRetry(ctx, newSession(make(chan domain.Event), deps), "/x")
	if called {
		t.Fatal("should skip")
	}
}

func TestOpenHotplugName(t *testing.T) {
	deps := testHookSet()
	setImmediateTimers(deps)

	sess := newSession(make(chan domain.Event, 1), deps)
	openHotplugName(context.Background(), sess, "mouse0")

	stubOpen(deps, map[string]*fakeDev{"/dev/input/event4": kbdFake()})
	sess.deps = deps
	ctx, cancel := context.WithCancel(context.Background())
	openHotplugName(ctx, sess, "event4")
	cancel()
	sess.reg.wg.Wait()
}

func TestCloseWatchOK(t *testing.T) {
	deps := testHookSet()
	deps.unixClose = func(int) error { return nil }
	closeWatch(deps, 1)
}

func TestCloseAllSnapshot(t *testing.T) {
	dev := (&fakeDev{})
	reg := &registry{opened: map[string]evdevHandle{"p": dev}}

	closeAll(reg)
	if dev.closed == 0 {
		t.Fatal("expected close")
	}
}

func TestDeliverMouseIgnored(t *testing.T) {
	events := make(chan domain.Event, 1)

	deliverMouseButton(context.Background(), events, mouseSample{
		code: btnMiddle, value: 1,
	})
	deliverMouseSyn(context.Background(), events, mouseSample{
		state: &mouseState{}, code: evKey, value: 0,
	})

	select {
	case <-events:
		t.Fatal("unexpected event")
	default:
	}
}

func TestFillInotifyEarlyStop(t *testing.T) {
	good := inotifyBuf("event0")
	buf := append(good, make([]byte, unixSizeofInotifyEvent)...)
	names := make([]string, 1)
	fillInotifyNames(buf, names)

	if names[0] != "event0" {
		t.Fatalf("names=%v", names)
	}
}

func TestPumpHotplugCancel(t *testing.T) {
	deps := testHookSet()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	deps.unixRead = func(int, []byte) (int, error) {
		t.Fatal("should not read")

		return 0, nil
	}

	pumpHotplug(ctx, newSession(make(chan domain.Event), deps), 1)
}

func TestPumpHotplugReadStop(t *testing.T) {
	deps := testHookSet()
	deps.unixRead = func(int, []byte) (int, error) {
		return 0, errors.New("stop")
	}

	pumpHotplug(context.Background(), newSession(make(chan domain.Event), deps), 1)
}

func TestReadHotplugErr(t *testing.T) {
	deps := testHookSet()
	deps.unixRead = func(int, []byte) (int, error) {
		return 0, errors.New("stop")
	}

	ok := readHotplug(context.Background(), &hotplugRead{
		sess:    newSession(make(chan domain.Event), deps),
		buf:     make([]byte, 1),
		watchFD: 1,
	})
	if ok {
		t.Fatal("expected stop")
	}
}

func TestStoreNameEmpty(t *testing.T) {
	names := make([]string, 2)
	if storeName(names, 0, emptyName) != 0 {
		t.Fatal("empty")
	}
}

func TestCodeSetCapable(t *testing.T) {
	set := codeSet([]evdev.EvCode{evdev.KEY_A, evdev.KEY_Z})
	if !set[evdev.KEY_A] || !set[evdev.KEY_Z] {
		t.Fatal(set)
	}
}

func TestAdapterRun(t *testing.T) {
	t.Setenv(grantEnvName, grantEnvAllow)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := (&Adapter{}).Run(ctx, make(chan domain.Event))
	if err == nil {
		t.Fatal("expected error")
	}
}
