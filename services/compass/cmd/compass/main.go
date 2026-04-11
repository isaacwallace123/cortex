// Compass — project planning service.
package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"

	compassv1 "github.com/isaacwallace123/cortex/services/compass/gen/compass/v1"
	"github.com/isaacwallace123/cortex/services/compass/internal/bootstrap"
)

func main() {
	app := bootstrap.Wire()

	port := envOr("COMPASS_PORT", "5151")
	metricsPort := envOr("COMPASS_METRICS_PORT", "5152")

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "compass: listen: %v\n", err)
		os.Exit(1)
	}

	srv := grpc.NewServer()
	compassv1.RegisterCompassServiceServer(srv, app.GRPCServer)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"compass"}`))
	})
	httpSrv := &http.Server{Addr: ":" + metricsPort, Handler: mux, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "compass: metrics server: %v\n", err)
		}
	}()

	go func() {
		<-ctx.Done()
		srv.GracefulStop()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutdownCtx)
	}()

	app.Logger.Info("[Compass] listening", "port", port, "metrics", ":"+metricsPort+"/metrics")
	if err := srv.Serve(lis); err != nil {
		fmt.Fprintf(os.Stderr, "compass: serve: %v\n", err)
		os.Exit(1)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
