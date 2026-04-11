package application

import (
	"context"
	"fmt"

	"github.com/isaacwallace123/cortex/services/beacon/internal/domain"
	"github.com/isaacwallace123/cortex/services/beacon/internal/ports"
)

type SearchUseCase struct {
	searcher ports.Searcher
}

func NewSearchUseCase(searcher ports.Searcher) *SearchUseCase {
	return &SearchUseCase{searcher: searcher}
}

type SearchInput struct {
	Query      string
	MaxResults int
}

type SearchOutput struct {
	Results []domain.SearchResult
	Engine  string
}

func (uc *SearchUseCase) Execute(ctx context.Context, in SearchInput) (SearchOutput, error) {
	if in.Query == "" {
		return SearchOutput{}, fmt.Errorf("query must not be empty")
	}
	if in.MaxResults <= 0 {
		in.MaxResults = 5
	}
	results, engine, err := uc.searcher.Search(ctx, in.Query, in.MaxResults)
	if err != nil {
		return SearchOutput{}, fmt.Errorf("search: %w", err)
	}
	return SearchOutput{Results: results, Engine: engine}, nil
}
