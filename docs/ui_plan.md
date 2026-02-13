# Chirp Desktop UI Plan (Wails)

## Goal
Build a desktop GUI for Chirp that reuses the existing Go radio/service code and makes common radio operations easier than CLI usage.

## Primary User Workflows
1. Select a USB serial port and connect to a target radio.
2. View current device/radio configuration, channels, and node summaries.
3. Run a live debug listener with readable, filterable traffic output.
4. View node history on a map, including where nodes were seen over time.

## Scope (V1)
- Desktop app built with Wails.
- Frontend implemented with Svelte + TypeScript (no untyped JavaScript app code).
- Single active radio connection at a time.
- Read-only config/channel overview initially.
- Live listener with category filters and search.
- Node history map using stored location sightings from received packets/events.

## Non-goals (V1)
- BLE/TCP transports.
- Full radio config editing parity with Meshtastic tooling.
- Cloud sync or multi-user collaboration.
- Mobile app.

## Why Wails
- Keeps the backend in Go and allows direct reuse of existing packages:
  - `internal/app/node`
  - `pkg/radio`
  - `pkg/serial`
- Better fit for rich map + log visualization than pure native Go widget toolkits.
- Produces native desktop binaries without requiring a separate sidecar process.

## Architecture

### Backend (Go)
Create a Wails-facing app layer that wraps existing domain code.

Proposed package:
- `internal/uiapp`

Responsibilities:
1. Connection manager
   - `ListPorts() []PortInfo`
   - `Connect(port string) error`
   - `Disconnect() error`
   - `ConnectionStatus()`
2. Radio queries
   - `LoadInfo()`
   - `LoadConfigs()`
   - `LoadChannels()`
3. Actions
   - `SendText(message, to, channel)`
   - `SetOwner(name)`
   - `SetModem(mode)`
   - `SetLocation(latI, lonI, alt)`
   - `FactoryReset(confirm bool)`
4. Listener stream
   - `StartListener(filters)`
   - `StopListener()`
   - emits typed stream events to frontend
5. Node history store
   - persist sightings to SQLite
   - query by time window and node id

### Frontend (Wails UI)
Pages/panels:
1. Connect
   - serial port dropdown + refresh
   - connect/disconnect button
   - current device summary
2. Overview
   - info summary cards (firmware/hw/role/node counts/channels/configs)
   - config/channel inspectors
3. Live Listener
   - scrolling stream view with category badges (`EVT`, `PKT`, `TEL`, `MSG`)
   - filters (events/packets/telemetry/messages)
   - pause/resume + clear + search
4. Map
   - node markers and optional trails
   - timeline window selector
   - node selector and details panel

## Data Model (Initial)

### Stream Event
- `timestamp`
- `label` (`EVT|PKT|TEL|MSG`)
- `category`
- `message`
- optional parsed fields (`from`, `to`, `port`, `rssi`, `snr`)

### Node Sighting
- `node_num`
- `lat`
- `lon`
- `alt` (optional)
- `seen_at`
- optional radio metrics (`rssi`, `snr`)

## Implementation Phases

### Phase 1: Wails Bootstrap
1. Create Wails app skeleton under `cmd/chirp-ui` (or similar).
2. Add backend app struct with lifecycle hooks.
3. Wire basic frontend shell + navigation.

### Phase 2: Connection + Overview
1. Expose serial port listing via `pkg/serial.GetPorts`.
2. Add connect/disconnect flow using `pkg/radio`.
3. Reuse `internal/app/node.Service.Info` for overview data.

### Phase 3: Live Listener
1. Reuse rendering logic from `internal/app/node/events.go`.
2. Implement listener goroutine and frontend event bridge.
3. Add filters and log controls.

### Phase 4: Node History + Map
1. Capture node/location telemetry from listener stream.
2. Persist sightings to SQLite.
3. Add map view (Leaflet or MapLibre GL JS) and timeline filtering.

### Phase 5: Command Parity and Polish
1. Add send/set/factory reset actions to UI.
2. Improve error presentation and reconnect handling.
3. Add import/export of listener logs and map data.

## Suggested Technical Choices
- Frontend framework: Svelte + TypeScript (selected).
- Mapping: Leaflet (simpler) or MapLibre GL JS (more advanced styling/perf).
- Local persistence: SQLite with `modernc.org/sqlite` or `mattn/go-sqlite3`.
- State management: Svelte stores.

## Testing Strategy
1. Unit tests for backend app layer (mock radio/service).
2. Integration tests for connect/info/listen against real hardware (opt-in).
3. UI smoke tests for key flows:
   - connect
   - info render
   - listener start/stop/filter
   - map points render

## Risks and Mitigations
- Serial contention with other apps:
  - clear error + retry path and auto-refresh ports.
- Listener volume causing UI lag:
  - bounded buffers and batched UI updates.
- Incomplete location data:
  - fallback to last known location and clear missing-state UI.

## Open Decisions
1. Map engine: Leaflet vs MapLibre.
2. Node history retention policy (for example 30/90/unlimited days).

## Development Workflow (Hot Reload)
1. Use Wails dev mode while building features:
   - `wails dev`
2. Frontend changes (Svelte/TS/CSS) hot reload in the running desktop window.
3. Go backend changes trigger rebuild/restart of the app process in dev mode.
4. For fastest iteration:
   - keep connection state resilient (auto-reconnect toggle or quick reconnect action)
   - keep seed/mock data mode for UI work when hardware is disconnected
5. When needed, split workflow:
   - run frontend-only dev server for pure UI work
   - run `wails dev` for integrated frontend+backend testing with real serial access
