package bootstrap

import (
	"log/slog"

	cortexlog "github.com/isaacwallace123/cortex/services/crucible/internal/log"
	"github.com/isaacwallace123/cortex/services/crucible/internal/sandbox"
	grpctransport "github.com/isaacwallace123/cortex/services/crucible/internal/transport/grpc"
)

type App struct {
	GRPCServer *grpctransport.Server
	Executor   *sandbox.Executor
	Logger     *slog.Logger
}

func Wire() *App {
	log := cortexlog.Crucible()
	log.Info("[Crucible] initialising nerdctl executor")

	exec := sandbox.NewExecutor()

	return &App{
		GRPCServer: grpctransport.NewServer(exec),
		Executor:   exec,
		Logger:     log,
	}
}
