package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResolveEnrollmentForCourse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/me/enrollments" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"enrollments": []any{map[string]any{"id": "e1", "courseCode": "PY101"}},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	c := httptestClient(srv.URL)
	id, err := resolveEnrollmentForCourse(c, "PY101")
	if err != nil || id != "e1" {
		t.Fatalf("id=%s err=%v", id, err)
	}
}

func TestDiagnosticConfig_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/courses/CS101/diagnostic-config" {
			_ = json.NewEncoder(w).Encode(map[string]any{"enabled": true})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	setCfg(srv.URL, "tok")
	var out strings.Builder
	diagnosticConfigCmd.SetOut(&out)
	if err := diagnosticConfigCmd.RunE(diagnosticConfigCmd, []string{"CS101"}); err != nil {
		t.Fatalf("config: %v", err)
	}
}

func TestTutorEval_RequiresYes(t *testing.T) {
	tutorEvalFlags.file = writeTempJSONL(t, `{"course":"CS101","prompt":"hi"}`)
	tutorEvalFlags.yes = false
	defer func() {
		tutorEvalFlags.file = ""
		tutorEvalFlags.yes = false
	}()
	err := tutorEvalCmd.RunE(tutorEvalCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("err=%v", err)
	}
}