package workspace

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	workspacev1 "github.com/isaacwallace123/cortex/services/workspace/gen/workspace/v1"
)

// Client wraps the Workspace gRPC client.
type Client struct {
	inner workspacev1.WorkspaceServiceClient
}

func NewClient(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial workspace at %s: %w", addr, err)
	}
	return &Client{inner: workspacev1.NewWorkspaceServiceClient(conn)}, nil
}

type Workspace struct {
	WorkspaceID string `json:"workspace_id"`
	UserID      string `json:"user_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	RootPath    string `json:"root_path"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

func (c *Client) CreateWorkspace(ctx context.Context, userID, name, description, rootPath string) (Workspace, error) {
	resp, err := c.inner.CreateWorkspace(ctx, &workspacev1.CreateWorkspaceRequest{
		UserId:      userID,
		Name:        name,
		Description: description,
		RootPath:    rootPath,
	})
	if err != nil {
		return Workspace{}, err
	}
	return protoToWorkspace(resp.GetWorkspace()), nil
}

func (c *Client) GetWorkspace(ctx context.Context, workspaceID, userID string) (Workspace, error) {
	resp, err := c.inner.GetWorkspace(ctx, &workspacev1.GetWorkspaceRequest{
		WorkspaceId: workspaceID,
		UserId:      userID,
	})
	if err != nil {
		return Workspace{}, err
	}
	return protoToWorkspace(resp.GetWorkspace()), nil
}

func (c *Client) ListWorkspaces(ctx context.Context, userID string) ([]Workspace, error) {
	resp, err := c.inner.ListWorkspaces(ctx, &workspacev1.ListWorkspacesRequest{UserId: userID})
	if err != nil {
		return nil, err
	}
	out := make([]Workspace, 0, len(resp.GetWorkspaces()))
	for _, w := range resp.GetWorkspaces() {
		out = append(out, protoToWorkspace(w))
	}
	return out, nil
}

func (c *Client) UpdateWorkspace(ctx context.Context, workspaceID, userID, name, description, rootPath string) (Workspace, error) {
	resp, err := c.inner.UpdateWorkspace(ctx, &workspacev1.UpdateWorkspaceRequest{
		WorkspaceId: workspaceID,
		UserId:      userID,
		Name:        name,
		Description: description,
		RootPath:    rootPath,
	})
	if err != nil {
		return Workspace{}, err
	}
	return protoToWorkspace(resp.GetWorkspace()), nil
}

func (c *Client) DeleteWorkspace(ctx context.Context, workspaceID, userID string) (bool, error) {
	resp, err := c.inner.DeleteWorkspace(ctx, &workspacev1.DeleteWorkspaceRequest{
		WorkspaceId: workspaceID,
		UserId:      userID,
	})
	if err != nil {
		return false, err
	}
	return resp.GetDeleted(), nil
}

func protoToWorkspace(w *workspacev1.Workspace) Workspace {
	if w == nil {
		return Workspace{}
	}
	return Workspace{
		WorkspaceID: w.GetWorkspaceId(),
		UserID:      w.GetUserId(),
		Name:        w.GetName(),
		Description: w.GetDescription(),
		RootPath:    w.GetRootPath(),
		CreatedAt:   w.GetCreatedAt(),
		UpdatedAt:   w.GetUpdatedAt(),
	}
}

