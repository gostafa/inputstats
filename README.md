# inputstats

Cross-platform Go package that monitors global keyboard and mouse **activity** and emits aggregated counts on a fixed interval.

This is an activity statistics library, **not a keylogger**. The public API exposes only totals (`KeyboardClicks`, `MouseMoves`, `LeftClicks`, `RightClicks`) for each window — never key identities, pointer coordinates, or application context.

```go
func Start(ctx context.Context, interval time.Duration) (<-chan Stats, error)
```

```go
type Stats struct {
    From, To                                            time.Time
    KeyboardClicks, MouseMoves, LeftClicks, RightClicks uint64
}
```

`Start` returns a channel that emits one `Stats` value per interval until `ctx` is cancelled (or the platform adapter exits). Cancel the context to stop monitoring; the channel is then closed.

Supported platforms (compile-time build tags):

| OS      | Backend                                      |
| ------- | -------------------------------------------- |
| Linux   | `/dev/input/event*` via evdev                |
| Windows | Raw Input API (`RIDEV_INPUTSINK`)            |
| macOS   | `CGEventTap` (listen-only)                   |

## Install

```bash
go get github.com/gostafa/inputstats
```

## Usage

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gostafa/inputstats"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	statsCh, err := inputstats.Start(ctx, 5*time.Second)
	if err != nil {
		log.Fatal(err)
	}

	for stats := range statsCh {
		fmt.Printf(
			"keys=%d moves=%d left=%d right=%d\n",
			stats.KeyboardClicks,
			stats.MouseMoves,
			stats.LeftClicks,
			stats.RightClicks,
		)
	}
}
```

See [`examples/basic`](examples/basic) for a runnable copy.

Key-down and OS-paced autorepeat both increment `KeyboardClicks`. Key-up is ignored. Mouse movement is counted as events (not pixels); left/right button downs increment the click counters.

## Platform notes

### Linux

Requires **read** permission on:

```text
/dev/input/event*
```

These nodes are security-sensitive: anyone who can read them can observe raw input. Prefer least-privilege access:

- Add the process user to a dedicated group that owns the input devices (often `input` on distributions that ship that group).
- Or install a **udev** rule that grants the device nodes to a specific group, then run the service as a user in that group.
- For long-running daemons, run under systemd (or similar) with a dedicated user/group and the minimum device ACLs you need — not world-writable nodes.

**Do not** use insecure blanket permissions such as:

```bash
chmod 777 /dev/input/event*
```

If no usable keyboard/mouse devices are found, `Start` returns `ErrNoInputDevices`. If the process cannot open devices due to permissions, expect `ErrPermissionDenied`.

### Windows

Uses the native **Raw Input** API. Devices are registered with `RIDEV_INPUTSINK` so monitoring continues while the process is in the background (global/sink delivery to a message-only window).

**Process-level caveat:** Raw Input registration is process-wide — typically one HWND per device class (keyboard / mouse). Loading this package in a process that already registers Raw Input for the same classes, or registering another consumer afterward, can conflict. Prefer a single Raw Input owner per process.

No typed characters, virtual-key identities, or scan codes are exposed through the public API — only counts.

### macOS

Uses a listen-only **CGEventTap** attached to a run loop on a dedicated OS thread.

Grant **Input Monitoring** and/or **Accessibility** permission to the host binary (Terminal, your app bundle, etc.), depending on macOS version and how the process is launched. Without permission the tap may fail to enable; `Start` / the adapter surface that as `ErrPermissionDenied`.

The package records **counts only**. It does not expose typed text or key codes.

## Privacy and security

The public API must not (and does not) expose:

```text
key code
scan code
virtual key
Unicode character
typed text
active application contents
window title
clipboard
pointer coordinates
```

Keyboard activity is reported only as `KeyboardClicks++`. Mouse motion is reported only as a move count, not positions.

## Errors

| Error                     | When                                              |
| ------------------------- | ------------------------------------------------- |
| `ErrNilContext`           | `Start` called with a nil context                 |
| `ErrInvalidInterval`      | `interval` is not positive                        |
| `ErrPermissionDenied`     | OS denies input monitoring access                 |
| `ErrNoInputDevices`       | No usable keyboard/mouse devices found            |
| `ErrUnsupportedPlatform`  | Build target is not Linux, Windows, or macOS      |
