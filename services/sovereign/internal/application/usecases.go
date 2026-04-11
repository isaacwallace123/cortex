// Package application implements Sovereign authentication use cases.
package application

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/isaacwallace123/cortex/services/sovereign/internal/adapters/store"
	"github.com/isaacwallace123/cortex/services/sovereign/internal/domain"
	"github.com/isaacwallace123/cortex/services/sovereign/internal/ports"
)

const (
	defaultTokenTTL = 24 * time.Hour
	bcryptCost      = 12
)

// UseCases bundles all Sovereign operations.
type UseCases struct {
	store    ports.IdentityStore
	tokenTTL time.Duration
}

func New(s ports.IdentityStore, tokenTTL time.Duration) *UseCases {
	if tokenTTL <= 0 {
		tokenTTL = defaultTokenTTL
	}
	return &UseCases{store: s, tokenTTL: tokenTTL}
}

// Register creates a new user account, hashes the password, and opens a
// session for the device. The role defaults to "user" when empty.
func (uc *UseCases) Register(ctx context.Context, username, email, password, role, deviceID, deviceName string) (*domain.User, *domain.Session, error) {
	if username == "" || password == "" {
		return nil, nil, fmt.Errorf("username and password are required")
	}
	// Normalize role.
	if role == "" {
		role = string(domain.RoleUser)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return nil, nil, fmt.Errorf("hash password: %w", err)
	}

	user := &domain.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(hash),
		Roles:        []string{role},
	}
	if err := uc.store.CreateUser(ctx, user); err != nil {
		return nil, nil, fmt.Errorf("create user: %w", err)
	}

	sess, err := uc.newSession(ctx, user.UserID, deviceID, deviceName)
	if err != nil {
		return user, nil, err
	}
	return user, sess, nil
}

// Login verifies credentials and opens a new session for the device.
// Returns an error with a generic message to avoid leaking whether the
// username or the password was wrong.
func (uc *UseCases) Login(ctx context.Context, username, password, deviceID, deviceName string) (*domain.User, *domain.Session, error) {
	user, err := uc.store.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, nil, fmt.Errorf("lookup user: %w", err)
	}
	if user == nil {
		return nil, nil, fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, nil, fmt.Errorf("invalid credentials")
	}

	_ = uc.store.UpdateLastActive(ctx, user.UserID)

	sess, err := uc.newSession(ctx, user.UserID, deviceID, deviceName)
	if err != nil {
		return user, nil, err
	}
	return user, sess, nil
}

// TokenIdentity is the result of validating a session token. When Valid is
// false, Reason contains a human-readable explanation of why validation failed.
type TokenIdentity struct {
	Valid    bool
	UserID   string
	Username string
	Roles    []string
	Reason   string
}

func (uc *UseCases) ValidateToken(ctx context.Context, token string) TokenIdentity {
	if token == "" {
		return TokenIdentity{Reason: "empty token"}
	}

	sess, err := uc.store.GetSessionByToken(ctx, token)
	if err != nil || sess == nil {
		return TokenIdentity{Reason: "token not found"}
	}
	if sess.IsExpired() {
		_, _ = uc.store.RevokeSession(ctx, token)
		return TokenIdentity{Reason: "token expired"}
	}

	user, err := uc.store.GetUserByID(ctx, sess.UserID)
	if err != nil || user == nil {
		return TokenIdentity{Reason: "user not found"}
	}

	_ = uc.store.UpdateLastActive(ctx, user.UserID)

	return TokenIdentity{
		Valid:    true,
		UserID:   user.UserID,
		Username: user.Username,
		Roles:    user.Roles,
	}
}

func (uc *UseCases) Logout(ctx context.Context, token string) (bool, error) {
	return uc.store.RevokeSession(ctx, token)
}

func (uc *UseCases) GetUser(ctx context.Context, userID string) (*domain.User, error) {
	u, err := uc.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return u, nil
}

func (uc *UseCases) ListUsers(ctx context.Context) ([]*domain.User, error) {
	return uc.store.ListUsers(ctx)
}

func (uc *UseCases) newSession(ctx context.Context, userID, deviceID, deviceName string) (*domain.Session, error) {
	sess := &domain.Session{
		UserID:     userID,
		DeviceID:   deviceID,
		DeviceName: deviceName,
		Token:      store.GenerateToken(),
		ExpiresAt:  time.Now().Add(uc.tokenTTL),
	}
	if err := uc.store.CreateSession(ctx, sess); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return sess, nil
}

