// Sovereign — identity and authentication service.
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

	sovereignv1 "github.com/isaacwallace123/cortex/services/sovereign/gen/sovereign/v1"
	"github.com/isaacwallace123/cortex/services/sovereign/internal/bootstrap"
)

func main() {
	app := bootstrap.Wire()

	addr := envOr("SOVEREIGN_ADDR", ":4141")
	metricsPort := envOr("SOVEREIGN_METRICS_PORT", "4142")

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sovereign: listen: %v\n", err)
		os.Exit(1)
	}

	srv := grpc.NewServer()
	sovereignv1.RegisterSovereignServiceServer(srv, app.GRPCServer)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"sovereign"}`))
	})
	httpSrv := &http.Server{Addr: ":" + metricsPort, Handler: mux, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "sovereign: metrics server: %v\n", err)
		}
	}()

	go func() {
		<-ctx.Done()
		srv.GracefulStop()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutdownCtx)
	}()

	app.Logger.Info("[Sovereign] listening", "addr", addr, "metrics", ":"+metricsPort+"/metrics")
	if err := srv.Serve(lis); err != nil {
		fmt.Fprintf(os.Stderr, "sovereign: serve: %v\n", err)
		os.Exit(1)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
