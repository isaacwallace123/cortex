package bootstrap

import (
	"log/slog"
	"os"

	"github.com/isaacwallace123/cortex/services/workspace/internal/adapters/store"
	"github.com/isaacwallace123/cortex/services/workspace/internal/application"
	cortexlog "github.com/isaacwallace123/cortex/services/workspace/internal/log"
	grpctransport "github.com/isaacwallace123/cortex/services/workspace/internal/transport/grpc"
)

type App struct {
	GRPCServer *grpctransport.Server
	Logger     *slog.Logger
}

func Wire() *App {
	log := cortexlog.Workspace()

	dbPath := os.Getenv("WORKSPACE_DB")
	if dbPath == "" {
		dbPath = "/data/workspace.db"
	}

	workspaceStore, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		log.Error("[Workspace] failed to open database",
			slog.String("path", dbPath),
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}

	uc := application.New(workspaceStore)
	grpcServer := grpctransport.NewServer(uc)

	log.Info("[Workspace] wired", slog.String("db", dbPath))
	return &App{GRPCServer: grpcServer, Logger: log}
}
