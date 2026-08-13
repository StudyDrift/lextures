package kernel

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lextures/lextures/server/internal/apierr"
)

func TestMap_Table(t *testing.T) {
	t.Parallel()
	sqlLeak := errors.New(`ERROR: duplicate key value violates unique constraint "users_email_key" (SQLSTATE 23505)`)
	pgUnique := &pgconn.PgError{Code: "23505", Message: `duplicate key value violates unique constraint "users_email_key"`}
	pgFK := &pgconn.PgError{Code: "23503", Message: `insert or update on table "x" violates foreign key constraint`}

	cases := []struct {
		name   string
		err    error
		status int
		code   string
		msg    string
		log    bool
	}{
		{"nil", nil, 0, "", "", false},
		{"no_rows", pgx.ErrNoRows, 404, apierr.CodeNotFound, "Not found.", true},
		{"wrapped_no_rows", errors.Join(errors.New("load"), pgx.ErrNoRows), 404, apierr.CodeNotFound, "Not found.", true},
		{"unique", pgUnique, 409, apierr.CodeConflict, "Conflict.", true},
		{"fk", pgFK, 409, apierr.CodeConflict, "Conflict.", true},
		{"canceled", context.Canceled, 400, apierr.CodeInvalidInput, "Request canceled.", true},
		{"deadline", context.DeadlineExceeded, 503, apierr.CodeServiceUnavailable, "The request timed out.", true},
		{"too_large", &http.MaxBytesError{Limit: 12}, 413, apierr.CodeInvalidInput, "Request body too large.", false},
		{"explicit_forbidden", Forbidden("You do not have permission to edit outcomes."), 403, apierr.CodeForbidden, "You do not have permission to edit outcomes.", false},
		{"internal_wrap", Internal("Failed to create outcome.", sqlLeak), 500, apierr.CodeInternal, "Failed to create outcome.", true},
		{"leaked_explicit", &Error{Status: 500, Code: apierr.CodeInternal, Message: "SQLSTATE 23505 boom"}, 500, apierr.CodeInternal, "Something went wrong.", true},
		{"unmapped", sqlLeak, 500, apierr.CodeInternal, "Something went wrong.", true},
		{"ai_unconfigured", ErrAINotConfigured, 503, apierr.CodeAiNotConfigured, "AI is not configured.", true},
		{"ai_rate", ErrAIRateLimited, 503, apierr.CodeServiceUnavailable, "AI provider is temporarily unavailable.", true},
		{"ai_budget", ErrAIBudgetExceeded, 402, apierr.CodePaymentRequired, "AI budget exceeded.", false},
		{"ai_filter", ErrAIContentFiltered, 422, apierr.CodeInvalidInput, "The request was blocked by a content filter.", false},
		{"validation", Validation("Fix the highlighted fields", FieldError("title", "required", "Title is required.")), 422, apierr.ErrorValidationFailed, "Fix the highlighted fields", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := Map(tc.err)
			if m.Status != tc.status || m.Code != tc.code || m.Message != tc.msg {
				t.Fatalf("Map() = %+v, want status=%d code=%q msg=%q", m, tc.status, tc.code, tc.msg)
			}
			if m.LogInternal != tc.log {
				t.Fatalf("LogInternal=%v want %v", m.LogInternal, tc.log)
			}
			if looksInternal(m.Message) {
				t.Fatalf("client message leaked internals: %q", m.Message)
			}
			if tc.err != nil && strings.Contains(strings.ToLower(m.Message), "sqlstate") {
				t.Fatalf("SQLSTATE in client message: %q", m.Message)
			}
		})
	}
}

func TestWriteError_NoSQLInBody(t *testing.T) {
	t.Parallel()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/x", nil)
	WriteError(rr, req, &pgconn.PgError{Code: "23505", Message: `duplicate key "users_email_key" SQLSTATE 23505`})
	if rr.Code != http.StatusConflict {
		t.Fatalf("status=%d", rr.Code)
	}
	body := rr.Body.String()
	if strings.Contains(body, "SQLSTATE") || strings.Contains(body, "users_email_key") {
		t.Fatalf("leaked SQL: %s", body)
	}
	var env apierr.Body
	if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env.Error.Code != apierr.CodeConflict || env.Error.Message != "Conflict." {
		t.Fatalf("envelope=%+v", env)
	}
}

func TestWriteError_NoRows(t *testing.T) {
	t.Parallel()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	WriteError(rr, req, pgx.ErrNoRows)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rr.Code)
	}
	var env apierr.Body
	if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env.Error.Code != apierr.CodeNotFound || env.Error.Message != "Not found." {
		t.Fatalf("envelope=%+v", env)
	}
}

func TestWriteError_ValidationEnvelope(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	WriteError(rr, req, Validation("Fix the highlighted fields", FieldError("title", "required", "Title is required.")))
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d", rr.Code)
	}
	var body apierr.ValidationFailedBody
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Error != apierr.ErrorValidationFailed || len(body.Fields) != 1 || body.Fields[0].Path != "title" {
		t.Fatalf("body=%+v", body)
	}
}

func TestWriteError_RetryAfter(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	WriteError(rr, httptest.NewRequest(http.MethodPost, "/x", nil), ErrAIRateLimited)
	if rr.Header().Get("Retry-After") != "30" {
		t.Fatalf("Retry-After=%q", rr.Header().Get("Retry-After"))
	}
}

func TestWriteError_AlreadyWritten(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	WriteError(rr, httptest.NewRequest(http.MethodGet, "/x", nil), ErrWritten)
	if rr.Code != http.StatusOK && rr.Body.Len() != 0 {
		// httptest default code is 200 until WriteHeader; body must stay empty.
		t.Fatalf("wrote response for ErrWritten: status=%d body=%q", rr.Code, rr.Body.String())
	}
	if rr.Body.Len() != 0 {
		t.Fatalf("body=%q", rr.Body.String())
	}
}

func TestMap_DoesNotDowngradeUnknown(t *testing.T) {
	t.Parallel()
	m := Map(errors.New("weird driver failure"))
	if m.Status != http.StatusInternalServerError {
		t.Fatalf("status=%d", m.Status)
	}
}

func TestLooksInternal(t *testing.T) {
	t.Parallel()
	if !looksInternal("SQLSTATE 23505") {
		t.Fatal("expected SQLSTATE to look internal")
	}
	if looksInternal("You do not have permission to edit outcomes.") {
		t.Fatal("user message flagged internal")
	}
}
