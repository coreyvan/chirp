package radio

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetRadioInfoIntegration(t *testing.T) {
	r, err := NewRadio("/dev/cu.usbmodem101")
	require.NoError(t, err)
	defer r.Close()

	responses, err := r.GetRadioInfo()
	require.NoError(t, err)
	require.NotEmpty(t, responses)
	t.Log(responses)
}
