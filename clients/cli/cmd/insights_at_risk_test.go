package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/lextures/lextures/clients/cli/internal/config"
)

func TestParseAtRiskThreshold(t *testing.T) {
	v, ok, err := parseAtRiskThreshold("high")
	if err != nil || !ok || v != 75 {
		t.Fatalf("high: v=%v ok=%v err=%v", v, ok, err)
	}
	if _, _, err := parseAtRiskThreshold("bogus"); err == nil {
		t.Fatal("expected error for bogus threshold")
	}
}

func TestFilterAtRiskAlerts(t *testing.T) {
	alerts := []atRiskAlert{
		{Score: 80, TopFactor: "quiz"},
		{Score: 40, TopFactor: "inactive"},
	}
	got := filterAtRiskAlerts(alerts, 50, true, "quiz")
	if len(got) != 1 || got[0].TopFactor != "quiz" {
		t.Fatalf("filter = %+v", got)
	}
}

func TestAtRiskAlertsToCSV(t *testing.T) {
	data, rows, err := atRiskAlertsToCSV([]atRiskAlert{
		{CourseCode: "CS101", UserID: "u1", EnrollmentID: "e1", DisplayName: "Ada", Score: 88, TopFactor: "quiz", TopFactorLabel: "Failing", Status: "active", TriggeredDate: "2026-01-01"},
	})
	if err != nil || rows != 2 {
		t.Fatalf("rows=%d err=%v", rows, err)
	}
	if !strings.Contains(string(data), "CS101") {
		t.Fatalf("csv = %q", string(data))
	}
}

func TestAtRiskList_RequiresYesForExport(t *testing.T) {
	atRiskListFlags.export = true
	atRiskListFlags.yes = false
	defer func() {
		atRiskListFlags.export = false
		atRiskListFlags.yes = false
	}()
	err := atRiskListCmd.RunE(atRiskListCmd, []string{"CS101"})
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("err = %v", err)
	}
}

func TestAtRiskList_CourseSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/courses/CS101/at-risk" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"alerts": []any{map[string]any{
					"id": "a1", "userId": "u1", "enrollmentId": "e1", "displayName": "Ada",
					"score": 90, "status": "active", "topFactor": "quiz", "topFactorLabel": "Failing quiz",
					"triggeredDate": "2026-01-01",
				}},
				"resolved": []any{},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	atRiskListFlags.threshold = "high"
	defer func() { atRiskListFlags.threshold = "" }()

	globalFlags.jsonOut = true
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	atRiskListCmd.SetOut(&out)
	if err := atRiskListCmd.RunE(atRiskListCmd, []string{"CS101"}); err != nil {
		t.Fatalf("at-risk list: %v", err)
	}
	if !strings.Contains(out.String(), "Ada") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestInsightsCourse_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/courses/CS101/analytics/insights" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"weekOf": "2026-01-06", "generatedAt": "2026-01-07T00:00:00Z",
				"workingWell":    []any{map[string]any{"signalKey": "engagement", "title": "Strong participation"}},
				"needsAttention": []any{},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	insightsCourseCmd.SetOut(&out)
	if err := insightsCourseCmd.RunE(insightsCourseCmd, []string{"CS101"}); err != nil {
		t.Fatalf("insights course: %v", err)
	}
	if !strings.Contains(out.String(), "Strong participation") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestAtRiskExport_ToDir(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/courses/CS101/at-risk" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"alerts": []any{map[string]any{
					"id": "a1", "userId": "u1", "enrollmentId": "e1", "displayName": "Ada",
					"score": 90, "status": "active", "topFactor": "quiz", "triggeredDate": "2026-01-01",
				}},
				"resolved": []any{},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	dir := t.TempDir()
	atRiskListFlags.export = true
	atRiskListFlags.yes = true
	atRiskListFlags.out = dir
	defer func() {
		atRiskListFlags.export = false
		atRiskListFlags.yes = false
		atRiskListFlags.out = ""
	}()

	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	if err := atRiskListCmd.RunE(atRiskListCmd, []string{"CS101"}); err != nil {
		t.Fatalf("export: %v", err)
	}
	data, err := os.ReadFile(dir + "/at-risk-cohort.csv")
	if err != nil || !strings.Contains(string(data), "Ada") {
		t.Fatalf("file err=%v data=%q", err, string(data))
	}
}