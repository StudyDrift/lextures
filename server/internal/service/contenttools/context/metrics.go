package context

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	contextBuildSeconds = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "lextures_content_tool_context_build_seconds",
		Help:    "CT.6 context pack build latency in seconds",
		Buckets: prometheus.DefBuckets,
	})
	linkFetchTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lextures_content_tool_link_fetch_total",
		Help: "CT.6 link fetch outcomes",
	}, []string{"outcome"})
	linkCacheHitsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "lextures_content_tool_link_cache_hits_total",
		Help: "CT.6 link cache hits",
	})
	aiCallsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lextures_content_tool_ai_calls_total",
		Help: "CT.6 AI calls through the context rails",
	}, []string{"tool_id", "outcome"})
	aiBudgetDenialsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lextures_content_tool_ai_budget_denials_total",
		Help: "CT.6 AI budget denials by level",
	}, []string{"level"})
)

func observeBuild(d time.Duration) { contextBuildSeconds.Observe(d.Seconds()) }

func observeFetch(outcome string) { linkFetchTotal.WithLabelValues(outcome).Inc() }

func observeCacheHit() { linkCacheHitsTotal.Inc() }

func observeAICall(toolID, outcome string) {
	aiCallsTotal.WithLabelValues(toolID, outcome).Inc()
}

func observeBudgetDenial(level string) {
	aiBudgetDenialsTotal.WithLabelValues(level).Inc()
}
