// Genesis — tool auto-generation service.
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

	genesisv1 "github.com/isaacwallace123/cortex/services/genesis/gen/genesis/v1"
	"github.com/isaacwallace123/cortex/services/genesis/internal/bootstrap"
)

func main() {
	app := bootstrap.Wire()

	port := envOr("GENESIS_PORT", "4343")
	metricsPort := envOr("GENESIS_METRICS_PORT", "4344")

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "genesis: listen: %v\n", err)
		os.Exit(1)
	}

	srv := grpc.NewServer()
	genesisv1.RegisterGenesisServiceServer(srv, app.GRPCServer)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"genesis"}`))
	})
	httpSrv := &http.Server{Addr: ":" + metricsPort, Handler: mux, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "genesis: metrics server: %v\n", err)
		}
	}()

	go func() {
		<-ctx.Done()
		srv.GracefulStop()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutdownCtx)
	}()

	app.Logger.Info("[Genesis] listening", "port", port, "metrics", ":"+metricsPort+"/metrics")
	if err := srv.Serve(lis); err != nil {
		fmt.Fprintf(os.Stderr, "genesis: serve: %v\n", err)
		os.Exit(1)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
