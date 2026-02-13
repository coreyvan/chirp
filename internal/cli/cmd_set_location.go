package cli

import (
	"context"
	"encoding/json"
	"fmt"

	appnode "github.com/coreyvan/chirp/internal/app/node"
	"github.com/spf13/cobra"
)

func newSetLocationCommand(cliCtx *Context, opener radioOpener) *cobra.Command {
	var (
		latI int64
		lonI int64
		alt  int64
	)

	cmd := &cobra.Command{
		Use:   "location",
		Short: "Set fixed location payload",
		Args:  wrapPositionalArgs(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if _, err := appnode.ValidateAndConvertSetLocationRequest(appnode.SetLocationRequest{
				LatI: latI,
				LonI: lonI,
				Alt:  alt,
			}); err != nil {
				return mapServiceError(err)
			}

			return runWithRadio(cmd.Context(), cliCtx, opener, RadioRunnerFunc(func(_ context.Context, radio Radio) error {
				service := appnode.NewService(radio)
				result, err := service.SetLocation(cmd.Context(), appnode.SetLocationRequest{
					LatI: latI,
					LonI: lonI,
					Alt:  alt,
				})
				if err != nil {
					return mapServiceError(err)
				}

				if cliCtx.JSON {
					return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
						"ok":    true,
						"lat_i": result.LatI,
						"lon_i": result.LonI,
						"alt":   result.Alt,
					})
				}

				_, err = fmt.Fprintf(cmd.OutOrStdout(), "location set lat_i=%d lon_i=%d alt=%d\n", result.LatI, result.LonI, result.Alt)
				return err
			}))
		},
	}

	cmd.Flags().Int64Var(&latI, "lat-i", 0, "latitude in 1e-7 degree integer format")
	cmd.Flags().Int64Var(&lonI, "lon-i", 0, "longitude in 1e-7 degree integer format")
	cmd.Flags().Int64Var(&alt, "alt", 0, "altitude in meters")
	_ = cmd.MarkFlagRequired("lat-i")
	_ = cmd.MarkFlagRequired("lon-i")
	_ = cmd.MarkFlagRequired("alt")

	return cmd
}
