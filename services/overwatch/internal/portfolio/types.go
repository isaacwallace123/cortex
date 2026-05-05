package portfolio

import "time"

type Anomaly struct {
	Severity    string `json:"severity"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Affected    string `json:"affected"`
}

type Insight struct {
	ID              *int       `json:"id,omitempty"`
	CollectedAt     *time.Time `json:"collected_at"`
	Status          string     `json:"status"`
	Summary         string     `json:"summary"`
	Anomalies       []Anomaly  `json:"anomalies"`
	Recommendations []string   `json:"recommendations"`
}

type PodInsight struct {
	Namespace   string    `json:"namespace"`
	App         string    `json:"app"`
	AnalyzedAt  time.Time `json:"analyzed_at"`
	Status      string    `json:"status"`
	Diagnosis   string    `json:"diagnosis"`
	RootCause   string    `json:"root_cause"`
	Suggestions []string  `json:"suggestions"`
}
