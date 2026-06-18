package studyreminders

import (
	"sync/atomic"
)

var studyRemindersSentTotal int64

// RecordReminderSent increments observability counters for reminder delivery.
func RecordReminderSent(channel string) {
	atomic.AddInt64(&studyRemindersSentTotal, 1)
	_ = channel
}

// RemindersSentTotal returns the total reminders sent (for tests/metrics).
func RemindersSentTotal() int64 {
	return atomic.LoadInt64(&studyRemindersSentTotal)
}
