package tests

import (
	"context"
	"io"
	"log/slog"
	"testing"

	arsenaladapter "github.com/isaacwallace123/cortex/services/brain/internal/adapters/arsenal"
	compassadapter "github.com/isaacwallace123/cortex/services/brain/internal/adapters/compass"
	courieradapter "github.com/isaacwallace123/cortex/services/brain/internal/adapters/courier"
	"github.com/isaacwallace123/cortex/services/brain/internal/adapters/inference"
	memoryadapter "github.com/isaacwallace123/cortex/services/brain/internal/adapters/memory"
	policyadapter "github.com/isaacwallace123/cortex/services/brain/internal/adapters/policy"
	"github.com/isaacwallace123/cortex/services/brain/internal/adapters/telemetry"
	vaultadapter "github.com/isaacwallace123/cortex/services/brain/internal/adapters/vault"
	"github.com/isaacwallace123/cortex/services/brain/internal/application"
	"github.com/isaacwallace123/cortex/services/brain/internal/domain"
	"github.com/isaacwallace123/cortex/services/brain/internal/recall"
)

func TestCreatePlan_Deploy(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	uc := application.NewCreatePlanUseCase(
		inference.NewStubAdapter(),
		telemetry.NewLogAdapter(logger),
		memoryadapter.NewNoopAdapter(),
		policyadapter.NewNoopAdapter(),
		courieradapter.NewNoopAdapter(),
		recall.NewAssembler(memoryadapter.NewNoopAdapter(), vaultadapter.NewNoopAdapter()),
		arsenaladapter.NoopAdapter{},
		compassadapter.NoopAdapter{},
	)

	out, err := uc.Execute(context.Background(), application.CreatePlanInput{
		Task: domain.Task{
			SessionID: "test-session-3",
			Intent:    "deploy application",
			Entities:  []string{"application", "kubernetes"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Plan.ID == "" {
		t.Fatal("expected non-empty plan ID")
	}
	if len(out.Plan.Steps) == 0 {
		t.Fatal("expected at least one step")
	}
	t.Logf("plan_id=%s steps=%d", out.Plan.ID, len(out.Plan.Steps))
	for _, s := range out.Plan.Steps {
		t.Logf("  step %d: %s [%s]", s.Index, s.Description, s.Executor)
	}
}

func TestCreatePlan_UnknownIntent(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	uc := application.NewCreatePlanUseCase(
		inference.NewStubAdapter(),
		telemetry.NewLogAdapter(logger),
		memoryadapter.NewNoopAdapter(),
		policyadapter.NewNoopAdapter(),
		courieradapter.NewNoopAdapter(),
		recall.NewAssembler(memoryadapter.NewNoopAdapter(), vaultadapter.NewNoopAdapter()),
		arsenaladapter.NoopAdapter{},
		compassadapter.NoopAdapter{},
	)

	_, err := uc.Execute(context.Background(), application.CreatePlanInput{
		Task: domain.Task{
			SessionID: "test-session-4",
			Intent:    "unknown",
		},
	})
	if err == nil {
		t.Fatal("expected error for unknown intent")
	}
}
