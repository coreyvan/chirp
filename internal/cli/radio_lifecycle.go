package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/coreyvan/chirp/pkg/radio"
	pb "github.com/coreyvan/chirp/protogen/meshtastic"
	"github.com/spf13/cobra"
)

type runtimeError struct {
	err error
}

func (e *runtimeError) Error() string {
	return e.err.Error()
}

func (e *runtimeError) Unwrap() error {
	return e.err
}

func newRuntimeError(err error) error {
	if err == nil {
		return nil
	}

	var runErr *runtimeError
	if errors.As(err, &runErr) {
		return err
	}

	return &runtimeError{err: err}
}

func validateContext(ctx *Context) error {
	if strings.TrimSpace(ctx.Port) == "" {
		return newUserInputError(fmt.Errorf("--port cannot be empty"))
	}
	if ctx.Timeout <= 0 {
		return newUserInputError(fmt.Errorf("--timeout must be greater than 0"))
	}
	return nil
}

func formatTimeoutError(timeout string) error {
	return newRuntimeError(fmt.Errorf("command timed out after %s", timeout))
}

func formatOpenRadioError(port string, err error) error {
	hint := ""
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "resource busy"), strings.Contains(msg, "in use"):
		hint = " (port may already be in use by another process)"
	case strings.Contains(msg, "permission denied"), strings.Contains(msg, "operation not permitted"):
		hint = " (check serial port permissions)"
	}

	return newRuntimeError(fmt.Errorf("failed to open radio on %q%s: %w", port, hint, err))
}

// Radio describes the radio surface used by CLI commands.
type Radio interface {
	Close() error
	ReadResponse(timeout bool) ([]*pb.FromRadio, error)
	GetRadioInfo() ([]*pb.FromRadio, error)
	SendTextMessage(message string, to int64, channel int64) error
	SetRadioOwner(name string) error
	SetModemMode(mode string) error
	SetLocation(lat int32, long int32, alt int32) error
	FactoryReset() error
}

type radioOpener func(port string) (Radio, error)

func defaultRadioOpener(port string) (Radio, error) {
	return radio.NewRadio(port)
}

// RadioRunner executes command logic using an opened radio instance.
type RadioRunner interface {
	Run(ctx context.Context, radio Radio) error
}

// RadioRunnerFunc adapts a function into a RadioRunner.
type RadioRunnerFunc func(ctx context.Context, radio Radio) error

func (f RadioRunnerFunc) Run(ctx context.Context, radio Radio) error {
	return f(ctx, radio)
}

func runWithRadio(parent context.Context, cliCtx *Context, opener radioOpener, runner RadioRunner) (err error) {
	if err := validateContext(cliCtx); err != nil {
		return err
	}
	if opener == nil {
		opener = defaultRadioOpener
	}
	if runner == nil {
		return newRuntimeError(fmt.Errorf("internal error: missing command runner"))
	}

	r, err := opener(cliCtx.Port)
	if err != nil {
		return formatOpenRadioError(cliCtx.Port, err)
	}

	var closeOnce sync.Once
	closeRadio := func() error {
		var closeErr error
		closeOnce.Do(func() {
			closeErr = r.Close()
		})
		return closeErr
	}

	defer func() {
		closeErr := closeRadio()
		if closeErr != nil {
			err = errors.Join(err, newRuntimeError(fmt.Errorf("close radio: %w", closeErr)))
		}
	}()

	base := parent
	if base == nil {
		base = context.Background()
	}
	runCtx, cancel := context.WithTimeout(base, cliCtx.Timeout)
	defer cancel()

	result := make(chan error, 1)
	go func() {
		result <- runner.Run(runCtx, r)
	}()

	select {
	case runErr := <-result:
		if runErr == nil {
			return nil
		}
		if errors.Is(runErr, context.DeadlineExceeded) {
			return formatTimeoutError(cliCtx.Timeout.String())
		}
		return newRuntimeError(runErr)
	case <-runCtx.Done():
		_ = closeRadio()
		if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
			return formatTimeoutError(cliCtx.Timeout.String())
		}
		return newRuntimeError(runCtx.Err())
	}
}

func runWithRadioNoTimeout(parent context.Context, cliCtx *Context, opener radioOpener, runner RadioRunner) (err error) {
	if err := validateContext(cliCtx); err != nil {
		return err
	}
	if opener == nil {
		opener = defaultRadioOpener
	}
	if runner == nil {
		return newRuntimeError(fmt.Errorf("internal error: missing command runner"))
	}

	r, err := opener(cliCtx.Port)
	if err != nil {
		return formatOpenRadioError(cliCtx.Port, err)
	}

	defer func() {
		closeErr := r.Close()
		if closeErr != nil {
			err = errors.Join(err, newRuntimeError(fmt.Errorf("close radio: %w", closeErr)))
		}
	}()

	base := parent
	if base == nil {
		base = context.Background()
	}

	runErr := runner.Run(base, r)
	if runErr == nil {
		return nil
	}
	if errors.Is(runErr, context.DeadlineExceeded) {
		return formatTimeoutError(cliCtx.Timeout.String())
	}
	return newRuntimeError(runErr)
}

func newRadioCommand(use, short string, cliCtx *Context, opener radioOpener, runner RadioRunner) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  wrapPositionalArgs(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runWithRadio(cmd.Context(), cliCtx, opener, runner)
		},
	}
}
