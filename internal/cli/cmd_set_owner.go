package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newSetOwnerCommand(cliCtx *Context, opener radioOpener) *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "owner",
		Short: "Set owner long name",
		Args:  wrapPositionalArgs(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(name) == "" {
				return newUserInputError(fmt.Errorf("--name cannot be empty"))
			}

			return runWithRadio(cmd.Context(), cliCtx, opener, RadioRunnerFunc(func(_ context.Context, radio Radio) error {
				if err := radio.SetRadioOwner(name); err != nil {
					return fmt.Errorf("set owner: %w", err)
				}

				if cliCtx.JSON {
					return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
						"ok":   true,
						"name": name,
					})
				}

				_, err := fmt.Fprintf(cmd.OutOrStdout(), "owner set to %q\n", name)
				return err
			}))
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "owner long name")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}
