package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/isaacwallace123/cortex/services/workspace/internal/domain"
	"github.com/isaacwallace123/cortex/services/workspace/internal/ports"
)

type UseCases struct {
	store ports.StorePort
}

func New(store ports.StorePort) *UseCases {
	return &UseCases{store: store}
}

func (uc *UseCases) CreateWorkspace(ctx context.Context, userID, name, description, rootPath string) (domain.Workspace, error) {
	if userID == "" {
		return domain.Workspace{}, fmt.Errorf("user_id required")
	}
	if name == "" {
		return domain.Workspace{}, fmt.Errorf("name required")
	}
	now := time.Now().UTC()
	w := domain.Workspace{
		WorkspaceID: uuid.New().String(),
		UserID:      userID,
		Name:        name,
		Description: description,
		RootPath:    rootPath,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	return uc.store.Create(ctx, w)
}

func (uc *UseCases) GetWorkspace(ctx context.Context, workspaceID, userID string) (domain.Workspace, error) {
	if workspaceID == "" {
		return domain.Workspace{}, fmt.Errorf("workspace_id required")
	}
	return uc.store.Get(ctx, workspaceID, userID)
}

func (uc *UseCases) ListWorkspaces(ctx context.Context, userID string) ([]domain.Workspace, error) {
	if userID == "" {
		return nil, fmt.Errorf("user_id required")
	}
	return uc.store.List(ctx, userID)
}

func (uc *UseCases) UpdateWorkspace(ctx context.Context, workspaceID, userID, name, description, rootPath string) (domain.Workspace, error) {
	existing, err := uc.store.Get(ctx, workspaceID, userID)
	if err != nil {
		return domain.Workspace{}, err
	}
	if name != "" {
		existing.Name = name
	}
	if description != "" {
		existing.Description = description
	}
	if rootPath != "" {
		existing.RootPath = rootPath
	}
	existing.UpdatedAt = time.Now().UTC()
	return uc.store.Update(ctx, existing)
}

func (uc *UseCases) DeleteWorkspace(ctx context.Context, workspaceID, userID string) (bool, error) {
	return uc.store.Delete(ctx, workspaceID, userID)
}
