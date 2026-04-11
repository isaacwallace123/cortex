package application

import (
	"context"
	"fmt"

	"github.com/isaacwallace123/cortex/services/beacon/internal/domain"
	"github.com/isaacwallace123/cortex/services/beacon/internal/ports"
)

type FetchUseCase struct {
	fetcher ports.Fetcher
}

func NewFetchUseCase(fetcher ports.Fetcher) *FetchUseCase {
	return &FetchUseCase{fetcher: fetcher}
}

type FetchInput struct {
	URL            string
	TimeoutSeconds int
}

type FetchOutput struct {
	Result domain.FetchResult
}

func (uc *FetchUseCase) Execute(ctx context.Context, in FetchInput) (FetchOutput, error) {
	if in.URL == "" {
		return FetchOutput{}, fmt.Errorf("url must not be empty")
	}
	result, err := uc.fetcher.Fetch(ctx, in.URL, in.TimeoutSeconds)
	if err != nil {
		return FetchOutput{}, fmt.Errorf("fetch: %w", err)
	}
	return FetchOutput{Result: result}, nil
}
