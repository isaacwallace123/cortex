package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/isaacwallace123/cortex/apps/cli/internal/client"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CLI configuration",
}

var configSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Persist API URL and key to ~/.config/cortex/config.json",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _ := client.LoadConfig()

		if u, _ := cmd.Flags().GetString("api"); u != "" {
			cfg.APIURL = u
		}
		if k, _ := cmd.Flags().GetString("key"); k != "" {
			cfg.APIKey = k
		}

		if err := client.SaveConfig(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		fmt.Println("Config saved.")
		if cfg.APIURL != "" {
			fmt.Println("  api_url:", cfg.APIURL)
		}
		if cfg.APIKey != "" {
			fmt.Println("  api_key: (set)")
		}
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Print the current config file path and values",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := client.LoadConfig()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		apiKeyDisplay := "(not set)"
		if cfg.APIKey != "" {
			apiKeyDisplay = "(set)"
		}
		apiURLDisplay := cfg.APIURL
		if apiURLDisplay == "" {
			apiURLDisplay = "(not set)"
		}
		fmt.Println("api_url:", apiURLDisplay)
		fmt.Println("api_key:", apiKeyDisplay)
		return nil
	},
}

func init() {
	configSetCmd.Flags().String("api", "", "Cortex API URL to persist")
	configSetCmd.Flags().String("key", "", "API key to persist")

	configCmd.AddCommand(configSetCmd, configShowCmd)
	rootCmd.AddCommand(configCmd)
}
