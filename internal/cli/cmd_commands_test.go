package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	pb "github.com/coreyvan/chirp/protogen/meshtastic"
)

type commandTestRadio struct {
	factoryResetCalls int
	sendTextCalls     int
	sendTextMessage   string
	sendTextTo        int64
	sendTextChannel   int64
	setOwnerCalls     int
	setOwnerName      string
	setModemCalls     int
	setModemMode      string
	setLocationCalls  int
	setLocationLat    int32
	setLocationLon    int32
	setLocationAlt    int32
	infoCalls         int
	infoResponses     []*pb.FromRadio
}

func (f *commandTestRadio) Close() error { return nil }
func (f *commandTestRadio) ReadResponse(bool) ([]*pb.FromRadio, error) {
	return nil, nil
}
func (f *commandTestRadio) GetRadioInfo() ([]*pb.FromRadio, error) {
	f.infoCalls++
	return f.infoResponses, nil
}
func (f *commandTestRadio) SendTextMessage(message string, to int64, channel int64) error {
	f.sendTextCalls++
	f.sendTextMessage = message
	f.sendTextTo = to
	f.sendTextChannel = channel
	return nil
}
func (f *commandTestRadio) SetRadioOwner(name string) error {
	f.setOwnerCalls++
	f.setOwnerName = name
	return nil
}
func (f *commandTestRadio) SetModemMode(mode string) error {
	f.setModemCalls++
	f.setModemMode = mode
	return nil
}
func (f *commandTestRadio) SetLocation(lat int32, lon int32, alt int32) error {
	f.setLocationCalls++
	f.setLocationLat = lat
	f.setLocationLon = lon
	f.setLocationAlt = alt
	return nil
}
func (f *commandTestRadio) FactoryReset() error {
	f.factoryResetCalls++
	return nil
}

func TestSendTextRejectsEmptyMessage(t *testing.T) {
	cliCtx := &Context{Port: "/dev/test", Timeout: time.Second}
	cmd := newSendTextCommand(cliCtx, func(string) (Radio, error) {
		t.Fatalf("opener should not be called for invalid args")
		return nil, nil
	})
	cmd.SetArgs([]string{"--message", "   "})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected error")
	}
	if ExitCode(err) != 2 {
		t.Fatalf("exit code = %d, want 2", ExitCode(err))
	}
	if !strings.Contains(err.Error(), "--message cannot be empty") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestSetModemRejectsInvalidMode(t *testing.T) {
	cliCtx := &Context{Port: "/dev/test", Timeout: time.Second}
	cmd := newSetModemCommand(cliCtx, func(string) (Radio, error) {
		t.Fatalf("opener should not be called for invalid args")
		return nil, nil
	})
	cmd.SetArgs([]string{"--mode", "bad"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected error")
	}
	if ExitCode(err) != 2 {
		t.Fatalf("exit code = %d, want 2", ExitCode(err))
	}
	if !strings.Contains(err.Error(), "--mode must be one of") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestSetLocationRejectsOutOfRange(t *testing.T) {
	cliCtx := &Context{Port: "/dev/test", Timeout: time.Second}
	cmd := newSetLocationCommand(cliCtx, func(string) (Radio, error) {
		t.Fatalf("opener should not be called for invalid args")
		return nil, nil
	})
	cmd.SetArgs([]string{"--lat-i", "2147483648", "--lon-i", "0", "--alt", "0"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected error")
	}
	if ExitCode(err) != 2 {
		t.Fatalf("exit code = %d, want 2", ExitCode(err))
	}
	if !strings.Contains(err.Error(), "--lat-i must be in int32 range") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestFactoryResetCancelled(t *testing.T) {
	cliCtx := &Context{Port: "/dev/test", Timeout: time.Second}
	cmd := newFactoryResetCommand(cliCtx, func(string) (Radio, error) {
		t.Fatalf("opener should not be called when user declines")
		return nil, nil
	})
	cmd.SetIn(strings.NewReader("n\n"))

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected cancellation error")
	}
	if ExitCode(err) != 1 {
		t.Fatalf("exit code = %d, want 1", ExitCode(err))
	}
	if !strings.Contains(err.Error(), "factory-reset cancelled") {
		t.Fatalf("error = %q", err.Error())
	}
	if !strings.Contains(out.String(), "This action is destructive. Continue? [y/N]") {
		t.Fatalf("missing confirmation prompt in output: %q", out.String())
	}
}

func TestFactoryResetYesSkipsPromptAndRuns(t *testing.T) {
	cliCtx := &Context{Port: "/dev/test", Timeout: time.Second}
	r := &commandTestRadio{}
	cmd := newFactoryResetCommand(cliCtx, func(string) (Radio, error) {
		return r, nil
	})
	cmd.SetArgs([]string{"--yes"})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.factoryResetCalls != 1 {
		t.Fatalf("factory reset calls = %d, want 1", r.factoryResetCalls)
	}
	if !strings.Contains(out.String(), "factory reset command sent") {
		t.Fatalf("missing success output: %q", out.String())
	}
}

func TestFactoryResetPromptConfirmRuns(t *testing.T) {
	cliCtx := &Context{Port: "/dev/test", Timeout: time.Second}
	r := &commandTestRadio{}
	cmd := newFactoryResetCommand(cliCtx, func(string) (Radio, error) {
		return r, nil
	})
	cmd.SetIn(strings.NewReader("y\n"))

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.factoryResetCalls != 1 {
		t.Fatalf("factory reset calls = %d, want 1", r.factoryResetCalls)
	}
	if !strings.Contains(out.String(), "This action is destructive. Continue? [y/N]") {
		t.Fatalf("missing confirmation prompt in output: %q", out.String())
	}
}

func TestSendTextSuccessTextOutput(t *testing.T) {
	cliCtx := &Context{Port: "/dev/test", Timeout: time.Second}
	r := &commandTestRadio{}
	cmd := newSendTextCommand(cliCtx, func(string) (Radio, error) { return r, nil })
	cmd.SetArgs([]string{"--message", "hello mesh", "--to", "123", "--channel", "2"})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.sendTextCalls != 1 {
		t.Fatalf("send text calls = %d, want 1", r.sendTextCalls)
	}
	if r.sendTextMessage != "hello mesh" || r.sendTextTo != 123 || r.sendTextChannel != 2 {
		t.Fatalf("unexpected send args: message=%q to=%d channel=%d", r.sendTextMessage, r.sendTextTo, r.sendTextChannel)
	}
	if !strings.Contains(out.String(), "sent text to=123 channel=2") {
		t.Fatalf("missing success output: %q", out.String())
	}
}

func TestSendTextSuccessJSONOutput(t *testing.T) {
	cliCtx := &Context{Port: "/dev/test", Timeout: time.Second, JSON: true}
	r := &commandTestRadio{}
	cmd := newSendTextCommand(cliCtx, func(string) (Radio, error) { return r, nil })
	cmd.SetArgs([]string{"--message", "hi"})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal output: %v, output=%q", err, out.String())
	}
	if got["ok"] != true {
		t.Fatalf("expected ok=true, got %#v", got["ok"])
	}
	if got["message"] != "hi" {
		t.Fatalf("expected message=hi, got %#v", got["message"])
	}
}

func TestSetOwnerSuccess(t *testing.T) {
	cliCtx := &Context{Port: "/dev/test", Timeout: time.Second}
	r := &commandTestRadio{}
	cmd := newSetOwnerCommand(cliCtx, func(string) (Radio, error) { return r, nil })
	cmd.SetArgs([]string{"--name", "Moon Station"})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.setOwnerCalls != 1 || r.setOwnerName != "Moon Station" {
		t.Fatalf("unexpected set owner call: calls=%d name=%q", r.setOwnerCalls, r.setOwnerName)
	}
	if !strings.Contains(out.String(), `owner set to "Moon Station"`) {
		t.Fatalf("missing owner output: %q", out.String())
	}
}

func TestSetModemSuccessNormalizesInput(t *testing.T) {
	cliCtx := &Context{Port: "/dev/test", Timeout: time.Second}
	r := &commandTestRadio{}
	cmd := newSetModemCommand(cliCtx, func(string) (Radio, error) { return r, nil })
	cmd.SetArgs([]string{"--mode", "LF"})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.setModemCalls != 1 || r.setModemMode != "lf" {
		t.Fatalf("unexpected modem call: calls=%d mode=%q", r.setModemCalls, r.setModemMode)
	}
	if !strings.Contains(out.String(), `modem mode set to "lf"`) {
		t.Fatalf("missing modem output: %q", out.String())
	}
}

func TestSetLocationSuccess(t *testing.T) {
	cliCtx := &Context{Port: "/dev/test", Timeout: time.Second}
	r := &commandTestRadio{}
	cmd := newSetLocationCommand(cliCtx, func(string) (Radio, error) { return r, nil })
	cmd.SetArgs([]string{"--lat-i", "123", "--lon-i", "-456", "--alt", "7"})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.setLocationCalls != 1 || r.setLocationLat != 123 || r.setLocationLon != -456 || r.setLocationAlt != 7 {
		t.Fatalf(
			"unexpected set location call: calls=%d lat=%d lon=%d alt=%d",
			r.setLocationCalls,
			r.setLocationLat,
			r.setLocationLon,
			r.setLocationAlt,
		)
	}
	if !strings.Contains(out.String(), "location set lat_i=123 lon_i=-456 alt=7") {
		t.Fatalf("missing location output: %q", out.String())
	}
}

func TestInfoSuccessTextAndJSON(t *testing.T) {
	testCases := []struct {
		name   string
		json   bool
		assert func(t *testing.T, out string)
	}{
		{
			name: "text",
			assert: func(t *testing.T, out string) {
				t.Helper()
				if !strings.Contains(out, "radio info") {
					t.Fatalf("missing header: %q", out)
				}
				if !strings.Contains(out, "my_node") || !strings.Contains(out, "!16c3f424") {
					t.Fatalf("missing node summary: %q", out)
				}
			},
		},
		{
			name: "json",
			json: true,
			assert: func(t *testing.T, out string) {
				t.Helper()
				var got map[string]any
				if err := json.Unmarshal([]byte(out), &got); err != nil {
					t.Fatalf("unmarshal output: %v", err)
				}
				if _, ok := got["summary"]; !ok {
					t.Fatalf("missing summary field: %q", out)
				}
				if _, ok := got["responses"]; !ok {
					t.Fatalf("missing responses field: %q", out)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cliCtx := &Context{Port: "/dev/test", Timeout: time.Second, JSON: tc.json}
			r := &commandTestRadio{
				infoResponses: []*pb.FromRadio{
					{PayloadVariant: &pb.FromRadio_MyInfo{MyInfo: &pb.MyNodeInfo{MyNodeNum: 0x16c3f424}}},
					{PayloadVariant: &pb.FromRadio_Metadata{Metadata: &pb.DeviceMetadata{FirmwareVersion: "2.6.0"}}},
				},
			}
			cmd := newInfoCommand(cliCtx, func(string) (Radio, error) { return r, nil })

			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&out)

			if err := cmd.Execute(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if r.infoCalls != 1 {
				t.Fatalf("info calls = %d, want 1", r.infoCalls)
			}
			tc.assert(t, out.String())
		})
	}
}
