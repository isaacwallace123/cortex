package bootstrap

import (
	"log/slog"
	"os"
	"time"

	arsenaladapter "github.com/isaacwallace123/cortex/services/brain/internal/adapters/arsenal"
	beaconadapter "github.com/isaacwallace123/cortex/services/brain/internal/adapters/beacon"
	compassadapter "github.com/isaacwallace123/cortex/services/brain/internal/adapters/compass"
	atlasadapter "github.com/isaacwallace123/cortex/services/brain/internal/adapters/atlas"
	courieradapter "github.com/isaacwallace123/cortex/services/brain/internal/adapters/courier"
	crucibleadapter "github.com/isaacwallace123/cortex/services/brain/internal/adapters/crucible"
	forgeadapter "github.com/isaacwallace123/cortex/services/brain/internal/adapters/forge"
	vaultadapter "github.com/isaacwallace123/cortex/services/brain/internal/adapters/vault"
	inferenceadapter "github.com/isaacwallace123/cortex/services/brain/internal/adapters/inference"
	memoryadapter "github.com/isaacwallace123/cortex/services/brain/internal/adapters/memory"
	nervaadapter "github.com/isaacwallace123/cortex/services/brain/internal/adapters/nerva"
	policyadapter "github.com/isaacwallace123/cortex/services/brain/internal/adapters/policy"
	shelladapter "github.com/isaacwallace123/cortex/services/brain/internal/adapters/shell"
	"github.com/isaacwallace123/cortex/services/brain/internal/adapters/telemetry"
	"github.com/isaacwallace123/cortex/services/brain/internal/application"
	cortexlog "github.com/isaacwallace123/cortex/services/brain/internal/log"
	"github.com/isaacwallace123/cortex/services/brain/internal/ports"
	"github.com/isaacwallace123/cortex/services/brain/internal/recall"
	"github.com/isaacwallace123/cortex/services/brain/internal/sentinel"
	braingrpc "github.com/isaacwallace123/cortex/services/brain/internal/transport/grpc"
)

type App struct {
	GRPCServer *braingrpc.Server
	Logger     *slog.Logger
}

func Wire() *App {
	log := cortexlog.Brain()

	var inferencePort ports.InferencePort
	if addr := os.Getenv("INFERENCE_ADDR"); addr != "" {
		adapter, err := inferenceadapter.NewGRPCAdapter(addr)
		if err != nil {
			log.Error("[brain] failed to connect to inference, falling back to stub",
				slog.String("addr", addr), slog.String("error", err.Error()))
			inferencePort = inferenceadapter.NewStubAdapter()
		} else {
			log.Info("[brain] inference adapter ready", slog.String("addr", addr))
			inferencePort = adapter
		}
	} else {
		log.Warn("[brain] INFERENCE_ADDR not set â€” using stub inference adapter")
		inferencePort = inferenceadapter.NewStubAdapter()
	}

	var memoryPort ports.MemoryPort
	if addr := os.Getenv("MEMORY_ADDR"); addr != "" {
		adapter, err := memoryadapter.NewGRPCAdapter(addr)
		if err != nil {
			log.Error("[brain] failed to connect to Echo (memory), falling back to noop",
				slog.String("addr", addr), slog.String("error", err.Error()))
			memoryPort = memoryadapter.NewNoopAdapter()
		} else {
			log.Info("[brain] Echo (memory) adapter ready", slog.String("addr", addr))
			memoryPort = adapter
		}
	} else {
		log.Warn("[brain] MEMORY_ADDR not set â€” session events will not be persisted")
		memoryPort = memoryadapter.NewNoopAdapter()
	}

	var policyPort ports.PolicyPort
	if addr := os.Getenv("POLICY_ADDR"); addr != "" {
		adapter, err := policyadapter.NewGRPCAdapter(addr)
		if err != nil {
			log.Error("[brain] failed to connect to Aegis (policy), falling back to noop",
				slog.String("addr", addr), slog.String("error", err.Error()))
			policyPort = policyadapter.NewNoopAdapter()
		} else {
			log.Info("[brain] Aegis (policy) adapter ready", slog.String("addr", addr))
			policyPort = adapter
		}
	} else {
		log.Warn("[brain] POLICY_ADDR not set â€” all steps will be allowed by default")
		policyPort = policyadapter.NewNoopAdapter()
	}

	var shellPort ports.ExecutorPort
	if addr := os.Getenv("SHELL_ADDR"); addr != "" {
		adapter, err := shelladapter.NewGRPCAdapter(addr)
		if err != nil {
			log.Error("[brain] failed to connect to Shell executor, falling back to noop",
				slog.String("addr", addr), slog.String("error", err.Error()))
			shellPort = shelladapter.NewNoopAdapter()
		} else {
			log.Info("[brain] Shell executor adapter ready", slog.String("addr", addr))
			shellPort = adapter
		}
	} else {
		log.Warn("[brain] SHELL_ADDR not set â€” shell steps will be skipped")
		shellPort = shelladapter.NewNoopAdapter()
	}

	var forgePort ports.ForgePort
	if addr := os.Getenv("FORGE_ADDR"); addr != "" {
		adapter, err := forgeadapter.NewGRPCAdapter(addr)
		if err != nil {
			log.Error("[brain] failed to connect to Forge (filesystem), falling back to noop",
				slog.String("addr", addr), slog.String("error", err.Error()))
			forgePort = forgeadapter.NewNoopAdapter()
		} else {
			log.Info("[brain] Forge (filesystem) adapter ready", slog.String("addr", addr))
			forgePort = adapter
		}
	} else {
		log.Warn("[brain] FORGE_ADDR not set â€” filesystem steps will be skipped")
		forgePort = forgeadapter.NewNoopAdapter()
	}

	var telemetryPort ports.TelemetryPort
	if addr := os.Getenv("NERVA_ADDR"); addr != "" {
		adapter, err := nervaadapter.NewGRPCAdapter(addr)
		if err != nil {
			log.Error("[brain] failed to connect to Nerva, falling back to log adapter",
				slog.String("addr", addr), slog.String("error", err.Error()))
			telemetryPort = telemetry.NewLogAdapter(cortexlog.Nerva())
		} else {
			log.Info("[brain] Nerva (event bus) adapter ready", slog.String("addr", addr))
			telemetryPort = adapter
		}
	} else {
		log.Warn("[brain] NERVA_ADDR not set â€” events will be logged locally")
		telemetryPort = telemetry.NewLogAdapter(cortexlog.Nerva())
	}

	var courierPort ports.CourierPort
	if addr := os.Getenv("COURIER_ADDR"); addr != "" {
		adapter, err := courieradapter.NewGRPCAdapter(addr)
		if err != nil {
			log.Error("[brain] failed to connect to Courier, falling back to noop",
				slog.String("addr", addr), slog.String("error", err.Error()))
			courierPort = courieradapter.NewNoopAdapter()
		} else {
			log.Info("[brain] Courier adapter ready", slog.String("addr", addr))
			courierPort = adapter
		}
	} else {
		log.Warn("[brain] COURIER_ADDR not set â€” courier steps will be skipped")
		courierPort = courieradapter.NewNoopAdapter()
	}

	var vaultPort ports.VaultPort
	if addr := os.Getenv("VAULT_ADDR"); addr != "" {
		adapter, err := vaultadapter.NewGRPCAdapter(addr)
		if err != nil {
			log.Error("[brain] failed to connect to Vault, falling back to noop",
				slog.String("addr", addr), slog.String("error", err.Error()))
			vaultPort = vaultadapter.NewNoopAdapter()
		} else {
			log.Info("[brain] Vault adapter ready", slog.String("addr", addr))
			vaultPort = adapter
		}
	} else {
		log.Warn("[brain] VAULT_ADDR not set â€” long-term knowledge will not be persisted")
		vaultPort = vaultadapter.NewNoopAdapter()
	}

	parseInputUC := application.NewParseInputUseCase(inferencePort, telemetryPort, memoryPort) // Prism
	var cruciblePort ports.CruciblePort
	if addr := os.Getenv("CRUCIBLE_ADDR"); addr != "" {
		adapter, err := crucibleadapter.NewGRPCAdapter(addr)
		if err != nil {
			log.Error("[brain] failed to connect to Crucible, falling back to noop",
				slog.String("addr", addr), slog.String("error", err.Error()))
			cruciblePort = crucibleadapter.NewNoopAdapter()
		} else {
			log.Info("[brain] Crucible adapter ready", slog.String("addr", addr))
			cruciblePort = adapter
		}
	} else {
		log.Warn("[brain] CRUCIBLE_ADDR not set â€” crucible steps will fail")
		cruciblePort = crucibleadapter.NewNoopAdapter()
	}

	sentinelLevel := sentinel.Level(os.Getenv("SENTINEL_LEVEL"))
	if sentinelLevel == "" {
		sentinelLevel = sentinel.LevelStandard
	}
	safetyFilter := sentinel.New(sentinelLevel)
	if rulesFile := os.Getenv("SENTINEL_RULES_FILE"); rulesFile != "" {
		if err := safetyFilter.LoadRulesFile(rulesFile); err != nil {
			log.Error("[Sentinel] failed to load custom rules file, using built-in rules only",
				slog.String("path", rulesFile),
				slog.String("error", err.Error()),
			)
		} else {
			log.Info("[Sentinel] custom rules loaded", slog.String("path", rulesFile))
			// Hot-reload: check every 30 s and reload if the file changed.
			go func() {
				t := time.NewTicker(30 * time.Second)
				defer t.Stop()
				for range t.C {
					if safetyFilter.ReloadIfChanged() {
						log.Info("[Sentinel] rules reloaded from file", slog.String("path", rulesFile))
					}
				}
			}()
		}
	}
	log.Info("[brain] Sentinel active", slog.String("level", string(sentinelLevel)))

	var arsenalPort ports.ArsenalPort
	if addr := os.Getenv("ARSENAL_ADDR"); addr != "" {
		adapter, err := arsenaladapter.NewGRPCAdapter(addr)
		if err != nil {
			log.Error("[brain] failed to connect to Arsenal, falling back to noop",
				slog.String("addr", addr), slog.String("error", err.Error()))
			arsenalPort = arsenaladapter.NoopAdapter{}
		} else {
			log.Info("[brain] Arsenal adapter ready", slog.String("addr", addr))
			arsenalPort = adapter
		}
	} else {
		log.Warn("[brain] ARSENAL_ADDR not set â€” tool registry unavailable, using built-in rules")
		arsenalPort = arsenaladapter.NoopAdapter{}
	}

	var compassPort ports.CompassPort
	if addr := os.Getenv("COMPASS_ADDR"); addr != "" {
		adapter, err := compassadapter.NewGRPCAdapter(addr)
		if err != nil {
			log.Error("[brain] failed to connect to Compass, falling back to noop",
				slog.String("addr", addr), slog.String("error", err.Error()))
			compassPort = compassadapter.NoopAdapter{}
		} else {
			log.Info("[brain] Compass adapter ready", slog.String("addr", addr))
			compassPort = adapter
		}
	} else {
		log.Warn("[brain] COMPASS_ADDR not set â€” project planning will not persist")
		compassPort = compassadapter.NoopAdapter{}
	}

	recallAssembler := recall.NewAssembler(memoryPort, vaultPort)

	createPlanUC := application.NewCreatePlanUseCase(inferencePort, telemetryPort, memoryPort, policyPort, courierPort, recallAssembler, arsenalPort, compassPort)
	var atlasPort ports.AtlasPort
	if addr := os.Getenv("ATLAS_ADDR"); addr != "" {
		adapter, err := atlasadapter.NewGRPCAdapter(addr)
		if err != nil {
			log.Error("[brain] failed to connect to Atlas, falling back to noop",
				slog.String("addr", addr), slog.String("error", err.Error()))
			atlasPort = atlasadapter.NewNoopAdapter()
		} else {
			log.Info("[brain] Atlas adapter ready", slog.String("addr", addr))
			atlasPort = adapter
		}
	} else {
		log.Warn("[brain] ATLAS_ADDR not set â€” atlas steps will fail")
		atlasPort = atlasadapter.NewNoopAdapter()
	}

	var beaconPort ports.BeaconPort
	if addr := os.Getenv("BEACON_ADDR"); addr != "" {
		adapter, err := beaconadapter.NewGRPCAdapter(addr)
		if err != nil {
			log.Error("[brain] failed to connect to Beacon, falling back to noop",
				slog.String("addr", addr), slog.String("error", err.Error()))
			beaconPort = beaconadapter.NewNoopAdapter()
		} else {
			log.Info("[brain] Beacon adapter ready", slog.String("addr", addr))
			beaconPort = adapter
		}
	} else {
		log.Warn("[brain] BEACON_ADDR not set â€” web search and fetch will be unavailable")
		beaconPort = beaconadapter.NewNoopAdapter()
	}

	executePlanUC := application.NewExecutePlanUseCase(shellPort, forgePort, courierPort, cruciblePort, atlasPort, beaconPort, inferencePort, memoryPort, vaultPort, telemetryPort, safetyFilter) // Vector
	getSessionUC := application.NewGetSessionUseCase(memoryPort)
	streamChatUC := application.NewStreamChatUseCase(parseInputUC, createPlanUC, executePlanUC, inferencePort)

	grpcServer := braingrpc.NewServer(parseInputUC, createPlanUC, executePlanUC, getSessionUC, streamChatUC)

	return &App{
		GRPCServer: grpcServer,
		Logger:     log,
	}
}

