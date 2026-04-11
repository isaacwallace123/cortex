package bootstrap

import (
	"log/slog"
	"os"

	"github.com/isaacwallace123/cortex/services/chat/internal/adapters/store"
	"github.com/isaacwallace123/cortex/services/chat/internal/application"
	cortexlog "github.com/isaacwallace123/cortex/services/chat/internal/log"
	grpctransport "github.com/isaacwallace123/cortex/services/chat/internal/transport/grpc"
)

const defaultDBPath = "/data/chat.db"

type App struct {
	GRPCServer *grpctransport.Server
	Logger     *slog.Logger
}

func Wire() *App {
	log := cortexlog.Chat()

	dbPath := os.Getenv("CHAT_DB_PATH")
	if dbPath == "" {
		dbPath = defaultDBPath
	}

	sqliteStore, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		log.Error("[Chat] failed to open store",
			slog.String("path", dbPath),
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
	log.Info("[Chat] store ready", slog.String("path", dbPath))

	return &App{
		GRPCServer: grpctransport.NewServer(
			application.NewCreateChatUseCase(sqliteStore),
			application.NewGetChatUseCase(sqliteStore),
			application.NewListChatsUseCase(sqliteStore),
			application.NewDeleteChatUseCase(sqliteStore),
			application.NewAppendMessageUseCase(sqliteStore),
			application.NewGetMessagesUseCase(sqliteStore),
		),
		Logger: log,
	}
}
