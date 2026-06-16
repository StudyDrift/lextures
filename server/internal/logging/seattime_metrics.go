package logging

import "sync/atomic"

// SeatTimeMetrics tracks seat-time aggregation counters (plan 14.17).
type SeatTimeMetrics struct {
	MinutesTotal atomic.Int64
	Heartbeats   atomic.Int64
	Anomalies    atomic.Int64
	Awards       atomic.Int64
}

// GlobalSeatTimeMetrics is incremented by seat-time HTTP handlers and background flush.
var GlobalSeatTimeMetrics = &SeatTimeMetrics{}

// AddMinutes records aggregated seat-time minutes for a course.
func (m *SeatTimeMetrics) AddMinutes(n int64) {
	if n > 0 {
		m.MinutesTotal.Add(n)
	}
}

// IncHeartbeats records a processed heartbeat.
func (m *SeatTimeMetrics) IncHeartbeats() {
	m.Heartbeats.Add(1)
}

// IncAnomalies records a session flagged for anomalous heartbeat patterns.
func (m *SeatTimeMetrics) IncAnomalies() {
	m.Anomalies.Add(1)
}

// IncAwards records a newly issued CEU award.
func (m *SeatTimeMetrics) IncAwards() {
	m.Awards.Add(1)
}
