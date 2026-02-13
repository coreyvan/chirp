package node

import (
	"strings"
	"testing"

	pb "github.com/coreyvan/chirp/protogen/meshtastic"
)

func TestRenderMeshPacketIncludesPacketAndMessage(t *testing.T) {
	lines := RenderMeshPacket(&pb.MeshPacket{
		From: 1,
		To:   2,
		PayloadVariant: &pb.MeshPacket_Decoded{
			Decoded: &pb.Data{
				Portnum: pb.PortNum_TEXT_MESSAGE_APP,
				Payload: []byte(" hello "),
			},
		},
	})

	if len(lines) != 2 {
		t.Fatalf("line count = %d, want 2", len(lines))
	}
	if lines[0].Label != "PKT" || lines[0].Category != StreamCategoryPacket {
		t.Fatalf("unexpected first line: %+v", lines[0])
	}
	if lines[1].Label != "MSG" || lines[1].Category != StreamCategoryMessage {
		t.Fatalf("unexpected second line: %+v", lines[1])
	}
	if !strings.Contains(lines[1].Message, `text="hello"`) {
		t.Fatalf("unexpected msg line: %+v", lines[1])
	}
}

func TestRenderFromRadioMetadata(t *testing.T) {
	lines := RenderFromRadio(&pb.FromRadio{
		PayloadVariant: &pb.FromRadio_Metadata{
			Metadata: &pb.DeviceMetadata{
				FirmwareVersion: "2.6.0",
				HasWifi:         true,
			},
		},
	})

	if len(lines) != 1 {
		t.Fatalf("line count = %d, want 1", len(lines))
	}
	if lines[0].Label != "EVT" || lines[0].Category != StreamCategoryEvent {
		t.Fatalf("unexpected line: %+v", lines[0])
	}
	if !strings.Contains(lines[0].Message, `metadata fw="2.6.0"`) {
		t.Fatalf("unexpected metadata line: %+v", lines[0])
	}
}
