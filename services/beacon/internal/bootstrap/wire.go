package bootstrap

import (
	"log/slog"
	"os"

	"github.com/isaacwallace123/cortex/services/beacon/internal/adapters/fetch"
	"github.com/isaacwallace123/cortex/services/beacon/internal/adapters/search"
	"github.com/isaacwallace123/cortex/services/beacon/internal/application"
	cortexlog "github.com/isaacwallace123/cortex/services/beacon/internal/log"
	"github.com/isaacwallace123/cortex/services/beacon/internal/ports"
	grpctransport "github.com/isaacwallace123/cortex/services/beacon/internal/transport/grpc"
)

type App struct {
	GRPCServer *grpctransport.Server
	Logger     *slog.Logger
}

func Wire() *App {
	log := cortexlog.Beacon()

	var searcher ports.Searcher
	switch {
	case os.Getenv("BEACON_SEARXNG_URL") != "":
		url := os.Getenv("BEACON_SEARXNG_URL")
		searcher = search.NewSearXNG(url)
		log.Info("[Beacon] search backend: SearXNG", slog.String("url", url))
	case os.Getenv("BEACON_BRAVE_API_KEY") != "":
		searcher = search.NewBrave(os.Getenv("BEACON_BRAVE_API_KEY"))
		log.Info("[Beacon] search backend: Brave Search")
	default:
		log.Warn("[Beacon] no search backend configured â€” set BEACON_SEARXNG_URL or BEACON_BRAVE_API_KEY")
		searcher = search.Noop{}
	}

	fetcher := fetch.NewHTTPFetcher()
	log.Info("[Beacon] HTTP fetcher ready")

	searchUC := application.NewSearchUseCase(searcher)
	fetchUC := application.NewFetchUseCase(fetcher)

	grpcServer := grpctransport.NewServer(searchUC, fetchUC)

	return &App{
		GRPCServer: grpcServer,
		Logger:     log,
	}
}

