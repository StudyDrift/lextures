package telemetry

import (
	"context"
	"log/slog"

	"github.com/getsentry/sentry-go"
)

// sentrySlogHandler forwards ERROR-level slog records to Sentry (plan 17.7 FR-3:
// "log/slog integration — ERROR-level events sent to Sentry"). It is chained
// downstream of the PII-redacting handler, so records it receives are already
// scrubbed; the Sentry before_send hook is a second line of defence. It never
// emits output itself — it taps records and hands off to the inner handler.
type sentrySlogHandler struct {
	inner slog.Handler
}

// newSentrySlogHandler wraps inner so ERROR records are also captured by Sentry.
func newSentrySlogHandler(inner slog.Handler) slog.Handler {
	return &sentrySlogHandler{inner: inner}
}

func (h *sentrySlogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *sentrySlogHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level >= slog.LevelError {
		h.capture(ctx, r)
	}
	return h.inner.Handle(ctx, r)
}

func (h *sentrySlogHandler) capture(ctx context.Context, r slog.Record) {
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub()
	}
	hub.WithScope(func(scope *sentry.Scope) {
		scope.SetLevel(sentry.LevelError)
		r.Attrs(func(a slog.Attr) bool {
			// Attrs are already PII-redacted upstream; record them as tags for
			// triage. before_send still strips known-sensitive keys.
			scope.SetTag(a.Key, a.Value.String())
			return true
		})
		hub.CaptureMessage(r.Message)
	})
}

func (h *sentrySlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &sentrySlogHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *sentrySlogHandler) WithGroup(name string) slog.Handler {
	return &sentrySlogHandler{inner: h.inner.WithGroup(name)}
}

var _ slog.Handler = (*sentrySlogHandler)(nil)
