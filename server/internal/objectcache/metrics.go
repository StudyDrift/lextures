package objectcache

import "log/slog"

// ResourceType labels cache metrics (plan 17.5 observability).
type ResourceType string

const (
	ResourceCourseStructure   ResourceType = "course_structure"
	ResourceCourseEnrollments ResourceType = "course_enrollments"
	ResourceCatalogPage       ResourceType = "catalog_page"
	ResourceUserCalendar      ResourceType = "user_calendar"
)

// RecordHit logs a cache hit for observability (Prometheus-compatible structured field names).
func RecordHit(resource ResourceType) {
	slog.Info("cache_hits_total", "resource_type", string(resource))
}

// RecordMiss logs a cache miss for observability.
func RecordMiss(resource ResourceType) {
	slog.Info("cache_misses_total", "resource_type", string(resource))
}

// RecordStaleHit logs a stale-while-revalidate serve.
func RecordStaleHit(resource ResourceType) {
	slog.Info("cache_stale_hits_total", "resource_type", string(resource))
}
