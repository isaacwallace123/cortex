package vault

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/isaacwallace123/cortex/pkg/observe"
	"github.com/isaacwallace123/cortex/services/brain/internal/ports"
	vaultv1 "github.com/isaacwallace123/cortex/services/vault/gen/vault/v1"
)

// GRPCAdapter implements VaultPort via the Vault service.
type GRPCAdapter struct {
	client vaultv1.VaultServiceClient
}

func NewGRPCAdapter(addr string) (*GRPCAdapter, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(observe.UnaryClientInterceptor()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial vault at %s: %w", addr, err)
	}
	return &GRPCAdapter{client: vaultv1.NewVaultServiceClient(conn)}, nil
}

func (a *GRPCAdapter) Store(ctx context.Context, userID, workspaceID, content, source string) error {
	_, err := a.client.Store(ctx, &vaultv1.StoreRequest{
		UserId:      userID,
		WorkspaceId: workspaceID,
		Content:     content,
		Source:      source,
	})
	return err
}

func (a *GRPCAdapter) Search(ctx context.Context, userID, workspaceID, query string, limit int) ([]ports.KnowledgeEntry, error) {
	resp, err := a.client.Search(ctx, &vaultv1.SearchRequest{
		UserId:      userID,
		WorkspaceId: workspaceID,
		Query:       query,
		Limit:       int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("vault.Search: %w", err)
	}
	entries := make([]ports.KnowledgeEntry, 0, len(resp.GetEntries()))
	for _, e := range resp.GetEntries() {
		entries = append(entries, ports.KnowledgeEntry{
			EntryID:   e.GetEntryId(),
			Content:   e.GetContent(),
			Source:    e.GetSource(),
			CreatedAt: e.GetCreatedAt(),
		})
	}
	return entries, nil
}
