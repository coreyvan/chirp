package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

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
			lat, err := int32FromInt64Flag("--lat-i", latI)
			if err != nil {
				return err
			}
			lon, err := int32FromInt64Flag("--lon-i", lonI)
			if err != nil {
				return err
			}
			altitude, err := int32FromInt64Flag("--alt", alt)
			if err != nil {
				return err
			}

			return runWithRadio(cmd.Context(), cliCtx, opener, RadioRunnerFunc(func(_ context.Context, radio Radio) error {
				if err := radio.SetLocation(lat, lon, altitude); err != nil {
					return fmt.Errorf("set location: %w", err)
				}

				if cliCtx.JSON {
					return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
						"ok":    true,
						"lat_i": lat,
						"lon_i": lon,
						"alt":   altitude,
					})
				}

				_, err := fmt.Fprintf(cmd.OutOrStdout(), "location set lat_i=%d lon_i=%d alt=%d\n", lat, lon, altitude)
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

func int32FromInt64Flag(flagName string, value int64) (int32, error) {
	if value < math.MinInt32 || value > math.MaxInt32 {
		return 0, newUserInputError(fmt.Errorf("%s must be in int32 range", flagName))
	}
	return int32(value), nil
}
