package main

import (
	"flag"
	"log"
	"strings"
	"time"

	"github.com/coreyvan/chirp/pkg/radio"
	pb "github.com/coreyvan/chirp/protogen/meshtastic"
	"google.golang.org/protobuf/proto"
)

func main() {
	port := flag.String("port", "/dev/cu.usbmodem101", "serial port for the Meshtastic node")
	idleLog := flag.Duration("idle-log", 10*time.Second, "how often to print idle message when no packets arrive")
	flag.Parse()

	r, err := radio.NewRadio(*port)
	if err != nil {
		log.Fatalf("open radio: %v", err)
	}
	defer r.Close()

	log.Printf("rx listener started on %s", *port)
	lastIdleLog := time.Now()

	for {
		fromRadioPackets, err := r.ReadResponse(true)
		if err != nil {
			log.Printf("[ERR] read response: %v", err)
			time.Sleep(300 * time.Millisecond)
			continue
		}

		if len(fromRadioPackets) == 0 {
			if time.Since(lastIdleLog) >= *idleLog {
				log.Printf("[IDLE] no packets")
				lastIdleLog = time.Now()
			}
			continue
		}

		for _, fr := range fromRadioPackets {
			logFromRadio(fr)
		}
	}
}

func logFromRadio(fr *pb.FromRadio) {
	switch v := fr.GetPayloadVariant().(type) {
	case *pb.FromRadio_Packet:
		logMeshPacket(v.Packet)
	case *pb.FromRadio_LogRecord:
		if v.LogRecord == nil {
			log.Printf("[EVT] log_record=nil")
			return
		}
		log.Printf(
			"[EVT] log level=%s source=%s msg=%q",
			v.LogRecord.GetLevel().String(),
			v.LogRecord.GetSource(),
			v.LogRecord.GetMessage(),
		)
	case *pb.FromRadio_QueueStatus:
		if v.QueueStatus == nil {
			log.Printf("[EVT] queue_status=nil")
			return
		}
		log.Printf(
			"[EVT] queue res=%d free=%d/%d mesh_packet_id=%d",
			v.QueueStatus.GetRes(),
			v.QueueStatus.GetFree(),
			v.QueueStatus.GetMaxlen(),
			v.QueueStatus.GetMeshPacketId(),
		)
	case *pb.FromRadio_Rebooted:
		log.Printf("[EVT] rebooted=%t", v.Rebooted)
	case *pb.FromRadio_ConfigCompleteId:
		log.Printf("[EVT] config_complete_id=%d", v.ConfigCompleteId)
	case *pb.FromRadio_MyInfo:
		if v.MyInfo == nil {
			log.Printf("[EVT] my_info=nil")
			return
		}
		log.Printf("[EVT] my_info node_num=!%08x", v.MyInfo.GetMyNodeNum())
	case *pb.FromRadio_NodeInfo:
		if v.NodeInfo == nil {
			log.Printf("[EVT] node_info=nil")
			return
		}
		log.Printf(
			"[EVT] node_info node_num=!%08x user=%q",
			v.NodeInfo.GetNum(),
			v.NodeInfo.GetUser().GetLongName(),
		)
	default:
		log.Printf("[EVT] variant=%T", fr.GetPayloadVariant())
	}
}

func logMeshPacket(mp *pb.MeshPacket) {
	if mp == nil {
		log.Printf("[PKT] nil")
		return
	}

	portLabel := "UNKNOWN"
	payloadLen := 0
	if decoded := mp.GetDecoded(); decoded != nil {
		portLabel = decoded.GetPortnum().String()
		payloadLen = len(decoded.GetPayload())
	}

	log.Printf(
		"[PKT] from=!%08x to=!%08x ch=%d id=%d hop=%d rssi=%ddBm snr=%.2f rx_time=%s port=%s bytes=%d",
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

	decoded := mp.GetDecoded()
	if decoded == nil {
		return
	}

	switch decoded.GetPortnum() {
	case pb.PortNum_TEXT_MESSAGE_APP:
		text := strings.TrimSpace(string(decoded.GetPayload()))
		log.Printf("[MSG] text=%q", text)
	case pb.PortNum_TELEMETRY_APP:
		logTelemetry(decoded.GetPayload())
	}
}

func logTelemetry(payload []byte) {
	var t pb.Telemetry
	if err := proto.Unmarshal(payload, &t); err != nil {
		log.Printf("[TEL] decode_error=%v", err)
		return
	}

	tstamp := formatUnixSeconds(t.GetTime())

	if dm := t.GetDeviceMetrics(); dm != nil {
		log.Printf(
			"[TEL] type=device time=%s batt=%d%% volt=%.2fV ch_util=%.2f%% air_tx=%.2f%% uptime=%ds",
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
		log.Printf(
			"[TEL] type=local time=%s uptime=%ds nodes=%d/%d pkts_tx=%d pkts_rx=%d bad_rx=%d ch_util=%.2f%% air_tx=%.2f%% noise_floor=%ddBm",
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
		log.Printf(
			"[TEL] type=env time=%s temp=%.2fC hum=%.2f%% pressure=%.2fhPa",
			tstamp,
			em.GetTemperature(),
			em.GetRelativeHumidity(),
			em.GetBarometricPressure(),
		)
		return
	}

	log.Printf("[TEL] type=other time=%s variant=%T", tstamp, t.GetVariant())
}

func formatUnixSeconds(ts uint32) string {
	if ts == 0 {
		return "-"
	}
	return time.Unix(int64(ts), 0).UTC().Format(time.RFC3339)
}
