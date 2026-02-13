package cli

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	pb "github.com/coreyvan/chirp/protogen/meshtastic"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

type listenOptions struct {
	idleLog     time.Duration
	noTelemetry bool
	noEvents    bool
	noPackets   bool
}

func newListenCommand(cliCtx *Context, opener radioOpener) *cobra.Command {
	opts := &listenOptions{
		idleLog: 10 * time.Second,
	}

	cmd := &cobra.Command{
		Use:   "listen",
		Short: "Stream incoming packets, events, and telemetry",
		Args:  wrapPositionalArgs(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if opts.idleLog <= 0 {
				return newUserInputError(fmt.Errorf("--idle-log must be greater than 0"))
			}

			return runWithRadioNoTimeout(cmd.Context(), cliCtx, opener, RadioRunnerFunc(func(runCtx context.Context, radio Radio) error {
				return runListen(runCtx, cmd.OutOrStdout(), radio, cliCtx.Port, opts)
			}))
		},
	}

	cmd.Flags().DurationVar(&opts.idleLog, "idle-log", opts.idleLog, "how often to print idle message when no packets arrive")
	cmd.Flags().BoolVar(&opts.noTelemetry, "no-telemetry", false, "suppress telemetry output")
	cmd.Flags().BoolVar(&opts.noEvents, "no-events", false, "suppress event output")
	cmd.Flags().BoolVar(&opts.noPackets, "no-packets", false, "suppress packet output")

	return cmd
}

func runListen(ctx context.Context, out io.Writer, radio Radio, port string, opts *listenOptions) error {
	_, _ = fmt.Fprintf(out, "rx listener started on %s\n", port)

	// Prime the device so nodes that stay quiet until polled begin streaming updates.
	if responses, err := radio.GetRadioInfo(); err != nil {
		_, _ = fmt.Fprintf(out, "[ERR] get radio info: %v\n", err)
	} else {
		for _, fr := range responses {
			logFromRadio(out, fr, opts)
		}
	}

	lastIdleLog := time.Now()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		fromRadioPackets, err := radio.ReadResponse(true)
		if err != nil {
			_, _ = fmt.Fprintf(out, "[ERR] read response: %v\n", err)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(300 * time.Millisecond):
			}
			continue
		}

		if len(fromRadioPackets) == 0 {
			if time.Since(lastIdleLog) >= opts.idleLog {
				_, _ = fmt.Fprintln(out, "[IDLE] no packets")
				lastIdleLog = time.Now()
			}
			continue
		}

		for _, fr := range fromRadioPackets {
			logFromRadio(out, fr, opts)
		}
	}
}

func logFromRadio(out io.Writer, fr *pb.FromRadio, opts *listenOptions) {
	switch v := fr.GetPayloadVariant().(type) {
	case *pb.FromRadio_Packet:
		logMeshPacket(out, v.Packet, opts)
	case *pb.FromRadio_LogRecord:
		if opts.noEvents {
			return
		}
		if v.LogRecord == nil {
			_, _ = fmt.Fprintln(out, "[EVT] log_record=nil")
			return
		}
		_, _ = fmt.Fprintf(
			out,
			"[EVT] log level=%s source=%s msg=%q\n",
			v.LogRecord.GetLevel().String(),
			v.LogRecord.GetSource(),
			v.LogRecord.GetMessage(),
		)
	case *pb.FromRadio_QueueStatus:
		if opts.noEvents {
			return
		}
		if v.QueueStatus == nil {
			_, _ = fmt.Fprintln(out, "[EVT] queue_status=nil")
			return
		}
		_, _ = fmt.Fprintf(
			out,
			"[EVT] queue res=%d free=%d/%d mesh_packet_id=%d\n",
			v.QueueStatus.GetRes(),
			v.QueueStatus.GetFree(),
			v.QueueStatus.GetMaxlen(),
			v.QueueStatus.GetMeshPacketId(),
		)
	case *pb.FromRadio_Rebooted:
		if opts.noEvents {
			return
		}
		_, _ = fmt.Fprintf(out, "[EVT] rebooted=%t\n", v.Rebooted)
	case *pb.FromRadio_ConfigCompleteId:
		if opts.noEvents {
			return
		}
		_, _ = fmt.Fprintf(out, "[EVT] config_complete_id=%d\n", v.ConfigCompleteId)
	case *pb.FromRadio_MyInfo:
		if opts.noEvents {
			return
		}
		if v.MyInfo == nil {
			_, _ = fmt.Fprintln(out, "[EVT] my_info=nil")
			return
		}
		_, _ = fmt.Fprintf(out, "[EVT] my_info node_num=!%08x\n", v.MyInfo.GetMyNodeNum())
	case *pb.FromRadio_NodeInfo:
		if opts.noEvents {
			return
		}
		if v.NodeInfo == nil {
			_, _ = fmt.Fprintln(out, "[EVT] node_info=nil")
			return
		}
		_, _ = fmt.Fprintf(
			out,
			"[EVT] node_info node_num=!%08x user=%q\n",
			v.NodeInfo.GetNum(),
			v.NodeInfo.GetUser().GetLongName(),
		)
	case *pb.FromRadio_Metadata:
		if opts.noEvents {
			return
		}
		if v.Metadata == nil {
			_, _ = fmt.Fprintln(out, "[EVT] metadata=nil")
			return
		}
		_, _ = fmt.Fprintf(
			out,
			"[EVT] metadata fw=%q state_ver=%d hw=%s role=%s wifi=%t bt=%t eth=%t remote_hw=%t pkc=%t excluded_modules=0x%x\n",
			v.Metadata.GetFirmwareVersion(),
			v.Metadata.GetDeviceStateVersion(),
			v.Metadata.GetHwModel().String(),
			v.Metadata.GetRole().String(),
			v.Metadata.GetHasWifi(),
			v.Metadata.GetHasBluetooth(),
			v.Metadata.GetHasEthernet(),
			v.Metadata.GetHasRemoteHardware(),
			v.Metadata.GetHasPKC(),
			v.Metadata.GetExcludedModules(),
		)
	case *pb.FromRadio_Channel:
		if opts.noEvents {
			return
		}
		if v.Channel == nil {
			_, _ = fmt.Fprintln(out, "[EVT] channel=nil")
			return
		}
		settingsName := ""
		settingsID := uint32(0)
		uplink := false
		downlink := false
		if s := v.Channel.GetSettings(); s != nil {
			settingsName = s.GetName()
			settingsID = s.GetId()
			uplink = s.GetUplinkEnabled()
			downlink = s.GetDownlinkEnabled()
		}
		_, _ = fmt.Fprintf(
			out,
			"[EVT] channel index=%d role=%s name=%q id=%d uplink=%t downlink=%t\n",
			v.Channel.GetIndex(),
			v.Channel.GetRole().String(),
			settingsName,
			settingsID,
			uplink,
			downlink,
		)
	case *pb.FromRadio_Config:
		if opts.noEvents {
			return
		}
		if v.Config == nil {
			_, _ = fmt.Fprintln(out, "[EVT] config=nil")
			return
		}
		_, _ = fmt.Fprintf(out, "[EVT] config section=%s\n", configSectionName(v.Config))
	case *pb.FromRadio_ModuleConfig:
		if opts.noEvents {
			return
		}
		if v.ModuleConfig == nil {
			_, _ = fmt.Fprintln(out, "[EVT] module_config=nil")
			return
		}
		_, _ = fmt.Fprintf(out, "[EVT] module_config section=%s\n", moduleConfigSectionName(v.ModuleConfig))
	case *pb.FromRadio_FileInfo:
		if opts.noEvents {
			return
		}
		if v.FileInfo == nil {
			_, _ = fmt.Fprintln(out, "[EVT] file_info=nil")
			return
		}
		_, _ = fmt.Fprintf(
			out,
			"[EVT] file_info name=%q size_bytes=%d\n",
			v.FileInfo.GetFileName(),
			v.FileInfo.GetSizeBytes(),
		)
	default:
		if opts.noEvents {
			return
		}
		_, _ = fmt.Fprintf(out, "[EVT] variant=%T\n", fr.GetPayloadVariant())
	}
}

func logMeshPacket(out io.Writer, mp *pb.MeshPacket, opts *listenOptions) {
	if mp == nil {
		if !opts.noPackets {
			_, _ = fmt.Fprintln(out, "[PKT] nil")
		}
		return
	}

	portLabel := "UNKNOWN"
	payloadLen := 0
	if decoded := mp.GetDecoded(); decoded != nil {
		portLabel = decoded.GetPortnum().String()
		payloadLen = len(decoded.GetPayload())
	}

	if !opts.noPackets {
		_, _ = fmt.Fprintf(
			out,
			"[PKT] from=!%08x to=!%08x ch=%d id=%d hop=%d rssi=%ddBm snr=%.2f rx_time=%s port=%s bytes=%d\n",
			mp.GetFrom(),
			mp.GetTo(),
			mp.GetChannel(),
			mp.GetId(),
			mp.GetHopLimit(),
			mp.GetRxRssi(),
			mp.GetRxSnr(),
			formatUnixSeconds(mp.GetRxTime()),
			portLabel,
			payloadLen,
		)
	}

	decoded := mp.GetDecoded()
	if decoded == nil {
		return
	}

	switch decoded.GetPortnum() {
	case pb.PortNum_TEXT_MESSAGE_APP:
		text := strings.TrimSpace(string(decoded.GetPayload()))
		_, _ = fmt.Fprintf(out, "[MSG] text=%q\n", text)
	case pb.PortNum_TELEMETRY_APP:
		if opts.noTelemetry {
			return
		}
		logTelemetry(out, decoded.GetPayload())
	}
}

func logTelemetry(out io.Writer, payload []byte) {
	var t pb.Telemetry
	if err := proto.Unmarshal(payload, &t); err != nil {
		_, _ = fmt.Fprintf(out, "[TEL] decode_error=%v\n", err)
		return
	}

	tstamp := formatUnixSeconds(t.GetTime())

	if dm := t.GetDeviceMetrics(); dm != nil {
		_, _ = fmt.Fprintf(
			out,
			"[TEL] type=device time=%s batt=%d%% volt=%.2fV ch_util=%.2f%% air_tx=%.2f%% uptime=%ds\n",
			tstamp,
			dm.GetBatteryLevel(),
			dm.GetVoltage(),
			dm.GetChannelUtilization(),
			dm.GetAirUtilTx(),
			dm.GetUptimeSeconds(),
		)
		return
	}

	if ls := t.GetLocalStats(); ls != nil {
		_, _ = fmt.Fprintf(
			out,
			"[TEL] type=local time=%s uptime=%ds nodes=%d/%d pkts_tx=%d pkts_rx=%d bad_rx=%d ch_util=%.2f%% air_tx=%.2f%% noise_floor=%ddBm\n",
			tstamp,
			ls.GetUptimeSeconds(),
			ls.GetNumOnlineNodes(),
			ls.GetNumTotalNodes(),
			ls.GetNumPacketsTx(),
			ls.GetNumPacketsRx(),
			ls.GetNumPacketsRxBad(),
			ls.GetChannelUtilization(),
			ls.GetAirUtilTx(),
			ls.GetNoiseFloor(),
		)
		return
	}

	if em := t.GetEnvironmentMetrics(); em != nil {
		_, _ = fmt.Fprintf(
			out,
			"[TEL] type=env time=%s temp=%.2fC hum=%.2f%% pressure=%.2fhPa\n",
			tstamp,
			em.GetTemperature(),
			em.GetRelativeHumidity(),
			em.GetBarometricPressure(),
		)
		return
	}

	_, _ = fmt.Fprintf(out, "[TEL] type=other time=%s variant=%T\n", tstamp, t.GetVariant())
}

func formatUnixSeconds(ts uint32) string {
	if ts == 0 {
		return "-"
	}
	return time.Unix(int64(ts), 0).UTC().Format(time.RFC3339)
}

func configSectionName(c *pb.Config) string {
	if c == nil || c.GetPayloadVariant() == nil {
		return "unknown"
	}

	switch c.GetPayloadVariant().(type) {
	case *pb.Config_Device:
		return "device"
	case *pb.Config_Position:
		return "position"
	case *pb.Config_Power:
		return "power"
	case *pb.Config_Network:
		return "network"
	case *pb.Config_Display:
		return "display"
	case *pb.Config_Lora:
		return "lora"
	case *pb.Config_Bluetooth:
		return "bluetooth"
	case *pb.Config_Security:
		return "security"
	case *pb.Config_Sessionkey:
		return "sessionkey"
	case *pb.Config_DeviceUi:
		return "device_ui"
	default:
		return "unknown"
	}
}

func moduleConfigSectionName(c *pb.ModuleConfig) string {
	if c == nil || c.GetPayloadVariant() == nil {
		return "unknown"
	}

	switch c.GetPayloadVariant().(type) {
	case *pb.ModuleConfig_Mqtt:
		return "mqtt"
	case *pb.ModuleConfig_Serial:
		return "serial"
	case *pb.ModuleConfig_ExternalNotification:
		return "external_notification"
	case *pb.ModuleConfig_StoreForward:
		return "store_forward"
	case *pb.ModuleConfig_RangeTest:
		return "range_test"
	case *pb.ModuleConfig_Telemetry:
		return "telemetry"
	case *pb.ModuleConfig_CannedMessage:
		return "canned_message"
	case *pb.ModuleConfig_Audio:
		return "audio"
	case *pb.ModuleConfig_RemoteHardware:
		return "remote_hardware"
	case *pb.ModuleConfig_NeighborInfo:
		return "neighbor_info"
	case *pb.ModuleConfig_AmbientLighting:
		return "ambient_lighting"
	case *pb.ModuleConfig_DetectionSensor:
		return "detection_sensor"
	case *pb.ModuleConfig_Paxcounter:
		return "paxcounter"
	case *pb.ModuleConfig_Statusmessage:
		return "statusmessage"
	default:
		return "unknown"
	}
}
