package kernel

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"unicode"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lextures/lextures/server/internal/apierr"
)

// Error is an explicit HTTP-mapped failure. Handlers should use this (or the
// helpers below) when adopting the mapper so client messages stay stable.
type Error struct {
	Status     int
	Code       string
	Message    string
	Cause      error
	RetryAfter int // seconds; set on the response when > 0
	Fields     []apierr.FieldViolation
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// Mapped is the result of Map.
type Mapped struct {
	Status     int
	Code       string
	Message    string
	RetryAfter int
	Fields     []apierr.FieldViolation
	// LogInternal is true when the original error should be logged server-side
	// and must not appear in the client body.
	LogInternal bool
	Cause       error
}

// Sentinel errors for AI-backed endpoints (§11). Handlers opt in by returning
// these (or wrapping them). Status codes match the agreed mapping: provider
// 429 → 503 + Retry-After; budget exceeded → 402; content filter → 422.
var (
	ErrAINotConfigured = &Error{
		Status:  http.StatusServiceUnavailable,
		Code:    apierr.CodeAiNotConfigured,
		Message: "AI is not configured.",
	}
	ErrAIGenerationFailed = &Error{
		Status:  http.StatusServiceUnavailable,
		Code:    apierr.CodeAiGenerationFailed,
		Message: "AI generation failed.",
	}
	ErrAIRateLimited = &Error{
		Status:     http.StatusServiceUnavailable,
		Code:       apierr.CodeServiceUnavailable,
		Message:    "AI provider is temporarily unavailable.",
		RetryAfter: 30,
	}
	ErrAIBudgetExceeded = &Error{
		Status:  http.StatusPaymentRequired,
		Code:    apierr.CodePaymentRequired,
		Message: "AI budget exceeded.",
	}
	ErrAIContentFiltered = &Error{
		Status:  http.StatusUnprocessableEntity,
		Code:    apierr.CodeInvalidInput,
		Message: "The request was blocked by a content filter.",
	}
)

// InvalidInput returns 400 INVALID_INPUT.
func InvalidInput(message string) error {
	return &Error{Status: http.StatusBadRequest, Code: apierr.CodeInvalidInput, Message: message}
}

// NotFound returns 404 NOT_FOUND.
func NotFound(message string) error {
	return &Error{Status: http.StatusNotFound, Code: apierr.CodeNotFound, Message: message}
}

// Forbidden returns 403 FORBIDDEN.
func Forbidden(message string) error {
	return &Error{Status: http.StatusForbidden, Code: apierr.CodeForbidden, Message: message}
}

// Unauthorized returns 401 UNAUTHORIZED.
func Unauthorized(message string) error {
	return &Error{Status: http.StatusUnauthorized, Code: apierr.CodeUnauthorized, Message: message}
}

// Conflict returns 409 CONFLICT.
func Conflict(message string) error {
	return &Error{Status: http.StatusConflict, Code: apierr.CodeConflict, Message: message}
}

// Internal returns 500 INTERNAL with a safe client message. The cause is logged
// when WriteError runs and is never copied into the response.
func Internal(message string, cause error) error {
	if message == "" {
		message = "Something went wrong."
	}
	return &Error{Status: http.StatusInternalServerError, Code: apierr.CodeInternal, Message: message, Cause: cause}
}

// PayloadTooLarge returns 413 INVALID_INPUT.
func PayloadTooLarge() error {
	return &Error{Status: http.StatusRequestEntityTooLarge, Code: apierr.CodeInvalidInput, Message: "Request body too large."}
}

// UnsupportedMediaType returns 415 INVALID_INPUT.
func UnsupportedMediaType() error {
	return &Error{Status: http.StatusUnsupportedMediaType, Code: apierr.CodeInvalidInput, Message: "Content-Type must be application/json."}
}

// Map translates err to an HTTP status and safe apierr code/message.
// Unmapped errors become 500 INTERNAL with a generic message.
func Map(err error) Mapped {
	return sanitizeMapped(mapErr(err))
}

func mapErr(err error) Mapped {
	if err == nil {
		return Mapped{}
	}

	var httpErr *Error
	if errors.As(err, &httpErr) && httpErr != nil {
		msg := httpErr.Message
		if msg == "" {
			msg = http.StatusText(httpErr.Status)
		}
		code := httpErr.Code
		if code == "" {
			code = defaultCode(httpErr.Status)
		}
		m := Mapped{
			Status:      httpErr.Status,
			Code:        code,
			Message:     msg,
			RetryAfter:  httpErr.RetryAfter,
			Fields:      httpErr.Fields,
			LogInternal: httpErr.Status >= http.StatusInternalServerError || httpErr.Cause != nil,
			Cause:       httpErr.Cause,
		}
		if m.Cause == nil {
			m.Cause = err
		}
		return m
	}

	var valErr *ValidationError
	if errors.As(err, &valErr) && valErr != nil {
		return Mapped{
			Status:  http.StatusUnprocessableEntity,
			Code:    apierr.ErrorValidationFailed,
			Message: valErr.Message,
			Fields:  valErr.Fields,
		}
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return Mapped{
			Status:      http.StatusNotFound,
			Code:        apierr.CodeNotFound,
			Message:     "Not found.",
			LogInternal: true,
			Cause:       err,
		}
	}

	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		return Mapped{
			Status:  http.StatusRequestEntityTooLarge,
			Code:    apierr.CodeInvalidInput,
			Message: "Request body too large.",
		}
	}

	if errors.Is(err, context.Canceled) {
		return Mapped{
			Status:      http.StatusBadRequest,
			Code:        apierr.CodeInvalidInput,
			Message:     "Request canceled.",
			LogInternal: true,
			Cause:       err,
		}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return Mapped{
			Status:      http.StatusServiceUnavailable,
			Code:        apierr.CodeServiceUnavailable,
			Message:     "The request timed out.",
			LogInternal: true,
			Cause:       err,
		}
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr != nil && isIntegrityConstraint(pgErr.Code) {
		return Mapped{
			Status:      http.StatusConflict,
			Code:        apierr.CodeConflict,
			Message:     "Conflict.",
			LogInternal: true,
			Cause:       err,
		}
	}

	return Mapped{
		Status:      http.StatusInternalServerError,
		Code:        apierr.CodeInternal,
		Message:     "Something went wrong.",
		LogInternal: true,
		Cause:       err,
	}
}

func sanitizeMapped(m Mapped) Mapped {
	if looksInternal(m.Message) {
		m.Message = "Something went wrong."
		m.LogInternal = true
	}
	return m
}

func isIntegrityConstraint(code string) bool {
	if len(code) < 2 {
		return false
	}
	return code[0] == '2' && code[1] == '3'
}

func defaultCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return apierr.CodeInvalidInput
	case http.StatusUnauthorized:
		return apierr.CodeUnauthorized
	case http.StatusForbidden:
		return apierr.CodeForbidden
	case http.StatusNotFound:
		return apierr.CodeNotFound
	case http.StatusConflict:
		return apierr.CodeConflict
	case http.StatusPaymentRequired:
		return apierr.CodePaymentRequired
	case http.StatusUnprocessableEntity:
		return apierr.CodeUnprocessableEntity
	case http.StatusTooManyRequests:
		return apierr.CodeRateLimited
	case http.StatusServiceUnavailable:
		return apierr.CodeServiceUnavailable
	default:
		if status >= http.StatusInternalServerError {
			return apierr.CodeInternal
		}
		return apierr.CodeInvalidInput
	}
}

// WriteError maps err and writes the standard envelope. It never writes
// internal detail, SQL text, or stack traces to the client.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil || errors.Is(err, ErrWritten) {
		return
	}
	m := Map(err)
	if m.Status == 0 {
		return
	}
	observeMapped(r, m)
	if m.LogInternal {
		logMapped(r, m)
	}
	if len(m.Fields) > 0 || m.Code == apierr.ErrorValidationFailed {
		apierr.WriteValidationFailed(w, m.Message, m.Fields)
		return
	}
	if m.RetryAfter > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(m.RetryAfter))
	}
	if m.Status >= http.StatusInternalServerError {
		apierr.WriteJSONWithErr(w, r, m.Status, m.Code, m.Message, m.Cause)
		return
	}
	apierr.WriteJSON(w, m.Status, m.Code, m.Message)
}

func logMapped(r *http.Request, m Mapped) {
	attrs := []any{
		"status", m.Status,
		"code", m.Code,
	}
	if r != nil {
		attrs = append(attrs, "request_id", middleware.GetReqID(r.Context()))
		if r.URL != nil {
			attrs = append(attrs, "path", r.URL.Path)
		}
	}
	if m.Cause != nil {
		attrs = append(attrs, "err", m.Cause)
	}
	slog.Error("mapped api error", attrs...)
}

func routeClass(r *http.Request) string {
	if r == nil {
		return "unknown"
	}
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		if p := rctx.RoutePattern(); p != "" && p != "/*" {
			return p
		}
	}
	return "unmatched"
}

// looksInternal reports whether s looks like an infrastructure leak.
func looksInternal(s string) bool {
	if s == "" {
		return false
	}
	lower := strings.ToLower(s)
	needles := []string{
		"sqlstate", "pq:", "pgx", "select ", "insert ", "update ", "delete from",
		"goroutine ", "stack trace", "panic:", "connection refused",
		"password=", "postgres://",
	}
	for _, n := range needles {
		if strings.Contains(lower, n) {
			return true
		}
	}
	// Bare hex/error codes with lots of punctuation often indicate driver text.
	nonPrint := 0
	for _, r := range s {
		if !unicode.IsPrint(r) {
			nonPrint++
		}
	}
	return nonPrint > 0
}
