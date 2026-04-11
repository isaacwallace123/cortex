package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/isaacwallace123/cortex/apps/cli/internal/client"
	"github.com/isaacwallace123/cortex/apps/cli/internal/render"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in and save a session token (~/.cortex/session.json)",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Print("Username: ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		username := strings.TrimSpace(scanner.Text())
		if username == "" {
			render.Error(os.Stderr, "username required")
			os.Exit(1)
		}
		fmt.Print("Password: ")
		pwBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			render.Error(os.Stderr, "could not read password: "+err.Error())
			os.Exit(1)
		}
		
		unauthAPI := client.NewAPI(apiURL, apiKey)
		h, _ := os.Hostname()
		resp, err := unauthAPI.Login(username, string(pwBytes), h)
		if err != nil {
			render.Error(os.Stderr, "login failed: "+err.Error())
			os.Exit(1)
		}
		if err := client.SaveSession(client.SavedSession{
			Token:     resp.Session.Token,
			UserID:    resp.User.UserID,
			Username:  resp.User.Username,
			ExpiresAt: resp.Session.ExpiresAt,
		}); err != nil {
			render.Error(os.Stderr, "could not save session: "+err.Error())
			os.Exit(1)
		}
		fmt.Printf("Logged in as %s\n", resp.User.Username)
		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Revoke the current session token",
	RunE: func(cmd *cobra.Command, args []string) error {
		savedSess, _ := client.LoadSession()
		if savedSess.Token == "" {
			fmt.Println("Not logged in.")
			return nil
		}
		if _, err := apiClient.Logout(); err != nil {
			render.Error(os.Stderr, "logout failed: "+err.Error())
			os.Exit(1)
		}
		_ = client.ClearSession()
		fmt.Println("Logged out.")
		return nil
	},
}

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show the currently authenticated identity",
	RunE: func(cmd *cobra.Command, args []string) error {
		savedSess, _ := client.LoadSession()
		if savedSess.Token == "" && apiKey == "" {
			fmt.Println("Not logged in. Run: cortex login")
			return nil
		}
		me, err := apiClient.WhoAmI()
		if err != nil {
			render.Error(os.Stderr, err.Error())
			os.Exit(1)
		}
		fmt.Printf("user_id:  %s\nusername: %s\nroles:    %s\n", me.UserID, me.Username, strings.Join(me.Roles, ", "))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(whoamiCmd)
}
