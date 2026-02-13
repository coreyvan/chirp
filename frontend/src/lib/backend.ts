type AppBindings = NonNullable<NonNullable<NonNullable<Window["go"]>["uiapp"]>["App"]>;

function getBindings(): AppBindings {
  const app = window.go?.uiapp?.App;
  if (!app) {
    throw new Error("Wails bindings are not ready. Run inside `wails dev` or a built app.");
  }

  return app;
}

export async function health(): Promise<string> {
  return getBindings().Health();
}

export async function listPorts(): Promise<string[]> {
  return getBindings().ListPorts();
}

export async function connect(port: string): Promise<void> {
  return getBindings().Connect(port);
}

export async function disconnect(): Promise<void> {
  return getBindings().Disconnect();
}

export async function connectionStatus(): Promise<ChirpConnectionStatus> {
  return getBindings().ConnectionStatus();
}

export async function loadInfo(): Promise<ChirpInfoView> {
  return getBindings().LoadInfo();
}
