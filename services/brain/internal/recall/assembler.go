// Package recall implements the Recall context-assembly layer.
// Recall queries Echo (session memory) and Vault (long-term knowledge),
// ranks results, and assembles a token-budgeted context window that is
// injected into Prism and Axiom prompts before inference.
//
// Recall is stateless — it owns retrieval strategy but no data itself.
package recall

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/grpc/metadata"

	"github.com/isaacwallace123/cortex/services/brain/internal/ports"
)

const (
	// DefaultTokenBudget is the approximate token limit for assembled context.
	// 1 token ≈ 4 characters (rough English-language estimate).
	DefaultTokenBudget  = 2000
	approxCharsPerToken = 4
)

// Retrieved holds the raw results from Echo + Vault retrieval.
// The caller is responsible for formatting these into prompt sections.
type Retrieved struct {
	SessionEvents    []ports.StoredEvent
	KnowledgeEntries []ports.KnowledgeEntry
}

// Assembler queries Echo (session memory) and Vault (knowledge store)
// and assembles context for injection into Prism/Axiom prompts.
// Recall owns retrieval strategy — it decides what to fetch and how much.
type Assembler struct {
	memory ports.MemoryPort
	vault  ports.VaultPort
}

// NewAssembler creates a new Recall Assembler backed by the given ports.
func NewAssembler(memory ports.MemoryPort, vault ports.VaultPort) *Assembler {
	return &Assembler{memory: memory, vault: vault}
}

// metaVal extracts the first value of a gRPC incoming-metadata key from ctx.
func metaVal(ctx context.Context, key string) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get(key); len(vals) > 0 {
			return vals[0]
		}
	}
	return ""
}

// Retrieve fetches raw context from Echo and Vault for a given session + query.
// maxKnowledge limits the number of Vault entries returned (default 5 if 0).
// user_id and workspace_id are extracted from the incoming gRPC metadata so that
// Echo and Vault results are scoped to the authenticated caller.
// Errors from either store are silently swallowed — partial results are still
// useful and retrieval failures must not block plan creation.
func (a *Assembler) Retrieve(ctx context.Context, sessionID, query string, maxKnowledge int) Retrieved {
	if maxKnowledge <= 0 {
		maxKnowledge = 5
	}
	userID := metaVal(ctx, "x-user-id")
	workspaceID := metaVal(ctx, "x-workspace-id")

	var r Retrieved
	if events, err := a.memory.GetSession(ctx, sessionID); err == nil {
		r.SessionEvents = events
	}
	if query != "" {
		if entries, err := a.vault.Search(ctx, userID, workspaceID, query, maxKnowledge); err == nil {
			r.KnowledgeEntries = entries
		}
	}
	return r
}

// BuildConversationHistory reconstructs the prior user↔Cortex turns from Echo
// session events into a format suitable for injection into a conversational prompt.
// It pairs cortex.turn.user events with cortex.turn.assistant events in order.
// Returns an empty string when there are no prior turns.
func BuildConversationHistory(events []ports.StoredEvent) string {
	type turn struct {
		role    string // "User" | "Cortex"
		content string
	}
	var turns []turn

	for _, e := range events {
		switch e.Name {
		case "cortex.turn.user":
			var v struct {
				Content string `json:"content"`
			}
			if json.Unmarshal([]byte(e.Payload), &v) == nil && v.Content != "" {
				turns = append(turns, turn{role: "User", content: v.Content})
			}
		case "cortex.turn.assistant":
			var v struct {
				Content string `json:"content"`
			}
			if json.Unmarshal([]byte(e.Payload), &v) == nil && v.Content != "" {
				turns = append(turns, turn{role: "Cortex", content: v.Content})
			}
		}
	}

	if len(turns) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, t := range turns {
		sb.WriteString(t.role + ": " + t.content + "\n")
	}
	return sb.String()
}

// BuildSessionContext converts Echo session events into a concise prompt section
// showing prior turns: intent + the steps that were planned.
// Only prism.input.parsed and axiom.plan.created events are used.
func BuildSessionContext(events []ports.StoredEvent) string {
	type turn struct {
		intent string
		steps  []string
	}
	var turns []turn
	lastRawInput := ""

	for _, e := range events {
		switch e.Name {
		case "prism.input.parsed":
			var v struct {
				RawInput string `json:"raw_input"`
			}
			if json.Unmarshal([]byte(e.Payload), &v) == nil && v.RawInput != "" {
				lastRawInput = v.RawInput
			}
		case "axiom.plan.created":
			var v struct {
				Intent string `json:"intent"`
				Steps  []struct {
					Description string `json:"description"`
					Executor    string `json:"executor"`
					Command     string `json:"command"`
				} `json:"steps"`
			}
			if json.Unmarshal([]byte(e.Payload), &v) != nil {
				continue
			}
			intent := v.Intent
			if intent == "" {
				intent = lastRawInput
			}
			t := turn{intent: intent}
			for _, s := range v.Steps {
				cmd := s.Command
				if cmd == "" {
					cmd = s.Description
				}
				t.steps = append(t.steps, fmt.Sprintf("%s [%s]", cmd, s.Executor))
			}
			turns = append(turns, t)
		}
	}

	if len(turns) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Prior turns in this session:\n")
	for i, t := range turns {
		sb.WriteString(fmt.Sprintf("Turn %d: %q\n", i+1, t.intent))
		for _, s := range t.steps {
			sb.WriteString(fmt.Sprintf("  - %s\n", s))
		}
	}
	return sb.String()
}

// BuildKnowledgeContext formats Vault knowledge entries into a prompt section.
func BuildKnowledgeContext(entries []ports.KnowledgeEntry) string {
	if len(entries) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("Relevant knowledge from long-term memory:\n")
	for _, e := range entries {
		sb.WriteString(fmt.Sprintf("  - %s\n", e.Content))
	}
	return sb.String()
}

// AssembleContext combines session and knowledge context into a single string
// that fits within the given character budget.
// Knowledge context is placed first (highest relevance), session context second.
// Pass charBudget <= 0 to use the default (DefaultTokenBudget × approxCharsPerToken).
func AssembleContext(sessionContext, knowledgeContext string, charBudget int) string {
	if charBudget <= 0 {
		charBudget = DefaultTokenBudget * approxCharsPerToken
	}
	var parts []string
	remaining := charBudget

	if knowledgeContext != "" && remaining > 0 {
		kc := truncate(knowledgeContext, remaining)
		parts = append(parts, kc)
		remaining -= len(kc)
	}
	if sessionContext != "" && remaining > 0 {
		sc := truncate(sessionContext, remaining)
		parts = append(parts, sc)
	}
	return strings.Join(parts, "\n")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
