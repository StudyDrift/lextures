package logging

import "sync/atomic"

// TranscriptNotificationMetrics tracks transcript_notification_sent_total{event,channel} (T10).
type TranscriptNotificationMetrics struct {
	email atomic.Uint64
	push  atomic.Uint64
	inApp atomic.Uint64
}

// GlobalTranscriptNotificationMetrics is incremented by transcriptnotify sends.
var GlobalTranscriptNotificationMetrics = &TranscriptNotificationMetrics{}

func (m *TranscriptNotificationMetrics) Inc(channel string) {
	switch channel {
	case "email":
		m.email.Add(1)
	case "push":
		m.push.Add(1)
	case "in_app":
		m.inApp.Add(1)
	}
}

func (m *TranscriptNotificationMetrics) Snapshot() (email, push, inApp uint64) {
	return m.email.Load(), m.push.Load(), m.inApp.Load()
}
