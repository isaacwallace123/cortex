package application

import (
	"context"

	"github.com/isaacwallace123/cortex/services/vault/internal/domain"
)

type Store interface {
	Store(ctx context.Context, entry *domain.Entry) error
	Search(ctx context.Context, userID, workspaceID, query string, limit int) ([]*domain.Entry, error)
	Delete(ctx context.Context, entryID, userID string) (bool, error)
}

type StoreUseCase struct{ store Store }

func NewStoreUseCase(s Store) *StoreUseCase { return &StoreUseCase{store: s} }

type StoreInput struct {
	UserID      string
	WorkspaceID string
	Content     string
	Source      string
	Tags        []string
}

func (uc *StoreUseCase) Execute(ctx context.Context, in StoreInput) (*domain.Entry, error) {
	e := &domain.Entry{
		UserID:      in.UserID,
		WorkspaceID: in.WorkspaceID,
		Content:     in.Content,
		Source:      in.Source,
		Tags:        in.Tags,
	}
	if err := uc.store.Store(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}
