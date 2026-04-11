package tests

import (
	"context"
	"testing"

	"github.com/isaacwallace123/cortex/services/inference/internal/application"
	"github.com/isaacwallace123/cortex/services/inference/internal/domain"
)

// fakeProvider is a local test double for LLMProviderPort.
type fakeProvider struct{}

func (f *fakeProvider) Complete(_ context.Context, req domain.GenerationRequest) (domain.GenerationResponse, error) {
	return domain.GenerationResponse{
		Text:         "INTENT: test intent\nENTITIES: foo, bar",
		Model:        req.Model,
		InputTokens:  10,
		OutputTokens: 20,
	}, nil
}

func (f *fakeProvider) CompleteStream(_ context.Context, req domain.GenerationRequest) (<-chan domain.GenerationChunk, error) {
	ch := make(chan domain.GenerationChunk, 1)
	ch <- domain.GenerationChunk{Token: "INTENT: test intent\nENTITIES: foo, bar", Done: true, Model: req.Model}
	close(ch)
	return ch, nil
}

func (f *fakeProvider) ListModels(_ context.Context) ([]domain.Model, error) {
	return []domain.Model{
		{Name: "llama3.2", Size: "2GB", Modified: "2025-01-01"},
	}, nil
}

func TestCompleteUseCase(t *testing.T) {
	uc := application.NewCompleteUseCase(&fakeProvider{}, "llama3.2")

	out, err := uc.Execute(context.Background(), application.CompleteInput{
		Prompt: "what is 2 + 2?",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Response.Text == "" {
		t.Fatal("expected non-empty response text")
	}
	if out.Response.Model != "llama3.2" {
		t.Errorf("expected default model llama3.2, got %s", out.Response.Model)
	}
	t.Logf("response: %s", out.Response.Text)
}

func TestCompleteUseCase_EmptyPrompt(t *testing.T) {
	uc := application.NewCompleteUseCase(&fakeProvider{}, "llama3.2")

	_, err := uc.Execute(context.Background(), application.CompleteInput{Prompt: ""})
	if err == nil {
		t.Fatal("expected error for empty prompt")
	}
}

func TestListModelsUseCase(t *testing.T) {
	uc := application.NewListModelsUseCase(&fakeProvider{})

	out, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Models) == 0 {
		t.Fatal("expected at least one model")
	}
	t.Logf("models: %v", out.Models)
}
