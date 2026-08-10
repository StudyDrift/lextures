package apierr

import (
	"encoding/json"
	"net/http"
)

// ErrorValidationFailed is the UX.6 machine code for field-addressable 422 responses.
// Distinct from the legacy nested {error:{code,message}} envelope.
const ErrorValidationFailed = "validation_failed"

// FieldViolation is one field-level validation failure (UX.6 §9).
// Path uses dot/bracket notation (e.g. "sections[0].code").
// Code is a stable machine string (e.g. "already_taken"); Message is fallback human text.
// Params are optional ICU interpolation values for the client (never include secrets).
type FieldViolation struct {
	Path    string         `json:"path"`
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Params  map[string]any `json:"params,omitempty"`
}

// ValidationFailedBody is the UX.6 HTTP 422 envelope.
// Adopted incrementally; clients tolerate the legacy shape and fall back to a banner.
type ValidationFailedBody struct {
	Error   string           `json:"error"`
	Message string           `json:"message"`
	Fields  []FieldViolation `json:"fields"`
}

// WriteValidationFailed writes HTTP 422 with the field-addressable validation envelope.
// message should be a short human summary (i18n key preferred on the client).
// fields must not leak existence of records the caller cannot see.
func WriteValidationFailed(w http.ResponseWriter, message string, fields []FieldViolation) {
	if message == "" {
		message = "Validation failed"
	}
	if fields == nil {
		fields = []FieldViolation{}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnprocessableEntity)
	_ = json.NewEncoder(w).Encode(ValidationFailedBody{
		Error:   ErrorValidationFailed,
		Message: message,
		Fields:  fields,
	})
}
