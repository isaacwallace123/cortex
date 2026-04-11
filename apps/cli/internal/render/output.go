package render

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/isaacwallace123/cortex/apps/cli/internal/client"
)

var sep = "  " + strings.Repeat("─", 60)

// InputHeader prints the user's prompt at the top of a command.
// The spinner will appear on the line immediately following this.
func InputHeader(w io.Writer, input string) {
	fmt.Fprintf(w, "\n  %s>%s  %s\n", aBold+aGreen, aReset, input)
}

// ChatResult renders the plan with a "To run:" hint.
// Call after stopping the thinking spinner so there is no leading blank line needed.
func ChatResult(w io.Writer, resp client.ChatResponse, input string) {
	printPlanHeader(w, resp)
	fmt.Fprintf(w, "\n  %s  cortex run \"%s\"\n", gray("To run:  "), input)
	fmt.Fprintf(w, "  %s  cortex run --session %s --plan %s\n",
		gray("Re-run:  "), resp.SessionID, resp.Plan.PlanID,
	)
	fmt.Fprintf(w, "\n")
}

// RunHeader renders the plan before execution begins (no "To run:" hint).
func RunHeader(w io.Writer, resp client.ChatResponse) {
	printPlanHeader(w, resp)
	fmt.Fprintf(w, "\n")
}

func printPlanHeader(w io.Writer, resp client.ChatResponse) {
	stepCount := len(resp.Plan.Steps)
	stepWord := "step"
	if stepCount != 1 {
		stepWord = "steps"
	}

	fmt.Fprintf(w, "\n  %s  %s·%s  %s%d %s%s  %ssession %s%s\n",
		bold(resp.Intent),
		aDim, aReset,
		aDim, stepCount, stepWord, aReset,
		aDim, shortID(resp.SessionID), aReset,
	)
	fmt.Fprintf(w, "\n%s\n", sep)
	for _, step := range resp.Plan.Steps {
		executor := step.Executor
		if executor == "" {
			executor = "?"
		}
		desc := truncate(step.Description, 38)
		fmt.Fprintf(w, "   %s%2d%s  %-38s  %s%10s%s  %s\n",
			aCyan, step.Index, aReset,
			desc,
			aDim, executor, aReset,
			verdictStr(step.Verdict),
		)
		if step.Executor == "infer" {
			fmt.Fprintf(w, "       %sdirect answer%s\n", aDim, aReset)
		} else if step.Command != "" {
			fmt.Fprintf(w, "       %s$%s %s\n", aDim, aReset, step.Command)
		}
	}
	fmt.Fprintf(w, "%s\n", sep)
}

// AegisBlockedStep prints the Aegis block notice for a step that was denied or
// is pending human review. The caller reads the override response from stdin.
func AegisBlockedStep(w io.Writer, step client.Step) {
	fmt.Fprintf(w, "\n  %s⚠%s  %sstep %d blocked by Aegis%s\n",
		aYellow, aReset, aBold, step.Index, aReset,
	)
	if step.Description != "" {
		fmt.Fprintf(w, "     %s\n", step.Description)
	}
	fmt.Fprintf(w, "     executor  %s%s%s\n", aCyan, step.Executor, aReset)
	if step.Command != "" {
		fmt.Fprintf(w, "     command   %s\n", step.Command)
	}
	fmt.Fprintf(w, "     verdict   %s\n\n", verdictStr(step.Verdict))
	fmt.Fprintf(w, "  Allow anyway? %s[y/N]%s  ", aDim, aReset)
}

// PlanList renders a list of plan summaries for a session.
func PlanList(w io.Writer, sessionID string, plans []client.PlanSummary) {
	if len(plans) == 0 {
		fmt.Fprintf(w, "\n  %s  no plans found\n\n", gray("session "+sessionID))
		return
	}
	fmt.Fprintf(w, "\n  %ssession %s%s  %s%d plans%s\n",
		aDim, sessionID, aReset,
		aDim, len(plans), aReset,
	)
	fmt.Fprintf(w, "%s\n", sep)
	for _, p := range plans {
		age := time.Since(time.Unix(p.CreatedAt, 0)).Round(time.Second)
		stepWord := "steps"
		if p.StepCount == 1 {
			stepWord = "step"
		}
		fmt.Fprintf(w, "  %s%s%s  %-42s  %s%d %s%s  %s\n",
			aCyan, shortID(p.PlanID), aReset,
			truncate(p.Intent, 42),
			aDim, p.StepCount, stepWord, aReset,
			gray(age.String()+" ago"),
		)
	}
	fmt.Fprintf(w, "%s\n", sep)
	fmt.Fprintf(w, "\n  %s  cortex run --session %s --plan <id>\n\n",
		gray("To run:"), sessionID,
	)
}

// RunConfirmPrompt prints the interactive confirmation line.
// The caller reads the response from stdin; this only writes the prompt text.
func RunConfirmPrompt(w io.Writer, steps int) {
	word := "step"
	if steps != 1 {
		word = "steps"
	}
	fmt.Fprintf(w, "  Run %d %s? %s[y/N]%s  ", steps, word, aDim, aReset)
}

// Aborted prints the abort message.
func Aborted(w io.Writer) {
	fmt.Fprintf(w, "\n  %s\n\n", dim("aborted"))
}

// ExecuteHeader prints the "Executing" section divider.
func ExecuteHeader(w io.Writer) {
	fmt.Fprintf(w, "  %s◆%s  %s\n\n", aCyan, aReset, bold("Executing"))
}

// StepResult prints the result of a single executed step.
func StepResult(w io.Writer, r client.StepResult) {
	if r.Skipped {
		fmt.Fprintf(w, "   %s%2d%s  %s  %s\n\n",
			aCyan, r.StepIndex, aReset,
			dim("–"),
			gray(r.SkipReason),
		)
		return
	}

	// infer steps: show the answer as a plain block, no timing metadata.
	if r.Executor == "infer" {
		fmt.Fprintf(w, "   %s%2d%s  %s\n", aCyan, r.StepIndex, aReset, dim("answer"))
		if r.Stdout != "" {
			fmt.Fprintf(w, "\n")
			for _, line := range strings.Split(strings.TrimRight(r.Stdout, "\n"), "\n") {
				fmt.Fprintf(w, "       %s\n", line)
			}
		}
		fmt.Fprintf(w, "\n")
		return
	}

	icon := green("✓")
	if r.ExitCode != 0 {
		icon = red("✗")
	}
	fmt.Fprintf(w, "   %s%2d%s  %s  %s\n",
		aCyan, r.StepIndex, aReset,
		icon,
		gray(fmt.Sprintf("%dms", r.DurationMs)),
	)
	if r.Stdout != "" {
		for _, line := range strings.Split(strings.TrimRight(r.Stdout, "\n"), "\n") {
			fmt.Fprintf(w, "       %s\n", line)
		}
	}
	if r.Stderr != "" {
		for _, line := range strings.Split(strings.TrimRight(r.Stderr, "\n"), "\n") {
			fmt.Fprintf(w, "       %s\n", red(line))
		}
	}
	fmt.Fprintf(w, "\n")
}

// RunSummary prints the final execution summary line.
func RunSummary(w io.Writer, total, executed, skipped int) {
	summary := bold(fmt.Sprintf("%d/%d complete", executed, total))
	if skipped > 0 {
		summary = fmt.Sprintf("%d/%d executed  %s", executed, total, gray(fmt.Sprintf("%d skipped", skipped)))
	}
	fmt.Fprintf(w, "  %s  %s\n\n", green("✓"), summary)
}

// SubscribeHeader prints the live-event stream header.
func SubscribeHeader(w io.Writer, filter string) {
	label := filter
	if label == "" {
		label = "all events"
	}
	fmt.Fprintf(w, "\n  %s◆%s  %ssubscribed%s  %s(%s)%s\n\n",
		aCyan, aReset,
		aBold, aReset,
		aDim, label, aReset,
	)
	fmt.Fprintf(w, "  %s%-8s  %-10s  %-28s  %s%s\n",
		aDim, "TIME", "SUBSYSTEM", "EVENT", "PAYLOAD", aReset,
	)
	fmt.Fprintf(w, "  %s%s%s\n", aDim, strings.Repeat("─", 72), aReset)
}

// EventLine prints a single Nerva event with timestamp in streaming format.
func EventLine(w io.Writer, ts, subsystem, eventName, sessionID, payload string) {
	if len(payload) > 52 {
		payload = payload[:49] + "..."
	}
	fmt.Fprintf(w, "  %s  %s%-10s%s  %s%-28s%s  %s\n",
		gray(ts),
		aCyan, subsystem, aReset,
		aDim, eventName, aReset,
		dim(payload),
	)
}

// SessionHistory prints a session's event timeline.
func SessionHistory(w io.Writer, resp client.SessionResponse) {
	if len(resp.Events) == 0 {
		fmt.Fprintf(w, "\n  %s  no events found\n\n", gray("session "+resp.SessionID))
		return
	}
	fmt.Fprintf(w, "\n  %ssession %s%s  %s%d events%s\n",
		aDim, resp.SessionID, aReset,
		aDim, len(resp.Events), aReset,
	)
	fmt.Fprintf(w, "%s\n", sep)
	for _, e := range resp.Events {
		t := time.Unix(e.StoredAt, 0).Local().Format("15:04:05")
		payload := e.Payload
		if len(payload) > 56 {
			payload = payload[:53] + "..."
		}
		fmt.Fprintf(w, "  %s  %s%-28s%s  %s\n",
			gray(t),
			aCyan, e.EventName, aReset,
			dim(payload),
		)
	}
	fmt.Fprintf(w, "\n")
}

// Error prints a styled error message.
func Error(w io.Writer, msg string) {
	fmt.Fprintf(w, "\n  %s  %s\n\n", red("error"), msg)
}
