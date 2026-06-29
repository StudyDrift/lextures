package telemetry

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

// TestSentrySlogHandler_PassesThroughToInner verifies the bridge still emits the
// record to the inner JSON handler (it only taps ERROR records for Sentry; it
// must never swallow log output).
func TestSentrySlogHandler_PassesThroughToInner(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	h := newSentrySlogHandler(inner)
	logger := slog.New(h)

	logger.Info("info line", "k", "v")
	logger.Error("error line", "course_id", "c1")

	out := buf.String()
	if !strings.Contains(out, "info line") || !strings.Contains(out, "error line") {
		t.Errorf("inner handler did not receive both records:\n%s", out)
	}
}

func TestSentrySlogHandler_EnabledDelegates(t *testing.T) {
	inner := slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelWarn})
	h := newSentrySlogHandler(inner)
	if h.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("Enabled should delegate to inner (Info below Warn threshold)")
	}
	if !h.Enabled(context.Background(), slog.LevelError) {
		t.Error("Enabled should be true for Error")
	}
}

func TestSentrySlogHandler_WithAttrsAndGroup(t *testing.T) {
	inner := slog.NewJSONHandler(&bytes.Buffer{}, nil)
	h := newSentrySlogHandler(inner)
	if _, ok := h.WithAttrs([]slog.Attr{slog.String("a", "b")}).(*sentrySlogHandler); !ok {
		t.Error("WithAttrs should preserve handler type")
	}
	if _, ok := h.WithGroup("g").(*sentrySlogHandler); !ok {
		t.Error("WithGroup should preserve handler type")
	}
}
