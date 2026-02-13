package cli

import (
	"context"
	"encoding/json"
	"fmt"

	appnode "github.com/coreyvan/chirp/internal/app/node"
	"github.com/spf13/cobra"
)

func newSetOwnerCommand(cliCtx *Context, opener radioOpener) *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "owner",
		Short: "Set owner long name",
		Args:  wrapPositionalArgs(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := appnode.ValidateSetOwnerRequest(appnode.SetOwnerRequest{Name: name}); err != nil {
				return mapServiceError(err)
			}

			return runWithRadio(cmd.Context(), cliCtx, opener, RadioRunnerFunc(func(_ context.Context, radio Radio) error {
				service := appnode.NewService(radio)
				result, err := service.SetOwner(cmd.Context(), appnode.SetOwnerRequest{Name: name})
				if err != nil {
					return mapServiceError(err)
				}

				if cliCtx.JSON {
					return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
						"ok":   true,
						"name": result.Name,
					})
				}

				_, err = fmt.Fprintf(cmd.OutOrStdout(), "owner set to %q\n", result.Name)
				return err
			}))
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "owner long name")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}
