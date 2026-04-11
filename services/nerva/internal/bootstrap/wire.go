package bootstrap

import (
	"log/slog"
	"os"

	"github.com/isaacwallace123/cortex/services/nerva/internal/adapters/store"
	"github.com/isaacwallace123/cortex/services/nerva/internal/application"
	cortexlog "github.com/isaacwallace123/cortex/services/nerva/internal/log"
	grpctransport "github.com/isaacwallace123/cortex/services/nerva/internal/transport/grpc"
)

const defaultDBPath = "/data/nerva.db"

type App struct {
	GRPCServer *grpctransport.Server
	Logger     *slog.Logger
}

func Wire() *App {
	log := cortexlog.Nerva()

	dbPath := os.Getenv("NERVA_DB_PATH")
	if dbPath == "" {
		dbPath = defaultDBPath
	}

	sqliteStore, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		log.Error("[Nerva] failed to open event store",
			slog.String("path", dbPath),
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}

	log.Info("[Nerva] event store ready", slog.String("path", dbPath))

	bus := application.NewBus()
	publishUC := application.NewPublishUseCase(sqliteStore, bus)
	grpcServer := grpctransport.NewServer(publishUC, bus)

	return &App{GRPCServer: grpcServer, Logger: log}
}
