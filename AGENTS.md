## Learned User Preferences
- Fix golangci-lint findings by changing code; do not use `nolint` and do not modify `.golangci.yml`.
- Prefer ports-and-adapters (onion) architecture with OS-independent core.
- Keep packages split into `consts.go`, `funcs.go`, `vars.go`, `types.go`, `doc.go`, and `PACKAGE_NAME_test.go` when restructuring.

## Learned Workspace Facts
- Go module `github.com/gostafa/inputstats` is a cross-platform keyboard/mouse activity stats library (not a keylogger); public API is interval `Stats` totals only.
- Layout uses `internal/{domain,ports,app,adapters}` with platform adapters under `adapters/{darwin,linux,windows}` (evdev, Raw Input, CGEventTap/CGO).
- Lint is enforced via a custom golangci-lint setup in this repo (`custom-golangci-lint` / `.lint`); treat that config as off-limits unless the user explicitly asks to change it.
