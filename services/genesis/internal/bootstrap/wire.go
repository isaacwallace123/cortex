package bootstrap

import (
	"log/slog"
	"os"

	arsenaladapter "github.com/isaacwallace123/cortex/services/genesis/internal/adapters/arsenal"
	inferenceadapter "github.com/isaacwallace123/cortex/services/genesis/internal/adapters/inference"
	"github.com/isaacwallace123/cortex/services/genesis/internal/adapters/store"
	"github.com/isaacwallace123/cortex/services/genesis/internal/application"
	genesislog "github.com/isaacwallace123/cortex/services/genesis/internal/log"
	grpctransport "github.com/isaacwallace123/cortex/services/genesis/internal/transport/grpc"
)

// App holds the wired Genesis service components.
type App struct {
	GRPCServer *grpctransport.Server
	Logger     *slog.Logger
}

func Wire() *App {
	log := genesislog.Genesis()

	dbPath := envOr("GENESIS_DB", "/data/genesis.db")
	proposalStore, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		log.Error("[Genesis] failed to open database",
			slog.String("path", dbPath),
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
	log.Info("[Genesis] store ready", slog.String("path", dbPath))

	inferenceAddr := envOr("INFERENCE_ADDR", "inference:9090")
	inferenceAdapter, err := inferenceadapter.NewGRPCAdapter(inferenceAddr)
	if err != nil {
		log.Error("[Genesis] failed to connect to Inference",
			slog.String("addr", inferenceAddr),
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}

	arsenalAddr := envOr("ARSENAL_ADDR", "arsenal:7373")
	arsenalAdapter, err := arsenaladapter.NewGRPCAdapter(arsenalAddr)
	if err != nil {
		log.Error("[Genesis] failed to connect to Arsenal",
			slog.String("addr", arsenalAddr),
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}

	generateUC := application.NewGenerateToolUseCase(inferenceAdapter, arsenalAdapter, proposalStore, log)
	listUC := application.NewListProposalsUseCase(proposalStore)

	grpcServer := grpctransport.NewServer(generateUC, listUC)

	log.Info("[Genesis] wired",
		slog.String("inference", inferenceAddr),
		slog.String("arsenal", arsenalAddr),
	)
	return &App{GRPCServer: grpcServer, Logger: log}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
