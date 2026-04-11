// Workspace HTTP handlers — CRUD operations for user workspaces.
// All endpoints require authentication; the user_id is always sourced from
// the verified token rather than the request body.
package http

import (
	"encoding/json"
	"net/http"
)

type CreateWorkspaceRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	RootPath    string `json:"root_path"`
}

func (s *Server) handleCreateWorkspace(w http.ResponseWriter, r *http.Request) {
	if s.workspace == nil {
		writeError(w, http.StatusServiceUnavailable, "workspace service unavailable")
		return
	}
	var req CreateWorkspaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name required")
		return
	}
	userID := userIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	ws, err := s.workspace.CreateWorkspace(r.Context(), userID, req.Name, req.Description, req.RootPath)
	if err != nil {
		s.logger.Error("[api] create_workspace failed", "error", err.Error())
		writeError(w, http.StatusBadGateway, "workspace service error")
		return
	}
	writeJSON(w, http.StatusCreated, ws)
}

func (s *Server) handleListWorkspaces(w http.ResponseWriter, r *http.Request) {
	if s.workspace == nil {
		writeError(w, http.StatusServiceUnavailable, "workspace service unavailable")
		return
	}
	userID := userIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	workspaces, err := s.workspace.ListWorkspaces(r.Context(), userID)
	if err != nil {
		s.logger.Error("[api] list_workspaces failed", "error", err.Error())
		writeError(w, http.StatusBadGateway, "workspace service error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"workspaces": workspaces})
}

func (s *Server) handleGetWorkspace(w http.ResponseWriter, r *http.Request) {
	if s.workspace == nil {
		writeError(w, http.StatusServiceUnavailable, "workspace service unavailable")
		return
	}
	workspaceID := r.PathValue("id")
	userID := userIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	ws, err := s.workspace.GetWorkspace(r.Context(), workspaceID, userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "workspace not found")
		return
	}
	writeJSON(w, http.StatusOK, ws)
}

type UpdateWorkspaceRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	RootPath    string `json:"root_path"`
}

func (s *Server) handleUpdateWorkspace(w http.ResponseWriter, r *http.Request) {
	if s.workspace == nil {
		writeError(w, http.StatusServiceUnavailable, "workspace service unavailable")
		return
	}
	workspaceID := r.PathValue("id")
	userID := userIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	var req UpdateWorkspaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	ws, err := s.workspace.UpdateWorkspace(r.Context(), workspaceID, userID, req.Name, req.Description, req.RootPath)
	if err != nil {
		s.logger.Error("[api] update_workspace failed", "error", err.Error())
		writeError(w, http.StatusBadGateway, "workspace service error")
		return
	}
	writeJSON(w, http.StatusOK, ws)
}

func (s *Server) handleDeleteWorkspace(w http.ResponseWriter, r *http.Request) {
	if s.workspace == nil {
		writeError(w, http.StatusServiceUnavailable, "workspace service unavailable")
		return
	}
	workspaceID := r.PathValue("id")
	userID := userIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	deleted, err := s.workspace.DeleteWorkspace(r.Context(), workspaceID, userID)
	if err != nil {
		s.logger.Error("[api] delete_workspace failed", "error", err.Error())
		writeError(w, http.StatusBadGateway, "workspace service error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": deleted})
}

