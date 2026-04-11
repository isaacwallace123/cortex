// Auth handlers for the Cortex HTTP gateway.
// Register, login, logout, and identity verification are handled here.
// These endpoints are public — they do not require a token, since they are
// responsible for issuing and revoking tokens.
package http

import (
	"encoding/json"
	"net/http"
	"strings"

	sovereignadapter "github.com/isaacwallace123/cortex/services/api/internal/adapters/sovereign"
)

// handleAuthRegister handles POST /v1/auth/register. Creates a new user account
// and returns the user record along with a session token.
func (s *Server) handleAuthRegister(w http.ResponseWriter, r *http.Request) {
	if s.sovereign == nil {
		http.Error(w, `{"error":"auth service unavailable"}`, http.StatusServiceUnavailable)
		return
	}
	var body struct {
		Username   string `json:"username"`
		Email      string `json:"email"`
		Password   string `json:"password"`
		Role       string `json:"role"`
		DeviceID   string `json:"device_id"`
		DeviceName string `json:"device_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Username == "" || body.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password are required")
		return
	}
	user, sess, err := s.sovereign.Register(r.Context(),
		body.Username, body.Email, body.Password, body.Role,
		body.DeviceID, body.DeviceName,
	)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"user":    user,
		"session": sess,
	})
}

// handleAuthLogin handles POST /v1/auth/login. Validates credentials and returns
// the user record and a fresh session token.
func (s *Server) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if s.sovereign == nil {
		http.Error(w, `{"error":"auth service unavailable"}`, http.StatusServiceUnavailable)
		return
	}
	var body struct {
		Username   string `json:"username"`
		Password   string `json:"password"`
		DeviceID   string `json:"device_id"`
		DeviceName string `json:"device_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	user, sess, err := s.sovereign.Login(r.Context(),
		body.Username, body.Password,
		body.DeviceID, body.DeviceName,
	)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user":    user,
		"session": sess,
	})
}

// handleAuthLogout handles POST /v1/auth/logout. Revokes the caller's session token.
func (s *Server) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	if s.sovereign == nil {
		http.Error(w, `{"error":"auth service unavailable"}`, http.StatusServiceUnavailable)
		return
	}
	token := bearerToken(r)
	if token == "" {
		writeError(w, http.StatusBadRequest, "no token provided")
		return
	}
	revoked, err := s.sovereign.Logout(r.Context(), token)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"revoked": revoked})
}

// handleAuthMe handles GET /v1/auth/me. Returns identity information for the
// currently authenticated caller.
func (s *Server) handleAuthMe(w http.ResponseWriter, r *http.Request) {
	if s.sovereign == nil {
		http.Error(w, `{"error":"auth service unavailable"}`, http.StatusServiceUnavailable)
		return
	}
	token := bearerToken(r)
	id := s.sovereign.ValidateToken(r.Context(), token)
	if !id.Valid {
		writeError(w, http.StatusUnauthorized, id.Reason)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user_id":  id.UserID,
		"username": id.Username,
		"roles":    id.Roles,
	})
}

// handleListUsers handles GET /v1/auth/users. Admin-only endpoint that returns
// all registered users.
func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	if s.sovereign == nil {
		http.Error(w, `{"error":"auth service unavailable"}`, http.StatusServiceUnavailable)
		return
	}
	id, ok := r.Context().Value(userIdentityKey).(sovereignadapter.TokenIdentity)
	if !ok || !id.Valid || !containsRole(id.Roles, "admin") {
		writeError(w, http.StatusForbidden, "admin role required")
		return
	}
	users, err := s.sovereign.ListUsers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": users})
}

// bearerToken extracts the Bearer token from the Authorization header.
func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if after, ok := strings.CutPrefix(h, "Bearer "); ok {
		return strings.TrimSpace(after)
	}
	return ""
}

// containsRole reports whether roles contains the target role.
func containsRole(roles []string, target string) bool {
	for _, r := range roles {
		if r == target {
			return true
		}
	}
	return false
}

