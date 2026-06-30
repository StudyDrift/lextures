package lessonplanai

import (
	"log/slog"
	"sync"
	"time"
)

var (
	metricsOnce sync.Once
	requests    uint64
	latencySum  int64
	failures    = map[string]uint64{}
	failMu      sync.Mutex
)

func recordRequest() {
	metricsOnce.Do(func() {})
	requests++
	slog.Info("lesson_generation_requests_total", "count", requests)
}

func recordLatency(d time.Duration) {
	latencySum += d.Milliseconds()
	slog.Info("lesson_generation_latency_ms", "ms", d.Milliseconds(), "cumulative_ms", latencySum)
}

func recordComponentFailure(component string) {
	failMu.Lock()
	defer failMu.Unlock()
	failures[component]++
	slog.Info("lesson_component_failures_total", "component", component, "count", failures[component])
}
