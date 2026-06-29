package ratelimit

import "log/slog"

// LimitType labels which tier rejected a request (plan 17.6 observability).
type LimitType string

const (
	LimitTypeAuth   LimitType = "auth"
	LimitTypeGlobal LimitType = "global"
	LimitTypeToken  LimitType = "token"
)

// RecordExceeded emits rate_limit_exceeded_total{endpoint_group,limit_type}
// (plan 17.6 NFR observability; alert on spikes). Field names are
// Prometheus-compatible for the structured-log → metrics pipeline (17.7).
func RecordExceeded(endpointGroup string, limitType LimitType) {
	slog.Info("rate_limit_exceeded_total",
		"endpoint_group", endpointGroup,
		"limit_type", string(limitType),
	)
}

// RecordRedisMiss emits rate_limit_redis_miss_total when a check failed open
// because Redis was unavailable (plan 17.6 AC-4).
func RecordRedisMiss(limitType LimitType) {
	slog.Warn("rate_limit_redis_miss_total", "limit_type", string(limitType))
}
