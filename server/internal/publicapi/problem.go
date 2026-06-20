package publicapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

const errorTypeBase = "https://lextures.io/errors/"

// Problem is an RFC 7807 problem details document.
type Problem struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
}

// WriteProblem writes application/problem+json.
func WriteProblem(w http.ResponseWriter, p Problem) {
	if p.Type != "" && !strings.HasPrefix(p.Type, "http") {
		p.Type = errorTypeBase + p.Type
	}
	if p.Title == "" {
		p.Title = http.StatusText(p.Status)
	}
	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p)
}

// Unauthorized writes a 401 problem response.
func Unauthorized(w http.ResponseWriter, instance, detail string) {
	if detail == "" {
		detail = "A valid Bearer token is required."
	}
	WriteProblem(w, Problem{
		Type:     "unauthorized",
		Title:    "Unauthorized",
		Status:   http.StatusUnauthorized,
		Detail:   detail,
		Instance: instance,
	})
}

// ForbiddenScope writes a 403 problem for a missing scope.
func ForbiddenScope(w http.ResponseWriter, instance, scope string) {
	WriteProblem(w, Problem{
		Type:     "forbidden",
		Title:    "Forbidden",
		Status:   http.StatusForbidden,
		Detail:   "Missing required scope: " + scope,
		Instance: instance,
	})
}

// RateLimited writes 429 with Retry-After.
func RateLimited(w http.ResponseWriter, instance string, retryAfterSec int) {
	if retryAfterSec < 1 {
		retryAfterSec = 60
	}
	w.Header().Set("Retry-After", fmtRetryAfter(retryAfterSec))
	WriteProblem(w, Problem{
		Type:     "rate-limited",
		Title:    "Too Many Requests",
		Status:   http.StatusTooManyRequests,
		Detail:   "Token quota exceeded.",
		Instance: instance,
	})
}

func fmtRetryAfter(sec int) string {
	return strconv.Itoa(sec)
}
