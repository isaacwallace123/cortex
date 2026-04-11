package grpc

import (
	"context"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sovereignv1 "github.com/isaacwallace123/cortex/services/sovereign/gen/sovereign/v1"
	"github.com/isaacwallace123/cortex/services/sovereign/internal/application"
	"github.com/isaacwallace123/cortex/services/sovereign/internal/domain"
	cortexlog "github.com/isaacwallace123/cortex/services/sovereign/internal/log"
)

// Server implements SovereignServiceServer.
type Server struct {
	sovereignv1.UnimplementedSovereignServiceServer
	uc  *application.UseCases
	log *slog.Logger
}

func NewServer(uc *application.UseCases) *Server {
	return &Server{uc: uc, log: cortexlog.Sovereign()}
}

func (s *Server) Register(ctx context.Context, req *sovereignv1.RegisterRequest) (*sovereignv1.RegisterResponse, error) {
	if req.GetUsername() == "" || req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "username and password are required")
	}
	user, sess, err := s.uc.Register(ctx,
		req.GetUsername(), req.GetEmail(), req.GetPassword(), req.GetRole(),
		req.GetDeviceId(), req.GetDeviceName(),
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "register: %v", err)
	}
	s.log.Info("[Sovereign] user registered", slog.String("user_id", user.UserID), slog.String("username", user.Username))
	return &sovereignv1.RegisterResponse{
		User:    domainUserToProto(user),
		Session: domainSessionToProto(sess),
	}, nil
}

func (s *Server) Login(ctx context.Context, req *sovereignv1.LoginRequest) (*sovereignv1.LoginResponse, error) {
	if req.GetUsername() == "" || req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "username and password are required")
	}
	user, sess, err := s.uc.Login(ctx,
		req.GetUsername(), req.GetPassword(),
		req.GetDeviceId(), req.GetDeviceName(),
	)
	if err != nil {
		// Don't leak whether username exists.
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}
	s.log.Info("[Sovereign] user logged in", slog.String("user_id", user.UserID))
	return &sovereignv1.LoginResponse{
		User:    domainUserToProto(user),
		Session: domainSessionToProto(sess),
	}, nil
}

func (s *Server) ValidateToken(ctx context.Context, req *sovereignv1.ValidateTokenRequest) (*sovereignv1.ValidateTokenResponse, error) {
	id := s.uc.ValidateToken(ctx, req.GetToken())
	return &sovereignv1.ValidateTokenResponse{
		Valid:    id.Valid,
		UserId:   id.UserID,
		Username: id.Username,
		Roles:    id.Roles,
		Reason:   id.Reason,
	}, nil
}

func (s *Server) Logout(ctx context.Context, req *sovereignv1.LogoutRequest) (*sovereignv1.LogoutResponse, error) {
	revoked, err := s.uc.Logout(ctx, req.GetToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "logout: %v", err)
	}
	return &sovereignv1.LogoutResponse{Revoked: revoked}, nil
}

func (s *Server) GetUser(ctx context.Context, req *sovereignv1.GetUserRequest) (*sovereignv1.GetUserResponse, error) {
	u, err := s.uc.GetUser(ctx, req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get_user: %v", err)
	}
	if u == nil {
		return nil, status.Errorf(codes.NotFound, "user %q not found", req.GetUserId())
	}
	return &sovereignv1.GetUserResponse{User: domainUserToProto(u)}, nil
}

func (s *Server) ListUsers(ctx context.Context, _ *sovereignv1.ListUsersRequest) (*sovereignv1.ListUsersResponse, error) {
	users, err := s.uc.ListUsers(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list_users: %v", err)
	}
	resp := &sovereignv1.ListUsersResponse{}
	for _, u := range users {
		resp.Users = append(resp.Users, domainUserToProto(u))
	}
	return resp, nil
}

func domainUserToProto(u *domain.User) *sovereignv1.User {
	if u == nil {
		return nil
	}
	return &sovereignv1.User{
		UserId:     u.UserID,
		Username:   u.Username,
		Email:      u.Email,
		Roles:      u.Roles,
		CreatedAt:  u.CreatedAt.UnixMilli(),
		LastActive: u.LastActive.UnixMilli(),
	}
}

func domainSessionToProto(s *domain.Session) *sovereignv1.Session {
	if s == nil {
		return nil
	}
	return &sovereignv1.Session{
		SessionId:  s.SessionID,
		UserId:     s.UserID,
		DeviceId:   s.DeviceID,
		DeviceName: s.DeviceName,
		Token:      s.Token,
		ExpiresAt:  s.ExpiresAt.UnixMilli(),
		CreatedAt:  s.CreatedAt.UnixMilli(),
	}
}

