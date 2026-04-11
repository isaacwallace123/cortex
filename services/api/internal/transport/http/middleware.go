package http

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	sovereignadapter "github.com/isaacwallace123/cortex/services/api/internal/adapters/sovereign"
)

type contextKey string

const (
	requestIDKey    contextKey = "request_id"
	userIdentityKey contextKey = "user_identity"
)

// withRequestID injects a unique X-Request-ID into every request context and response header.
func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.New().String()
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func requestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// userIDFromContext returns the authenticated user_id, or "" when auth is disabled.
func userIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(userIdentityKey).(sovereignadapter.TokenIdentity); ok && id.Valid {
		return id.UserID
	}
	return ""
}

// rolesFromContext returns the authenticated user's roles, or nil when auth is disabled.
func rolesFromContext(ctx context.Context) []string {
	if id, ok := ctx.Value(userIdentityKey).(sovereignadapter.TokenIdentity); ok && id.Valid {
		return id.Roles
	}
	return nil
}

// workspaceIDFromRequest extracts the workspace_id from the X-Workspace-ID header.
// Returns "" when the header is absent.
func workspaceIDFromRequest(r *http.Request) string {
	return r.Header.Get("X-Workspace-ID")
}

// withAuth validates requests using either a static API key or a Sovereign
// bearer token. When both are configured, either one grants access.
// If neither apiKey nor sovereign is configured, auth is disabled (dev mode).
func withAuth(apiKey string, sovereign *sovereignadapter.Client, next http.Handler) http.Handler {
	if apiKey == "" && sovereign == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization: Bearer <tok> or X-Api-Key: <key>
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token == r.Header.Get("Authorization") {
			token = "" // no Bearer prefix
		}
		staticKey := r.Header.Get("X-Api-Key")
		if staticKey == "" {
			staticKey = token
		}

		// 1. Static API key check.
		if apiKey != "" && staticKey == apiKey {
			next.ServeHTTP(w, r)
			return
		}

		// 2. Sovereign token check.
		if sovereign != nil && token != "" {
			id := sovereign.ValidateToken(r.Context(), token)
			if id.Valid {
				ctx := context.WithValue(r.Context(), userIdentityKey, id)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
	})
}

// withLogging logs each request: method, path, status, duration.
func withLogging(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		logger.Info("[api] request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", rw.status),
			slog.Duration("duration", time.Since(start)),
			slog.String("request_id", requestIDFromContext(r.Context())),
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}
