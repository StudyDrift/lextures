// Package publicapi implements the versioned public REST API surface (plan 16.1).
package publicapi

import (
	"encoding/json"
	"net/http"
	"strings"
)

const problemBaseType = "https://lextures.io/errors/"

// Problem is an RFC 7807 Problem Details object.
type Problem struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
}

// WriteProblem writes application/problem+json.
func WriteProblem(w http.ResponseWriter, p Problem) {
	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p)
}

// WriteUnauthorized returns 401 for missing/invalid bearer tokens.
func WriteUnauthorized(w http.ResponseWriter, instance string) {
	WriteProblem(w, Problem{
		Type:     problemBaseType + "unauthorized",
		Title:    "Unauthorized",
		Status:   http.StatusUnauthorized,
		Detail:   "A valid Bearer token is required.",
		Instance: instance,
	})
}

// WriteForbidden returns 403 for insufficient scope.
func WriteForbidden(w http.ResponseWriter, instance, detail string) {
	WriteProblem(w, Problem{
		Type:     problemBaseType + "forbidden",
		Title:    "Forbidden",
		Status:   http.StatusForbidden,
		Detail:   detail,
		Instance: instance,
	})
}

// WriteNotFound returns 404.
func WriteNotFound(w http.ResponseWriter, instance string) {
	WriteProblem(w, Problem{
		Type:     problemBaseType + "not-found",
		Title:    "Not Found",
		Status:   http.StatusNotFound,
		Detail:   "The requested resource was not found.",
		Instance: instance,
	})
}

// WriteRateLimited returns 429 with Retry-After.
func WriteRateLimited(w http.ResponseWriter, instance string, retryAfterSec int) {
	w.Header().Set("Retry-After", itoa(retryAfterSec))
	WriteProblem(w, Problem{
		Type:     problemBaseType + "rate-limited",
		Title:    "Too Many Requests",
		Status:   http.StatusTooManyRequests,
		Detail:   "Token quota exceeded.",
		Instance: instance,
	})
}

// WriteServiceUnavailable returns 503 when the public API feature flag is off.
func WriteServiceUnavailable(w http.ResponseWriter, instance string) {
	w.Header().Set("Retry-After", "300")
	WriteProblem(w, Problem{
		Type:     problemBaseType + "service-unavailable",
		Title:    "Service Unavailable",
		Status:   http.StatusServiceUnavailable,
		Detail:   "The public API is not enabled on this deployment.",
		Instance: instance,
	})
}

// WriteBadRequest returns 400 for malformed query parameters.
func WriteBadRequest(w http.ResponseWriter, instance, detail string) {
	WriteProblem(w, Problem{
		Type:     problemBaseType + "bad-request",
		Title:    "Bad Request",
		Status:   http.StatusBadRequest,
		Detail:   detail,
		Instance: instance,
	})
}

// WriteInternal returns 500.
func WriteInternal(w http.ResponseWriter, instance string) {
	WriteProblem(w, Problem{
		Type:     problemBaseType + "internal",
		Title:    "Internal Server Error",
		Status:   http.StatusInternalServerError,
		Detail:   "An unexpected error occurred.",
		Instance: instance,
	})
}

func itoa(n int) string {
	if n <= 0 {
		return "1"
	}
	return strings.TrimSpace(strings.ReplaceAll(jsonNumber(n), `"`, ""))
}

func jsonNumber(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}
