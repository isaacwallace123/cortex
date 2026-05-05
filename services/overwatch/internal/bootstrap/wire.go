package bootstrap

import (
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/isaacwallace123/cortex/services/overwatch/internal/application"
	overwatchlog "github.com/isaacwallace123/cortex/services/overwatch/internal/log"
	"github.com/isaacwallace123/cortex/services/overwatch/internal/portfolio"
)

type App struct {
	Analyzer        *application.Analyzer
	PortfolioEngine *portfolio.Engine
	Logger          *slog.Logger
}

func Wire() *App {
	log := overwatchlog.Overwatch()

	prometheusURL := envOr("PROMETHEUS_URL", "http://prometheus:9090")
	nervaAddr := os.Getenv("NERVA_ADDR")

	intervalSec := 60
	if v := os.Getenv("OVERWATCH_INTERVAL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			intervalSec = n
		}
	}
	interval := time.Duration(intervalSec) * time.Second

	analyzer, err := application.NewAnalyzer(prometheusURL, nervaAddr, interval, log)
	if err != nil {
		log.Error("[Overwatch] failed to initialise analyzer", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if nervaAddr != "" {
		log.Info("[Overwatch] Nerva event bus connected", slog.String("addr", nervaAddr))
	} else {
		log.Warn("[Overwatch] NERVA_ADDR not set — alerts will be logged only")
	}
	log.Info("[Overwatch] ready",
		slog.String("prometheus_url", prometheusURL),
		slog.Int("interval_sec", intervalSec),
	)

	ollamaURL := envOr("OLLAMA_URL", "http://ollama:11434")
	ollamaModel := envOr("OLLAMA_MODEL", "llama3.2")

	portfolioIntervalSec := 300
	if v := os.Getenv("PORTFOLIO_INTERVAL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			portfolioIntervalSec = n
		}
	}
	portfolioInterval := time.Duration(portfolioIntervalSec) * time.Second

	portfolioEngine := portfolio.NewEngine(prometheusURL, ollamaURL, ollamaModel, portfolioInterval, log)

	return &App{Analyzer: analyzer, PortfolioEngine: portfolioEngine, Logger: log}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
