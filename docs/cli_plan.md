# Slim Meshtastic CLI Plan (Cobra)

## Goal
Build a slimmed-down Meshtastic CLI focused on the most useful day-to-day workflows, using `github.com/spf13/cobra` for command/flag/config handling.

## Scope (V1)
- Connection over serial only (`--port`), default `/dev/cu.usbmodem101`.
- Core commands:
  - `info`: fetch and print radio info.
  - `send text`: send a text message.
  - `listen`: stream incoming packets/events/telemetry (migrated from `cmd/rxlisten`).
  - `set owner`: set device owner long/short name.
  - `set modem`: set modem preset (`lf|ls|vls|ms|mf|sl|sf|lm`).
  - `set location`: set fixed location payload.
  - `factory-reset`: send factory reset command.

## Non-goals (V1)
- BLE/TCP transports.
- Full parity with the official Meshtastic Python CLI.
- Config file import/export and full channel management.
- OTA/file-transfer workflows.

## CLI UX
- Binary name: `chirp`.
- Global flags (persistent):
  - `--port string` (default `/dev/cu.usbmodem101`)
  - `--timeout duration` (default `2s`)
  - `--json` (machine-readable output for non-listen commands)
  - `--verbose` (debug logs)
- Exit codes:
  - `0` success
  - `1` runtime/transport/protocol error
  - `2` invalid user input/flags

## Proposed Layout
```
cmd/chirp/main.go
internal/cli/root.go
internal/cli/context.go
internal/cli/cmd_info.go
internal/cli/cmd_send.go
internal/cli/cmd_send_text.go
internal/cli/cmd_listen.go
internal/cli/cmd_set.go
internal/cli/cmd_set_owner.go
internal/cli/cmd_set_modem.go
internal/cli/cmd_set_location.go
internal/cli/cmd_factory_reset.go
```

## Migration: `rxlisten` -> `chirp listen`
1. Move formatter/logging logic from `cmd/rxlisten/main.go` into `internal/cli/cmd_listen.go`.
2. Preserve current output labels (`[PKT]`, `[TEL]`, `[EVT]`, `[ERR]`, `[IDLE]`) for continuity.
3. Add flags:
   - `--idle-log duration` (keep existing behavior)
   - `--no-telemetry` (optional filter)
   - `--no-events` (optional filter)
   - `--no-packets` (optional filter)
4. After parity verification, remove `cmd/rxlisten`.

## Implementation Phases

### Phase 1: Cobra Bootstrap
1. Add dependency:
   - `go get github.com/spf13/cobra@latest`
2. Create `cmd/chirp/main.go` + root command wiring.
3. Add persistent global flags and shared CLI context builder.
4. Add `chirp version`.

### Phase 2: Shared Radio Lifecycle
1. Create helper to open/close `radio.Radio` once per command.
2. Standardize timeout handling and user-facing error messages.
3. Add command unit tests with mocked runner interfaces where possible.

### Phase 3: Listen Command
1. Port `cmd/rxlisten` logic into `chirp listen`.
2. Keep existing readable output format.
3. Add filtering flags and verify no regression in output semantics.

### Phase 4: Core Operational Commands
1. `chirp info`
2. `chirp send text --to <num|0> --channel <idx> --message "<text>"`
3. `chirp set owner --name "<name>"`
4. `chirp set modem --mode <preset>`
5. `chirp set location --lat-i <int> --lon-i <int> --alt <int>`
6. `chirp factory-reset`

### Phase 5: Output and Quality
1. Add `--json` support for non-streaming commands.
2. Add table/text formatting helpers for human output.
3. Add docs/examples in `README.md`.
4. Add CI checks for CLI build + tests.

## Testing Plan
- Unit tests for command argument validation.
- Unit tests for command handlers with mocked radio interface.
- Integration tests behind build tag `integration` and real hardware.
- Smoke checks:
  - `chirp listen --port ...`
  - `chirp info --port ...`
  - `chirp send text --message "test"`

## Hardware Integration Test Plan
### Preconditions
1. A Meshtastic radio is attached over serial and not in use by another process.
2. The test port is provided with an env var:
   - `CHIRP_TEST_PORT=/dev/cu.usbmodem101`
3. Integration tests are opt-in via build tags:
   - `//go:build integration`

### Test Layout
1. Keep integration tests next to command packages:
   - `cmd/chirp/*_integration_test.go`
2. Use a shared helper package for:
   - env var lookup and skip messages
   - short command timeouts
   - command execution wrappers

### Core Integration Cases (V1)
1. `listen` command starts, reads stream safely, and exits cleanly after bounded runtime.
2. `info` command connects and returns non-empty radio metadata.
3. `send text` command successfully writes a message frame to radio.
4. `set owner` / `set modem` / `set location` execute without transport/protocol errors.

### Guardrails
1. Use a bounded runtime flag for long-running commands (e.g. `listen --run-for 3s`) in tests.
2. Avoid destructive defaults in CI-like runs:
   - exclude `factory-reset` from default integration suite.
3. Run tests serially by default to avoid serial port contention.

### Command to Run
```bash
CHIRP_TEST_PORT=/dev/cu.usbmodem101 go test -tags=integration ./cmd/chirp/... -v
```

### Optional Destructive Suite
1. Gate destructive tests with a second env var:
   - `CHIRP_TEST_ALLOW_DESTRUCTIVE=1`
2. Include reset/reboot/DFU tests only when explicitly enabled.

## Risks and Mitigations
- Serial device contention (CLI vs other tools):
  - Mitigation: explicit error message naming the port and likely cause.
- Behavior drift from existing `rxlisten` output:
  - Mitigation: keep output contract and add golden tests.
- Scope creep toward full Meshtastic CLI parity:
  - Mitigation: enforce V1 non-goals; track extra features in a backlog.

## Backlog (Post-V1)
- `nodes` command and local node cache.
- Channel CRUD commands.
- Config transaction flow (`begin-edit`/`commit-edit`).
- Reboot/shutdown/DFU/time-set commands.
- Optional TCP/BLE transport support.
