package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/isaacwallace123/cortex/apps/cli/internal/render"
)

var plansCmd = &cobra.Command{
	Use:   "plans <session-id>",
	Short: "List stored plans for a session",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sid := args[0]
		resp, err := apiClient.ListPlans(sid)
		if err != nil {
			render.Error(os.Stderr, err.Error())
			os.Exit(1)
		}
		render.PlanList(os.Stdout, resp.SessionID, resp.Plans)
		return nil
	},
}

var sessionCmd = &cobra.Command{
	Use:   "session <id>",
	Short: "Show event history for a session",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := apiClient.GetSession(args[0])
		if err != nil {
			render.Error(os.Stderr, err.Error())
			os.Exit(1)
		}
		render.SessionHistory(os.Stdout, resp)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(plansCmd)
	rootCmd.AddCommand(sessionCmd)
}
