package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lextures/lextures/server/internal/service/codeexecution"
)

func TestHandleQuizQuestionRun_Unauthorized(t *testing.T) {
	h := NewHandler(Deps{})
	r := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/courses/demo/quizzes/00000000-0000-0000-0000-000000000001/attempts/00000000-0000-0000-0000-000000000002/questions/q1/run",
		nil,
	)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestHandleQuizQuestionRun_MethodNotAllowed(t *testing.T) {
	h := NewHandler(Deps{})
	r := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/courses/demo/quizzes/00000000-0000-0000-0000-000000000001/attempts/00000000-0000-0000-0000-000000000002/questions/q1/run",
		nil,
	)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

func TestFilterPublicCodeTests(t *testing.T) {
	tests := filterPublicCodeTests([]codeexecution.TestCase{
		{ID: "a", IsHidden: false},
		{ID: "b", IsHidden: true},
		{ID: "c", IsHidden: false},
	})
	if len(tests) != 2 || tests[0].ID != "a" || tests[1].ID != "c" {
		t.Fatalf("unexpected filter result: %+v", tests)
	}
}

func TestParseQuizCodeTypeConfig(t *testing.T) {
	raw := []byte(`{"language":"javascript","testCases":[{"id":"t1","input":"","expectedOutput":"4","isHidden":false}]}`)
	runtime, cases := parseQuizCodeTypeConfig(raw)
	if runtime != "javascript" {
		t.Fatalf("runtime: %q", runtime)
	}
	if len(cases) != 1 || cases[0].ExpectedOutput != "4" {
		t.Fatalf("cases: %+v", cases)
	}
}
