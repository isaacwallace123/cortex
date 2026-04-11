package ports

import (
	"context"

	"github.com/isaacwallace123/cortex/services/sovereign/internal/domain"
)

// IdentityStore persists users and sessions.
type IdentityStore interface {
	// Users
	CreateUser(ctx context.Context, u *domain.User) error
	GetUserByID(ctx context.Context, userID string) (*domain.User, error)
	GetUserByUsername(ctx context.Context, username string) (*domain.User, error)
	UpdateLastActive(ctx context.Context, userID string) error
	ListUsers(ctx context.Context) ([]*domain.User, error)

	// Sessions
	CreateSession(ctx context.Context, s *domain.Session) error
	GetSessionByToken(ctx context.Context, token string) (*domain.Session, error)
	RevokeSession(ctx context.Context, token string) (bool, error)
	PruneExpired(ctx context.Context) error
}
