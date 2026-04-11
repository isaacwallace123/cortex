package bootstrap

import (
	"log/slog"
	"os"

	"github.com/isaacwallace123/cortex/services/inference/internal/adapters/ollama"
	"github.com/isaacwallace123/cortex/services/inference/internal/application"
	cortexlog "github.com/isaacwallace123/cortex/services/inference/internal/log"
	grpctransport "github.com/isaacwallace123/cortex/services/inference/internal/transport/grpc"
)

// App holds the composed inference service ready to serve.
type App struct {
	GRPCServer *grpctransport.Server
	Logger     *slog.Logger
}

// Wire composes all adapters, use cases, and transport.
func Wire() *App {
	log := cortexlog.Inference()

	ollamaURL := envOr("OLLAMA_URL", "http://localhost:11434")
	defaultModel := envOr("INFERENCE_DEFAULT_MODEL", "llama3.2")

	provider := ollama.NewAdapter(ollamaURL)

	completeUC := application.NewCompleteUseCase(provider, defaultModel)
	completeStreamUC := application.NewCompleteStreamUseCase(provider, defaultModel)
	listModelsUC := application.NewListModelsUseCase(provider)

	grpcServer := grpctransport.NewServer(completeUC, completeStreamUC, listModelsUC)

	log.Info("[inference] service wired",
		slog.String("ollama_url", ollamaURL),
		slog.String("default_model", defaultModel),
	)

	return &App{
		GRPCServer: grpcServer,
		Logger:     log,
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
