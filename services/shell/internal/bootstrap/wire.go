package bootstrap

import (
	"log/slog"

	"github.com/isaacwallace123/cortex/services/shell/internal/adapters/subprocess"
	"github.com/isaacwallace123/cortex/services/shell/internal/application"
	cortexlog "github.com/isaacwallace123/cortex/services/shell/internal/log"
	grpctransport "github.com/isaacwallace123/cortex/services/shell/internal/transport/grpc"
)

type App struct {
	GRPCServer *grpctransport.Server
	Logger     *slog.Logger
}

func Wire() *App {
	log := cortexlog.Shell()

	runner := subprocess.NewRunner()
	executeStepUC := application.NewExecuteStepUseCase(runner)
	grpcServer := grpctransport.NewServer(executeStepUC)

	log.Info("[Shell] executor wired", slog.String("runner", "subprocess"))

	return &App{GRPCServer: grpcServer, Logger: log}
}
