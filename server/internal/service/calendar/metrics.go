package calendar

import "log/slog"

// RecordFeedRequest logs a calendar feed request for observability (plan 16.5).
func RecordFeedRequest(feedType, cacheStatus string) {
	slog.Info("calendar_feed_requests_total", "type", feedType, "cache", cacheStatus)
}