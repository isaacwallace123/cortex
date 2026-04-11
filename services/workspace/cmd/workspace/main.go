// Workspace — workspace management service.
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

	workspacev1 "github.com/isaacwallace123/cortex/services/workspace/gen/workspace/v1"
	"github.com/isaacwallace123/cortex/services/workspace/internal/bootstrap"
)

func main() {
	app := bootstrap.Wire()

	port := envOr("WORKSPACE_PORT", "6262")
	metricsPort := envOr("WORKSPACE_METRICS_PORT", "6263")

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "workspace: listen: %v\n", err)
		os.Exit(1)
	}

	srv := grpc.NewServer()
	workspacev1.RegisterWorkspaceServiceServer(srv, app.GRPCServer)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"workspace"}`))
	})
	httpSrv := &http.Server{Addr: ":" + metricsPort, Handler: mux, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "workspace: metrics server: %v\n", err)
		}
	}()

	go func() {
		<-ctx.Done()
		srv.GracefulStop()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutdownCtx)
	}()

	app.Logger.Info("[Workspace] listening", "port", port, "metrics", ":"+metricsPort+"/metrics")
	if err := srv.Serve(lis); err != nil {
		fmt.Fprintf(os.Stderr, "workspace: serve: %v\n", err)
		os.Exit(1)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
