package bootstrap

import (
	"log/slog"
	"os"

	brainadapter "github.com/isaacwallace123/cortex/services/catalyst/internal/adapters/brain"
	"github.com/isaacwallace123/cortex/services/catalyst/internal/adapters/store"
	"github.com/isaacwallace123/cortex/services/catalyst/internal/application"
	catalystlog "github.com/isaacwallace123/cortex/services/catalyst/internal/log"
	grpctransport "github.com/isaacwallace123/cortex/services/catalyst/internal/transport/grpc"
)

// App holds the wired Catalyst service components.
type App struct {
	GRPCServer *grpctransport.Server
	Logger     *slog.Logger
}

func Wire() *App {
	log := catalystlog.Catalyst()

	dbPath := envOr("CATALYST_DB", "/data/catalyst.db")
	skillStore, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		log.Error("[Catalyst] failed to open database",
			slog.String("path", dbPath),
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
	log.Info("[Catalyst] skill store ready", slog.String("path", dbPath))

	// Brain adapter for delegating plan execution.
	brainAddr := envOr("BRAIN_ADDR", "brain:8090")
	brainAdapter, err := brainadapter.NewGRPCAdapter(brainAddr)
	if err != nil {
		log.Error("[Catalyst] failed to connect to Brain",
			slog.String("addr", brainAddr),
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
	log.Info("[Catalyst] Brain adapter ready", slog.String("addr", brainAddr))

	createUC := application.NewCreateSkillUseCase(skillStore)
	getUC := application.NewGetSkillUseCase(skillStore)
	listUC := application.NewListSkillsUseCase(skillStore)
	deleteUC := application.NewDeleteSkillUseCase(skillStore)
	executeUC := application.NewExecuteSkillUseCase(skillStore, brainAdapter)

	grpcServer := grpctransport.NewServer(createUC, getUC, listUC, deleteUC, executeUC)

	log.Info("[Catalyst] wired", slog.String("db", dbPath), slog.String("brain", brainAddr))
	return &App{GRPCServer: grpcServer, Logger: log}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
