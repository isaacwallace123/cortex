package bootstrap

import (
	"log/slog"
	"os"

	"github.com/isaacwallace123/cortex/services/compass/internal/adapters/store"
	"github.com/isaacwallace123/cortex/services/compass/internal/application"
	cortexlog "github.com/isaacwallace123/cortex/services/compass/internal/log"
	grpctransport "github.com/isaacwallace123/cortex/services/compass/internal/transport/grpc"
)

type App struct {
	GRPCServer *grpctransport.Server
	Logger     *slog.Logger
}

func Wire() *App {
	log := cortexlog.Compass()

	dbPath := os.Getenv("COMPASS_DB")
	if dbPath == "" {
		dbPath = "/data/compass.db"
	}

	projectStore, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		log.Error("[Compass] failed to open database",
			slog.String("path", dbPath),
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}

	uc := application.New(projectStore)
	grpcServer := grpctransport.NewServer(uc)

	log.Info("[Compass] wired", slog.String("db", dbPath))
	return &App{GRPCServer: grpcServer, Logger: log}
}
