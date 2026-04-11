package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/isaacwallace123/cortex/apps/cli/internal/render"
)

var chatSessionID string

var chatCmd = &cobra.Command{
	Use:   "chat <input>",
	Short: "Plan only — show intent, plan, and Aegis verdicts",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		input := strings.Join(args, " ")
		render.InputHeader(os.Stdout, input)

		spin := render.StartSpinner(os.Stdout, "Thinking...")
		resp, err := apiClient.Chat(chatSessionID, input)
		spin.Stop()

		if err != nil {
			render.Error(os.Stderr, err.Error())
			os.Exit(1)
		}
		render.ChatResult(os.Stdout, resp, input)
		return nil
	},
}

func init() {
	chatCmd.Flags().StringVar(&chatSessionID, "session", "", "Session ID to use or resume")
	rootCmd.AddCommand(chatCmd)
}
