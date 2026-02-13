export {};

declare global {
  type ChirpConnectionStatus = {
    connected: boolean;
    port: string;
  };

  type ChirpInfoSummary = {
    responses: number;
    my_node: string;
    firmware: string;
    hw_model: string;
    role: string;
    nodes: number;
    channels: number;
    configs: number;
    module_configs: number;
  };

  type ChirpInfoView = {
    summary: ChirpInfoSummary;
  };

  type ChirpListenerStatus = {
    running: boolean;
  };

  type ChirpListenerLine = {
    timestamp: string;
    label: string;
    message: string;
    category: "event" | "packet" | "telemetry" | "message";
  };

  interface Window {
    go?: {
      uiapp?: {
        App?: {
          Health: () => Promise<string>;
          ListPorts: () => Promise<string[]>;
          Connect: (port: string) => Promise<void>;
          Disconnect: () => Promise<void>;
          ConnectionStatus: () => Promise<ChirpConnectionStatus>;
          LoadInfo: () => Promise<ChirpInfoView>;
          StartListener: () => Promise<void>;
          StopListener: () => Promise<void>;
          GetListenerStatus: () => Promise<ChirpListenerStatus>;
        };
      };
    };
    runtime?: {
      EventsOn: (eventName: string, callback: (...args: unknown[]) => void) => () => void;
    };
  }
}
