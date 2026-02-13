# Chirp

A utility for configuring and managing Meshtastic nodes in Go.

## CLI

`chirp` is a slim Meshtastic serial CLI focused on common operational workflows.

### Global flags

- `--port` serial port (default: `/dev/cu.usbmodem101`)
- `--timeout` command timeout for non-streaming commands (default: `2s`)
- `--json` machine-readable output for non-streaming commands
- `--verbose` enable debug logging

### Commands

- `chirp version`
- `chirp listen [--idle-log 10s] [--no-telemetry] [--no-events] [--no-packets]`
- `chirp info`
- `chirp send text --to 0 --channel 0 --message "hello mesh"`
- `chirp set owner --name "Moon Station"`
- `chirp set modem --mode lf`
- `chirp set location --lat-i 377749000 --lon-i -1224194000 --alt 30`
- `chirp factory-reset` (interactive confirmation)
- `chirp factory-reset --yes` (non-interactive/automation-safe)

### Examples

```bash
# Listen for inbound packets/events
chirp listen --port /dev/cu.usbmodem101

# Fetch radio info as JSON
chirp info --json

# Send a broadcast text message on channel 0
chirp send text --message "test from chirp" --to 0 --channel 0

# Set device owner
chirp set owner --name "Field Node 01"

# Set modem preset
chirp set modem --mode mf

# Set fixed position payload
chirp set location --lat-i 377749000 --lon-i -1224194000 --alt 25

# Destructive command with explicit non-interactive confirmation
chirp factory-reset --yes
```

### Exit codes

- `0` success
- `1` runtime/transport/protocol error
- `2` invalid user input/flags

## Desktop UI (Wails)

The repo includes a Wails desktop scaffold using Svelte + TypeScript in
`frontend/`.

### Prerequisites

- Wails CLI installed
- Node.js 20+
- Go 1.25+

### Run in development mode

```bash
wails dev
```

Development behavior:

- Frontend Svelte/TS/CSS updates hot reload in the desktop window.
- Go backend updates rebuild/restart the app process automatically.
