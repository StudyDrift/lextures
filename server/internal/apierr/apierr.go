// Package apierr serializes client-facing errors in the same shape as the legacy Rust API.
package apierr

import (
	"encoding/json"
	"net/http"
)

// API error code strings (JSON "error.code"); messages stay human-readable.
const (
	CodeInvalidInput       = "INVALID_INPUT"
	CodeInvalidCredentials = "INVALID_CREDENTIALS"
	CodeEmailTaken         = "EMAIL_TAKEN"
	CodeInvalidResetToken  = "INVALID_RESET_TOKEN"
	CodeMagicLinkGone      = "MAGIC_LINK_GONE"
	CodeRateLimited        = "RATE_LIMITED"
	CodeUnauthorized       = "UNAUTHORIZED"
	CodeNotFound           = "NOT_FOUND"
	// CodeMethodNotAllowed is used when a path exists but not for this HTTP verb.
	CodeMethodNotAllowed = "METHOD_NOT_ALLOWED"
	// CodeNotImplemented is used for HTTP 501 (capability not available on this server build).
	CodeNotImplemented    = "NOT_IMPLEMENTED"
	CodeUnknownCourseCode = "UNKNOWN_COURSE_CODE"
	CodeForbidden         = "FORBIDDEN"
	CodeConflict          = "CONFLICT"
	CodeMFARequired       = "MFA_REQUIRED"
	CodeMFAEnrolRequired  = "MFA_ENROLMENT_REQUIRED"
	CodeInternal          = "INTERNAL"
	// CodeUnprocessableEntity is used when the request is well-formed but cannot be applied (e.g. revoke current session).
	CodeUnprocessableEntity = "UNPROCESSABLE_ENTITY"
	CodeAiNotConfigured          = "AI_NOT_CONFIGURED"
	CodeAiGenerationFailed       = "AI_GENERATION_FAILED"
	CodeAiProcessingDisabled     = "AI_PROCESSING_DISABLED"
	CodeTenantAIPolicyDisabled   = "TENANT_AI_POLICY_DISABLED"
	CodeOrgSuspended        = "ORG_SUSPENDED"
	CodePaymentRequired     = "PAYMENT_REQUIRED"
)

// Body matches server/src/error.rs JSON error envelope.
type Body struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// WriteJSON writes a JSON error body and sets Content-Type. Status is typically 4xx/5xx.
func WriteJSON(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body(code, message))
}

// WriteJSONWithErr writes a JSON error body and records server-side failures for access logging.
func WriteJSONWithErr(w http.ResponseWriter, r *http.Request, status int, code, message string, err error) {
	if status >= http.StatusInternalServerError && r != nil {
		RecordServerError(r, message, err)
	}
	WriteJSON(w, status, code, message)
}

// WriteInternal writes a 500 INTERNAL response and records err for access logging.
func WriteInternal(w http.ResponseWriter, r *http.Request, message string, err error) {
	WriteJSONWithErr(w, r, http.StatusInternalServerError, CodeInternal, message, err)
}

func body(code, message string) Body {
	var b Body
	b.Error.Code = code
	b.Error.Message = message
	return b
}
