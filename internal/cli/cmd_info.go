package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	appnode "github.com/coreyvan/chirp/internal/app/node"
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
				service := appnode.NewService(radio)
				result, err := service.Info(cmd.Context())
				if err != nil {
					return mapServiceError(err)
				}

				if cliCtx.JSON {
					return writeInfoJSON(result.Summary, result.Responses, cmd.OutOrStdout())
				}

				out := cmd.OutOrStdout()
				if _, err := fmt.Fprintln(out, "radio info"); err != nil {
					return err
				}
				return printKeyValueTable(out, []keyValueRow{
					{Key: "responses", Value: strconv.Itoa(result.Summary.Responses)},
					{Key: "my_node", Value: result.Summary.MyNode},
					{Key: "firmware", Value: result.Summary.Firmware},
					{Key: "hw_model", Value: result.Summary.HWModel},
					{Key: "role", Value: result.Summary.Role},
					{Key: "nodes", Value: strconv.Itoa(result.Summary.Nodes)},
					{Key: "channels", Value: strconv.Itoa(result.Summary.Channels)},
					{Key: "configs", Value: strconv.Itoa(result.Summary.Configs)},
					{Key: "module_configs", Value: strconv.Itoa(result.Summary.ModuleConfigs)},
				})
			}))
		},
	}
}

func writeInfoJSON(summary appnode.InfoSummary, responses []*pb.FromRadio, out io.Writer) error {
	items := make([]json.RawMessage, 0, len(responses))
	for _, fr := range responses {
		b, err := protojson.Marshal(fr)
		if err != nil {
			return fmt.Errorf("marshal radio info response: %w", err)
		}
		items = append(items, json.RawMessage(b))
	}

	return json.NewEncoder(out).Encode(map[string]any{
		"summary":   summary,
		"responses": items,
	})
}
