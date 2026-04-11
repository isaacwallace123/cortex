package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"

	"github.com/isaacwallace123/cortex/pkg/observe"
	brainv1 "github.com/isaacwallace123/cortex/services/brain/gen/brain/v1"
	"github.com/isaacwallace123/cortex/services/brain/internal/bootstrap"
	cortexlog "github.com/isaacwallace123/cortex/services/brain/internal/log"
	_ "github.com/isaacwallace123/cortex/services/brain/internal/metrics" // register prometheus metrics
)

func main() {
	log := cortexlog.Brain()

	log.Info("[brain] booting subsystems",
		slog.String("Prism", "input understanding"),
		slog.String("Axiom", "planning / reasoning"),
		slog.String("Vector", "routing / orchestration"),
		slog.String("Nerva", "telemetry / metrics"),
		slog.String("Pulse", "observability / tracing"),
	)

	app := bootstrap.Wire()

	httpAddr := envOr("BRAIN_HTTP_ADDR", ":8080")
	grpcAddr := envOr("BRAIN_GRPC_ADDR", ":8090")

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"brain"}`))
	})
	httpSrv := &http.Server{
		Addr:         httpAddr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Error("[brain] failed to listen on gRPC addr",
			slog.String("addr", grpcAddr),
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
	grpcSrv := grpc.NewServer(
		grpc.UnaryInterceptor(observe.UnaryServerInterceptor()),
	)
	brainv1.RegisterBrainServiceServer(grpcSrv, app.GRPCServer)

	log.Info("[brain] service ready",
		slog.String("http", httpAddr),
		slog.String("grpc", grpcAddr),
		slog.String("metrics", httpAddr+"/metrics"),
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("[brain] http server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	go func() {
		if err := grpcSrv.Serve(lis); err != nil {
			log.Error("[brain] grpc server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	log.Info("[brain] shutting down")

	grpcSrv.GracefulStop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		log.Error("[brain] http shutdown error", slog.String("error", err.Error()))
	}

	log.Info("[brain] stopped")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

