package node

import (
	"fmt"
	"strings"
	"time"

	pb "github.com/coreyvan/chirp/protogen/meshtastic"
	"google.golang.org/protobuf/proto"
)

type StreamCategory int

const (
	StreamCategoryEvent StreamCategory = iota
	StreamCategoryPacket
	StreamCategoryTelemetry
	StreamCategoryMessage
)

type StreamLine struct {
	Label    string
	Message  string
	Category StreamCategory
}

func RenderFromRadio(fr *pb.FromRadio) []StreamLine {
	switch v := fr.GetPayloadVariant().(type) {
	case *pb.FromRadio_Packet:
		return RenderMeshPacket(v.Packet)
	case *pb.FromRadio_LogRecord:
		if v.LogRecord == nil {
			return []StreamLine{{Label: "EVT", Message: "log_record=nil", Category: StreamCategoryEvent}}
		}
		return []StreamLine{{
			Label:    "EVT",
			Message:  fmt.Sprintf("log level=%s source=%s msg=%q", v.LogRecord.GetLevel().String(), v.LogRecord.GetSource(), v.LogRecord.GetMessage()),
			Category: StreamCategoryEvent,
		}}
	case *pb.FromRadio_QueueStatus:
		if v.QueueStatus == nil {
			return []StreamLine{{Label: "EVT", Message: "queue_status=nil", Category: StreamCategoryEvent}}
		}
		return []StreamLine{{
			Label: "EVT",
			Message: fmt.Sprintf(
				"queue res=%d free=%d/%d mesh_packet_id=%d",
				v.QueueStatus.GetRes(),
				v.QueueStatus.GetFree(),
				v.QueueStatus.GetMaxlen(),
				v.QueueStatus.GetMeshPacketId(),
			),
			Category: StreamCategoryEvent,
		}}
	case *pb.FromRadio_Rebooted:
		return []StreamLine{{Label: "EVT", Message: fmt.Sprintf("rebooted=%t", v.Rebooted), Category: StreamCategoryEvent}}
	case *pb.FromRadio_ConfigCompleteId:
		return []StreamLine{{Label: "EVT", Message: fmt.Sprintf("config_complete_id=%d", v.ConfigCompleteId), Category: StreamCategoryEvent}}
	case *pb.FromRadio_MyInfo:
		if v.MyInfo == nil {
			return []StreamLine{{Label: "EVT", Message: "my_info=nil", Category: StreamCategoryEvent}}
		}
		return []StreamLine{{Label: "EVT", Message: fmt.Sprintf("my_info node_num=!%08x", v.MyInfo.GetMyNodeNum()), Category: StreamCategoryEvent}}
	case *pb.FromRadio_NodeInfo:
		if v.NodeInfo == nil {
			return []StreamLine{{Label: "EVT", Message: "node_info=nil", Category: StreamCategoryEvent}}
		}
		return []StreamLine{{
			Label:    "EVT",
			Message:  fmt.Sprintf("node_info node_num=!%08x user=%q", v.NodeInfo.GetNum(), v.NodeInfo.GetUser().GetLongName()),
			Category: StreamCategoryEvent,
		}}
	case *pb.FromRadio_Metadata:
		if v.Metadata == nil {
			return []StreamLine{{Label: "EVT", Message: "metadata=nil", Category: StreamCategoryEvent}}
		}
		return []StreamLine{{
			Label: "EVT",
			Message: fmt.Sprintf(
				"metadata fw=%q state_ver=%d hw=%s role=%s wifi=%t bt=%t eth=%t remote_hw=%t pkc=%t excluded_modules=0x%x",
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
			),
			Category: StreamCategoryEvent,
		}}
	case *pb.FromRadio_Channel:
		if v.Channel == nil {
			return []StreamLine{{Label: "EVT", Message: "channel=nil", Category: StreamCategoryEvent}}
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
		return []StreamLine{{
			Label: "EVT",
			Message: fmt.Sprintf(
				"channel index=%d role=%s name=%q id=%d uplink=%t downlink=%t",
				v.Channel.GetIndex(),
				v.Channel.GetRole().String(),
				settingsName,
				settingsID,
				uplink,
				downlink,
			),
			Category: StreamCategoryEvent,
		}}
	case *pb.FromRadio_Config:
		if v.Config == nil {
			return []StreamLine{{Label: "EVT", Message: "config=nil", Category: StreamCategoryEvent}}
		}
		return []StreamLine{{Label: "EVT", Message: fmt.Sprintf("config section=%s", configSectionName(v.Config)), Category: StreamCategoryEvent}}
	case *pb.FromRadio_ModuleConfig:
		if v.ModuleConfig == nil {
			return []StreamLine{{Label: "EVT", Message: "module_config=nil", Category: StreamCategoryEvent}}
		}
		return []StreamLine{{Label: "EVT", Message: fmt.Sprintf("module_config section=%s", moduleConfigSectionName(v.ModuleConfig)), Category: StreamCategoryEvent}}
	case *pb.FromRadio_FileInfo:
		if v.FileInfo == nil {
			return []StreamLine{{Label: "EVT", Message: "file_info=nil", Category: StreamCategoryEvent}}
		}
		return []StreamLine{{
			Label:    "EVT",
			Message:  fmt.Sprintf("file_info name=%q size_bytes=%d", v.FileInfo.GetFileName(), v.FileInfo.GetSizeBytes()),
			Category: StreamCategoryEvent,
		}}
	default:
		return []StreamLine{{Label: "EVT", Message: fmt.Sprintf("variant=%T", fr.GetPayloadVariant()), Category: StreamCategoryEvent}}
	}
}

func RenderMeshPacket(mp *pb.MeshPacket) []StreamLine {
	if mp == nil {
		return []StreamLine{{Label: "PKT", Message: "nil", Category: StreamCategoryPacket}}
	}

	portLabel := "UNKNOWN"
	payloadLen := 0
	if decoded := mp.GetDecoded(); decoded != nil {
		portLabel = decoded.GetPortnum().String()
		payloadLen = len(decoded.GetPayload())
	}

	lines := []StreamLine{{
		Label: "PKT",
		Message: fmt.Sprintf(
			"from=!%08x to=!%08x ch=%d id=%d hop=%d rssi=%ddBm snr=%.2f rx_time=%s port=%s bytes=%d",
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
		),
		Category: StreamCategoryPacket,
	}}

	decoded := mp.GetDecoded()
	if decoded == nil {
		return lines
	}

	switch decoded.GetPortnum() {
	case pb.PortNum_TEXT_MESSAGE_APP:
		text := strings.TrimSpace(string(decoded.GetPayload()))
		lines = append(lines, StreamLine{
			Label:    "MSG",
			Message:  fmt.Sprintf("text=%q", text),
			Category: StreamCategoryMessage,
		})
	case pb.PortNum_TELEMETRY_APP:
		lines = append(lines, renderTelemetry(decoded.GetPayload())...)
	}

	return lines
}

func renderTelemetry(payload []byte) []StreamLine {
	var t pb.Telemetry
	if err := proto.Unmarshal(payload, &t); err != nil {
		return []StreamLine{{Label: "TEL", Message: fmt.Sprintf("decode_error=%v", err), Category: StreamCategoryTelemetry}}
	}

	tstamp := formatUnixSeconds(t.GetTime())

	if dm := t.GetDeviceMetrics(); dm != nil {
		return []StreamLine{{
			Label: "TEL",
			Message: fmt.Sprintf(
				"type=device time=%s batt=%d%% volt=%.2fV ch_util=%.2f%% air_tx=%.2f%% uptime=%ds",
				tstamp,
				dm.GetBatteryLevel(),
				dm.GetVoltage(),
				dm.GetChannelUtilization(),
				dm.GetAirUtilTx(),
				dm.GetUptimeSeconds(),
			),
			Category: StreamCategoryTelemetry,
		}}
	}

	if ls := t.GetLocalStats(); ls != nil {
		return []StreamLine{{
			Label: "TEL",
			Message: fmt.Sprintf(
				"type=local time=%s uptime=%ds nodes=%d/%d pkts_tx=%d pkts_rx=%d bad_rx=%d ch_util=%.2f%% air_tx=%.2f%% noise_floor=%ddBm",
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
			),
			Category: StreamCategoryTelemetry,
		}}
	}

	if em := t.GetEnvironmentMetrics(); em != nil {
		return []StreamLine{{
			Label: "TEL",
			Message: fmt.Sprintf(
				"type=env time=%s temp=%.2fC hum=%.2f%% pressure=%.2fhPa",
				tstamp,
				em.GetTemperature(),
				em.GetRelativeHumidity(),
				em.GetBarometricPressure(),
			),
			Category: StreamCategoryTelemetry,
		}}
	}

	return []StreamLine{{
		Label:    "TEL",
		Message:  fmt.Sprintf("type=other time=%s variant=%T", tstamp, t.GetVariant()),
		Category: StreamCategoryTelemetry,
	}}
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
