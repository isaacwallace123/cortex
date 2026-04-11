// Catalyst — skill composition service.
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

	catalystv1 "github.com/isaacwallace123/cortex/services/catalyst/gen/catalyst/v1"
	"github.com/isaacwallace123/cortex/services/catalyst/internal/bootstrap"
)

func main() {
	app := bootstrap.Wire()

	port := envOr("CATALYST_PORT", "5353")
	metricsPort := envOr("CATALYST_METRICS_PORT", "5354")

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "catalyst: listen: %v\n", err)
		os.Exit(1)
	}

	srv := grpc.NewServer()
	catalystv1.RegisterCatalystServiceServer(srv, app.GRPCServer)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"catalyst"}`))
	})
	httpSrv := &http.Server{Addr: ":" + metricsPort, Handler: mux, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "catalyst: metrics server: %v\n", err)
		}
	}()

	go func() {
		<-ctx.Done()
		srv.GracefulStop()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutdownCtx)
	}()

	app.Logger.Info("[Catalyst] listening", "port", port, "metrics", ":"+metricsPort+"/metrics")
	if err := srv.Serve(lis); err != nil {
		fmt.Fprintf(os.Stderr, "catalyst: serve: %v\n", err)
		os.Exit(1)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
