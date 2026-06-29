package telemetry

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/getsentry/sentry-go"
)

func TestScrubEvent_RemovesPII(t *testing.T) {
	event := &sentry.Event{
		Message: "failed to grade submission for alice@school.edu",
		User: sentry.User{
			ID:        "user-123",
			Email:     "alice@school.edu",
			Name:      "Alice Student",
			IPAddress: "203.0.113.7",
		},
		Tags: map[string]string{
			"email":     "bob@school.edu",
			"course_id": "course-9",
			"note":      "contact carol@school.edu",
		},
		Request: &sentry.Request{
			Cookies:     "session=secret",
			QueryString: "token=abc",
			Data:        `{"password":"hunter2"}`,
			Headers: map[string]string{
				"Authorization": "Bearer xyz",
				"User-Agent":    "test",
			},
		},
		Exception: []sentry.Exception{
			{Value: "panic processing dave@school.edu record"},
		},
		Contexts: map[string]sentry.Context{
			"extra": {"first_name": "Eve", "trace_id": "keep-me"},
		},
	}

	out := scrubEvent(event, nil)

	// User identity fully cleared (plan 17.7 AC-3 / FERPA).
	if out.User.Email != "" || out.User.ID != "" || out.User.Name != "" || out.User.IPAddress != "" {
		t.Errorf("user identity not cleared: %+v", out.User)
	}
	// Sensitive tag keys redacted, non-sensitive kept, emails in values scrubbed.
	if out.Tags["email"] != "[redacted]" {
		t.Errorf("email tag = %q", out.Tags["email"])
	}
	if out.Tags["course_id"] != "course-9" {
		t.Errorf("non-PII tag dropped: %q", out.Tags["course_id"])
	}
	if strings.Contains(out.Tags["note"], "carol@school.edu") {
		t.Errorf("email not scrubbed from tag value: %q", out.Tags["note"])
	}
	// Request scrubbed.
	if out.Request.Cookies != "" || out.Request.QueryString != "" || out.Request.Data != "" {
		t.Error("request cookies/query/data not scrubbed")
	}
	if _, ok := out.Request.Headers["Authorization"]; ok {
		t.Error("authorization header not removed")
	}
	if out.Request.Headers["User-Agent"] != "test" {
		t.Error("non-sensitive header dropped")
	}
	// Message + exception emails scrubbed.
	if strings.Contains(out.Message, "alice@school.edu") {
		t.Errorf("message email not scrubbed: %q", out.Message)
	}
	if strings.Contains(out.Exception[0].Value, "dave@school.edu") {
		t.Errorf("exception email not scrubbed: %q", out.Exception[0].Value)
	}
	// Context sensitive key redacted, non-sensitive preserved.
	if out.Contexts["extra"]["first_name"] != "[redacted]" {
		t.Errorf("context first_name = %v", out.Contexts["extra"]["first_name"])
	}
	if out.Contexts["extra"]["trace_id"] != "keep-me" {
		t.Errorf("context trace_id dropped: %v", out.Contexts["extra"]["trace_id"])
	}
}

func TestScrubEvent_Nil(t *testing.T) {
	if scrubEvent(nil, nil) != nil {
		t.Error("nil event should return nil")
	}
}

func TestScrubText(t *testing.T) {
	in := "user a.b+1@x.co and c@y.io failed"
	out := scrubText(in)
	if strings.Contains(out, "@x.co") || strings.Contains(out, "@y.io") {
		t.Errorf("emails not scrubbed: %q", out)
	}
	if !strings.Contains(out, "[redacted-email]") {
		t.Errorf("expected redaction marker in %q", out)
	}
	if scrubText("") != "" {
		t.Error("empty string should stay empty")
	}
}

func TestInitSentry_DisabledWithoutDSN(t *testing.T) {
	flush, enabled, err := initSentry(SentryConfig{})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if enabled {
		t.Error("sentry should be disabled without a DSN")
	}
	flush(0) // no-op must not panic
}

func TestTelemetry_SentryRecoverDisabledPassthrough(t *testing.T) {
	tel := &Telemetry{sentryEnabled: false}
	called := false
	h := tel.SentryRecoverMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	if !called {
		t.Error("disabled recover middleware must pass through")
	}
	// CaptureError on disabled telemetry is a no-op (no panic).
	tel.CaptureError(errors.New("x"))
}
