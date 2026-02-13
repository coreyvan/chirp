package commands

import (
	"context"
	"encoding/json"
	"fmt"

	appnode "github.com/coreyvan/chirp/internal/app/node"
	"github.com/spf13/cobra"
)

func newSetModemCommand(cliCtx *Context, opener radioOpener) *cobra.Command {
	var mode string

	cmd := &cobra.Command{
		Use:   "modem",
		Short: "Set modem preset",
		Args:  wrapPositionalArgs(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			normalizedMode, err := appnode.NormalizeAndValidateModemMode(mode)
			if err != nil {
				return mapServiceError(err)
			}

			return runWithRadio(cmd.Context(), cliCtx, opener, RadioRunnerFunc(func(_ context.Context, radio Radio) error {
				service := appnode.NewService(radio)
				result, err := service.SetModem(cmd.Context(), appnode.SetModemRequest{Mode: normalizedMode})
				if err != nil {
					return mapServiceError(err)
				}

				if cliCtx.JSON {
					return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
						"ok":   true,
						"mode": result.Mode,
					})
				}

				_, err = fmt.Fprintf(cmd.OutOrStdout(), "modem mode set to %q\n", result.Mode)
				return err
			}))
		},
	}

	cmd.Flags().StringVar(&mode, "mode", "", "modem preset (lf|ls|vls|ms|mf|sl|sf|lm)")
	_ = cmd.MarkFlagRequired("mode")

	return cmd
}
