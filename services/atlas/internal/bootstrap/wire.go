package bootstrap

import (
	"log/slog"
	"os"

	"github.com/isaacwallace123/cortex/services/atlas/internal/executor"
	cortexlog "github.com/isaacwallace123/cortex/services/atlas/internal/log"
	grpctransport "github.com/isaacwallace123/cortex/services/atlas/internal/transport/grpc"
)

type App struct {
	GRPCServer *grpctransport.Server
	Logger     *slog.Logger
}

func Wire() *App {
	log := cortexlog.Atlas()

	kubeconfig := os.Getenv("ATLAS_KUBECONFIG")
	if kubeconfig != "" {
		log.Info("[Atlas] using kubeconfig", slog.String("path", kubeconfig))
	} else {
		log.Info("[Atlas] using default kubeconfig (~/.kube/config or in-cluster)")
	}

	kubectl := executor.New(kubeconfig)

	return &App{
		GRPCServer: grpctransport.NewServer(kubectl),
		Logger:     log,
	}
}
