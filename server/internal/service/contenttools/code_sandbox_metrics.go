package contenttools

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	codeSandboxMetricsOnce sync.Once

	codeSandboxRunsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_code_runs_total",
		Help:      "Code Sandbox run/check outcomes (CT.17).",
	}, []string{"language", "action", "status"})

	codeSandboxTestPassRatio = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "lextures",
		Name:      "content_tool_code_test_pass_ratio",
		Help:      "Most recent check pass ratio by language (CT.17).",
	}, []string{"language"})
)

func registerCodeSandboxMetrics() {
	codeSandboxMetricsOnce.Do(func() {
		prometheus.MustRegister(codeSandboxRunsTotal, codeSandboxTestPassRatio)
		codeSandboxRunsTotal.WithLabelValues("_", "_", "_reserved").Add(0)
	})
}

// ObserveCodeSandboxRun increments lextures_content_tool_code_runs_total.
func ObserveCodeSandboxRun(language, action, status string) {
	registerCodeSandboxMetrics()
	if language == "" {
		language = "_unknown"
	}
	if action == "" {
		action = "_unknown"
	}
	if status == "" {
		status = "_unknown"
	}
	codeSandboxRunsTotal.WithLabelValues(language, action, status).Inc()
}

// ObserveCodeSandboxTests updates the pass-ratio gauge after a check.
func ObserveCodeSandboxTests(language string, passed, total int) {
	registerCodeSandboxMetrics()
	if language == "" {
		language = "_unknown"
	}
	if total <= 0 {
		return
	}
	codeSandboxTestPassRatio.WithLabelValues(language).Set(float64(passed) / float64(total))
}
