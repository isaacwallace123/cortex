package portfolio

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// Engine periodically collects cluster state from Prometheus and builds insights
// for the portfolio REST API.
type Engine struct {
	prom     *promClient
	ollama   *ollamaClient
	store    *Store
	interval time.Duration
	log      *slog.Logger
}

func NewEngine(prometheusURL, ollamaURL, ollamaModel string, interval time.Duration, log *slog.Logger) *Engine {
	return &Engine{
		prom:     newPromClient(prometheusURL),
		ollama:   newOllamaClient(ollamaURL, ollamaModel),
		store:    NewStore(),
		interval: interval,
		log:      log,
	}
}

func (e *Engine) Store() *Store { return e.store }

func (e *Engine) Run(ctx context.Context) {
	e.log.Info("[Portfolio] engine started", slog.Duration("interval", e.interval))
	e.refresh(ctx)
	t := time.NewTicker(e.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			e.log.Info("[Portfolio] engine stopped")
			return
		case <-t.C:
			e.refresh(ctx)
		}
	}
}

// ── cluster-level insight ────────────────────────────────────────────────────

type clusterMetrics struct {
	readyNodes  int
	totalNodes  int
	runningPods int
	problemPods int
	restartRate float64 // restarts/min
	oomCount    int
}

func (e *Engine) refresh(ctx context.Context) {
	now := time.Now()

	cm := e.collectCluster(ctx)
	ins := e.buildInsight(ctx, now, cm)
	e.store.AddInsight(ins)

	pods := e.collectAllPodMetrics(ctx)
	for key, pm := range pods {
		ns, app, _ := strings.Cut(key, "/")
		e.store.SetPodInsight(ns, app, e.buildPodInsight(now, ns, app, pm))
	}
	e.log.Info("[Portfolio] refresh complete",
		slog.String("status", ins.Status),
		slog.Int("apps", len(pods)),
	)
}

func (e *Engine) collectCluster(ctx context.Context) clusterMetrics {
	var cm clusterMetrics
	if rows, err := e.prom.queryVector(ctx, `kube_node_status_condition{condition="Ready",status="true"}`); err == nil {
		cm.readyNodes = len(rows)
	}
	if rows, err := e.prom.queryVector(ctx, `kube_node_info`); err == nil {
		cm.totalNodes = len(rows)
	}
	if v, ok, _ := e.prom.queryScalar(ctx, `count(kube_pod_status_phase{phase="Running"})`); ok {
		cm.runningPods = int(v)
	}
	if v, ok, _ := e.prom.queryScalar(ctx, `count(kube_pod_status_phase{phase=~"Failed|Pending|Unknown"})`); ok {
		cm.problemPods = int(v)
	}
	if v, ok, _ := e.prom.queryScalar(ctx, `sum(rate(kube_pod_container_status_restarts_total[15m]))*60`); ok {
		cm.restartRate = v
	}
	if v, ok, _ := e.prom.queryScalar(ctx, `count(kube_pod_container_status_last_terminated_reason{reason="OOMKilled"})`); ok {
		cm.oomCount = int(v)
	}
	return cm
}

func (e *Engine) buildInsight(ctx context.Context, t time.Time, cm clusterMetrics) Insight {
	now := t
	ins := Insight{
		CollectedAt:     &now,
		Anomalies:       []Anomaly{},
		Recommendations: []string{},
	}

	if cm.totalNodes > 0 && cm.readyNodes < cm.totalNodes {
		ins.Anomalies = append(ins.Anomalies, Anomaly{
			Severity:    "critical",
			Type:        "node_not_ready",
			Description: fmt.Sprintf("%d of %d nodes not Ready", cm.totalNodes-cm.readyNodes, cm.totalNodes),
			Affected:    "cluster",
		})
		ins.Recommendations = append(ins.Recommendations, "Investigate not-ready nodes: check kubelet logs and node conditions.")
	}
	if cm.problemPods > 0 {
		ins.Anomalies = append(ins.Anomalies, Anomaly{
			Severity:    "warning",
			Type:        "pods_not_running",
			Description: fmt.Sprintf("%d pods in Failed/Pending/Unknown state", cm.problemPods),
			Affected:    "cluster",
		})
		ins.Recommendations = append(ins.Recommendations, "Check pod events and logs for Failed/Pending pods.")
	}
	if cm.restartRate > 0.5 {
		ins.Anomalies = append(ins.Anomalies, Anomaly{
			Severity:    "warning",
			Type:        "high_restart_rate",
			Description: fmt.Sprintf("Container restart rate %.2f/min", cm.restartRate),
			Affected:    "cluster",
		})
		ins.Recommendations = append(ins.Recommendations, "Review logs for crash-looping containers.")
	}
	if cm.oomCount > 0 {
		ins.Anomalies = append(ins.Anomalies, Anomaly{
			Severity:    "warning",
			Type:        "oom_killed",
			Description: fmt.Sprintf("%d container(s) recently OOM-killed", cm.oomCount),
			Affected:    "cluster",
		})
		ins.Recommendations = append(ins.Recommendations, "Increase memory limits for OOM-killed containers.")
	}

	for _, a := range ins.Anomalies {
		if a.Severity == "critical" {
			ins.Status = "critical"
			break
		}
		ins.Status = "warning"
	}
	if ins.Status == "" {
		ins.Status = "healthy"
	}

	ins.Summary = e.summarizeCluster(ctx, cm, ins)
	return ins
}

func (e *Engine) summarizeCluster(ctx context.Context, cm clusterMetrics, ins Insight) string {
	if ins.Status == "healthy" {
		return fmt.Sprintf("Cluster healthy: %d/%d nodes ready, %d pods running.", cm.readyNodes, cm.totalNodes, cm.runningPods)
	}

	var anomalyLines strings.Builder
	for _, a := range ins.Anomalies {
		fmt.Fprintf(&anomalyLines, "- [%s] %s: %s\n", a.Severity, a.Type, a.Description)
	}
	prompt := fmt.Sprintf(
		"You are a Kubernetes monitoring assistant. Write a single concise sentence summarizing the cluster state.\n\nMetrics:\n- Nodes: %d/%d ready\n- Running pods: %d, problem pods: %d\n- Restart rate: %.2f/min, OOM kills: %d\n\nAnomalies:\n%s\nWrite only the sentence, no JSON or markdown.",
		cm.readyNodes, cm.totalNodes, cm.runningPods, cm.problemPods, cm.restartRate, cm.oomCount, anomalyLines.String(),
	)

	lctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	text, err := e.ollama.chat(lctx, prompt)
	if err != nil || strings.TrimSpace(text) == "" {
		e.log.Warn("[Portfolio] ollama summary failed", slog.String("err", fmt.Sprintf("%v", err)))
		return fmt.Sprintf("Cluster %s: %d/%d nodes ready, %d running pods, %d anomaly(s) detected.", ins.Status, cm.readyNodes, cm.totalNodes, cm.runningPods, len(ins.Anomalies))
	}
	return strings.TrimSpace(text)
}

// ── pod-level insights ───────────────────────────────────────────────────────

type podAgg struct {
	running  int
	failed   int
	pending  int
	restarts float64
}

// collectAllPodMetrics fetches phase + restart data for all pods in one pass
// and groups them by "namespace/label_app".
func (e *Engine) collectAllPodMetrics(ctx context.Context) map[string]podAgg {
	// app label assignments: pod key → app name
	appOf := make(map[string]string) // "ns/pod" → app
	if rows, err := e.prom.queryVector(ctx, `kube_pod_labels{label_app!=""}`); err == nil {
		for _, r := range rows {
			ns, pod, app := r.Labels["namespace"], r.Labels["pod"], r.Labels["label_app"]
			if ns != "" && pod != "" && app != "" {
				appOf[ns+"/"+pod] = app
			}
		}
	}
	if len(appOf) == 0 {
		return nil
	}

	result := make(map[string]podAgg) // "ns/app" → aggregated metrics

	if rows, err := e.prom.queryVector(ctx, `kube_pod_status_phase`); err == nil {
		for _, r := range rows {
			ns, pod, phase := r.Labels["namespace"], r.Labels["pod"], r.Labels["phase"]
			app, ok := appOf[ns+"/"+pod]
			if !ok {
				continue
			}
			key := ns + "/" + app
			agg := result[key]
			switch phase {
			case "Running":
				agg.running++
			case "Failed":
				agg.failed++
			case "Pending":
				agg.pending++
			}
			result[key] = agg
		}
	}

	if rows, err := e.prom.queryVector(ctx, `kube_pod_container_status_restarts_total`); err == nil {
		for _, r := range rows {
			ns, pod := r.Labels["namespace"], r.Labels["pod"]
			app, ok := appOf[ns+"/"+pod]
			if !ok {
				continue
			}
			key := ns + "/" + app
			agg := result[key]
			agg.restarts += r.Value
			result[key] = agg
		}
	}

	return result
}

func (e *Engine) buildPodInsight(t time.Time, namespace, app string, pm podAgg) PodInsight {
	pi := PodInsight{
		Namespace:   namespace,
		App:         app,
		AnalyzedAt:  t,
		Suggestions: []string{},
	}

	total := pm.running + pm.failed + pm.pending
	if total == 0 {
		pi.Status = "healthy"
		pi.Diagnosis = "No metrics available yet."
		pi.RootCause = "Prometheus may not have scraped this pod yet."
		pi.Suggestions = []string{"Wait for the next scrape interval."}
		return pi
	}

	if pm.failed == 0 && pm.pending == 0 && pm.restarts < 5 {
		pi.Status = "healthy"
		pi.Diagnosis = fmt.Sprintf("%d pod(s) running normally.", pm.running)
		pi.RootCause = "No issues detected."
		pi.Suggestions = []string{"No action required."}
		return pi
	}

	if pm.failed > 0 {
		pi.Status = "critical"
	} else {
		pi.Status = "warning"
	}
	pi.Diagnosis = fmt.Sprintf("%d running, %d failed, %d pending, %.0f total restarts.", pm.running, pm.failed, pm.pending, pm.restarts)

	if pm.restarts >= 5 && pm.failed == 0 {
		pi.RootCause = "Containers are crash-looping — check application logs."
		pi.Suggestions = []string{
			"Run: kubectl logs -n " + namespace + " -l app=" + app + " --previous",
			"Check resource limits: pod may be OOM-killed.",
		}
	} else if pm.failed > 0 {
		pi.RootCause = "One or more pods are in a Failed state."
		pi.Suggestions = []string{
			"Run: kubectl describe pod -n " + namespace + " -l app=" + app,
			"Check events: kubectl get events -n " + namespace + " --sort-by=.lastTimestamp",
		}
	} else {
		pi.RootCause = "Pods are pending — likely waiting for resources or volume mounts."
		pi.Suggestions = []string{
			"Check: kubectl describe pod -n " + namespace + " -l app=" + app,
			"Verify PVCs are bound and node resources are available.",
		}
	}
	return pi
}
