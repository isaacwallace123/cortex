// Package sovereign provides a gRPC client for the Sovereign identity service.
package sovereign

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	sovereignv1 "github.com/isaacwallace123/cortex/services/sovereign/gen/sovereign/v1"
)

// Client wraps the Sovereign gRPC client.
type Client struct {
	inner sovereignv1.SovereignServiceClient
}

func NewClient(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial sovereign at %s: %w", addr, err)
	}
	return &Client{inner: sovereignv1.NewSovereignServiceClient(conn)}, nil
}

type User struct {
	UserID     string   `json:"user_id"`
	Username   string   `json:"username"`
	Email      string   `json:"email"`
	Roles      []string `json:"roles"`
	CreatedAt  int64    `json:"created_at"`
	LastActive int64    `json:"last_active"`
}

type Session struct {
	SessionID  string `json:"session_id"`
	UserID     string `json:"user_id"`
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
	Token      string `json:"token"`
	ExpiresAt  int64  `json:"expires_at"`
	CreatedAt  int64  `json:"created_at"`
}

type TokenIdentity struct {
	Valid    bool
	UserID   string
	Username string
	Roles    []string
	Reason   string
}

func (c *Client) Register(ctx context.Context, username, email, password, role, deviceID, deviceName string) (User, Session, error) {
	resp, err := c.inner.Register(ctx, &sovereignv1.RegisterRequest{
		Username:   username,
		Email:      email,
		Password:   password,
		Role:       role,
		DeviceId:   deviceID,
		DeviceName: deviceName,
	})
	if err != nil {
		return User{}, Session{}, err
	}
	return protoUser(resp.GetUser()), protoSession(resp.GetSession()), nil
}

func (c *Client) Login(ctx context.Context, username, password, deviceID, deviceName string) (User, Session, error) {
	resp, err := c.inner.Login(ctx, &sovereignv1.LoginRequest{
		Username:   username,
		Password:   password,
		DeviceId:   deviceID,
		DeviceName: deviceName,
	})
	if err != nil {
		return User{}, Session{}, err
	}
	return protoUser(resp.GetUser()), protoSession(resp.GetSession()), nil
}

func (c *Client) ValidateToken(ctx context.Context, token string) TokenIdentity {
	resp, err := c.inner.ValidateToken(ctx, &sovereignv1.ValidateTokenRequest{Token: token})
	if err != nil || resp == nil {
		return TokenIdentity{Reason: "sovereign unavailable"}
	}
	return TokenIdentity{
		Valid:    resp.GetValid(),
		UserID:   resp.GetUserId(),
		Username: resp.GetUsername(),
		Roles:    resp.GetRoles(),
		Reason:   resp.GetReason(),
	}
}

func (c *Client) Logout(ctx context.Context, token string) (bool, error) {
	resp, err := c.inner.Logout(ctx, &sovereignv1.LogoutRequest{Token: token})
	if err != nil {
		return false, err
	}
	return resp.GetRevoked(), nil
}

func (c *Client) ListUsers(ctx context.Context) ([]User, error) {
	resp, err := c.inner.ListUsers(ctx, &sovereignv1.ListUsersRequest{})
	if err != nil {
		return nil, err
	}
	out := make([]User, 0, len(resp.GetUsers()))
	for _, u := range resp.GetUsers() {
		out = append(out, protoUser(u))
	}
	return out, nil
}

func protoUser(u *sovereignv1.User) User {
	if u == nil {
		return User{}
	}
	return User{
		UserID:     u.GetUserId(),
		Username:   u.GetUsername(),
		Email:      u.GetEmail(),
		Roles:      u.GetRoles(),
		CreatedAt:  u.GetCreatedAt(),
		LastActive: u.GetLastActive(),
	}
}

func protoSession(s *sovereignv1.Session) Session {
	if s == nil {
		return Session{}
	}
	return Session{
		SessionID:  s.GetSessionId(),
		UserID:     s.GetUserId(),
		DeviceID:   s.GetDeviceId(),
		DeviceName: s.GetDeviceName(),
		Token:      s.GetToken(),
		ExpiresAt:  s.GetExpiresAt(),
		CreatedAt:  s.GetCreatedAt(),
	}
}

