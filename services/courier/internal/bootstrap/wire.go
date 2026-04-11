package bootstrap

import (
	"log/slog"
	"os"

	"github.com/isaacwallace123/cortex/services/courier/internal/adapters/nerva"
	"github.com/isaacwallace123/cortex/services/courier/internal/application"
	cortexlog "github.com/isaacwallace123/cortex/services/courier/internal/log"
	grpctransport "github.com/isaacwallace123/cortex/services/courier/internal/transport/grpc"
)

type App struct {
	GRPCServer *grpctransport.Server
	Logger     *slog.Logger
}

func Wire() *App {
	log := cortexlog.Courier()

	var publisher application.EventPublisher
	nervaAddr := os.Getenv("NERVA_ADDR")
	if nervaAddr != "" {
		p, err := nerva.NewPublisher(nervaAddr)
		if err != nil {
			log.Warn("[Courier] Nerva unavailable, events disabled",
				slog.String("addr", nervaAddr),
				slog.String("error", err.Error()),
			)
			publisher = nerva.NoopPublisher{}
		} else {
			log.Info("[Courier] Nerva connected", slog.String("addr", nervaAddr))
			publisher = p
		}
	} else {
		publisher = nerva.NoopPublisher{}
	}

	registry := application.NewRegistry()
	connectUC := application.NewConnectUseCase(registry, publisher, log)
	dispatchUC := application.NewDispatchUseCase(registry)
	listAgentsUC := application.NewListAgentsUseCase(registry)

	grpcServer := grpctransport.NewServer(connectUC, dispatchUC, listAgentsUC)

	log.Info("[Courier] wired â€” awaiting agent connections")
	return &App{GRPCServer: grpcServer, Logger: log}
}

