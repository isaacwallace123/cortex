package portfolio

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// Engine periodically collects cluster state and builds insights for the portfolio REST API.
type Engine struct {
	prom     *promClient
	k8s      *k8sClient
	ollama   *ollamaClient
	store    *Store
	interval time.Duration
	log      *slog.Logger
}

func NewEngine(prometheusURL, ollamaURL, ollamaModel string, interval time.Duration, log *slog.Logger) *Engine {
	k8s, err := newInClusterK8sClient()
	if err != nil {
		log.Warn("[Portfolio] k8s in-cluster client unavailable — pod/node data will be empty", slog.String("err", err.Error()))
	}
	return &Engine{
		prom:     newPromClient(prometheusURL),
		k8s:      k8s,
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
	readyNodes      int
	totalNodes      int
	runningPods     int
	problemPods     int
	highRestartPods int
	oomCount        int
	avgCPUPct       float64
	avgMemPct       float64
}

func (e *Engine) refresh(ctx context.Context) {
	now := time.Now()

	// Fetch all pods once and reuse for both cluster and per-app insights.
	var pods *PodList
	if e.k8s != nil {
		if p, err := e.k8s.ListPods(ctx); err == nil {
			pods = p
		} else {
			e.log.Warn("[Portfolio] ListPods failed", slog.String("err", err.Error()))
		}
	}

	cm := e.collectCluster(ctx, pods)
	ins := e.buildInsight(ctx, now, cm)
	e.store.AddInsight(ins)

	appMap := e.collectAllPodMetrics(pods)
	for key, pm := range appMap {
		ns, app, _ := strings.Cut(key, "/")
		e.store.SetPodInsight(ns, app, e.buildPodInsight(now, ns, app, pm))
	}
	e.log.Info("[Portfolio] refresh complete",
		slog.String("status", ins.Status),
		slog.Int("nodes", cm.totalNodes),
		slog.Int("pods", cm.runningPods),
		slog.Int("apps", len(appMap)),
	)
}

func (e *Engine) collectCluster(ctx context.Context, pods *PodList) clusterMetrics {
	var cm clusterMetrics

	// Node data from k8s API.
	if e.k8s != nil {
		if nodes, err := e.k8s.ListNodes(ctx); err == nil {
			for _, n := range nodes.Items {
				cm.totalNodes++
				for _, c := range n.Status.Conditions {
					if c.Type == "Ready" && c.Status == "True" {
						cm.readyNodes++
					}
				}
			}
		} else {
			e.log.Warn("[Portfolio] ListNodes failed", slog.String("err", err.Error()))
		}
	}

	// Pod data from the already-fetched list.
	if pods != nil {
		for _, p := range pods.Items {
			switch p.Status.Phase {
			case "Running":
				cm.runningPods++
			case "Failed", "Unknown", "Pending":
				cm.problemPods++
			}
			restarts := 0
			for _, cs := range p.Status.ContainerStatuses {
				restarts += cs.RestartCount
				if cs.LastState.Terminated != nil && cs.LastState.Terminated.Reason == "OOMKilled" {
					cm.oomCount++
				}
			}
			if restarts >= 5 {
				cm.highRestartPods++
			}
		}
	}

	// Performance metrics from node-exporter via Prometheus.
	if v, ok, _ := e.prom.queryScalar(ctx, `100-(avg(rate(node_cpu_seconds_total{mode="idle"}[5m]))*100)`); ok {
		cm.avgCPUPct = v
	}
	if v, ok, _ := e.prom.queryScalar(ctx, `(1-avg(node_memory_MemAvailable_bytes/node_memory_MemTotal_bytes))*100`); ok {
		cm.avgMemPct = v
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
	if cm.highRestartPods > 0 {
		ins.Anomalies = append(ins.Anomalies, Anomaly{
			Severity:    "warning",
			Type:        "crash_looping",
			Description: fmt.Sprintf("%d pod(s) have ≥5 container restarts", cm.highRestartPods),
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
	if cm.avgCPUPct > 85 {
		ins.Anomalies = append(ins.Anomalies, Anomaly{
			Severity:    "warning",
			Type:        "high_cpu",
			Description: fmt.Sprintf("Average node CPU usage %.1f%%", cm.avgCPUPct),
			Affected:    "cluster",
		})
		ins.Recommendations = append(ins.Recommendations, "Identify high-CPU pods and review resource limits.")
	}
	if cm.avgMemPct > 85 {
		ins.Anomalies = append(ins.Anomalies, Anomaly{
			Severity:    "warning",
			Type:        "high_memory",
			Description: fmt.Sprintf("Average node memory usage %.1f%%", cm.avgMemPct),
			Affected:    "cluster",
		})
		ins.Recommendations = append(ins.Recommendations, "Review memory limits and consider scaling workloads.")
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
		return fmt.Sprintf("Cluster healthy: %d/%d nodes ready, %d pods running, CPU %.0f%%, mem %.0f%%.",
			cm.readyNodes, cm.totalNodes, cm.runningPods, cm.avgCPUPct, cm.avgMemPct)
	}

	var anomalyLines strings.Builder
	for _, a := range ins.Anomalies {
		fmt.Fprintf(&anomalyLines, "- [%s] %s: %s\n", a.Severity, a.Type, a.Description)
	}
	prompt := fmt.Sprintf(
		"You are a Kubernetes monitoring assistant. Write a single concise sentence summarizing the cluster state.\n\nMetrics:\n- Nodes: %d/%d ready\n- Running pods: %d, problem pods: %d\n- Restart pods: %d, OOM kills: %d\n- CPU: %.0f%%, Memory: %.0f%%\n\nAnomalies:\n%s\nWrite only the sentence, no JSON or markdown.",
		cm.readyNodes, cm.totalNodes, cm.runningPods, cm.problemPods,
		cm.highRestartPods, cm.oomCount, cm.avgCPUPct, cm.avgMemPct,
		anomalyLines.String(),
	)

	lctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	text, err := e.ollama.chat(lctx, prompt)
	if err != nil || strings.TrimSpace(text) == "" {
		return fmt.Sprintf("Cluster %s: %d/%d nodes ready, %d running pods, %d anomaly(s) detected.",
			ins.Status, cm.readyNodes, cm.totalNodes, cm.runningPods, len(ins.Anomalies))
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

func (e *Engine) collectAllPodMetrics(pods *PodList) map[string]podAgg {
	if pods == nil {
		return nil
	}
	result := make(map[string]podAgg)
	for _, p := range pods.Items {
		app := p.Metadata.Labels["app"]
		if app == "" {
			app = p.Metadata.Labels["app.kubernetes.io/name"]
		}
		if app == "" {
			continue
		}
		key := p.Metadata.Namespace + "/" + app
		agg := result[key]
		switch p.Status.Phase {
		case "Running":
			agg.running++
		case "Failed":
			agg.failed++
		case "Pending":
			agg.pending++
		}
		for _, cs := range p.Status.ContainerStatuses {
			agg.restarts += float64(cs.RestartCount)
		}
		result[key] = agg
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
		pi.Diagnosis = "No pods found for this app."
		pi.RootCause = "App may have scaled to zero or not yet scheduled."
		pi.Suggestions = []string{"Check: kubectl get pods -n " + namespace + " -l app=" + app}
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

	switch {
	case pm.failed > 0:
		pi.RootCause = "One or more pods are in a Failed state."
		pi.Suggestions = []string{
			"kubectl describe pod -n " + namespace + " -l app=" + app,
			"kubectl get events -n " + namespace + " --sort-by=.lastTimestamp",
		}
	case pm.pending > 0:
		pi.RootCause = "Pods are pending — likely waiting for resources or volume mounts."
		pi.Suggestions = []string{
			"kubectl describe pod -n " + namespace + " -l app=" + app,
			"Verify PVCs are bound and node resources are available.",
		}
	default:
		pi.RootCause = "Containers are crash-looping — check application logs."
		pi.Suggestions = []string{
			"kubectl logs -n " + namespace + " -l app=" + app + " --previous",
			"Check resource limits: pod may be OOM-killed.",
		}
	}
	return pi
}
