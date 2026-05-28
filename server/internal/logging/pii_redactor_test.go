package logging

import (
	"log/slog"
	"strings"
	"testing"
)

func TestRedactor_RedactsRegisteredField(t *testing.T) {
	t.Parallel()
	r := NewRedactor(RedactorConfig{Registry: NewFieldRegistry()})
	got, ok := r.RedactValue("email", "student@school.edu")
	if !ok {
		t.Fatal("expected redaction")
	}
	if got != "[REDACTED:email]" {
		t.Fatalf("got %v", got)
	}
}

func TestRedactor_PassThroughUnknownField(t *testing.T) {
	t.Parallel()
	r := NewRedactor(RedactorConfig{Registry: NewFieldRegistry()})
	got, ok := r.RedactValue("request_id", "abc-123")
	if ok {
		t.Fatalf("unexpected redaction: %v", got)
	}
	if got != "abc-123" {
		t.Fatalf("got %v", got)
	}
}

func TestRedactor_AlwaysRedactsJWTWhenDisabled(t *testing.T) {
	t.Parallel()
	r := NewRedactor(RedactorConfig{Disabled: true, Registry: NewFieldRegistry()})
	got, ok := r.RedactValue("access_token", "eyJhbGciOiJIUzI1NiJ9.payload.sig")
	if !ok {
		t.Fatal("expected redaction")
	}
	if !strings.HasPrefix(got.(string), "[REDACTED:access_token]") {
		t.Fatalf("got %v", got)
	}
}

func TestRedactor_ExtraRegistryField(t *testing.T) {
	t.Parallel()
	r := NewRedactor(RedactorConfig{Registry: NewFieldRegistry("ssn")})
	got, ok := r.RedactValue("ssn", "123-45-6789")
	if !ok || got != "[REDACTED:ssn]" {
		t.Fatalf("got %v ok=%v", got, ok)
	}
}

func TestRedactor_HashUserID(t *testing.T) {
	t.Parallel()
	r := NewRedactor(RedactorConfig{
		Registry:   NewFieldRegistry(),
		HMACSecret: []byte("test-secret-key-32-bytes-long!!"),
	})
	a, _ := r.RedactValue("user_id", "550e8400-e29b-41d4-a716-446655440000")
	b, _ := r.RedactValue("user_id", "550e8400-e29b-41d4-a716-446655440000")
	if a != b {
		t.Fatalf("hash not stable: %v vs %v", a, b)
	}
	if !strings.HasPrefix(a.(string), "hmac:") {
		t.Fatalf("expected hmac prefix, got %v", a)
	}
}

func TestRedactHandler_EmitsRedactedJSON(t *testing.T) {
	t.Parallel()
	GlobalRedactionMetrics.Reset()
	var buf strings.Builder
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	h := NewRedactHandler(inner, NewRedactor(RedactorConfig{
		Registry:   NewFieldRegistry(),
		HMACSecret: []byte("secret"),
	}))
	log := slog.New(h)
	log.Info("login", "email", "user@example.com", "status", 200)
	out := buf.String()
	if strings.Contains(out, "user@example.com") {
		t.Fatalf("plaintext email in log: %s", out)
	}
	if !strings.Contains(out, "[REDACTED:email]") {
		t.Fatalf("missing redaction marker: %s", out)
	}
	if GlobalRedactionMetrics.Snapshot()["email"] == 0 {
		t.Fatal("expected email redaction metric")
	}
}
