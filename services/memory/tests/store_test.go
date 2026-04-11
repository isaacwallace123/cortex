package tests

import (
	"context"
	"testing"

	"github.com/isaacwallace123/cortex/services/memory/internal/adapters/sqlite"
	"github.com/isaacwallace123/cortex/services/memory/internal/application"
)

func newTestStore(t *testing.T) *sqlite.Store {
	t.Helper()
	store, err := sqlite.NewStore(":memory:")
	if err != nil {
		t.Fatalf("open test store: %v", err)
	}
	return store
}

func TestStoreAndGetSession(t *testing.T) {
	store := newTestStore(t)
	storeUC := application.NewStoreEventUseCase(store)
	getUC := application.NewGetSessionUseCase(store)

	ctx := context.Background()
	sessionID := "test-session-1"

	_, err := storeUC.Execute(ctx, application.StoreEventInput{
		SessionID: sessionID,
		EventName: "brain.input.parsed",
		Payload:   `{"intent":"deploy application"}`,
	})
	if err != nil {
		t.Fatalf("store event: %v", err)
	}

	_, err = storeUC.Execute(ctx, application.StoreEventInput{
		SessionID: sessionID,
		EventName: "axiom.plan.created",
		Payload:   `{"plan_id":"plan-1","intent":"deploy application","step_count":4}`,
	})
	if err != nil {
		t.Fatalf("store plan event: %v", err)
	}

	out, err := getUC.Execute(ctx, sessionID, "")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if len(out.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(out.Events))
	}
	if out.Events[0].Name != "brain.input.parsed" {
		t.Errorf("expected first event brain.input.parsed, got %s", out.Events[0].Name)
	}
	t.Logf("session %s: %d events", sessionID, len(out.Events))
}

func TestListPlans(t *testing.T) {
	store := newTestStore(t)
	storeUC := application.NewStoreEventUseCase(store)
	listUC := application.NewListPlansUseCase(store)

	ctx := context.Background()
	sessionID := "test-session-2"

	_, err := storeUC.Execute(ctx, application.StoreEventInput{
		SessionID: sessionID,
		EventName: "axiom.plan.created",
		Payload:   `{"plan_id":"plan-abc","intent":"deploy application","step_count":5}`,
	})
	if err != nil {
		t.Fatalf("store plan: %v", err)
	}

	out, err := listUC.Execute(ctx, sessionID, "")
	if err != nil {
		t.Fatalf("list plans: %v", err)
	}
	if len(out.Plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(out.Plans))
	}
	if out.Plans[0].PlanID != "plan-abc" {
		t.Errorf("expected plan-abc, got %s", out.Plans[0].PlanID)
	}
	if out.Plans[0].StepCount != 5 {
		t.Errorf("expected 5 steps, got %d", out.Plans[0].StepCount)
	}
	t.Logf("plan %s: intent=%q steps=%d", out.Plans[0].PlanID, out.Plans[0].Intent, out.Plans[0].StepCount)
}

func TestEmptySession(t *testing.T) {
	store := newTestStore(t)
	getUC := application.NewGetSessionUseCase(store)

	out, err := getUC.Execute(context.Background(), "nonexistent-session", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Events) != 0 {
		t.Errorf("expected 0 events for unknown session, got %d", len(out.Events))
	}
}
