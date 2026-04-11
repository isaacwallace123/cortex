package cmd

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/isaacwallace123/cortex/apps/cli/internal/client"
	"github.com/isaacwallace123/cortex/apps/cli/internal/tui"
)

var (
	apiURL string
	apiKey string

	apiClient *client.API
)

var rootCmd = &cobra.Command{
	Use:   "cortex",
	Short: "Cortex AI control plane CLI",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		apiClient = client.NewAPI(apiURL, apiKey)
		savedSess, _ := client.LoadSession()
		if savedSess.Token != "" {
			apiClient = apiClient.WithToken(savedSess.Token)
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Build the API client with any saved token.
		apiClient = client.NewAPI(apiURL, apiKey)
		savedSess, _ := client.LoadSession()
		if savedSess.Token != "" {
			apiClient = apiClient.WithToken(savedSess.Token)
		}

		// Determine whether we need to show the login form.
		// We need login when: no static API key AND (no saved token OR token is invalid).
		needsLogin := false
		if apiKey == "" {
			if savedSess.Token == "" {
				needsLogin = true
			} else {
				// Validate the saved token against the API.
				if _, err := apiClient.WhoAmI(); err != nil {
					// Token expired or revoked — clear it and show login.
					if isAuthError(err) {
						_ = client.ClearSession()
						apiClient = client.NewAPI(apiURL, "")
						needsLogin = true
					}
					// Connection errors: keep token, let the TUI show the connection error naturally.
				}
			}
		}

		m := tui.New(apiClient, "", needsLogin)
		p := tea.NewProgram(m, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return err
		}
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&apiURL, "api", envOr("CORTEX_API_URL", "http://localhost:8000"), "Cortex API URL")
	rootCmd.PersistentFlags().StringVar(&apiKey, "key", envOr("CORTEX_API_KEY", ""), "API key")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// isAuthError returns true when the error is an HTTP 401/403 from the API.
func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "(401)") || strings.Contains(msg, "(403)")
}
