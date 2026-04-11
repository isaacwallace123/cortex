package bootstrap

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/isaacwallace123/cortex/services/memory/internal/adapters/sqlite"
	"github.com/isaacwallace123/cortex/services/memory/internal/application"
	cortexlog "github.com/isaacwallace123/cortex/services/memory/internal/log"
	grpctransport "github.com/isaacwallace123/cortex/services/memory/internal/transport/grpc"
)

type App struct {
	GRPCServer *grpctransport.Server
	Logger     *slog.Logger
}

func Wire() *App {
	log := cortexlog.Echo()

	dbPath := envOr("MEMORY_DB_PATH", "/data/memory.db")

	store, err := sqlite.NewStore(dbPath)
	if err != nil {
		log.Error("[Echo] failed to open memory store",
			slog.String("path", dbPath),
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
	log.Info("[Echo] memory store ready", slog.String("path", dbPath))

	// Background TTL pruner: delete expired events on a configurable interval.
	// MEMORY_TTL_INTERVAL controls the check period in seconds (default: 3600 = 1h).
	pruneInterval := 3600
	if v := os.Getenv("MEMORY_TTL_INTERVAL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			pruneInterval = n
		}
	}
	go func() {
		t := time.NewTicker(time.Duration(pruneInterval) * time.Second)
		defer t.Stop()
		for range t.C {
			n, err := store.PruneExpired(context.Background())
			if err != nil {
				log.Error("[Echo] TTL pruner failed", slog.String("error", err.Error()))
			} else if n > 0 {
				log.Info("[Echo] TTL pruner removed expired events", slog.Int64("count", n))
			}
		}
	}()

	grpcServer := grpctransport.NewServer(
		application.NewStoreEventUseCase(store),
		application.NewGetSessionUseCase(store),
		application.NewListPlansUseCase(store),
	)

	return &App{GRPCServer: grpcServer, Logger: log}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
