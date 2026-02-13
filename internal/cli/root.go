package cli

import (
	"errors"

	"github.com/spf13/cobra"
)

// Version is the CLI version and can be overridden at build time.
var Version = "dev"

type userInputError struct {
	err error
}

func (e *userInputError) Error() string {
	return e.err.Error()
}

func (e *userInputError) Unwrap() error {
	return e.err
}

func newUserInputError(err error) error {
	if err == nil {
		return nil
	}

	var inputErr *userInputError
	if errors.As(err, &inputErr) {
		return err
	}

	return &userInputError{err: err}
}

func wrapPositionalArgs(argsFn cobra.PositionalArgs) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := argsFn(cmd, args); err != nil {
			return newUserInputError(err)
		}
		return nil
	}
}

// ExitCode returns the CLI process exit code for the provided error.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}

	var inputErr *userInputError
	if errors.As(err, &inputErr) {
		return 2
	}

	return 1
}

func newRootCommand() *cobra.Command {
	ctx := &Context{
		Port:    defaultPort,
		Timeout: defaultTimeout,
	}

	cmd := &cobra.Command{
		Use:           "chirp",
		Short:         "A slim Meshtastic CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			return validateContext(ctx)
		},
	}

	cmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return newUserInputError(err)
	})

	cmd.PersistentFlags().StringVar(&ctx.Port, "port", defaultPort, "serial port for the Meshtastic node")
	cmd.PersistentFlags().DurationVar(&ctx.Timeout, "timeout", defaultTimeout, "command timeout")
	cmd.PersistentFlags().BoolVar(&ctx.JSON, "json", false, "print machine-readable output")
	cmd.PersistentFlags().BoolVar(&ctx.Verbose, "verbose", false, "enable debug logs")

	cmd.AddCommand(newVersionCommand(ctx))
	cmd.AddCommand(newListenCommand(ctx, nil))
	cmd.AddCommand(newInfoCommand(ctx, nil))
	cmd.AddCommand(newSendCommand(ctx, nil))
	cmd.AddCommand(newSetCommand(ctx, nil))
	cmd.AddCommand(newFactoryResetCommand(ctx, nil))

	return cmd
}

// Execute runs the root command.
func Execute() error {
	return newRootCommand().Execute()
}
