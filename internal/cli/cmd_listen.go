package cli

import (
	"context"
	"fmt"
	"io"
	"time"

	appnode "github.com/coreyvan/chirp/internal/app/node"
	pb "github.com/coreyvan/chirp/protogen/meshtastic"
	"github.com/spf13/cobra"
)

type listenOptions struct {
	idleLog     time.Duration
	noTelemetry bool
	noEvents    bool
	noPackets   bool
}

func newListenCommand(cliCtx *Context, opener radioOpener) *cobra.Command {
	opts := &listenOptions{
		idleLog: 10 * time.Second,
	}

	cmd := &cobra.Command{
		Use:   "listen",
		Short: "Stream incoming packets, events, and telemetry",
		Args:  wrapPositionalArgs(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if opts.idleLog <= 0 {
				return newUserInputError(fmt.Errorf("--idle-log must be greater than 0"))
			}

			return runWithRadioNoTimeout(cmd.Context(), cliCtx, opener, RadioRunnerFunc(func(runCtx context.Context, radio Radio) error {
				return runListen(runCtx, cmd.OutOrStdout(), radio, cliCtx.Port, opts)
			}))
		},
	}

	cmd.Flags().DurationVar(&opts.idleLog, "idle-log", opts.idleLog, "how often to print idle message when no packets arrive")
	cmd.Flags().BoolVar(&opts.noTelemetry, "no-telemetry", false, "suppress telemetry output")
	cmd.Flags().BoolVar(&opts.noEvents, "no-events", false, "suppress event output")
	cmd.Flags().BoolVar(&opts.noPackets, "no-packets", false, "suppress packet output")

	return cmd
}

func runListen(ctx context.Context, out io.Writer, radio Radio, port string, opts *listenOptions) error {
	_, _ = fmt.Fprintf(out, "rx listener started on %s\n", port)

	// Prime the device so nodes that stay quiet until polled begin streaming updates.
	if responses, err := radio.GetRadioInfo(); err != nil {
		_, _ = fmt.Fprintf(out, "[ERR] get radio info: %v\n", err)
	} else {
		for _, fr := range responses {
			logFromRadio(out, fr, opts)
		}
	}

	lastIdleLog := time.Now()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		fromRadioPackets, err := radio.ReadResponse(true)
		if err != nil {
			_, _ = fmt.Fprintf(out, "[ERR] read response: %v\n", err)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(300 * time.Millisecond):
			}
			continue
		}

		if len(fromRadioPackets) == 0 {
			if time.Since(lastIdleLog) >= opts.idleLog {
				_, _ = fmt.Fprintln(out, "[IDLE] no packets")
				lastIdleLog = time.Now()
			}
			continue
		}

		for _, fr := range fromRadioPackets {
			logFromRadio(out, fr, opts)
		}
	}
}

func logFromRadio(out io.Writer, fr *pb.FromRadio, opts *listenOptions) {
	writeStreamLines(out, appnode.RenderFromRadio(fr), opts)
}

func logMeshPacket(out io.Writer, mp *pb.MeshPacket, opts *listenOptions) {
	writeStreamLines(out, appnode.RenderMeshPacket(mp), opts)
}

func writeStreamLines(out io.Writer, lines []appnode.StreamLine, opts *listenOptions) {
	for _, line := range lines {
		if shouldSkipLine(line, opts) {
			continue
		}
		_, _ = fmt.Fprintf(out, "[%s] %s\n", line.Label, line.Message)
	}
}

func shouldSkipLine(line appnode.StreamLine, opts *listenOptions) bool {
	switch line.Category {
	case appnode.StreamCategoryEvent:
		return opts.noEvents
	case appnode.StreamCategoryPacket:
		return opts.noPackets
	case appnode.StreamCategoryTelemetry:
		return opts.noTelemetry
	default:
		return false
	}
}
