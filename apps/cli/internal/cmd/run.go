package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/isaacwallace123/cortex/apps/cli/internal/client"
	"github.com/isaacwallace123/cortex/apps/cli/internal/render"
)

var (
	runSessionID string
	runPlanID    string
	runYes       bool
)

var runCmd = &cobra.Command{
	Use:   "run <input>",
	Short: "Plan then execute all approved steps",
	RunE: func(cmd *cobra.Command, args []string) error {
		var steps []client.Step
		var sessionForRun, planForRun string

		if runPlanID != "" {
			if runSessionID == "" {
				render.Error(os.Stderr, "--session is required when using --plan")
				os.Exit(1)
			}
			spin := render.StartSpinner(os.Stdout, "Loading plan...")
			stored, err := apiClient.GetPlan(runSessionID, runPlanID)
			spin.Stop()
			if err != nil {
				render.Error(os.Stderr, err.Error())
				os.Exit(1)
			}
			chatResp := client.ChatResponse{
				SessionID: stored.SessionID,
				Intent:    stored.Intent,
				Plan:      client.Plan{PlanID: stored.PlanID, Steps: stored.Steps},
			}
			render.RunHeader(os.Stdout, chatResp)
			steps = stored.Steps
			sessionForRun = stored.SessionID
			planForRun = stored.PlanID
		} else {
			if len(args) == 0 {
				render.Error(os.Stderr, "run requires an input or --plan <id>\n  example: cortex run \"check disk usage\"")
				os.Exit(1)
			}
			input := strings.Join(args, " ")
			render.InputHeader(os.Stdout, input)

			spin := render.StartSpinner(os.Stdout, "Thinking...")
			chatResp, err := apiClient.Chat(runSessionID, input)
			spin.Stop()
			if err != nil {
				render.Error(os.Stderr, err.Error())
				os.Exit(1)
			}
			render.RunHeader(os.Stdout, chatResp)
			steps = chatResp.Plan.Steps
			sessionForRun = chatResp.SessionID
			planForRun = chatResp.Plan.PlanID
		}

		if !runYes {
			render.RunConfirmPrompt(os.Stdout, len(steps))
			scanner := bufio.NewScanner(os.Stdin)
			if !scanner.Scan() || strings.ToLower(strings.TrimSpace(scanner.Text())) != "y" {
				render.Aborted(os.Stdout)
				return nil
			}
			fmt.Fprintln(os.Stdout)
		}

		render.ExecuteHeader(os.Stdout)

		total := len(steps)
		executed, skipped := 0, 0

		for _, step := range steps {
			if step.Verdict != "allow" {
				render.AegisBlockedStep(os.Stdout, step)
				sc := bufio.NewScanner(os.Stdin)
				if sc.Scan() && strings.ToLower(strings.TrimSpace(sc.Text())) == "y" {
					fmt.Fprintln(os.Stdout)
					step.Verdict = "allow"
				} else {
					render.StepResult(os.Stdout, client.StepResult{
						StepIndex:  step.Index,
						Skipped:    true,
						SkipReason: "user declined Aegis override",
					})
					skipped++
					continue
				}
			}

			spin := render.StartSpinner(os.Stdout, fmt.Sprintf("%sstep %d of %d%s", "\033[2m", step.Index, total, "\033[0m"))
			result, err := apiClient.RunStep(sessionForRun, planForRun, step)
			spin.Stop()

			if err != nil {
				render.Error(os.Stderr, fmt.Sprintf("step %d: %s", step.Index, err.Error()))
				os.Exit(1)
			}
			render.StepResult(os.Stdout, result)
			if result.Skipped {
				skipped++
			} else {
				executed++
			}
		}
		render.RunSummary(os.Stdout, total, executed, skipped)
		return nil
	},
}

func init() {
	runCmd.Flags().StringVar(&runSessionID, "session", "", "Session ID to use or resume")
	runCmd.Flags().StringVar(&runPlanID, "plan", "", "Plan ID to re-execute")
	runCmd.Flags().BoolVarP(&runYes, "yes", "y", false, "Skip confirmation prompt")
	rootCmd.AddCommand(runCmd)
}
