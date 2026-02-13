package serial

import (
	"fmt"
	"time"

	sdkserial "go.bug.st/serial"
)

const (
	baudRate       int = 115200
	dataBits       int = 8
	stopBits       int = 0
	parityNoParity int = 0
)

type SerialStreamer struct {
	port   sdkserial.Port
	closer func() error
}

func NewSerialStreamer(device string) (*SerialStreamer, error) {
	port, err := sdkserial.Open(device, &sdkserial.Mode{
		BaudRate:          baudRate,
		DataBits:          dataBits,
		Parity:            sdkserial.Parity(parityNoParity),
		StopBits:          sdkserial.StopBits(stopBits),
		InitialStatusBits: &sdkserial.ModemOutputBits{},
	})
	if err != nil {
		return nil, fmt.Errorf("could not open serial connection: %w", err)
	}

	return &SerialStreamer{
		port:   port,
		closer: port.Close,
	}, nil
}

func (s *SerialStreamer) Close() error {
	return s.closer()
}

func GetPorts() ([]string, error) {
	p, err := sdkserial.GetPortsList()
	if err != nil {
		return nil, fmt.Errorf("could not get ports list: %w", err)
	}

	return p, nil
}

func (s *SerialStreamer) Read(p []byte) (int, error) {
	return s.port.Read(p)
}

func (s *SerialStreamer) Write(p []byte) (int, error) {
	n, err := s.port.Write(p)
	if err != nil {
		return 0, err
	}

	time.Sleep(100 * time.Millisecond)

	return n, nil
}
