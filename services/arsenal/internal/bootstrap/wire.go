// Package bootstrap wires the Arsenal service and seeds default tools.
package bootstrap

import (
	"context"
	"log/slog"
	"os"

	"github.com/isaacwallace123/cortex/services/arsenal/internal/adapters/store"
	"github.com/isaacwallace123/cortex/services/arsenal/internal/application"
	"github.com/isaacwallace123/cortex/services/arsenal/internal/domain"
	cortexlog "github.com/isaacwallace123/cortex/services/arsenal/internal/log"
	grpctransport "github.com/isaacwallace123/cortex/services/arsenal/internal/transport/grpc"
)

// App holds the wired service components.
type App struct {
	GRPCServer *grpctransport.Server
	Logger     *slog.Logger
}

func Wire() *App {
	log := cortexlog.Arsenal()

	dbPath := os.Getenv("ARSENAL_DB")
	if dbPath == "" {
		dbPath = "/data/arsenal.db"
	}

	toolStore, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		log.Error("[Arsenal] failed to open database",
			slog.String("path", dbPath),
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}

	registerUC := application.NewRegisterToolUseCase(toolStore)
	getUC := application.NewGetToolUseCase(toolStore)
	listUC := application.NewListToolsUseCase(toolStore)
	deleteUC := application.NewDeleteToolUseCase(toolStore)

	// Seed built-in tools (no-op if they already exist — upsert is idempotent).
	seedDefaults(context.Background(), registerUC, log)

	grpcServer := grpctransport.NewServer(registerUC, getUC, listUC, deleteUC)

	log.Info("[Arsenal] wired", slog.String("db", dbPath))
	return &App{GRPCServer: grpcServer, Logger: log}
}

func seedDefaults(ctx context.Context, uc *application.RegisterToolUseCase, log *slog.Logger) {
	for _, t := range domain.DefaultTools() {
		t := t // capture
		if _, err := uc.Execute(ctx, &t); err != nil {
			log.Warn("[Arsenal] failed to seed tool",
				slog.String("name", t.Name),
				slog.String("error", err.Error()),
			)
		}
	}
	log.Info("[Arsenal] default tools seeded", slog.Int("count", len(domain.DefaultTools())))
}
