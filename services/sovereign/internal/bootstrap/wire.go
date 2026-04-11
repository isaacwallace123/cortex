package bootstrap

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/isaacwallace123/cortex/services/sovereign/internal/adapters/store"
	"github.com/isaacwallace123/cortex/services/sovereign/internal/application"
	cortexlog "github.com/isaacwallace123/cortex/services/sovereign/internal/log"
	grpctransport "github.com/isaacwallace123/cortex/services/sovereign/internal/transport/grpc"
)

const sessionPruneInterval = 6 * time.Hour

type App struct {
	GRPCServer *grpctransport.Server
	Logger     *slog.Logger
}

func Wire() *App {
	log := cortexlog.Sovereign()

	dbPath := os.Getenv("SOVEREIGN_DB")
	if dbPath == "" {
		dbPath = "/data/sovereign.db"
	}

	identityStore, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		log.Error("[Sovereign] failed to open database",
			slog.String("path", dbPath),
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}

	tokenTTLHours := 24
	if v := os.Getenv("SOVEREIGN_TOKEN_TTL_HOURS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			tokenTTLHours = n
		}
	}

	uc := application.New(identityStore, time.Duration(tokenTTLHours)*time.Hour)

	// Seed an admin account if none exist and SOVEREIGN_ADMIN_PASSWORD is set.
	if pw := os.Getenv("SOVEREIGN_ADMIN_PASSWORD"); pw != "" {
		seedAdmin(context.Background(), uc, pw, log)
	}

	// Prune expired sessions at startup, then periodically every 6 hours.
	_ = identityStore.PruneExpired(context.Background())
	go func() {
		for range time.Tick(sessionPruneInterval) {
			if err := identityStore.PruneExpired(context.Background()); err != nil {
				log.Warn("[Sovereign] session prune failed", slog.String("error", err.Error()))
			}
		}
	}()

	grpcServer := grpctransport.NewServer(uc)
	log.Info("[Sovereign] wired",
		slog.String("db", dbPath),
		slog.Int("token_ttl_hours", tokenTTLHours),
	)
	return &App{GRPCServer: grpcServer, Logger: log}
}

func seedAdmin(ctx context.Context, uc *application.UseCases, password string, log *slog.Logger) {
	users, err := uc.ListUsers(ctx)
	if err != nil || len(users) > 0 {
		return // already seeded
	}
	adminUser := os.Getenv("SOVEREIGN_ADMIN_USER")
	if adminUser == "" {
		adminUser = "admin"
	}
	_, _, err = uc.Register(ctx, adminUser, adminUser+"@cortex.local", password, "admin", "", "")
	if err != nil {
		log.Warn("[Sovereign] failed to seed admin account", slog.String("error", err.Error()))
		return
	}
	log.Info("[Sovereign] admin account created", slog.String("username", adminUser))
}
