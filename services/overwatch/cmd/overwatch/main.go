package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/isaacwallace123/cortex/services/overwatch/internal/bootstrap"
	"github.com/isaacwallace123/cortex/services/overwatch/internal/portfolio"
)

func main() {
	app := bootstrap.Wire()
	log := app.Logger

	metricsAddr := envOr("OVERWATCH_METRICS_ADDR", ":8484")
	portfolioAddr := envOr("PORTFOLIO_API_ADDR", ":8000")

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	metricsMux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"overwatch"}`))
	})
	metricsSrv := &http.Server{Addr: metricsAddr, Handler: metricsMux, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second}

	portfolioMux := http.NewServeMux()
	portfolio.NewHandler(app.PortfolioEngine.Store()).Register(portfolioMux)
	portfolioSrv := &http.Server{Addr: portfolioAddr, Handler: portfolioMux, ReadTimeout: 15 * time.Second, WriteTimeout: 60 * time.Second}

	log.Info("[Overwatch] metrics server starting", slog.String("addr", metricsAddr))
	log.Info("[Overwatch] portfolio API starting", slog.String("addr", portfolioAddr))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := metricsSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("[Overwatch] metrics server error", slog.String("error", err.Error()))
		}
	}()
	go func() {
		if err := portfolioSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("[Overwatch] portfolio server error", slog.String("error", err.Error()))
		}
	}()

	go app.Analyzer.Run(ctx)
	go app.PortfolioEngine.Run(ctx)

	<-ctx.Done()
	log.Info("[Overwatch] shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = metricsSrv.Shutdown(shutdownCtx)
	_ = portfolioSrv.Shutdown(shutdownCtx)
	log.Info("[Overwatch] stopped")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
