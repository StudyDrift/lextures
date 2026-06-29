package telemetry

import (
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
)

// SentryConfig configures error/panic reporting (plan 17.7 FR-3, FR-4). When DSN
// is empty, Sentry is disabled and all helpers are no-ops.
type SentryConfig struct {
	// DSN is the project DSN (separate per environment; secrets manager in prod — FR-4).
	DSN string
	// Environment tags events (production/staging) so issues are filtered by env.
	Environment string
	// Release is the version/release identifier for source-mapped stack traces (FR-4).
	Release string
	// TracesSampleRate samples performance transactions (10% per FR-3).
	TracesSampleRate float64
}

// emailRe matches email addresses for scrubbing from free-text error messages.
var emailRe = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)

// sensitiveTagKeys and sensitiveExtraKeys are stripped from every event because
// they may carry PII / FERPA-protected fields (plan 17.7 NFR Privacy, 10.14).
var sensitiveKeys = map[string]struct{}{
	"email": {}, "user_email": {}, "parent_email": {},
	"first_name": {}, "last_name": {}, "full_name": {}, "display_name": {}, "name": {},
	"student_id": {}, "ssn": {}, "date_of_birth": {}, "dob": {},
	"password": {}, "authorization": {}, "access_token": {}, "refresh_token": {},
	"id_token": {}, "jwt_token": {}, "ip": {}, "ip_address": {},
}

// initSentry initialises the Sentry SDK. Returns a flush func (call on shutdown)
// and whether Sentry is enabled.
func initSentry(cfg SentryConfig) (func(time.Duration), bool, error) {
	if strings.TrimSpace(cfg.DSN) == "" {
		return func(time.Duration) {}, false, nil
	}
	rate := cfg.TracesSampleRate
	if rate <= 0 {
		rate = 0.1
	}
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.DSN,
		Environment:      cfg.Environment,
		Release:          cfg.Release,
		TracesSampleRate: rate,
		// SendDefaultPII MUST stay false; we additionally scrub in BeforeSend.
		SendDefaultPII: false,
		BeforeSend:     scrubEvent,
	})
	if err != nil {
		return func(time.Duration) {}, false, err
	}
	flush := func(d time.Duration) { sentry.Flush(d) }
	return flush, true, nil
}

// scrubEvent is the before_send hook that removes PII from an event before it
// leaves the process (plan 17.7 FR-3, NFR Privacy / FERPA, AC-3). It strips the
// user identity, request cookies/headers, and any sensitive tag/extra/context
// keys, and redacts email addresses from exception/message text.
func scrubEvent(event *sentry.Event, _ *sentry.EventHint) *sentry.Event {
	if event == nil {
		return nil
	}

	// Drop user identity entirely (id, email, ip, username, name).
	event.User = sentry.User{}

	// Scrub request: remove cookies, auth headers, query string, and body.
	if event.Request != nil {
		event.Request.Cookies = ""
		event.Request.QueryString = ""
		event.Request.Data = ""
		for k := range event.Request.Headers {
			if isSensitiveHeader(k) {
				delete(event.Request.Headers, k)
			}
		}
	}

	scrubStringMap(event.Tags)
	for _, ctx := range event.Contexts {
		scrubAnyMap(ctx)
	}

	event.Message = scrubText(event.Message)
	for i := range event.Exception {
		event.Exception[i].Value = scrubText(event.Exception[i].Value)
	}
	for i := range event.Breadcrumbs {
		if event.Breadcrumbs[i] != nil {
			event.Breadcrumbs[i].Message = scrubText(event.Breadcrumbs[i].Message)
			scrubAnyMap(event.Breadcrumbs[i].Data)
		}
	}
	return event
}

func isSensitiveHeader(k string) bool {
	switch strings.ToLower(strings.TrimSpace(k)) {
	case "authorization", "cookie", "set-cookie", "x-api-key", "proxy-authorization":
		return true
	}
	return false
}

func scrubStringMap(m map[string]string) {
	for k, v := range m {
		if _, ok := sensitiveKeys[strings.ToLower(k)]; ok {
			m[k] = "[redacted]"
			continue
		}
		m[k] = scrubText(v)
	}
}

func scrubAnyMap(m map[string]any) {
	for k, v := range m {
		if _, ok := sensitiveKeys[strings.ToLower(k)]; ok {
			m[k] = "[redacted]"
			continue
		}
		if s, ok := v.(string); ok {
			m[k] = scrubText(s)
		}
	}
}

// scrubText redacts email addresses embedded in free text.
func scrubText(s string) string {
	if s == "" {
		return s
	}
	return emailRe.ReplaceAllString(s, "[redacted-email]")
}

// SentryRecoverMiddleware recovers panics, reports them to Sentry, and re-panics
// so the standard chi Recoverer still returns 500 (plan 17.7 FR-3, AC-3). When
// Sentry is disabled it is a transparent pass-through.
func (t *Telemetry) SentryRecoverMiddleware(next http.Handler) http.Handler {
	if !t.sentryEnabled {
		return next
	}
	handler := sentryhttp.New(sentryhttp.Options{
		Repanic: true,
		Timeout: 2 * time.Second,
	})
	return handler.Handle(next)
}

// CaptureError reports a non-panic error to Sentry (no-op when disabled). The
// before_send hook scrubs any PII the error string may carry.
func (t *Telemetry) CaptureError(err error) {
	if !t.sentryEnabled || err == nil {
		return
	}
	sentry.CaptureException(err)
}
