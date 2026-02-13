package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newSendTextCommand(cliCtx *Context, opener radioOpener) *cobra.Command {
	var (
		to      int64
		channel int64
		message string
	)

	cmd := &cobra.Command{
		Use:   "text",
		Short: "Send a text message",
		Args:  wrapPositionalArgs(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(message) == "" {
				return newUserInputError(fmt.Errorf("--message cannot be empty"))
			}
			if to < 0 {
				return newUserInputError(fmt.Errorf("--to must be >= 0"))
			}
			if channel < 0 {
				return newUserInputError(fmt.Errorf("--channel must be >= 0"))
			}

			return runWithRadio(cmd.Context(), cliCtx, opener, RadioRunnerFunc(func(_ context.Context, radio Radio) error {
				if err := radio.SendTextMessage(message, to, channel); err != nil {
					return fmt.Errorf("send text: %w", err)
				}

				if cliCtx.JSON {
					return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
						"ok":      true,
						"to":      to,
						"channel": channel,
						"message": message,
					})
				}

				_, err := fmt.Fprintf(cmd.OutOrStdout(), "sent text to=%d channel=%d\n", to, channel)
				return err
			}))
		},
	}

	cmd.Flags().Int64Var(&to, "to", 0, "destination node number (0 for broadcast)")
	cmd.Flags().Int64Var(&channel, "channel", 0, "channel index")
	cmd.Flags().StringVar(&message, "message", "", "message text")
	_ = cmd.MarkFlagRequired("message")

	return cmd
}
