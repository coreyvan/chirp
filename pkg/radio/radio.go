package radio

import (
	"errors"
	"io"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/coreyvan/chirp/pkg/serial"
	pb "github.com/coreyvan/chirp/protogen/meshtastic"
	"google.golang.org/protobuf/proto"
)

const (
	start1                = byte(0x94)
	start2                = byte(0xc3)
	headerLen             = 4
	maxToFromRadioSize    = 512
	maxTextMessageLen     = 240
	maxPacketID           = 2386827
	broadcastNum          = uint32(0xffffffff)
	defaultHopLimit       = uint32(3)
	radioInfoConfigID     = 42
	radioInfoMaxPolls     = 5
	radioInfoPollInterval = 1 * time.Second
	readResponseTimeout   = 2 * time.Second
	readResponsePoll      = 200 * time.Millisecond
	wakeSendAttempts      = 1
	wakeSendInterval      = 300 * time.Millisecond
)

var (
	errNodeNumUnknown  = errors.New("failed to determine node number")
	errRadioInfoFailed = errors.New("failed to get radio info")
	errMessageTooLarge = errors.New("message too large")
	errNameTooShort    = errors.New("name too short")
	errInvalidModem    = errors.New("invalid modem mode")
)

type Streamer interface {
	Read(p []byte) (int, error)
	Write(p []byte) (int, error)
	Close() error
	SetReadTimeout(d time.Duration) error
}

// Radio holds the port and serial io.ReadWriteCloser struct to maintain one serial connection.
type Radio struct {
	streamer Streamer
	nodeNum  uint32
}

func NewRadio(device string) (*Radio, error) {
	streamer, err := serial.NewSerialStreamer(device)
	if err != nil {
		return nil, err
	}

	return &Radio{streamer: streamer}, nil
}

// Init initializes serial connection and caches the local node number.
func (r *Radio) Init(device string) error {
	streamer, err := serial.NewSerialStreamer(device)
	if err != nil {
		return err
	}

	r.streamer = streamer
	return r.getNodeNum()
}

func (r *Radio) Close() error {
	if r.streamer == nil {
		return nil
	}

	return r.streamer.Close()
}

// getNodeNum queries the radio and stores the local node number.
func (r *Radio) getNodeNum() error {
	radioResponses, err := r.GetRadioInfo()
	if err != nil {
		return err
	}

	var nodeNum uint32
	for _, response := range radioResponses {
		if myInfo := response.GetMyInfo(); myInfo != nil {
			nodeNum = myInfo.GetMyNodeNum()
			break
		}
	}

	if nodeNum == 0 {
		return errNodeNumUnknown
	}

	r.nodeNum = nodeNum
	return nil
}

// SendPacket takes a protobuf packet, constructs the appropriate header, and sends it to the radio.
func (r *Radio) SendPacket(protobufPacket []byte) error {
	packetLength := len(protobufPacket)
	header := []byte{start1, start2, byte(packetLength>>8) & 0xff, byte(packetLength) & 0xff}

	radioPacket := make([]byte, 0, len(header)+len(protobufPacket))
	radioPacket = append(radioPacket, header...)
	radioPacket = append(radioPacket, protobufPacket...)
	n, err := r.streamer.Write(radioPacket)
	if err != nil {
		return err
	}
	log.Printf("wrote %d bytes", n)

	return nil
}

// ReadResponse reads any responses in the serial port, converts them to FromRadio protobufs and returns them.
func (r *Radio) ReadResponse(timeout bool) ([]*pb.FromRadio, error) {
	if timeout {
		if err := r.streamer.SetReadTimeout(readResponsePoll); err != nil {
			return nil, err
		}
	}

	b := make([]byte, 1)
	buf := make([]byte, 0, maxToFromRadioSize+headerLen)
	var previousByte byte
	hasPrevious := false
	repeatByteCounter := 0

	var fromRadioPackets []*pb.FromRadio

	deadline := time.Now().Add(readResponseTimeout)
	for {
		n, err := r.streamer.Read(b)

		if n > 0 {
			currentByte := b[0]
			if hasPrevious && currentByte == previousByte {
				repeatByteCounter++
			} else {
				repeatByteCounter = 0
			}
			previousByte = currentByte
			hasPrevious = true
		}

		shouldBreakOnRepeat := repeatByteCounter > 20 && len(buf) < headerLen
		if err == io.EOF || shouldBreakOnRepeat || errors.Is(err, os.ErrDeadlineExceeded) {
			break
		} else if err != nil {
			return nil, err
		}

		if n == 0 {
			if timeout && time.Now().After(deadline) {
				break
			}
			continue
		}

		buf = append(buf, b[0])

		switch len(buf) {
		case 1:
			if buf[0] != start1 {
				buf = buf[:0]
			}
		case 2:
			if buf[1] != start2 {
				buf = buf[:0]
			}
		}

		if len(buf) < headerLen {
			continue
		}

		packetLength := int(buf[2])<<8 | int(buf[3])
		if len(buf) == headerLen && packetLength > maxToFromRadioSize {
			buf = buf[:0]
			continue
		}

		if len(buf) == packetLength+headerLen {
			fromRadio := pb.FromRadio{}
			if err := proto.Unmarshal(buf[headerLen:], &fromRadio); err != nil {
				return nil, err
			}

			fromRadioPackets = append(fromRadioPackets, &fromRadio)
			buf = buf[:0]
		}
	}

	return fromRadioPackets, nil
}

// GetRadioInfo retrieves information from the radio including config and adjacent node information.
func (r *Radio) GetRadioInfo() ([]*pb.FromRadio, error) {
	nodeInfo := pb.ToRadio{PayloadVariant: &pb.ToRadio_WantConfigId{WantConfigId: radioInfoConfigID}}

	out, err := proto.Marshal(&nodeInfo)
	if err != nil {
		return nil, err
	}

	for attempt := 0; attempt < wakeSendAttempts; attempt++ {
		if err := r.SendPacket(out); err != nil {
			return nil, err
		}
		time.Sleep(wakeSendInterval)
	}

	for checks := 0; checks <= radioInfoMaxPolls; checks++ {
		radioResponses, err := r.ReadResponse(true)
		if err != nil {
			return nil, err
		}

		if len(radioResponses) > 0 {
			return radioResponses, nil
		}

		time.Sleep(radioInfoPollInterval)
	}

	return nil, errRadioInfoFailed
}

// createAdminPacket builds an admin message packet to send to the radio.
func (r *Radio) createAdminPacket(nodeNum uint32, payload []byte) ([]byte, error) {
	radioMessage := pb.ToRadio{
		PayloadVariant: &pb.ToRadio_Packet{
			Packet: &pb.MeshPacket{
				To:      nodeNum,
				WantAck: true,
				PayloadVariant: &pb.MeshPacket_Decoded{
					Decoded: &pb.Data{
						Payload:      payload,
						Portnum:      pb.PortNum_ADMIN_APP,
						WantResponse: true,
					},
				},
			},
		},
	}

	packetOut, err := proto.Marshal(&radioMessage)
	if err != nil {
		return nil, err
	}

	return packetOut, nil
}

// SendTextMessage sends a text message to another radio (or broadcast if to == 0).
func (r *Radio) SendTextMessage(message string, to int64, channel int64) error {
	address := broadcastNum
	if to != 0 {
		address = uint32(to)
	}

	if len(message) > maxTextMessageLen {
		return errMessageTooLarge
	}

	packetID := uint32(rand.New(rand.NewSource(time.Now().UnixNano())).Intn(maxPacketID) + 1)
	radioMessage := pb.ToRadio{
		PayloadVariant: &pb.ToRadio_Packet{
			Packet: &pb.MeshPacket{
				To:       address,
				WantAck:  true,
				Id:       packetID,
				Channel:  uint32(channel),
				HopLimit: defaultHopLimit,
				PayloadVariant: &pb.MeshPacket_Decoded{
					Decoded: &pb.Data{
						Payload: []byte(message),
						Portnum: pb.PortNum_TEXT_MESSAGE_APP,
					},
				},
			},
		},
	}

	out, err := proto.Marshal(&radioMessage)
	if err != nil {
		return err
	}

	return r.SendPacket(out)
}

// SetRadioOwner sets the owner name reported by this radio.
func (r *Radio) SetRadioOwner(name string) error {
	if len(name) <= 2 {
		return errNameTooShort
	}

	shortName := name
	if len(shortName) > 3 {
		shortName = shortName[:3]
	}

	adminPacket := pb.AdminMessage{
		PayloadVariant: &pb.AdminMessage_SetOwner{
			SetOwner: &pb.User{
				LongName:  name,
				ShortName: shortName,
			},
		},
	}

	out, err := proto.Marshal(&adminPacket)
	if err != nil {
		return err
	}

	packet, err := r.createAdminPacket(r.nodeNum, out)
	if err != nil {
		return err
	}

	return r.SendPacket(packet)
}

// SetModemMode sets the LoRa modem preset.
func (r *Radio) SetModemMode(mode string) error {
	var modemSetting pb.Config_LoRaConfig_ModemPreset
	switch mode {
	case "lf":
		modemSetting = pb.Config_LoRaConfig_LONG_FAST
	case "ls":
		modemSetting = pb.Config_LoRaConfig_LONG_SLOW
	case "vls":
		modemSetting = pb.Config_LoRaConfig_VERY_LONG_SLOW
	case "ms":
		modemSetting = pb.Config_LoRaConfig_MEDIUM_SLOW
	case "mf":
		modemSetting = pb.Config_LoRaConfig_MEDIUM_FAST
	case "sl":
		modemSetting = pb.Config_LoRaConfig_SHORT_SLOW
	case "sf":
		modemSetting = pb.Config_LoRaConfig_SHORT_FAST
	case "lm":
		modemSetting = pb.Config_LoRaConfig_LONG_MODERATE
	default:
		return errInvalidModem
	}

	adminPacket := pb.AdminMessage{
		PayloadVariant: &pb.AdminMessage_SetConfig{
			SetConfig: &pb.Config{
				PayloadVariant: &pb.Config_Lora{
					Lora: &pb.Config_LoRaConfig{
						ModemPreset: modemSetting,
					},
				},
			},
		},
	}

	out, err := proto.Marshal(&adminPacket)
	if err != nil {
		return err
	}

	packet, err := r.createAdminPacket(r.nodeNum, out)
	if err != nil {
		return err
	}

	return r.SendPacket(packet)
}

// SetLocation sets a fixed position payload for the current node.
func (r *Radio) SetLocation(lat int32, long int32, alt int32) error {
	latCopy := lat
	longCopy := long
	altCopy := alt
	positionPacket := pb.Position{
		LatitudeI:  &latCopy,
		LongitudeI: &longCopy,
		Altitude:   &altCopy,
	}

	out, err := proto.Marshal(&positionPacket)
	if err != nil {
		return err
	}

	radioMessage := pb.ToRadio{
		PayloadVariant: &pb.ToRadio_Packet{
			Packet: &pb.MeshPacket{
				To:      r.nodeNum,
				WantAck: true,
				PayloadVariant: &pb.MeshPacket_Decoded{
					Decoded: &pb.Data{
						Payload:      out,
						Portnum:      pb.PortNum_POSITION_APP,
						WantResponse: true,
					},
				},
			},
		},
	}

	packet, err := proto.Marshal(&radioMessage)
	if err != nil {
		return err
	}

	return r.SendPacket(packet)
}

// FactoryReset sends a factory reset command to the radio.
func (r *Radio) FactoryReset() error {
	adminPacket := pb.AdminMessage{
		PayloadVariant: &pb.AdminMessage_FactoryResetDevice{
			FactoryResetDevice: 1,
		},
	}

	out, err := proto.Marshal(&adminPacket)
	if err != nil {
		return err
	}

	packet, err := r.createAdminPacket(r.nodeNum, out)
	if err != nil {
		return err
	}

	return r.SendPacket(packet)
}

// FactoryRest is kept for compatibility with older callers that used a typo.
func (r *Radio) FactoryRest() error {
	return r.FactoryReset()
}
