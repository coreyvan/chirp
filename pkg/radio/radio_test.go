package radio

import (
	"io"
	"os"
	"testing"
	"time"

	pb "github.com/coreyvan/chirp/protogen/meshtastic"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

type readStep struct {
	b   []byte
	n   int
	err error
}

type mockStreamer struct {
	readSteps       []readStep
	readIndex       int
	writes          [][]byte
	setReadTimeouts []time.Duration
	closeCalled     bool
	writeErr        error
	readErr         error
}

func (m *mockStreamer) Read(p []byte) (int, error) {
	if m.readErr != nil {
		return 0, m.readErr
	}

	if m.readIndex >= len(m.readSteps) {
		return 0, io.EOF
	}

	step := m.readSteps[m.readIndex]
	m.readIndex++

	if len(step.b) > 0 && len(p) > 0 {
		p[0] = step.b[0]
	}

	return step.n, step.err
}

func (m *mockStreamer) Write(p []byte) (int, error) {
	if m.writeErr != nil {
		return 0, m.writeErr
	}

	b := make([]byte, len(p))
	copy(b, p)
	m.writes = append(m.writes, b)
	return len(p), nil
}

func (m *mockStreamer) Close() error {
	m.closeCalled = true
	return nil
}

func (m *mockStreamer) SetReadTimeout(d time.Duration) error {
	m.setReadTimeouts = append(m.setReadTimeouts, d)
	return nil
}

func frame(payload []byte) []byte {
	header := []byte{start1, start2, byte(len(payload) >> 8), byte(len(payload))}
	out := make([]byte, 0, len(header)+len(payload))
	out = append(out, header...)
	out = append(out, payload...)
	return out
}

func stepsFromBytes(b []byte) []readStep {
	steps := make([]readStep, 0, len(b)+1)
	for _, c := range b {
		steps = append(steps, readStep{b: []byte{c}, n: 1, err: nil})
	}
	steps = append(steps, readStep{n: 0, err: io.EOF})
	return steps
}

func decodeToRadio(t *testing.T, packet []byte) *pb.ToRadio {
	t.Helper()
	require.GreaterOrEqual(t, len(packet), headerLen)
	require.Equal(t, start1, packet[0])
	require.Equal(t, start2, packet[1])

	var tr pb.ToRadio
	require.NoError(t, proto.Unmarshal(packet[headerLen:], &tr))
	return &tr
}

func TestSendPacketAddsHeader(t *testing.T) {
	m := &mockStreamer{}
	r := &Radio{streamer: m}

	payload := []byte{0x10, 0x20, 0x30}
	require.NoError(t, r.SendPacket(payload))
	require.Len(t, m.writes, 1)

	got := m.writes[0]
	require.Equal(t, []byte{start1, start2, 0x00, 0x03, 0x10, 0x20, 0x30}, got)
}

func TestReadResponseParsesSinglePacket(t *testing.T) {
	msg := pb.FromRadio{
		PayloadVariant: &pb.FromRadio_MyInfo{
			MyInfo: &pb.MyNodeInfo{MyNodeNum: 0x01020304},
		},
	}
	payload, err := proto.Marshal(&msg)
	require.NoError(t, err)

	streamBytes := append([]byte{0x00, 0x11, 0x22}, frame(payload)...)
	m := &mockStreamer{readSteps: stepsFromBytes(streamBytes)}
	r := &Radio{streamer: m}

	packets, err := r.ReadResponse(false)
	require.NoError(t, err)
	require.Len(t, packets, 1)
	require.Equal(t, uint32(0x01020304), packets[0].GetMyInfo().GetMyNodeNum())
	require.Empty(t, m.setReadTimeouts)
}

func TestReadResponseTimeoutSetsReadTimeoutAndReturnsOnDeadline(t *testing.T) {
	m := &mockStreamer{
		readSteps: []readStep{
			{n: 0, err: os.ErrDeadlineExceeded},
		},
	}
	r := &Radio{streamer: m}

	packets, err := r.ReadResponse(true)
	require.NoError(t, err)
	require.Empty(t, packets)
	require.Equal(t, []time.Duration{readResponsePoll}, m.setReadTimeouts)
}

func TestReadResponseIgnoresOversizedPacket(t *testing.T) {
	oversized := []byte{start1, start2, 0x02, 0x01, 0x00, 0x00, 0x00}
	m := &mockStreamer{readSteps: stepsFromBytes(oversized)}
	r := &Radio{streamer: m}

	packets, err := r.ReadResponse(false)
	require.NoError(t, err)
	require.Empty(t, packets)
}

func TestGetRadioInfoSendsWantConfigAndParsesResponse(t *testing.T) {
	response := pb.FromRadio{
		PayloadVariant: &pb.FromRadio_MyInfo{
			MyInfo: &pb.MyNodeInfo{MyNodeNum: 0x0badf00d},
		},
	}
	payload, err := proto.Marshal(&response)
	require.NoError(t, err)

	m := &mockStreamer{readSteps: stepsFromBytes(frame(payload))}
	r := &Radio{streamer: m}

	packets, err := r.GetRadioInfo()
	require.NoError(t, err)
	require.Len(t, packets, 1)
	require.Len(t, m.writes, wakeSendAttempts)

	written := decodeToRadio(t, m.writes[0])
	require.Equal(t, uint32(radioInfoConfigID), written.GetWantConfigId())
}

func TestGetNodeNumSetsNodeNum(t *testing.T) {
	response := pb.FromRadio{
		PayloadVariant: &pb.FromRadio_MyInfo{
			MyInfo: &pb.MyNodeInfo{MyNodeNum: 12345},
		},
	}
	payload, err := proto.Marshal(&response)
	require.NoError(t, err)

	m := &mockStreamer{readSteps: stepsFromBytes(frame(payload))}
	r := &Radio{streamer: m}

	require.NoError(t, r.getNodeNum())
	require.Equal(t, uint32(12345), r.nodeNum)
}

func TestSendTextMessageBroadcast(t *testing.T) {
	m := &mockStreamer{}
	r := &Radio{streamer: m}

	require.NoError(t, r.SendTextMessage("hello", 0, 1))
	require.Len(t, m.writes, 1)

	toRadio := decodeToRadio(t, m.writes[0])
	packet := toRadio.GetPacket()
	require.NotNil(t, packet)
	require.Equal(t, broadcastNum, packet.GetTo())
	require.Equal(t, uint32(1), packet.GetChannel())
	require.Equal(t, defaultHopLimit, packet.GetHopLimit())
	require.Equal(t, pb.PortNum_TEXT_MESSAGE_APP, packet.GetDecoded().GetPortnum())
	require.Equal(t, "hello", string(packet.GetDecoded().GetPayload()))
}

func TestSendTextMessageRejectsLargePayload(t *testing.T) {
	m := &mockStreamer{}
	r := &Radio{streamer: m}

	large := make([]byte, 241)
	for i := range large {
		large[i] = 'a'
	}

	err := r.SendTextMessage(string(large), 0, 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "message too large")
	require.Empty(t, m.writes)
}

func TestSetRadioOwnerBuildsAdminPacket(t *testing.T) {
	m := &mockStreamer{}
	r := &Radio{streamer: m, nodeNum: 77}

	require.NoError(t, r.SetRadioOwner("Corey"))
	require.Len(t, m.writes, 1)

	toRadio := decodeToRadio(t, m.writes[0])
	packet := toRadio.GetPacket()
	require.NotNil(t, packet)
	require.Equal(t, uint32(77), packet.GetTo())
	require.Equal(t, pb.PortNum_ADMIN_APP, packet.GetDecoded().GetPortnum())

	var admin pb.AdminMessage
	require.NoError(t, proto.Unmarshal(packet.GetDecoded().GetPayload(), &admin))
	require.Equal(t, "Corey", admin.GetSetOwner().GetLongName())
	require.Equal(t, "Cor", admin.GetSetOwner().GetShortName())
}

func TestSetModemModeBuildsAdminConfigPacket(t *testing.T) {
	m := &mockStreamer{}
	r := &Radio{streamer: m, nodeNum: 33}

	require.NoError(t, r.SetModemMode("ls"))
	require.Len(t, m.writes, 1)

	toRadio := decodeToRadio(t, m.writes[0])
	packet := toRadio.GetPacket()
	require.NotNil(t, packet)

	var admin pb.AdminMessage
	require.NoError(t, proto.Unmarshal(packet.GetDecoded().GetPayload(), &admin))
	require.Equal(t, pb.Config_LoRaConfig_LONG_SLOW, admin.GetSetConfig().GetLora().GetModemPreset())
}

func TestSetModemModeRejectsInvalid(t *testing.T) {
	m := &mockStreamer{}
	r := &Radio{streamer: m, nodeNum: 33}

	err := r.SetModemMode("bad")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid modem mode")
	require.Empty(t, m.writes)
}

func TestSetLocationBuildsPositionPacket(t *testing.T) {
	m := &mockStreamer{}
	r := &Radio{streamer: m, nodeNum: 88}

	require.NoError(t, r.SetLocation(123, 456, 789))
	require.Len(t, m.writes, 1)

	toRadio := decodeToRadio(t, m.writes[0])
	packet := toRadio.GetPacket()
	require.NotNil(t, packet)
	require.Equal(t, uint32(88), packet.GetTo())
	require.Equal(t, pb.PortNum_POSITION_APP, packet.GetDecoded().GetPortnum())

	var pos pb.Position
	require.NoError(t, proto.Unmarshal(packet.GetDecoded().GetPayload(), &pos))
	require.Equal(t, int32(123), pos.GetLatitudeI())
	require.Equal(t, int32(456), pos.GetLongitudeI())
	require.Equal(t, int32(789), pos.GetAltitude())
}

func TestFactoryResetAndAlias(t *testing.T) {
	m := &mockStreamer{}
	r := &Radio{streamer: m, nodeNum: 19}

	require.NoError(t, r.FactoryReset())
	require.NoError(t, r.FactoryRest())
	require.Len(t, m.writes, 2)

	for _, w := range m.writes {
		toRadio := decodeToRadio(t, w)
		packet := toRadio.GetPacket()
		var admin pb.AdminMessage
		require.NoError(t, proto.Unmarshal(packet.GetDecoded().GetPayload(), &admin))
		require.Equal(t, int32(1), admin.GetFactoryResetDevice())
	}
}

func TestCloseHandlesNilStreamer(t *testing.T) {
	r := &Radio{}
	require.NoError(t, r.Close())
}

func TestCloseDelegatesToStreamer(t *testing.T) {
	m := &mockStreamer{}
	r := &Radio{streamer: m}
	require.NoError(t, r.Close())
	require.True(t, m.closeCalled)
}

