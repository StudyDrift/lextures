package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/redisclient"
	"github.com/lextures/lextures/server/internal/telemetry"
)

const (
	postgresReadyTimeout = time.Second
	redisReadyTimeout    = 500 * time.Millisecond
)

// HealthProbe runs dependency checks against a dedicated DB pool so probes do
// not share the main request connection pool (plan 17.8).
type HealthProbe struct {
	dbPool  *pgxpool.Pool
	redis   *redisclient.Client
	metrics *telemetry.Metrics
}

// NewHealthProbe wires a probe against the given dedicated DB pool and optional
// shared Redis client. metrics may be nil.
func NewHealthProbe(dbPool *pgxpool.Pool, redis *redisclient.Client, metrics *telemetry.Metrics) *HealthProbe {
	return &HealthProbe{dbPool: dbPool, redis: redis, metrics: metrics}
}

// ReadyResponse is the public JSON body for GET /health/ready (plan 17.8 FR-3).
type ReadyResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

// DetailedCheck is one row in GET /health/detailed (plan 17.8 FR-4).
type DetailedCheck struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	LatencyMs int64  `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
}

// DetailedResponse is the authenticated detailed health payload.
type DetailedResponse struct {
	Checks []DetailedCheck `json:"checks"`
}

func handleLive(metrics *telemetry.Metrics) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		recordHealthMetric(metrics, "live", "200")
		writeHealthJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

func handleHealthAlias(metrics *telemetry.Metrics) http.HandlerFunc {
	return handleLive(metrics)
}

func handleReady(probe *HealthProbe) http.HandlerFunc {
	if probe == nil {
		probe = NewHealthProbe(nil, nil, nil)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		resp, code := probe.Ready(r.Context())
		statusLabel := "200"
		if code != http.StatusOK {
			statusLabel = "503"
		}
		recordHealthMetric(probe.metrics, "ready", statusLabel)
		writeHealthJSON(w, code, resp)
	}
}

func (d Deps) handleHealthDetailed(probe *HealthProbe) http.HandlerFunc {
	if probe == nil {
		probe = NewHealthProbe(nil, nil, nil)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		checks := probe.Detailed(r.Context())
		recordHealthMetric(probe.metrics, "detailed", "200")
		writeHealthJSON(w, http.StatusOK, DetailedResponse{Checks: checks})
	}
}

// Ready evaluates Postgres and Redis and returns the public readiness payload
// without leaking internal error details (plan 17.8 FR-2 / AC-2).
func (p *HealthProbe) Ready(ctx context.Context) (ReadyResponse, int) {
	pgOK, _ := p.checkPostgres(ctx)
	redisOK, redisConfigured := p.checkRedis(ctx)

	checks := map[string]string{
		"postgres": checkLabel(pgOK),
		"redis":    checkLabel(redisOK || !redisConfigured),
	}

	status := "ready"
	code := http.StatusOK
	if !pgOK || (redisConfigured && !redisOK) {
		status = "unhealthy"
		code = http.StatusServiceUnavailable
	}

	return ReadyResponse{Status: status, Checks: checks}, code
}

// Detailed returns per-component latency and safe error summaries for admins.
func (p *HealthProbe) Detailed(ctx context.Context) []DetailedCheck {
	pgOK, pgLatency, pgErr := p.checkPostgresDetailed(ctx)
	redisOK, redisConfigured, redisLatency, redisErr := p.checkRedisDetailed(ctx)

	checks := []DetailedCheck{
		{
			Name:      "postgres",
			Status:    checkLabel(pgOK),
			LatencyMs: pgLatency.Milliseconds(),
			Error:     safeErrorSummary(pgErr),
		},
	}
	if redisConfigured {
		checks = append(checks, DetailedCheck{
			Name:      "redis",
			Status:    checkLabel(redisOK),
			LatencyMs: redisLatency.Milliseconds(),
			Error:     safeErrorSummary(redisErr),
		})
	} else {
		checks = append(checks, DetailedCheck{
			Name:      "redis",
			Status:    "ok",
			LatencyMs: 0,
			Error:     "",
		})
	}
	return checks
}

func (p *HealthProbe) checkPostgres(ctx context.Context) (bool, error) {
	ok, _, err := p.checkPostgresDetailed(ctx)
	return ok, err
}

func (p *HealthProbe) checkPostgresDetailed(ctx context.Context) (bool, time.Duration, error) {
	if p.dbPool == nil {
		return false, 0, errNoDBPool
	}
	ctx, cancel := context.WithTimeout(ctx, postgresReadyTimeout)
	defer cancel()
	start := time.Now()
	var one int
	err := p.dbPool.QueryRow(ctx, "SELECT 1").Scan(&one)
	return err == nil && one == 1, time.Since(start), err
}

func (p *HealthProbe) checkRedis(ctx context.Context) (ok bool, configured bool) {
	ok, configured, _, _ = p.checkRedisDetailed(ctx)
	return ok, configured
}

func (p *HealthProbe) checkRedisDetailed(ctx context.Context) (ok bool, configured bool, latency time.Duration, err error) {
	if p.redis == nil {
		return true, false, 0, nil
	}
	ctx, cancel := context.WithTimeout(ctx, redisReadyTimeout)
	defer cancel()
	start := time.Now()
	err = p.redis.Ping(ctx)
	latency = time.Since(start)
	return err == nil, true, latency, err
}

func checkLabel(ok bool) string {
	if ok {
		return "ok"
	}
	return "fail"
}

func recordHealthMetric(metrics *telemetry.Metrics, endpoint, status string) {
	if metrics != nil {
		metrics.IncHealthCheck(endpoint, status)
	}
}

func writeHealthJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

// safeErrorSummary returns a generic operator-safe message without connection
// strings, passwords, or stack traces (plan 17.8 FR-2 / AC-2).
func safeErrorSummary(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "password"),
		strings.Contains(msg, "postgres://"),
		strings.Contains(msg, "postgresql://"),
		strings.Contains(msg, "rediss://"),
		strings.Contains(msg, "redis://"):
		return "connection failed"
	case strings.Contains(msg, "connection refused"):
		return "connection refused"
	case strings.Contains(msg, "not configured"):
		return "not configured"
	default:
		return "unavailable"
	}
}

var errNoDBPool = errors.New("database pool is not configured")
