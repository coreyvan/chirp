package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newFactoryResetCommand(cliCtx *Context, opener radioOpener) *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "factory-reset",
		Short: "Factory reset the radio",
		Args:  wrapPositionalArgs(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !yes {
				confirmed, err := promptConfirm(cmd, "This action is destructive. Continue? [y/N] ")
				if err != nil {
					return newRuntimeError(fmt.Errorf("read confirmation: %w", err))
				}
				if !confirmed {
					return newRuntimeError(fmt.Errorf("factory-reset cancelled"))
				}
			}

			return runWithRadio(cmd.Context(), cliCtx, opener, RadioRunnerFunc(func(_ context.Context, radio Radio) error {
				if err := radio.FactoryReset(); err != nil {
					return fmt.Errorf("factory-reset: %w", err)
				}

				if cliCtx.JSON {
					return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
						"ok": true,
					})
				}

				_, err := fmt.Fprintln(cmd.OutOrStdout(), "factory reset command sent")
				return err
			}))
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")
	return cmd
}

func promptConfirm(cmd *cobra.Command, prompt string) (bool, error) {
	if _, err := fmt.Fprint(cmd.OutOrStdout(), prompt); err != nil {
		return false, err
	}

	line, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	if err != nil {
		return false, err
	}

	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}
