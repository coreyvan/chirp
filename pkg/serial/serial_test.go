//go:build integration

package serial

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPortsList(t *testing.T) {
	ports, err := GetPorts()
	require.NoError(t, err)
	for _, p := range ports {
		t.Log(p)
	}
	require.NotEmpty(t, ports)
}

func TestOpenPort(t *testing.T) {
	port, err := NewSerialStreamer("/dev/cu.usbmodem101")
	require.NoError(t, err)
	assert.NotNil(t, port)
	err = port.Close()
	require.NoError(t, err)
}
