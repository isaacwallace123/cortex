// Package metrics registers Prometheus metrics for the brain service.
// All metrics are registered on the default registry and exposed via /metrics.
package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	// InputsParsedTotal counts inputs processed by Prism.
	InputsParsedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "cortex_inputs_parsed_total",
		Help: "Total number of raw inputs parsed by Prism.",
	})

	// PlansCreatedTotal counts plans produced by Axiom.
	PlansCreatedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "cortex_plans_created_total",
		Help: "Total number of plans created by Axiom.",
	})

	// StepsEvaluatedTotal counts policy verdicts issued by Aegis, per verdict label.
	StepsEvaluatedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cortex_steps_evaluated_total",
		Help: "Total plan steps evaluated by Aegis, labeled by verdict.",
	}, []string{"verdict"})

	// StepExecDurationSeconds records step execution latency across all executors (Vector).
	StepExecDurationSeconds = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "cortex_step_exec_duration_seconds",
		Help:    "Duration of step execution by Vector (shell, filesystem, and future executors).",
		Buckets: prometheus.DefBuckets,
	})
)

func init() {
	prometheus.MustRegister(
		InputsParsedTotal,
		PlansCreatedTotal,
		StepsEvaluatedTotal,
		StepExecDurationSeconds,
	)
}
