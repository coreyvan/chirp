package commands

import "github.com/spf13/cobra"

func newSendCommand(cliCtx *Context, opener radioOpener) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send outbound messages",
		Args:  wrapPositionalArgs(cobra.NoArgs),
	}

	cmd.AddCommand(newSendTextCommand(cliCtx, opener))
	return cmd
}
