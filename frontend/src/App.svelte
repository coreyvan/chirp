<script lang="ts">
  import { onMount } from "svelte";
  import {
    connect,
    connectionStatus,
    disconnect,
    health,
    listPorts,
    loadInfo
  } from "./lib/backend";

  let backendStatus = "Booting...";
  let ports: string[] = [];
  let selectedPort = "";
  let connected = false;
  let connectedPort = "";
  let info: ChirpInfoSummary | null = null;

  let loadingInfo = false;
  let loadingPorts = false;
  let connecting = false;
  let disconnecting = false;
  let error = "";

  function errorMessage(err: unknown, fallback: string): string {
    return err instanceof Error ? err.message : fallback;
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

  onMount(async () => {
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
      if (connected) {
        await refreshInfo();
      }
    } catch (err) {
      error = errorMessage(err, "Startup sync failed");
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
        <select bind:value={selectedPort} disabled={connecting || disconnecting}>
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
          disabled={connecting || disconnecting || ports.length === 0 || connected}
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

  {#if error}
    <section class="card error-card">
      <h2>Error</h2>
      <p class="error">{error}</p>
    </section>
  {/if}
</main>
