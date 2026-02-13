package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	pb "github.com/coreyvan/chirp/protogen/meshtastic"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
)

func newInfoCommand(cliCtx *Context, opener radioOpener) *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Fetch and print radio info",
		Args:  wrapPositionalArgs(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runWithRadio(cmd.Context(), cliCtx, opener, RadioRunnerFunc(func(_ context.Context, radio Radio) error {
				responses, err := radio.GetRadioInfo()
				if err != nil {
					return fmt.Errorf("get radio info: %w", err)
				}

				if cliCtx.JSON {
					return writeInfoJSON(responses, cmd.OutOrStdout())
				}

				out := cmd.OutOrStdout()
				_, _ = fmt.Fprintf(out, "radio info responses=%d\n", len(responses))
				for _, fr := range responses {
					logFromRadio(out, fr, &listenOptions{})
				}
				return nil
			}))
		},
	}
}

func writeInfoJSON(responses []*pb.FromRadio, out io.Writer) error {
	items := make([]json.RawMessage, 0, len(responses))
	for _, fr := range responses {
		b, err := protojson.Marshal(fr)
		if err != nil {
			return fmt.Errorf("marshal radio info response: %w", err)
		}
		items = append(items, json.RawMessage(b))
	}

	return json.NewEncoder(out).Encode(map[string]any{
		"responses": items,
	})
}
