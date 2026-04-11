package bootstrap

import (
	"log/slog"
	"os"

	"github.com/isaacwallace123/cortex/services/policy/internal/adapters/rules"
	"github.com/isaacwallace123/cortex/services/policy/internal/adapters/store"
	telemetryadapter "github.com/isaacwallace123/cortex/services/policy/internal/adapters/telemetry"
	"github.com/isaacwallace123/cortex/services/policy/internal/application"
	cortexlog "github.com/isaacwallace123/cortex/services/policy/internal/log"
	grpctransport "github.com/isaacwallace123/cortex/services/policy/internal/transport/grpc"
)

// App holds the composed policy service ready to serve.
type App struct {
	GRPCServer *grpctransport.Server
	Logger     *slog.Logger
}

func Wire() *App {
	log := cortexlog.Aegis()

	var engine *rules.Engine
	var err error
	if path := os.Getenv("POLICY_RULES_FILE"); path != "" {
		engine, err = rules.NewEngineFromFile(path)
		if err != nil {
			log.Error("[Aegis] failed to load rules file, falling back to defaults",
				slog.String("path", path),
				slog.String("error", err.Error()),
			)
			engine = mustEngine(rules.NewEngine(rules.DefaultRules()))
		} else {
			log.Info("[Aegis] rules loaded from file", slog.String("path", path))
		}
	} else {
		engine = mustEngine(rules.NewEngine(rules.DefaultRules()))
		log.Info("[Aegis] using default ruleset", slog.Int("rules", len(rules.DefaultRules())))
	}

	dbPath := os.Getenv("POLICY_DB")
	if dbPath == "" {
		dbPath = "/data/policy.db"
	}
	approvalStore, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		log.Error("[Aegis] failed to open approval database",
			slog.String("path", dbPath),
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
	log.Info("[Aegis] approval store ready", slog.String("db", dbPath))

	telemetryPort := telemetryadapter.NewLogAdapter(log)
	evaluatePlanUC := application.NewEvaluatePlanUseCase(engine, approvalStore, telemetryPort)
	submitApprovalUC := application.NewSubmitApprovalUseCase(approvalStore)
	listApprovalsUC := application.NewListApprovalsUseCase(approvalStore)

	grpcServer := grpctransport.NewServer(evaluatePlanUC, submitApprovalUC, listApprovalsUC)

	return &App{
		GRPCServer: grpcServer,
		Logger:     log,
	}
}

func mustEngine(e *rules.Engine, err error) *rules.Engine {
	if err != nil {
		panic("failed to compile Aegis rules: " + err.Error())
	}
	return e
}

