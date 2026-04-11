package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/isaacwallace123/cortex/apps/relay/internal/agent"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})).With(slog.String("subsystem", "Relay"))

	log.Info("[Relay] booting agent")

	// Load or generate a persistent agent ID so the agent survives restarts.
	idFile := envOr("RELAY_ID_FILE", filepath.Join(os.TempDir(), "cortex-relay-id"))
	agentID := os.Getenv("RELAY_AGENT_ID")
	if agentID == "" {
		var err error
		agentID, err = agent.PersistentID(idFile)
		if err != nil {
			log.Error("[Relay] failed to generate agent ID", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}

	cfg := agent.Config{
		AgentID:     agentID,
		AgentName:   envOr("RELAY_AGENT_NAME", ""),
		DeviceID:    envOr("RELAY_DEVICE_ID", agentID), // default device_id = agent_id
		CourierAddr: envOr("COURIER_ADDR", "localhost:2020"),
	}

	log.Info("[Relay] configuration",
		slog.String("agent_id", cfg.AgentID),
		slog.String("courier", cfg.CourierAddr),
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	agent.Run(ctx, cfg, log)
	log.Info("[Relay] agent stopped")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

