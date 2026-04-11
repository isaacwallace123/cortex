package tests

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/isaacwallace123/cortex/services/brain/internal/adapters/inference"
	memoryadapter "github.com/isaacwallace123/cortex/services/brain/internal/adapters/memory"
	"github.com/isaacwallace123/cortex/services/brain/internal/adapters/telemetry"
	"github.com/isaacwallace123/cortex/services/brain/internal/application"
)

func TestParseInput_Deploy(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	uc := application.NewParseInputUseCase(
		inference.NewStubAdapter(),
		telemetry.NewLogAdapter(logger),
		memoryadapter.NewNoopAdapter(),
	)

	out, err := uc.Execute(context.Background(), application.ParseInputInput{
		SessionID: "test-session-1",
		RawInput:  "deploy the application to kubernetes",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Task.Intent == "" {
		t.Fatal("expected non-empty intent")
	}
	if out.Task.SessionID != "test-session-1" {
		t.Errorf("expected session_id test-session-1, got %s", out.Task.SessionID)
	}
	t.Logf("intent=%q entities=%v", out.Task.Intent, out.Task.Entities)
}

func TestParseInput_EmptyInput(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	uc := application.NewParseInputUseCase(
		inference.NewStubAdapter(),
		telemetry.NewLogAdapter(logger),
		memoryadapter.NewNoopAdapter(),
	)

	_, err := uc.Execute(context.Background(), application.ParseInputInput{
		SessionID: "test-session-2",
		RawInput:  "   ",
	})
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}
