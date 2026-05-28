package logging

import (
	"context"
	"log/slog"
)

// redactHandler wraps an slog.Handler and redacts PII attributes before emission.
type redactHandler struct {
	inner    slog.Handler
	redactor *Redactor
}

// NewRedactHandler wraps inner with PII redaction (plan 10.14 FR-1).
func NewRedactHandler(inner slog.Handler, redactor *Redactor) slog.Handler {
	if inner == nil {
		inner = slog.Default().Handler()
	}
	if redactor == nil {
		redactor = NewRedactor(RedactorConfig{Registry: NewFieldRegistry()})
	}
	return &redactHandler{inner: inner, redactor: redactor}
}

func (h *redactHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *redactHandler) Handle(ctx context.Context, r slog.Record) (err error) {
	defer func() {
		if recover() != nil {
			// Fail-safe: drop event on panic (plan 10.14 NFR Reliability).
			err = nil
		}
	}()
	nr := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	r.Attrs(func(a slog.Attr) bool {
		nr.AddAttrs(h.redactor.RedactAttr(a))
		return true
	})
	return h.inner.Handle(ctx, nr)
}

func (h *redactHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	out := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		out[i] = h.redactor.RedactAttr(a)
	}
	return &redactHandler{inner: h.inner.WithAttrs(out), redactor: h.redactor}
}

func (h *redactHandler) WithGroup(name string) slog.Handler {
	return &redactHandler{inner: h.inner.WithGroup(name), redactor: h.redactor}
}

// Ensure redactHandler implements slog.Handler.
var _ slog.Handler = (*redactHandler)(nil)
