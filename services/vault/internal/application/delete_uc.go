package application

import "context"

type DeleteUseCase struct{ store Store }

func NewDeleteUseCase(s Store) *DeleteUseCase { return &DeleteUseCase{store: s} }

func (uc *DeleteUseCase) Execute(ctx context.Context, entryID, userID string) (bool, error) {
	return uc.store.Delete(ctx, entryID, userID)
}
