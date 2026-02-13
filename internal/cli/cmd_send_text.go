package cli

import (
	"context"
	"encoding/json"
	"fmt"

	appnode "github.com/coreyvan/chirp/internal/app/node"
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
			if err := appnode.ValidateSendTextRequest(appnode.SendTextRequest{
				Message: message,
				To:      to,
				Channel: channel,
			}); err != nil {
				return mapServiceError(err)
			}

			return runWithRadio(cmd.Context(), cliCtx, opener, RadioRunnerFunc(func(_ context.Context, radio Radio) error {
				service := appnode.NewService(radio)
				result, err := service.SendText(cmd.Context(), appnode.SendTextRequest{
					Message: message,
					To:      to,
					Channel: channel,
				})
				if err != nil {
					return mapServiceError(err)
				}

				if cliCtx.JSON {
					return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{
						"ok":      true,
						"to":      result.To,
						"channel": result.Channel,
						"message": result.Message,
					})
				}

				_, err = fmt.Fprintf(cmd.OutOrStdout(), "sent text to=%d channel=%d\n", result.To, result.Channel)
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
