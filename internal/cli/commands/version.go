package commands

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCommand(ctx *Context) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print chirp version",
		Args:  wrapPositionalArgs(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if ctx.JSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]string{
					"version": Version,
				})
			}

			_, err := fmt.Fprintln(cmd.OutOrStdout(), Version)
			return err
		},
	}
}
