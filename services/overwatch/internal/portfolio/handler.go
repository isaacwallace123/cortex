package portfolio

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

type Handler struct {
	store *Store
}

func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/insights", h.getInsights)
	mux.HandleFunc("/history", h.getHistory)
	mux.HandleFunc("/pod-insights/all", h.getPodInsightsAll)
	mux.HandleFunc("/pod-insights", h.getPodInsight)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (h *Handler) getInsights(w http.ResponseWriter, _ *http.Request) {
	ins, ok := h.store.GetLatest()
	if !ok {
		now := time.Now()
		ins = Insight{
			CollectedAt:     &now,
			Status:          "healthy",
			Summary:         "No data collected yet — first analysis cycle pending.",
			Anomalies:       []Anomaly{},
			Recommendations: []string{},
		}
	}
	writeJSON(w, ins)
}

func (h *Handler) getHistory(w http.ResponseWriter, r *http.Request) {
	limit := 48
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	writeJSON(w, h.store.GetHistory(limit))
}

func (h *Handler) getPodInsightsAll(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, h.store.GetAllPodInsights())
}

func (h *Handler) getPodInsight(w http.ResponseWriter, r *http.Request) {
	ns := r.URL.Query().Get("namespace")
	app := r.URL.Query().Get("app")
	if ns == "" || app == "" {
		http.Error(w, `{"error":"namespace and app required"}`, http.StatusBadRequest)
		return
	}
	pi, ok := h.store.GetPodInsight(ns, app)
	if !ok {
		pi = PodInsight{
			Namespace:   ns,
			App:         app,
			AnalyzedAt:  time.Now(),
			Status:      "healthy",
			Diagnosis:   "No data collected yet.",
			RootCause:   "First analysis cycle has not completed.",
			Suggestions: []string{"Wait for the next analysis cycle."},
		}
	}
	writeJSON(w, pi)
}
