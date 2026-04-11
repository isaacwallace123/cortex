package bootstrap

import (
	"log/slog"
	"net/http"
	"os"

	brainadapter "github.com/isaacwallace123/cortex/services/api/internal/adapters/brain"
	chatadapter "github.com/isaacwallace123/cortex/services/api/internal/adapters/chat"
	compassadapter "github.com/isaacwallace123/cortex/services/api/internal/adapters/compass"
	memoryadapter "github.com/isaacwallace123/cortex/services/api/internal/adapters/memory"
	policyadapter "github.com/isaacwallace123/cortex/services/api/internal/adapters/policy"
	sovereignadapter "github.com/isaacwallace123/cortex/services/api/internal/adapters/sovereign"
	workspaceadapter "github.com/isaacwallace123/cortex/services/api/internal/adapters/workspace"
	cortexlog "github.com/isaacwallace123/cortex/services/api/internal/log"
	apihttp "github.com/isaacwallace123/cortex/services/api/internal/transport/http"
)

// App holds the composed API gateway ready to serve.
type App struct {
	Handler http.Handler
	Logger  *slog.Logger
}

func Wire() *App {
	log := cortexlog.API()

	brainAddr := envOr("BRAIN_GRPC_ADDR", "localhost:8090")
	brain, err := brainadapter.NewClient(brainAddr)
	if err != nil {
		log.Error("[api] failed to connect to brain service",
			slog.String("addr", brainAddr),
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
	log.Info("[api] connected to brain service", slog.String("addr", brainAddr))

	memoryAddr := envOr("MEMORY_GRPC_ADDR", "localhost:7070")
	memory, err := memoryadapter.NewClient(memoryAddr)
	if err != nil {
		log.Error("[api] failed to connect to memory service",
			slog.String("addr", memoryAddr),
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
	log.Info("[api] connected to memory service", slog.String("addr", memoryAddr))

	var chatClient *chatadapter.Client
	if chatAddr := envOr("CHAT_GRPC_ADDR", ""); chatAddr != "" {
		c, err := chatadapter.NewClient(chatAddr)
		if err != nil {
			log.Warn("[api] failed to connect to chat service, chat endpoints disabled",
				slog.String("addr", chatAddr),
				slog.String("error", err.Error()),
			)
		} else {
			log.Info("[api] connected to chat service", slog.String("addr", chatAddr))
			chatClient = c
		}
	} else {
		log.Warn("[api] CHAT_GRPC_ADDR not set — chat endpoints disabled")
	}

	var compassClient *compassadapter.Client
	if compassAddr := envOr("COMPASS_GRPC_ADDR", ""); compassAddr != "" {
		c, err := compassadapter.NewClient(compassAddr)
		if err != nil {
			log.Warn("[api] failed to connect to compass service, project endpoints disabled",
				slog.String("addr", compassAddr),
				slog.String("error", err.Error()),
			)
		} else {
			log.Info("[api] connected to compass service", slog.String("addr", compassAddr))
			compassClient = c
		}
	} else {
		log.Warn("[api] COMPASS_GRPC_ADDR not set — project endpoints disabled")
	}

	var sovereignClient *sovereignadapter.Client
	if sovereignAddr := envOr("SOVEREIGN_GRPC_ADDR", ""); sovereignAddr != "" {
		c, err := sovereignadapter.NewClient(sovereignAddr)
		if err != nil {
			log.Warn("[api] failed to connect to sovereign service, token auth disabled",
				slog.String("addr", sovereignAddr),
				slog.String("error", err.Error()),
			)
		} else {
			log.Info("[api] connected to sovereign service", slog.String("addr", sovereignAddr))
			sovereignClient = c
		}
	} else {
		log.Warn("[api] SOVEREIGN_GRPC_ADDR not set — token auth disabled")
	}

	var policyClient *policyadapter.Client
	if policyAddr := envOr("POLICY_GRPC_ADDR", ""); policyAddr != "" {
		c, err := policyadapter.NewClient(policyAddr)
		if err != nil {
			log.Warn("[api] failed to connect to policy service, approval endpoints disabled",
				slog.String("addr", policyAddr),
				slog.String("error", err.Error()),
			)
		} else {
			log.Info("[api] connected to policy service", slog.String("addr", policyAddr))
			policyClient = c
		}
	} else {
		log.Warn("[api] POLICY_GRPC_ADDR not set — approval endpoints disabled")
	}

	var workspaceClient *workspaceadapter.Client
	if workspaceAddr := envOr("WORKSPACE_GRPC_ADDR", ""); workspaceAddr != "" {
		c, err := workspaceadapter.NewClient(workspaceAddr)
		if err != nil {
			log.Warn("[api] failed to connect to workspace service, workspace endpoints disabled",
				slog.String("addr", workspaceAddr),
				slog.String("error", err.Error()),
			)
		} else {
			log.Info("[api] connected to workspace service", slog.String("addr", workspaceAddr))
			workspaceClient = c
		}
	} else {
		log.Warn("[api] WORKSPACE_GRPC_ADDR not set — workspace endpoints disabled")
	}

	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		log.Warn("[api] API_KEY not set — authentication disabled")
	} else {
		log.Info("[api] authentication enabled")
	}

	server := apihttp.NewServer(brain, memory, chatClient, compassClient, policyClient, sovereignClient, workspaceClient, log, apiKey)

	return &App{
		Handler: server.Handler(),
		Logger:  log,
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
