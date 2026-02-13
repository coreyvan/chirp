package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

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
				summary := buildInfoSummary(responses)

				if cliCtx.JSON {
					return writeInfoJSON(summary, responses, cmd.OutOrStdout())
				}

				out := cmd.OutOrStdout()
				if _, err := fmt.Fprintln(out, "radio info"); err != nil {
					return err
				}
				return printKeyValueTable(out, []keyValueRow{
					{Key: "responses", Value: strconv.Itoa(summary.Responses)},
					{Key: "my_node", Value: summary.MyNode},
					{Key: "firmware", Value: summary.Firmware},
					{Key: "hw_model", Value: summary.HWModel},
					{Key: "role", Value: summary.Role},
					{Key: "nodes", Value: strconv.Itoa(summary.Nodes)},
					{Key: "channels", Value: strconv.Itoa(summary.Channels)},
					{Key: "configs", Value: strconv.Itoa(summary.Configs)},
					{Key: "module_configs", Value: strconv.Itoa(summary.ModuleConfigs)},
				})
			}))
		},
	}
}

type infoSummary struct {
	Responses     int    `json:"responses"`
	MyNode        string `json:"my_node"`
	Firmware      string `json:"firmware"`
	HWModel       string `json:"hw_model"`
	Role          string `json:"role"`
	Nodes         int    `json:"nodes"`
	Channels      int    `json:"channels"`
	Configs       int    `json:"configs"`
	ModuleConfigs int    `json:"module_configs"`
}

func buildInfoSummary(responses []*pb.FromRadio) infoSummary {
	summary := infoSummary{
		Responses: len(responses),
		MyNode:    "-",
		Firmware:  "-",
		HWModel:   "-",
		Role:      "-",
	}

	for _, fr := range responses {
		switch v := fr.GetPayloadVariant().(type) {
		case *pb.FromRadio_MyInfo:
			if v.MyInfo != nil {
				summary.MyNode = fmt.Sprintf("!%08x", v.MyInfo.GetMyNodeNum())
			}
		case *pb.FromRadio_Metadata:
			if v.Metadata != nil {
				if fw := v.Metadata.GetFirmwareVersion(); fw != "" {
					summary.Firmware = fw
				}
				summary.HWModel = v.Metadata.GetHwModel().String()
				summary.Role = v.Metadata.GetRole().String()
			}
		case *pb.FromRadio_NodeInfo:
			summary.Nodes++
		case *pb.FromRadio_Channel:
			summary.Channels++
		case *pb.FromRadio_Config:
			summary.Configs++
		case *pb.FromRadio_ModuleConfig:
			summary.ModuleConfigs++
		}
	}

	return summary
}

func writeInfoJSON(summary infoSummary, responses []*pb.FromRadio, out io.Writer) error {
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
