<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import {
    connect,
    connectionStatus,
    disconnect,
    health,
    listenerStatus,
    listPorts,
    loadInfo,
    startListener,
    stopListener
  } from "./lib/backend";

  const maxLines = 800;

  let backendStatus = "Booting...";
  let ports: string[] = [];
  let selectedPort = "";
  let connected = false;
  let connectedPort = "";
  let info: ChirpInfoSummary | null = null;

  let listenerRunning = false;
  let listenerBusy = false;
  let listenerLines: ChirpListenerLine[] = [];
  let showEvents = true;
  let showPackets = true;
  let showTelemetry = true;
  let showMessages = true;

  let loadingInfo = false;
  let loadingPorts = false;
  let connecting = false;
  let disconnecting = false;
  let error = "";

  let unsubscribeListener: (() => void) | null = null;

  $: filteredLines = listenerLines.filter((line) => {
    if (line.category === "event") return showEvents;
    if (line.category === "packet") return showPackets;
    if (line.category === "telemetry") return showTelemetry;
    if (line.category === "message") return showMessages;
    return true;
  });

  function errorMessage(err: unknown, fallback: string): string {
    return err instanceof Error ? err.message : fallback;
  }

  function appendListenerLine(line: ChirpListenerLine): void {
    const next = [...listenerLines, line];
    listenerLines = next.length > maxLines ? next.slice(next.length - maxLines) : next;
  }

  function subscribeListenerEvents(): void {
    if (!window.runtime?.EventsOn) {
      return;
    }

    unsubscribeListener = window.runtime.EventsOn("listener:line", (line) => {
      const raw = line as Partial<ChirpListenerLine> | undefined;
      if (!raw || typeof raw.message !== "string" || typeof raw.label !== "string") {
        return;
      }

      appendListenerLine({
        timestamp: typeof raw.timestamp === "string" ? raw.timestamp : new Date().toISOString(),
        label: raw.label,
        message: raw.message,
        category:
          raw.category === "packet" ||
          raw.category === "telemetry" ||
          raw.category === "message" ||
          raw.category === "event"
            ? raw.category
            : "event"
      });
    });
  }

  async function refreshPorts(): Promise<void> {
    loadingPorts = true;
    error = "";

    try {
      ports = await listPorts();
      if (!selectedPort || !ports.includes(selectedPort)) {
        selectedPort = ports[0] ?? "";
      }
    } catch (err) {
      error = errorMessage(err, "Failed to load ports");
    } finally {
      loadingPorts = false;
    }
  }

  async function refreshConnectionStatus(): Promise<void> {
    const status = await connectionStatus();
    connected = status.connected;
    connectedPort = status.port;
  }

  async function refreshListenerStatus(): Promise<void> {
    const status = await listenerStatus();
    listenerRunning = status.running;
  }

  async function handleConnect(): Promise<void> {
    if (!selectedPort) {
      error = "Select a serial port first.";
      return;
    }

    connecting = true;
    error = "";

    try {
      await connect(selectedPort);
      await refreshConnectionStatus();
      await refreshListenerStatus();
      await refreshInfo();
    } catch (err) {
      error = errorMessage(err, "Failed to connect");
    } finally {
      connecting = false;
    }
  }

  async function handleDisconnect(): Promise<void> {
    disconnecting = true;
    error = "";

    try {
      await disconnect();
      info = null;
      await refreshConnectionStatus();
      await refreshListenerStatus();
    } catch (err) {
      error = errorMessage(err, "Failed to disconnect");
    } finally {
      disconnecting = false;
    }
  }

  async function refreshInfo(): Promise<void> {
    loadingInfo = true;
    error = "";

    try {
      const result = await loadInfo();
      info = result.summary;
    } catch (err) {
      error = errorMessage(err, "Failed to load info");
    } finally {
      loadingInfo = false;
    }
  }

  async function handleStartListener(): Promise<void> {
    listenerBusy = true;
    error = "";

    try {
      await startListener();
      await refreshListenerStatus();
    } catch (err) {
      error = errorMessage(err, "Failed to start listener");
    } finally {
      listenerBusy = false;
    }
  }

  async function handleStopListener(): Promise<void> {
    listenerBusy = true;
    error = "";

    try {
      await stopListener();
      await refreshListenerStatus();
    } catch (err) {
      error = errorMessage(err, "Failed to stop listener");
    } finally {
      listenerBusy = false;
    }
  }

  function clearListener(): void {
    listenerLines = [];
  }

  onMount(async () => {
    subscribeListenerEvents();

    try {
      const result = await health();
      backendStatus = `Backend: ${result}`;
    } catch (err) {
      backendStatus = "Backend unavailable";
      error = errorMessage(err, "Unknown startup error");
      return;
    }

    try {
      await refreshPorts();
      await refreshConnectionStatus();
      await refreshListenerStatus();
      if (connected) {
        await refreshInfo();
      }
    } catch (err) {
      error = errorMessage(err, "Startup sync failed");
    }
  });

  onDestroy(() => {
    if (unsubscribeListener) {
      unsubscribeListener();
      unsubscribeListener = null;
    }
  });
</script>

<main class="page">
  <header class="header">
    <h1>Chirp UI</h1>
    <p>Wails + Svelte + TypeScript</p>
  </header>

  <section class="card">
    <h2>Backend Status</h2>
    <p>{backendStatus}</p>
  </section>

  <section class="card">
    <div class="row">
      <h2>Connection</h2>
      <button onclick={refreshPorts} disabled={loadingPorts}>
        {loadingPorts ? "Refreshing..." : "Refresh"}
      </button>
    </div>

    <div class="row stack-mobile">
      <label class="field">
        <span>Serial Port</span>
        <select bind:value={selectedPort} disabled={connecting || disconnecting || listenerRunning}>
          {#if ports.length === 0}
            <option value="">No ports found</option>
          {:else}
            {#each ports as port}
              <option value={port}>{port}</option>
            {/each}
          {/if}
        </select>
      </label>
      <div class="actions">
        <button
          onclick={handleConnect}
          disabled={connecting || disconnecting || ports.length === 0 || connected || listenerRunning}
        >
          {connecting ? "Connecting..." : "Connect"}
        </button>
        <button class="secondary" onclick={handleDisconnect} disabled={disconnecting || !connected}>
          {disconnecting ? "Disconnecting..." : "Disconnect"}
        </button>
      </div>
    </div>

    <p class="status">
      {#if connected}
        Connected to <code>{connectedPort}</code>
      {:else}
        Not connected
      {/if}
    </p>
  </section>

  <section class="card">
    <div class="row">
      <h2>Radio Overview</h2>
      <button class="secondary" onclick={refreshInfo} disabled={!connected || loadingInfo}>
        {loadingInfo ? "Loading..." : "Reload"}
      </button>
    </div>

    {#if info}
      <dl class="summary-grid">
        <div><dt>My Node</dt><dd>{info.my_node}</dd></div>
        <div><dt>Firmware</dt><dd>{info.firmware}</dd></div>
        <div><dt>HW Model</dt><dd>{info.hw_model}</dd></div>
        <div><dt>Role</dt><dd>{info.role}</dd></div>
        <div><dt>Nodes</dt><dd>{info.nodes}</dd></div>
        <div><dt>Channels</dt><dd>{info.channels}</dd></div>
        <div><dt>Configs</dt><dd>{info.configs}</dd></div>
        <div><dt>Module Configs</dt><dd>{info.module_configs}</dd></div>
        <div><dt>Responses</dt><dd>{info.responses}</dd></div>
      </dl>
    {:else}
      <p>{connected ? "No radio info loaded yet." : "Connect to a radio to view info."}</p>
    {/if}
  </section>

  <section class="card">
    <div class="row">
      <h2>Live Listener</h2>
      <div class="actions">
        <button onclick={handleStartListener} disabled={!connected || listenerRunning || listenerBusy}>
          {listenerBusy && !listenerRunning ? "Starting..." : "Start"}
        </button>
        <button
          class="secondary"
          onclick={handleStopListener}
          disabled={!listenerRunning || listenerBusy}
        >
          {listenerBusy && listenerRunning ? "Stopping..." : "Stop"}
        </button>
        <button class="secondary" onclick={clearListener} disabled={listenerLines.length === 0}>
          Clear
        </button>
      </div>
    </div>

    <p class="status">
      {listenerRunning ? "Listener running" : "Listener stopped"} · {listenerLines.length} lines buffered
    </p>

    <div class="listener-filters">
      <label><input type="checkbox" bind:checked={showEvents} /> Events</label>
      <label><input type="checkbox" bind:checked={showPackets} /> Packets</label>
      <label><input type="checkbox" bind:checked={showTelemetry} /> Telemetry</label>
      <label><input type="checkbox" bind:checked={showMessages} /> Messages</label>
    </div>

    <div class="listener-log" role="log" aria-live="polite">
      {#if filteredLines.length === 0}
        <p class="empty">No listener output yet.</p>
      {:else}
        {#each filteredLines as line}
          <div class="log-line">
            <span class={`badge ${line.category}`}>{line.label}</span>
            <span class="time">{line.timestamp}</span>
            <span class="text">{line.message}</span>
          </div>
        {/each}
      {/if}
    </div>
  </section>

  {#if error}
    <section class="card error-card">
      <h2>Error</h2>
      <p class="error">{error}</p>
    </section>
  {/if}
</main>
