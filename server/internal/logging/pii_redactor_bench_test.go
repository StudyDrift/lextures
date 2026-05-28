package logging

import (
	"bytes"
	"log/slog"
	"testing"
)

func BenchmarkRedactHandler_10000Events(b *testing.B) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, nil)
	h := NewRedactHandler(inner, NewRedactor(RedactorConfig{
		Registry:   NewFieldRegistry(),
		HMACSecret: []byte("benchmark-hmac-secret-key-32b"),
	}))
	log := slog.New(h)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.Info("event",
			"email", "user@example.com",
			"user_id", "550e8400-e29b-41d4-a716-446655440000",
			"status", 200,
			"request_id", "req-123",
		)
	}
}
