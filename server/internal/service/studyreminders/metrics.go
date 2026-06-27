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
