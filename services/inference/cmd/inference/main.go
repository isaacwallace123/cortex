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

	inferencev1 "github.com/isaacwallace123/cortex/services/inference/gen/inference/v1"
	"github.com/isaacwallace123/cortex/services/inference/internal/bootstrap"
	cortexlog "github.com/isaacwallace123/cortex/services/inference/internal/log"
)

func main() {
	log := cortexlog.Inference()

	app := bootstrap.Wire()

	grpcAddr := envOr("INFERENCE_ADDR", ":9090")
	metricsAddr := envOr("INFERENCE_METRICS_ADDR", ":9091")

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Error("[inference] failed to listen", slog.String("addr", grpcAddr), slog.String("error", err.Error()))
		os.Exit(1)
	}

	grpcSrv := grpc.NewServer()
	inferencev1.RegisterInferenceServiceServer(grpcSrv, app.GRPCServer)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"inference"}`))
	})
	httpSrv := &http.Server{Addr: metricsAddr, Handler: mux, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second}

	log.Info("[inference] service ready",
		slog.String("grpc", grpcAddr),
		slog.String("metrics", metricsAddr+"/metrics"),
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("[inference] metrics server error", slog.String("error", err.Error()))
		}
	}()

	go func() {
		if err := grpcSrv.Serve(lis); err != nil {
			log.Error("[inference] grpc server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	log.Info("[inference] shutting down")
	grpcSrv.GracefulStop()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutdownCtx)
	log.Info("[inference] stopped")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
