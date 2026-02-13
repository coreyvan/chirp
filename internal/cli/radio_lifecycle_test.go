package cli

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	pb "github.com/coreyvan/chirp/protogen/meshtastic"
)

type fakeRadio struct {
	closeCalls int
	closeErr   error
}

func (f *fakeRadio) Close() error {
	f.closeCalls++
	return f.closeErr
}

func (f *fakeRadio) ReadResponse(bool) ([]*pb.FromRadio, error) {
	return nil, nil
}

func (f *fakeRadio) GetRadioInfo() ([]*pb.FromRadio, error) {
	return nil, nil
}

func (f *fakeRadio) SendTextMessage(string, int64, int64) error {
	return nil
}

func (f *fakeRadio) SetRadioOwner(string) error {
	return nil
}

func (f *fakeRadio) SetModemMode(string) error {
	return nil
}

func (f *fakeRadio) SetLocation(int32, int32, int32) error {
	return nil
}

func (f *fakeRadio) FactoryReset() error {
	return nil
}

type fakeRunner struct {
	calls int
	run   func(ctx context.Context, radio Radio) error
}

func (f *fakeRunner) Run(ctx context.Context, radio Radio) error {
	f.calls++
	return f.run(ctx, radio)
}

func TestRunWithRadioSuccess(t *testing.T) {
	ctx := &Context{
		Port:    "/dev/test",
		Timeout: time.Second,
	}

	fRadio := &fakeRadio{}
	var openedPort string
	runner := &fakeRunner{
		run: func(_ context.Context, radio Radio) error {
			if radio != fRadio {
				t.Fatalf("runner received unexpected radio instance")
			}
			return nil
		},
	}

	err := runWithRadio(context.Background(), ctx, func(port string) (Radio, error) {
		openedPort = port
		return fRadio, nil
	}, runner)
	if err != nil {
		t.Fatalf("runWithRadio() error = %v", err)
	}
	if openedPort != ctx.Port {
		t.Fatalf("opener called with port %q, want %q", openedPort, ctx.Port)
	}
	if runner.calls != 1 {
		t.Fatalf("runner calls = %d, want 1", runner.calls)
	}
	if fRadio.closeCalls != 1 {
		t.Fatalf("close calls = %d, want 1", fRadio.closeCalls)
	}
}

func TestRunWithRadioInvalidContext(t *testing.T) {
	ctx := &Context{
		Port:    "",
		Timeout: time.Second,
	}

	err := runWithRadio(context.Background(), ctx, nil, &fakeRunner{run: func(context.Context, Radio) error { return nil }})
	if err == nil {
		t.Fatalf("expected error")
	}
	if ExitCode(err) != 2 {
		t.Fatalf("exit code = %d, want 2", ExitCode(err))
	}
	if !strings.Contains(err.Error(), "--port cannot be empty") {
		t.Fatalf("error = %q, expected port validation", err.Error())
	}
}

func TestRunWithRadioOpenErrorIncludesHint(t *testing.T) {
	ctx := &Context{
		Port:    "/dev/test",
		Timeout: time.Second,
	}

	err := runWithRadio(context.Background(), ctx, func(string) (Radio, error) {
		return nil, errors.New("could not open serial connection: resource busy")
	}, &fakeRunner{run: func(context.Context, Radio) error { return nil }})
	if err == nil {
		t.Fatalf("expected error")
	}
	if ExitCode(err) != 1 {
		t.Fatalf("exit code = %d, want 1", ExitCode(err))
	}
	if !strings.Contains(err.Error(), "failed to open radio on") {
		t.Fatalf("error = %q, missing open message", err.Error())
	}
	if !strings.Contains(err.Error(), "port may already be in use") {
		t.Fatalf("error = %q, missing contention hint", err.Error())
	}
}

func TestRunWithRadioTimeout(t *testing.T) {
	ctx := &Context{
		Port:    "/dev/test",
		Timeout: 10 * time.Millisecond,
	}

	fRadio := &fakeRadio{}
	runner := &fakeRunner{
		run: func(runCtx context.Context, _ Radio) error {
			<-runCtx.Done()
			return runCtx.Err()
		},
	}

	err := runWithRadio(context.Background(), ctx, func(string) (Radio, error) {
		return fRadio, nil
	}, runner)
	if err == nil {
		t.Fatalf("expected timeout error")
	}
	if ExitCode(err) != 1 {
		t.Fatalf("exit code = %d, want 1", ExitCode(err))
	}
	if !strings.Contains(err.Error(), "command timed out") {
		t.Fatalf("error = %q, missing timeout text", err.Error())
	}
	if fRadio.closeCalls == 0 {
		t.Fatalf("expected radio close to be called")
	}
}

func TestRunWithRadioCloseError(t *testing.T) {
	ctx := &Context{
		Port:    "/dev/test",
		Timeout: time.Second,
	}

	fRadio := &fakeRadio{closeErr: errors.New("close failed")}
	err := runWithRadio(context.Background(), ctx, func(string) (Radio, error) {
		return fRadio, nil
	}, &fakeRunner{run: func(context.Context, Radio) error { return nil }})
	if err == nil {
		t.Fatalf("expected close error")
	}
	if !strings.Contains(err.Error(), "close radio") {
		t.Fatalf("error = %q, missing close text", err.Error())
	}
}

func TestNewRadioCommandRejectsExtraArgs(t *testing.T) {
	cliCtx := &Context{
		Port:    "/dev/test",
		Timeout: time.Second,
	}
	cmd := newRadioCommand(
		"test",
		"test command",
		cliCtx,
		func(string) (Radio, error) { return &fakeRadio{}, nil },
		&fakeRunner{run: func(context.Context, Radio) error { return nil }},
	)
	cmd.SetArgs([]string{"extra"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected arg validation error")
	}
	if ExitCode(err) != 2 {
		t.Fatalf("exit code = %d, want 2", ExitCode(err))
	}
}
