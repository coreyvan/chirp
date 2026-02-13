package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	pb "github.com/coreyvan/chirp/protogen/meshtastic"
)

type commandTestRadio struct {
	factoryResetCalls int
}

func (f *commandTestRadio) Close() error { return nil }
func (f *commandTestRadio) ReadResponse(bool) ([]*pb.FromRadio, error) {
	return nil, nil
}
func (f *commandTestRadio) GetRadioInfo() ([]*pb.FromRadio, error) { return nil, nil }
func (f *commandTestRadio) SendTextMessage(string, int64, int64) error {
	return nil
}
func (f *commandTestRadio) SetRadioOwner(string) error            { return nil }
func (f *commandTestRadio) SetModemMode(string) error             { return nil }
func (f *commandTestRadio) SetLocation(int32, int32, int32) error { return nil }
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
