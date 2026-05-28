// Package logging provides structured log PII redaction (plan 10.14).
package logging

import (
	"strings"
)

// DefaultPIIFieldNames are structured log attribute keys redacted before emission.
var DefaultPIIFieldNames = []string{
	"email",
	"user_email",
	"parent_email",
	"first_name",
	"last_name",
	"full_name",
	"display_name",
	"date_of_birth",
	"dob",
	"ip_address",
	"ip",
	"client_ip",
	"remote_addr",
	"jwt_token",
	"access_token",
	"refresh_token",
	"id_token",
	"authorization",
	"password",
	"student_response",
	"assignment_text",
	"ssn",
}

// AlwaysRedactFieldNames are redacted even when DISABLE_PII_REDACTION=1 (plan 10.14 NFR Security).
var AlwaysRedactFieldNames = []string{
	"jwt_token",
	"access_token",
	"refresh_token",
	"id_token",
	"authorization",
	"password",
}

// HashFieldNames use stable HMAC-SHA256 instead of [REDACTED:name] (correlation without exposure).
var HashFieldNames = []string{
	"user_id",
	"actor_id",
	"target_id",
	"org_id",
	"enrollment_id",
}

// FieldRegistry holds normalized PII field names for redaction.
type FieldRegistry struct {
	redact      map[string]struct{}
	always      map[string]struct{}
	hash        map[string]struct{}
	orderedList []string
}

// NewFieldRegistry builds a registry from defaults plus extra field names.
func NewFieldRegistry(extra ...string) *FieldRegistry {
	r := &FieldRegistry{
		redact: make(map[string]struct{}),
		always: make(map[string]struct{}),
		hash:   make(map[string]struct{}),
	}
	seen := make(map[string]struct{})
	add := func(name string, bucket map[string]struct{}) {
		n := normalizeFieldName(name)
		if n == "" {
			return
		}
		bucket[n] = struct{}{}
		if _, ok := seen[n]; !ok {
			seen[n] = struct{}{}
			r.orderedList = append(r.orderedList, n)
		}
	}
	for _, f := range DefaultPIIFieldNames {
		add(f, r.redact)
	}
	for _, f := range AlwaysRedactFieldNames {
		add(f, r.always)
		add(f, r.redact)
	}
	for _, f := range HashFieldNames {
		add(f, r.hash)
		add(f, r.redact)
	}
	for _, f := range extra {
		add(f, r.redact)
	}
	return r
}

// Names returns the active field names (defaults + extras).
func (r *FieldRegistry) Names() []string {
	out := make([]string, len(r.orderedList))
	copy(out, r.orderedList)
	return out
}

func (r *FieldRegistry) shouldRedact(name string, disabled bool) bool {
	n := normalizeFieldName(name)
	if n == "" {
		return false
	}
	if _, ok := r.always[n]; ok {
		return true
	}
	if disabled {
		return false
	}
	_, ok := r.redact[n]
	return ok
}

func (r *FieldRegistry) useHash(name string) bool {
	_, ok := r.hash[normalizeFieldName(name)]
	return ok
}

func normalizeFieldName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
