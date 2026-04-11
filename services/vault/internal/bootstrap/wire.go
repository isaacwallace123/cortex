package bootstrap

import (
	"log/slog"
	"os"

	"github.com/isaacwallace123/cortex/services/vault/internal/adapters/store"
	"github.com/isaacwallace123/cortex/services/vault/internal/application"
	cortexlog "github.com/isaacwallace123/cortex/services/vault/internal/log"
	grpctransport "github.com/isaacwallace123/cortex/services/vault/internal/transport/grpc"
)

const defaultDBPath = "/data/vault.db"

type App struct {
	GRPCServer *grpctransport.Server
	Logger     *slog.Logger
}

func Wire() *App {
	log := cortexlog.Vault()

	dbPath := os.Getenv("VAULT_DB_PATH")
	if dbPath == "" {
		dbPath = defaultDBPath
	}

	sqliteStore, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		log.Error("[Vault] failed to open store",
			slog.String("path", dbPath),
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
	log.Info("[Vault] store ready", slog.String("path", dbPath))

	storeUC := application.NewStoreUseCase(sqliteStore)
	searchUC := application.NewSearchUseCase(sqliteStore)
	deleteUC := application.NewDeleteUseCase(sqliteStore)

	return &App{
		GRPCServer: grpctransport.NewServer(storeUC, searchUC, deleteUC),
		Logger:     log,
	}
}
