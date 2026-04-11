package application

import (
	"context"

	"github.com/isaacwallace123/cortex/services/vault/internal/domain"
)

type SearchUseCase struct{ store Store }

func NewSearchUseCase(s Store) *SearchUseCase { return &SearchUseCase{store: s} }

type SearchInput struct {
	UserID      string
	WorkspaceID string
	Query       string
	Limit       int
}

func (uc *SearchUseCase) Execute(ctx context.Context, in SearchInput) ([]*domain.Entry, error) {
	return uc.store.Search(ctx, in.UserID, in.WorkspaceID, in.Query, in.Limit)
}
