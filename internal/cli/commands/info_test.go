package commands

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	appnode "github.com/coreyvan/chirp/internal/app/node"
	pb "github.com/coreyvan/chirp/protogen/meshtastic"
)

func TestBuildInfoSummary(t *testing.T) {
	responses := []*pb.FromRadio{
		{PayloadVariant: &pb.FromRadio_MyInfo{MyInfo: &pb.MyNodeInfo{MyNodeNum: 0x16c3f424}}},
		{PayloadVariant: &pb.FromRadio_Metadata{Metadata: &pb.DeviceMetadata{
			FirmwareVersion: "2.6.0",
			HwModel:         pb.HardwareModel_TRACKER_T1000_E,
			Role:            pb.Config_DeviceConfig_ROUTER,
		}}},
		{PayloadVariant: &pb.FromRadio_NodeInfo{NodeInfo: &pb.NodeInfo{}}},
		{PayloadVariant: &pb.FromRadio_Channel{Channel: &pb.Channel{}}},
		{PayloadVariant: &pb.FromRadio_Config{Config: &pb.Config{}}},
		{PayloadVariant: &pb.FromRadio_ModuleConfig{ModuleConfig: &pb.ModuleConfig{}}},
	}

	s := appnode.BuildInfoSummary(responses)
	if s.Responses != 6 || s.Nodes != 1 || s.Channels != 1 || s.Configs != 1 || s.ModuleConfigs != 1 {
		t.Fatalf("unexpected summary counts: %+v", s)
	}
	if s.MyNode != "!16c3f424" {
		t.Fatalf("my node = %q, want !16c3f424", s.MyNode)
	}
	if s.Firmware != "2.6.0" {
		t.Fatalf("firmware = %q, want 2.6.0", s.Firmware)
	}
	if s.HWModel == "-" || s.Role == "-" {
		t.Fatalf("expected model/role to be populated: %+v", s)
	}
}

func TestWriteInfoJSONIncludesSummaryAndResponses(t *testing.T) {
	summary := appnode.InfoSummary{Responses: 1, MyNode: "!00000001"}
	responses := []*pb.FromRadio{
		{PayloadVariant: &pb.FromRadio_Rebooted{Rebooted: true}},
	}

	var out bytes.Buffer
	if err := writeInfoJSON(summary, responses, &out); err != nil {
		t.Fatalf("writeInfoJSON() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if _, ok := got["summary"]; !ok {
		t.Fatalf("missing summary field in JSON output: %s", out.String())
	}
	if _, ok := got["responses"]; !ok {
		t.Fatalf("missing responses field in JSON output: %s", out.String())
	}
}

func TestPrintKeyValueTable(t *testing.T) {
	var out bytes.Buffer
	err := printKeyValueTable(&out, []keyValueRow{
		{Key: "alpha", Value: "one"},
		{Key: "b", Value: "two"},
	})
	if err != nil {
		t.Fatalf("printKeyValueTable() error = %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "alpha  one") || !strings.Contains(got, "b      two") {
		t.Fatalf("unexpected table output:\n%s", got)
	}
}
