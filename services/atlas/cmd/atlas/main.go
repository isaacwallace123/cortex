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
	atlasv1 "github.com/isaacwallace123/cortex/services/atlas/gen/atlas/v1"
	"github.com/isaacwallace123/cortex/services/atlas/internal/bootstrap"
	cortexlog "github.com/isaacwallace123/cortex/services/atlas/internal/log"
)

func main() {
	log := cortexlog.New("Atlas")
	log.Info("[Atlas] booting service")

	app := bootstrap.Wire()

	grpcAddr := envOr("ATLAS_ADDR", ":6161")
	metricsAddr := envOr("ATLAS_METRICS_ADDR", ":6162")

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Error("[Atlas] failed to listen",
			slog.String("addr", grpcAddr),
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}

	grpcSrv := grpc.NewServer(
		grpc.UnaryInterceptor(observe.UnaryServerInterceptor()),
	)
	atlasv1.RegisterAtlasServiceServer(grpcSrv, app.GRPCServer)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"atlas"}`))
	})
	httpSrv := &http.Server{Addr: metricsAddr, Handler: mux, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second}

	log.Info("[Atlas] service ready",
		slog.String("grpc", grpcAddr),
		slog.String("metrics", metricsAddr+"/metrics"),
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("[Atlas] metrics server error", slog.String("error", err.Error()))
		}
	}()

	go func() {
		if err := grpcSrv.Serve(lis); err != nil {
			log.Error("[Atlas] grpc server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	log.Info("[Atlas] shutting down")
	grpcSrv.GracefulStop()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutdownCtx)
	log.Info("[Atlas] stopped")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
