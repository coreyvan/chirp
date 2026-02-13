package cli

import "github.com/spf13/cobra"

func newSetCommand(cliCtx *Context, opener radioOpener) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set device configuration values",
		Args:  wrapPositionalArgs(cobra.NoArgs),
	}

	cmd.AddCommand(newSetOwnerCommand(cliCtx, opener))
	cmd.AddCommand(newSetModemCommand(cliCtx, opener))
	cmd.AddCommand(newSetLocationCommand(cliCtx, opener))
	return cmd
}
