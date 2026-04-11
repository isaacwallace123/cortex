package ports

import (
	"context"

	"github.com/isaacwallace123/cortex/services/workspace/internal/domain"
)

type StorePort interface {
	Create(ctx context.Context, w domain.Workspace) (domain.Workspace, error)
	Get(ctx context.Context, workspaceID, userID string) (domain.Workspace, error)
	List(ctx context.Context, userID string) ([]domain.Workspace, error)
	Update(ctx context.Context, w domain.Workspace) (domain.Workspace, error)
	Delete(ctx context.Context, workspaceID, userID string) (bool, error)
}
