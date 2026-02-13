package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var validModemModes = map[string]struct{}{
	"lf":  {},
	"ls":  {},
	"vls": {},
	"ms":  {},
	"mf":  {},
	"sl":  {},
	"sf":  {},
	"lm":  {},
}

func newSetModemCommand(cliCtx *Context, opener radioOpener) *cobra.Command {
	var mode string

	cmd := &cobra.Command{
		Use:   "modem",
		Short: "Set modem preset",
		Args:  wrapPositionalArgs(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			mode = strings.ToLower(strings.TrimSpace(mode))
			if mode == "" {
				return newUserInputError(fmt.Errorf("--mode cannot be empty"))
			}
			if _, ok := validModemModes[mode]; !ok {
				return newUserInputError(fmt.Errorf("--mode must be one of: lf|ls|vls|ms|mf|sl|sf|lm"))
			}

			return runWithRadio(cmd.Context(), cliCtx, opener, RadioRunnerFunc(func(_ context.Context, radio Radio) error {
				if err := radio.SetModemMode(mode); err != nil {
					return fmt.Errorf("set modem: %w", err)
				}

				if cliCtx.JSON {
					return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
						"ok":   true,
						"mode": mode,
					})
				}

				_, err := fmt.Fprintf(cmd.OutOrStdout(), "modem mode set to %q\n", mode)
				return err
			}))
		},
	}

	cmd.Flags().StringVar(&mode, "mode", "", "modem preset (lf|ls|vls|ms|mf|sl|sf|lm)")
	_ = cmd.MarkFlagRequired("mode")

	return cmd
}
