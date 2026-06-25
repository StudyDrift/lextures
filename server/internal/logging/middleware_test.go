package logging

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/apierr"
)

func TestAccessLog_LogsServerErrorOn500(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	handler := AccessLog(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apierr.WriteInternal(w, r, "Failed to load submissions.", errTestFailure)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses/C-TEST/assignments/item/submissions", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status: got %d want %d", rr.Code, http.StatusInternalServerError)
	}
	out := buf.String()
	if !strings.Contains(out, `"level":"ERROR"`) {
		t.Fatalf("expected ERROR log, got: %s", out)
	}
	if !strings.Contains(out, `"error_message":"Failed to load submissions."`) {
		t.Fatalf("expected error_message in log, got: %s", out)
	}
	if !strings.Contains(out, `"err":"database unavailable"`) {
		t.Fatalf("expected err in log, got: %s", out)
	}
}

var errTestFailure = errString("database unavailable")

type errString string

func (e errString) Error() string { return string(e) }