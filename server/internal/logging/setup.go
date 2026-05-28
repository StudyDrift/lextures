package logging

import (
	"log/slog"
	"os"
	"strings"
)

// Settings configures process-wide slog redaction.
type Settings struct {
	DisableRedaction bool
	ExtraFields      []string
	HMACSecret       []byte
	AppEnv           string
}

// Configure installs the default slog logger with a PII redacting handler.
func Configure(s Settings) {
	registry := NewFieldRegistry(s.ExtraFields...)
	redactor := NewRedactor(RedactorConfig{
		Enabled:    !s.DisableRedaction,
		Disabled:   s.DisableRedaction,
		Registry:   registry,
		HMACSecret: s.HMACSecret,
	})
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	inner := slog.NewJSONHandler(os.Stderr, opts)
	if lvl := strings.TrimSpace(os.Getenv("LOG_LEVEL")); lvl != "" {
		switch strings.ToLower(lvl) {
		case "debug":
			opts.Level = slog.LevelDebug
		case "warn", "warning":
			opts.Level = slog.LevelWarn
		case "error":
			opts.Level = slog.LevelError
		}
		inner = slog.NewJSONHandler(os.Stderr, opts)
	}
	h := NewRedactHandler(inner, redactor)
	slog.SetDefault(slog.New(h))

	env := strings.ToLower(strings.TrimSpace(s.AppEnv))
	if s.DisableRedaction && env != "" && env != "local" && env != "development" && env != "dev" {
		slog.Warn("PII redaction is disabled; operational logs may contain personal data", "app_env", s.AppEnv)
	}
}
