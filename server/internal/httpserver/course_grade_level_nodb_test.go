package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
)

// Regression: PUT /api/v1/courses/{code} must accept gradeLevel in body (no 405).
func TestPutCourse_AcceptsGradeLevel(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: config.Config{}})
	rr := httptest.NewRecorder()
	body := `{"title":"T","description":"","published":false,"scheduleMode":"fixed",` +
		`"startsAt":null,"endsAt":null,"visibleFrom":null,"hiddenAt":null,` +
		`"relativeEndAfter":null,"relativeHiddenAfter":null,"gradeLevel":"5"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/courses/C-TEST", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	tok, _ := signer.Sign(context.Background(), "00000000-0000-0000-0000-000000000001", "u@test.com", "", "", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, req)
	if rr.Code == http.StatusMethodNotAllowed {
		t.Fatalf("expected PUT to be registered, got 405: %s", rr.Body.String())
	}
}

// Regression: GET /api/v1/courses must accept grade_level query parameter (no 405).
func TestListCourses_AcceptsGradeLevelFilter(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: config.Config{}})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses?grade_level=5", nil)
	tok, _ := signer.Sign(context.Background(), "00000000-0000-0000-0000-000000000001", "u@test.com", "", "", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, req)
	if rr.Code == http.StatusMethodNotAllowed {
		t.Fatalf("expected GET to be registered, got 405: %s", rr.Body.String())
	}
}

// Regression: GET /api/v1/courses with invalid grade_level must not return 405 (route registered).
// Full 400 validation is covered by e2e/tests/grade-level.spec.ts with a real DB.
func TestListCourses_InvalidGradeLevelNotMethodNotAllowed(t *testing.T) {
	signer := auth.NewJWTSigner("01234567890123456789012345678901")
	h := NewHandler(Deps{Pool: nil, JWTSigner: signer, Config: config.Config{}})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/courses?grade_level=INVALID", nil)
	tok, _ := signer.Sign(context.Background(), "00000000-0000-0000-0000-000000000001", "u@test.com", "", "", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(rr, req)
	if rr.Code == http.StatusMethodNotAllowed {
		t.Fatalf("expected route to be registered, got 405: %s", rr.Body.String())
	}
}
