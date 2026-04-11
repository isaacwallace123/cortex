// Compass HTTP handlers — project, milestone, task, and decision management.
// All endpoints proxy to the Compass gRPC service via compassadapter.Client.
package http

import (
	"encoding/json"
	"net/http"
)

// compassUnavailable returns 503 when the Compass client is not wired.
func (s *Server) compassUnavailable(w http.ResponseWriter) bool {
	if s.compass == nil {
		http.Error(w, `{"error":"compass service unavailable"}`, http.StatusServiceUnavailable)
		return true
	}
	return false
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	if s.compassUnavailable(w) {
		return
	}
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		http.Error(w, `{"error":"name is required"}`, http.StatusBadRequest)
		return
	}
	project, err := s.compass.CreateProject(r.Context(), body.Name, body.Description)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, project)
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	if s.compassUnavailable(w) {
		return
	}
	projects, err := s.compass.ListProjects(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"projects": projects})
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	if s.compassUnavailable(w) {
		return
	}
	projectID := r.PathValue("id")
	detail, err := s.compass.GetProject(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) handleUpdateProjectStatus(w http.ResponseWriter, r *http.Request) {
	if s.compassUnavailable(w) {
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Status == "" {
		http.Error(w, `{"error":"status is required"}`, http.StatusBadRequest)
		return
	}
	project, err := s.compass.UpdateProjectStatus(r.Context(), r.PathValue("id"), body.Status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, project)
}

func (s *Server) handleCreateMilestone(w http.ResponseWriter, r *http.Request) {
	if s.compassUnavailable(w) {
		return
	}
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Order       int32  `json:"order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		http.Error(w, `{"error":"name is required"}`, http.StatusBadRequest)
		return
	}
	m, err := s.compass.CreateMilestone(r.Context(), r.PathValue("id"), body.Name, body.Description, body.Order)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, m)
}

func (s *Server) handleListMilestones(w http.ResponseWriter, r *http.Request) {
	if s.compassUnavailable(w) {
		return
	}
	milestones, err := s.compass.ListMilestones(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"milestones": milestones})
}

func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	if s.compassUnavailable(w) {
		return
	}
	var body struct {
		Name               string   `json:"name"`
		Description        string   `json:"description"`
		AcceptanceCriteria []string `json:"acceptance_criteria"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		http.Error(w, `{"error":"name is required"}`, http.StatusBadRequest)
		return
	}
	t, err := s.compass.CreateTask(r.Context(), r.PathValue("id"), body.Name, body.Description, body.AcceptanceCriteria)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	if s.compassUnavailable(w) {
		return
	}
	tasks, err := s.compass.ListTasks(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tasks": tasks})
}

func (s *Server) handleUpdateTaskStatus(w http.ResponseWriter, r *http.Request) {
	if s.compassUnavailable(w) {
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Status == "" {
		http.Error(w, `{"error":"status is required"}`, http.StatusBadRequest)
		return
	}
	t, err := s.compass.UpdateTaskStatus(r.Context(), r.PathValue("id"), body.Status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) handleAddDecision(w http.ResponseWriter, r *http.Request) {
	if s.compassUnavailable(w) {
		return
	}
	var body struct {
		Title        string   `json:"title"`
		Rationale    string   `json:"rationale"`
		Alternatives []string `json:"alternatives"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Title == "" {
		http.Error(w, `{"error":"title is required"}`, http.StatusBadRequest)
		return
	}
	d, err := s.compass.AddDecision(r.Context(), r.PathValue("id"), body.Title, body.Rationale, body.Alternatives)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, d)
}

func (s *Server) handleListDecisions(w http.ResponseWriter, r *http.Request) {
	if s.compassUnavailable(w) {
		return
	}
	decisions, err := s.compass.ListDecisions(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"decisions": decisions})
}

