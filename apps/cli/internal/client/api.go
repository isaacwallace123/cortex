// Package client provides a typed HTTP client for the Cortex API gateway.
// It handles authentication, request serialization, and error mapping so
// callers can interact with the API without dealing with raw HTTP.
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// API is a typed client for the Cortex API gateway.
type API struct {
	base   string
	apiKey string // static key â€” sent as X-Api-Key
	token  string // Sovereign bearer token â€” sent as Authorization: Bearer
	http   *http.Client
}

func NewAPI(baseURL, apiKey string) *API {
	return &API{
		base:   baseURL,
		apiKey: apiKey,
		http:   &http.Client{Timeout: 120 * time.Second},
	}
}

// WithToken returns a copy of the client using a Sovereign bearer token.
func (a *API) WithToken(token string) *API {
	cp := *a
	cp.token = token
	return &cp
}

type ChatRequest struct {
	SessionID string `json:"session_id,omitempty"`
	Input     string `json:"input"`
}

type ChatResponse struct {
	RequestID string   `json:"request_id"`
	SessionID string   `json:"session_id"`
	Intent    string   `json:"intent"`
	Entities  []string `json:"entities"`
	Plan      Plan     `json:"plan"`
}

type Plan struct {
	PlanID string `json:"plan_id"`
	Steps  []Step `json:"steps"`
}

type Step struct {
	Index       int32  `json:"index"`
	Description string `json:"description"`
	Executor    string `json:"executor"`
	Command     string `json:"command,omitempty"`
	Verdict     string `json:"verdict"`
}

func (a *API) Chat(sessionID, input string) (ChatResponse, error) {
	var resp ChatResponse
	err := a.post("/v1/chat", ChatRequest{SessionID: sessionID, Input: input}, &resp)
	return resp, err
}

type RunRequest struct {
	SessionID string `json:"session_id"`
	PlanID    string `json:"plan_id"`
	Steps     []Step `json:"steps"`
}

type StepResult struct {
	StepIndex  int32  `json:"step_index"`
	Executor   string `json:"executor"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	ExitCode   int32  `json:"exit_code"`
	DurationMs int64  `json:"duration_ms"`
	Skipped    bool   `json:"skipped"`
	SkipReason string `json:"skip_reason,omitempty"`
}

type RunResponse struct {
	SessionID string       `json:"session_id"`
	PlanID    string       `json:"plan_id"`
	Results   []StepResult `json:"results"`
}

func (a *API) Run(sessionID, planID string, steps []Step) (RunResponse, error) {
	var resp RunResponse
	err := a.post("/v1/run", RunRequest{SessionID: sessionID, PlanID: planID, Steps: steps}, &resp)
	return resp, err
}

// RunStep executes a single step and returns its result.
// Used by the CLI to stream step results one at a time.
func (a *API) RunStep(sessionID, planID string, step Step) (StepResult, error) {
	resp, err := a.Run(sessionID, planID, []Step{step})
	if err != nil {
		return StepResult{}, err
	}
	if len(resp.Results) == 0 {
		return StepResult{StepIndex: step.Index, Skipped: true, SkipReason: "no result"}, nil
	}
	return resp.Results[0], nil
}

type PlanSummary struct {
	PlanID    string `json:"plan_id"`
	SessionID string `json:"session_id"`
	Intent    string `json:"intent"`
	StepCount int    `json:"step_count"`
	CreatedAt int64  `json:"created_at"`
}

type ListPlansResponse struct {
	SessionID string        `json:"session_id"`
	Plans     []PlanSummary `json:"plans"`
}

type StoredPlan struct {
	PlanID    string `json:"plan_id"`
	SessionID string `json:"session_id"`
	Intent    string `json:"intent"`
	Steps     []Step `json:"steps"`
}

func (a *API) ListPlans(sessionID string) (ListPlansResponse, error) {
	var resp ListPlansResponse
	err := a.get("/v1/plans?session_id="+sessionID, &resp)
	return resp, err
}

func (a *API) GetPlan(sessionID, planID string) (StoredPlan, error) {
	var resp StoredPlan
	err := a.get("/v1/plans/"+planID+"?session_id="+sessionID, &resp)
	return resp, err
}

type SessionResponse struct {
	SessionID string         `json:"session_id"`
	Events    []SessionEvent `json:"events"`
}

type SessionEvent struct {
	EventID   string `json:"event_id"`
	EventName string `json:"event_name"`
	Payload   string `json:"payload"`
	StoredAt  int64  `json:"stored_at"`
}

func (a *API) GetSession(sessionID string) (SessionResponse, error) {
	var resp SessionResponse
	err := a.get("/v1/session/"+sessionID, &resp)
	return resp, err
}

type LoginRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	DeviceName string `json:"device_name,omitempty"`
}

type AuthUser struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
}

type AuthSession struct {
	SessionID string `json:"session_id"`
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}

type LoginResponse struct {
	User    AuthUser    `json:"user"`
	Session AuthSession `json:"session"`
}

func (a *API) Login(username, password, deviceName string) (LoginResponse, error) {
	var resp LoginResponse
	err := a.post("/v1/auth/login", LoginRequest{Username: username, Password: password, DeviceName: deviceName}, &resp)
	return resp, err
}

type WhoAmIResponse struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
}

func (a *API) WhoAmI() (WhoAmIResponse, error) {
	var resp WhoAmIResponse
	err := a.get("/v1/auth/me", &resp)
	return resp, err
}

type LogoutResponse struct {
	Revoked bool `json:"revoked"`
}

func (a *API) Logout() (LogoutResponse, error) {
	var resp LogoutResponse
	err := a.post("/v1/auth/logout", nil, &resp)
	return resp, err
}

type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

func (a *API) Health() (HealthResponse, error) {
	var resp HealthResponse
	err := a.get("/healthz", &resp)
	return resp, err
}

// setAuth attaches the appropriate auth header to req. Bearer token takes
// precedence over the static API key.
func (a *API) setAuth(req *http.Request) {
	if a.token != "" {
		req.Header.Set("Authorization", "Bearer "+a.token)
	} else if a.apiKey != "" {
		req.Header.Set("X-Api-Key", a.apiKey)
	}
}

func (a *API) post(path string, body, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, a.base+path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	a.setAuth(req)
	resp, err := a.http.Do(req)
	if err != nil {
		return fmt.Errorf("POST %s: %w", path, err)
	}
	return a.decode(resp, out)
}

func (a *API) get(path string, out any) error {
	req, err := http.NewRequest(http.MethodGet, a.base+path, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	a.setAuth(req)
	resp, err := a.http.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", path, err)
	}
	return a.decode(resp, out)
}

func (a *API) delete(path string, out any) error {
	req, err := http.NewRequest(http.MethodDelete, a.base+path, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	a.setAuth(req)
	resp, err := a.http.Do(req)
	if err != nil {
		return fmt.Errorf("DELETE %s: %w", path, err)
	}
	return a.decode(resp, out)
}

func (a *API) decode(resp *http.Response, out any) error {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		var apiErr struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(body, &apiErr)
		if apiErr.Error != "" {
			return fmt.Errorf("api error (%d): %s", resp.StatusCode, apiErr.Error)
		}
		return fmt.Errorf("http %d", resp.StatusCode)
	}
	return json.Unmarshal(body, out)
}

