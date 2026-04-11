// Package application contains the Overwatch analysis engine.
// It periodically queries the Prometheus HTTP API, evaluates anomaly rules,
// and emits overwatch.alert events to Nerva when thresholds are exceeded.
package application

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	nervav1 "github.com/isaacwallace123/cortex/services/nerva/gen/nerva/v1"
)

// Rule defines a single anomaly threshold.
type Rule struct {
	// Name is the human-readable rule identifier, e.g. "high_deny_rate".
	Name string
	// Query is a PromQL instant-query expression.
	Query string
	// Threshold is the value above which the rule fires.
	Threshold float64
	// Severity is "warning" or "critical".
	Severity string
	// Message is the recommendation emitted in the Nerva event payload.
	Message string
}

// defaultRules is the built-in set of anomaly rules Overwatch evaluates.
var defaultRules = []Rule{
	{
		Name:      "high_deny_rate",
		Query:     `cortex_steps_evaluated_total{verdict="deny"}`,
		Threshold: 10,
		Severity:  "warning",
		Message:   "High number of Aegis denials — review policy rules or user intent classification.",
	},
	{
		Name:      "high_pending_approvals",
		Query:     `cortex_steps_evaluated_total{verdict="pending"}`,
		Threshold: 20,
		Severity:  "warning",
		Message:   "Many steps are awaiting approval — consider delegating or relaxing policy thresholds.",
	},
	{
		Name:      "goroutine_leak_brain",
		Query:     `go_goroutines{job="brain"}`,
		Threshold: 500,
		Severity:  "critical",
		Message:   "Brain goroutine count is abnormally high — possible goroutine leak.",
	},
	{
		Name:      "high_memory_brain",
		Query:     `go_memstats_alloc_bytes{job="brain"}`,
		Threshold: 512 * 1024 * 1024, // 512 MiB
		Severity:  "warning",
		Message:   "Brain memory allocation exceeds 512 MiB — consider restarting or investigating leaks.",
	},
	{
		Name:      "slow_step_execution",
		Query:     `histogram_quantile(0.95, rate(cortex_step_exec_duration_seconds_bucket[5m]))`,
		Threshold: 30,
		Severity:  "warning",
		Message:   "95th-percentile step execution time exceeds 30 s — check executor health.",
	},
}

// Analyzer is the Overwatch analysis engine.
type Analyzer struct {
	prometheusURL string
	nerva         nervav1.NervaServiceClient
	rules         []Rule
	interval      time.Duration
	log           *slog.Logger
	httpClient    *http.Client
}

// NewAnalyzer creates a new Analyzer.
// prometheusURL is the base URL of the Prometheus HTTP API (e.g. "http://prometheus:9090").
// nervaAddr is the gRPC address of the Nerva event bus.
func NewAnalyzer(prometheusURL, nervaAddr string, interval time.Duration, log *slog.Logger) (*Analyzer, error) {
	var nervaClient nervav1.NervaServiceClient
	if nervaAddr != "" {
		conn, err := grpc.NewClient(nervaAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, fmt.Errorf("dial nerva at %s: %w", nervaAddr, err)
		}
		nervaClient = nervav1.NewNervaServiceClient(conn)
	}
	return &Analyzer{
		prometheusURL: prometheusURL,
		nerva:         nervaClient,
		rules:         defaultRules,
		interval:      interval,
		log:           log,
		httpClient:    &http.Client{Timeout: 10 * time.Second},
	}, nil
}

// Run starts the analysis loop. It blocks until ctx is cancelled.
func (a *Analyzer) Run(ctx context.Context) {
	a.log.Info("[Overwatch] analysis engine started",
		slog.String("prometheus", a.prometheusURL),
		slog.Duration("interval", a.interval),
		slog.Int("rules", len(a.rules)),
	)
	t := time.NewTicker(a.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			a.log.Info("[Overwatch] analysis engine stopped")
			return
		case <-t.C:
			a.analyze(ctx)
		}
	}
}

// analyze runs all rules against Prometheus and emits alerts for violated thresholds.
func (a *Analyzer) analyze(ctx context.Context) {
	for _, rule := range a.rules {
		value, ok, err := a.queryInstant(ctx, rule.Query)
		if err != nil {
			a.log.Warn("[Overwatch] prometheus query failed",
				slog.String("rule", rule.Name),
				slog.String("error", err.Error()),
			)
			continue
		}
		if !ok {
			// No data — metric not yet emitted; skip silently.
			continue
		}
		if value > rule.Threshold {
			a.log.Warn("[Overwatch] anomaly detected",
				slog.String("rule", rule.Name),
				slog.String("severity", rule.Severity),
				slog.Float64("value", value),
				slog.Float64("threshold", rule.Threshold),
				slog.String("message", rule.Message),
			)
			a.emit(ctx, rule, value)
		}
	}
}

// queryInstant sends an instant PromQL query and returns the first scalar value.
func (a *Analyzer) queryInstant(ctx context.Context, query string) (float64, bool, error) {
	endpoint := a.prometheusURL + "/api/v1/query"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, false, err
	}
	q := url.Values{"query": {query}}
	req.URL.RawQuery = q.Encode()

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return 0, false, fmt.Errorf("prometheus request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Parse the outer envelope first to get resultType.
	var envelope struct {
		Status string          `json:"status"`
		Error  string          `json:"error,omitempty"`
		Data   json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return 0, false, fmt.Errorf("decode prometheus envelope: %w", err)
	}
	if envelope.Status != "success" {
		return 0, false, fmt.Errorf("prometheus error: %s", envelope.Error)
	}

	// Decode just the resultType to branch.
	var dataType struct {
		ResultType string `json:"resultType"`
	}
	if err := json.Unmarshal(envelope.Data, &dataType); err != nil {
		return 0, false, fmt.Errorf("decode prometheus data type: %w", err)
	}

	switch dataType.ResultType {
	case "vector":
		var data struct {
			Result []struct {
				Value [2]json.RawMessage `json:"value"` // [timestamp, "value_string"]
			} `json:"result"`
		}
		if err := json.Unmarshal(envelope.Data, &data); err != nil || len(data.Result) == 0 {
			return 0, false, nil
		}
		var valStr string
		if err := json.Unmarshal(data.Result[0].Value[1], &valStr); err != nil {
			return 0, false, nil
		}
		v, err := strconv.ParseFloat(valStr, 64)
		if err != nil {
			return 0, false, nil
		}
		return v, true, nil

	case "scalar":
		// scalar result is [timestamp, "value_string"] directly in result field.
		var data struct {
			Result [2]json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(envelope.Data, &data); err != nil {
			return 0, false, nil
		}
		var valStr string
		if err := json.Unmarshal(data.Result[1], &valStr); err != nil {
			return 0, false, nil
		}
		v, err := strconv.ParseFloat(valStr, 64)
		if err != nil {
			return 0, false, nil
		}
		return v, true, nil
	}
	return 0, false, nil
}

// emit fires an overwatch.alert event to Nerva.
func (a *Analyzer) emit(ctx context.Context, rule Rule, value float64) {
	if a.nerva == nil {
		return
	}
	payload, _ := json.Marshal(map[string]any{
		"rule":      rule.Name,
		"severity":  rule.Severity,
		"value":     value,
		"threshold": rule.Threshold,
		"message":   rule.Message,
	})
	_, err := a.nerva.Publish(ctx, &nervav1.PublishRequest{
		Subsystem: "Overwatch",
		EventName: "overwatch.alert",
		Payload:   string(payload),
	})
	if err != nil {
		a.log.Warn("[Overwatch] failed to emit alert to Nerva",
			slog.String("rule", rule.Name),
			slog.String("error", err.Error()),
		)
	}
}
