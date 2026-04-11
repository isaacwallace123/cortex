package bootstrap

import (
	"log/slog"
	"os"

	"github.com/isaacwallace123/cortex/services/forge/internal/adapters/fs"
	"github.com/isaacwallace123/cortex/services/forge/internal/application"
	cortexlog "github.com/isaacwallace123/cortex/services/forge/internal/log"
	grpctransport "github.com/isaacwallace123/cortex/services/forge/internal/transport/grpc"
)

const defaultRoot = "/workspace"

type App struct {
	GRPCServer *grpctransport.Server
	Logger     *slog.Logger
}

func Wire() *App {
	log := cortexlog.Forge()

	root := os.Getenv("FORGE_ROOT")
	if root == "" {
		root = defaultRoot
	}

	executor, err := fs.NewExecutor(root)
	if err != nil {
		log.Error("[Forge] failed to initialize filesystem executor",
			slog.String("root", root),
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}

	log.Info("[Forge] filesystem executor ready",
		slog.String("root", executor.Root),
	)

	executeStepUC := application.NewExecuteStepUseCase(executor)
	grpcServer := grpctransport.NewServer(executeStepUC)

	return &App{GRPCServer: grpcServer, Logger: log}
}
