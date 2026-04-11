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

	"github.com/isaacwallace123/cortex/services/api/internal/bootstrap"
	cortexlog "github.com/isaacwallace123/cortex/services/api/internal/log"
)

func main() {
	log := cortexlog.API()

	app := bootstrap.Wire()

	addr := envOr("API_ADDR", ":8000")

	srv := &http.Server{
		Addr:         addr,
		Handler:      app.Handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 120 * time.Second, // LLM responses can take time
		IdleTimeout:  60 * time.Second,
	}

	log.Info("[api] gateway ready", slog.String("addr", addr))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("[api] server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	log.Info("[api] shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("[api] shutdown error", slog.String("error", err.Error()))
	}

	log.Info("[api] stopped")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
