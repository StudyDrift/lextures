package telemetry

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResourceCollector_AllSources(t *testing.T) {
	m := NewMetrics()
	srcs := Sources{
		DBPool: func() DBPoolSnapshot {
			return DBPoolSnapshot{Total: 8, Acquired: 4, Idle: 4, Max: 10}
		},
		Redis: func() RedisPoolSnapshot {
			return RedisPoolSnapshot{Total: 5, Idle: 3, Hits: 100, Misses: 7, Timeouts: 1}
		},
		JobQueue: func() (JobQueueSnapshot, bool) {
			return JobQueueSnapshot{
				Pending: 3, Running: 1, Failed: 2, DeadLetters: 5, Depth: 6,
				ByType: map[string]int{"email": 4, "report": 2},
			}, true
		},
	}
	if err := m.RegisterCollector(newResourceCollector(srcs)); err != nil {
		t.Fatalf("register: %v", err)
	}
	body := scrape(t, m)

	for _, want := range []string{
		"lextures_db_pool_total_connections 8",
		"lextures_db_pool_acquired_connections 4",
		"lextures_db_pool_max_connections 10",
		"lextures_db_pool_utilization_ratio 0.4",
		"lextures_redis_pool_total_connections 5",
		"lextures_redis_pool_hits_total 100",
		"lextures_job_queue_depth 6",
		"lextures_job_queue_dead_letters 5",
		`lextures_job_queue_jobs{status="pending"} 3`,
		`lextures_job_queue_depth_by_type{job_type="email"} 4`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing series %q", want)
		}
	}
}

func TestResourceCollector_NilSourcesOmitted(t *testing.T) {
	m := NewMetrics()
	// Redis disabled (nil), no job queue; only DB present.
	if err := m.RegisterCollector(newResourceCollector(Sources{
		DBPool: func() DBPoolSnapshot { return DBPoolSnapshot{Max: 0, Acquired: 1} },
	})); err != nil {
		t.Fatalf("register: %v", err)
	}
	body := scrape(t, m)
	if strings.Contains(body, "redis_pool_total_connections") {
		t.Error("redis series should be omitted when source is nil")
	}
	if strings.Contains(body, "job_queue_depth") {
		t.Error("job queue series should be omitted when source is nil")
	}
	// Utilization with Max=0 must not divide by zero; it should report 0.
	if !strings.Contains(body, "lextures_db_pool_utilization_ratio 0") {
		t.Error("utilization with zero max should be 0")
	}
}

func TestResourceCollector_JobQueueUnavailable(t *testing.T) {
	m := NewMetrics()
	if err := m.RegisterCollector(newResourceCollector(Sources{
		JobQueue: func() (JobQueueSnapshot, bool) { return JobQueueSnapshot{}, false },
	})); err != nil {
		t.Fatalf("register: %v", err)
	}
	if strings.Contains(scrape(t, m), "job_queue_depth") {
		t.Error("unavailable job queue (ok=false) must not emit series")
	}
}

func scrape(t *testing.T, m *Metrics) string {
	t.Helper()
	rr := httptest.NewRecorder()
	m.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("scrape status %d", rr.Code)
	}
	return rr.Body.String()
}
