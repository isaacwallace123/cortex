package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/isaacwallace123/cortex/apps/cli/internal/client"
	"github.com/isaacwallace123/cortex/apps/cli/internal/render"
)

var (
	subscribeFilter string
	nervaAddr       string
)

var subscribeCmd = &cobra.Command{
	Use:   "subscribe",
	Short: "Tail live events from the Nerva event bus",
	RunE: func(cmd *cobra.Command, args []string) error {
		nerva, err := client.NewNervaClient(nervaAddr)
		if err != nil {
			render.Error(os.Stderr, err.Error())
			os.Exit(1)
		}

		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		render.SubscribeHeader(os.Stdout, subscribeFilter)

		events, errc := nerva.Subscribe(ctx, subscribeFilter)
		for {
			select {
			case ev, ok := <-events:
				if !ok {
					return nil
				}
				ts := time.Now().Format("15:04:05")
				render.EventLine(os.Stdout, ts, ev.Subsystem, ev.EventName, ev.SessionID, ev.Payload)
			case err := <-errc:
				if err != nil {
					render.Error(os.Stderr, err.Error())
					os.Exit(1)
				}
				return nil
			}
		}
	},
}

func init() {
	subscribeCmd.Flags().StringVar(&subscribeFilter, "filter", "", "Event filter for subscribe (subsystem name, event prefix, or empty for all)")
	subscribeCmd.Flags().StringVar(&nervaAddr, "nerva", envOr("CORTEX_NERVA_ADDR", "localhost:3030"), "Nerva gRPC address")
	rootCmd.AddCommand(subscribeCmd)
}
