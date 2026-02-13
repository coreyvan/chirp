package commands

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	pb "github.com/coreyvan/chirp/protogen/meshtastic"
	"google.golang.org/protobuf/proto"
)

type listenTestRadio struct {
	readResults [][]*pb.FromRadio
	readErrors  []error
	readIndex   int
	infoResults []*pb.FromRadio
	infoErr     error
	infoCalls   int
}

func (f *listenTestRadio) Close() error { return nil }

func (f *listenTestRadio) ReadResponse(bool) ([]*pb.FromRadio, error) {
	i := f.readIndex
	f.readIndex++

	var packets []*pb.FromRadio
	var err error
	if i < len(f.readResults) {
		packets = f.readResults[i]
	}
	if i < len(f.readErrors) {
		err = f.readErrors[i]
	}
	return packets, err
}

func (f *listenTestRadio) GetRadioInfo() ([]*pb.FromRadio, error) {
	f.infoCalls++
	return f.infoResults, f.infoErr
}
func (f *listenTestRadio) SendTextMessage(string, int64, int64) error { return nil }
func (f *listenTestRadio) SetRadioOwner(string) error                 { return nil }
func (f *listenTestRadio) SetModemMode(string) error                  { return nil }
func (f *listenTestRadio) SetLocation(int32, int32, int32) error      { return nil }
func (f *listenTestRadio) FactoryReset() error                        { return nil }

func TestListenCommandRejectsNonPositiveIdleLog(t *testing.T) {
	cliCtx := &Context{Port: "/dev/test", Timeout: time.Second}
	cmd := newListenCommand(cliCtx, func(string) (Radio, error) {
		t.Fatalf("radio opener should not be called for invalid flags")
		return nil, nil
	})
	cmd.SetArgs([]string{"--idle-log=0s"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if ExitCode(err) != 2 {
		t.Fatalf("exit code = %d, want 2", ExitCode(err))
	}
	if !strings.Contains(err.Error(), "--idle-log must be greater than 0") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestLogMeshPacketTelemetryFilters(t *testing.T) {
	telPayload, err := proto.Marshal(&pb.Telemetry{
		Variant: &pb.Telemetry_DeviceMetrics{
			DeviceMetrics: &pb.DeviceMetrics{},
		},
	})
	if err != nil {
		t.Fatalf("marshal telemetry: %v", err)
	}

	packet := &pb.MeshPacket{
		From: 1,
		To:   2,
		PayloadVariant: &pb.MeshPacket_Decoded{
			Decoded: &pb.Data{
				Portnum: pb.PortNum_TELEMETRY_APP,
				Payload: telPayload,
			},
		},
	}

	var out bytes.Buffer
	logMeshPacket(&out, packet, &listenOptions{})
	got := out.String()
	if !strings.Contains(got, "[PKT]") || !strings.Contains(got, "[TEL]") {
		t.Fatalf("expected packet and telemetry logs, got:\n%s", got)
	}

	out.Reset()
	logMeshPacket(&out, packet, &listenOptions{noPackets: true})
	got = out.String()
	if strings.Contains(got, "[PKT]") {
		t.Fatalf("did not expect packet log when --no-packets, got:\n%s", got)
	}
	if !strings.Contains(got, "[TEL]") {
		t.Fatalf("expected telemetry log to remain visible, got:\n%s", got)
	}

	out.Reset()
	logMeshPacket(&out, packet, &listenOptions{noTelemetry: true})
	got = out.String()
	if !strings.Contains(got, "[PKT]") {
		t.Fatalf("expected packet log, got:\n%s", got)
	}
	if strings.Contains(got, "[TEL]") {
		t.Fatalf("did not expect telemetry log when --no-telemetry, got:\n%s", got)
	}
}

func TestLogFromRadioEventFilter(t *testing.T) {
	fr := &pb.FromRadio{
		PayloadVariant: &pb.FromRadio_Rebooted{Rebooted: true},
	}

	var out bytes.Buffer
	logFromRadio(&out, fr, &listenOptions{})
	if !strings.Contains(out.String(), "[EVT] rebooted=true") {
		t.Fatalf("expected event line, got:\n%s", out.String())
	}

	out.Reset()
	logFromRadio(&out, fr, &listenOptions{noEvents: true})
	if out.Len() != 0 {
		t.Fatalf("expected no output with --no-events, got:\n%s", out.String())
	}
}

func TestLogFromRadioFileInfoEvent(t *testing.T) {
	fr := &pb.FromRadio{
		PayloadVariant: &pb.FromRadio_FileInfo{
			FileInfo: &pb.FileInfo{
				FileName:  "/prefs/config.proto",
				SizeBytes: 1536,
			},
		},
	}

	var out bytes.Buffer
	logFromRadio(&out, fr, &listenOptions{})
	got := out.String()
	if !strings.Contains(got, `[EVT] file_info name="/prefs/config.proto" size_bytes=1536`) {
		t.Fatalf("expected formatted file_info event, got:\n%s", got)
	}

	out.Reset()
	logFromRadio(&out, fr, &listenOptions{noEvents: true})
	if out.Len() != 0 {
		t.Fatalf("expected no output with --no-events, got:\n%s", out.String())
	}
}

func TestLogFromRadioMetadataEvent(t *testing.T) {
	fr := &pb.FromRadio{
		PayloadVariant: &pb.FromRadio_Metadata{
			Metadata: &pb.DeviceMetadata{
				FirmwareVersion:    "2.6.0",
				DeviceStateVersion: 7,
				HasWifi:            true,
				HasBluetooth:       true,
				HasEthernet:        false,
				HasRemoteHardware:  true,
				HasPKC:             true,
			},
		},
	}

	var out bytes.Buffer
	logFromRadio(&out, fr, &listenOptions{})
	got := out.String()
	if !strings.Contains(got, `[EVT] metadata fw="2.6.0" state_ver=7`) {
		t.Fatalf("expected metadata summary, got:\n%s", got)
	}
	if !strings.Contains(got, "wifi=true bt=true") {
		t.Fatalf("expected metadata capabilities, got:\n%s", got)
	}
}

func TestLogFromRadioChannelEvent(t *testing.T) {
	fr := &pb.FromRadio{
		PayloadVariant: &pb.FromRadio_Channel{
			Channel: &pb.Channel{
				Index: 1,
				Role:  pb.Channel_PRIMARY,
				Settings: &pb.ChannelSettings{
					Name:            "primary",
					Id:              42,
					UplinkEnabled:   true,
					DownlinkEnabled: false,
				},
			},
		},
	}

	var out bytes.Buffer
	logFromRadio(&out, fr, &listenOptions{})
	got := out.String()
	if !strings.Contains(got, `[EVT] channel index=1 role=PRIMARY name="primary" id=42 uplink=true downlink=false`) {
		t.Fatalf("expected channel summary, got:\n%s", got)
	}
}

func TestLogFromRadioConfigEvent(t *testing.T) {
	fr := &pb.FromRadio{
		PayloadVariant: &pb.FromRadio_Config{
			Config: &pb.Config{
				PayloadVariant: &pb.Config_Lora{
					Lora: &pb.Config_LoRaConfig{},
				},
			},
		},
	}

	var out bytes.Buffer
	logFromRadio(&out, fr, &listenOptions{})
	got := out.String()
	if !strings.Contains(got, "[EVT] config section=lora") {
		t.Fatalf("expected config section, got:\n%s", got)
	}
}

func TestLogFromRadioModuleConfigEvent(t *testing.T) {
	fr := &pb.FromRadio{
		PayloadVariant: &pb.FromRadio_ModuleConfig{
			ModuleConfig: &pb.ModuleConfig{
				PayloadVariant: &pb.ModuleConfig_Mqtt{
					Mqtt: &pb.ModuleConfig_MQTTConfig{},
				},
			},
		},
	}

	var out bytes.Buffer
	logFromRadio(&out, fr, &listenOptions{})
	got := out.String()
	if !strings.Contains(got, "[EVT] module_config section=mqtt") {
		t.Fatalf("expected module_config section, got:\n%s", got)
	}
}

func TestRunListenEmitsStartupAndError(t *testing.T) {
	r := &listenTestRadio{
		readResults: [][]*pb.FromRadio{{}},
		readErrors:  []error{errors.New("boom")},
	}
	opts := &listenOptions{idleLog: time.Hour}

	var out bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := runListen(ctx, &out, r, "/dev/test", opts)
	if err != nil {
		t.Fatalf("runListen() error = %v", err)
	}

	logs := out.String()
	if !strings.Contains(logs, "rx listener started on /dev/test") {
		t.Fatalf("missing startup log:\n%s", logs)
	}
	if !strings.Contains(logs, "[ERR] read response: boom") {
		t.Fatalf("missing read error log:\n%s", logs)
	}
	if r.infoCalls != 1 {
		t.Fatalf("GetRadioInfo() calls = %d, want 1", r.infoCalls)
	}
}

func TestRunListenEmitsIdle(t *testing.T) {
	r := &listenTestRadio{
		readResults: [][]*pb.FromRadio{{}, {}, {}, {}},
	}
	opts := &listenOptions{idleLog: 1 * time.Millisecond}

	var out bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Millisecond)
	defer cancel()

	err := runListen(ctx, &out, r, "/dev/test", opts)
	if err != nil {
		t.Fatalf("runListen() error = %v", err)
	}

	logs := out.String()
	if !strings.Contains(logs, "[IDLE] no packets") {
		t.Fatalf("missing idle log:\n%s", logs)
	}
}

func TestRunListenLogsGetRadioInfoError(t *testing.T) {
	r := &listenTestRadio{
		infoErr: errors.New("info fail"),
	}
	opts := &listenOptions{idleLog: time.Hour}

	var out bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	err := runListen(ctx, &out, r, "/dev/test", opts)
	if err != nil {
		t.Fatalf("runListen() error = %v", err)
	}

	logs := out.String()
	if !strings.Contains(logs, "[ERR] get radio info: info fail") {
		t.Fatalf("missing radio info error log:\n%s", logs)
	}
}
