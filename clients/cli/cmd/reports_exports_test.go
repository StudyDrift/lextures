package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lextures/lextures/clients/cli/internal/config"
)

func TestLookupReportType(t *testing.T) {
	if _, ok := lookupReportType("learning-activity"); !ok {
		t.Fatal("expected learning-activity")
	}
	if _, ok := lookupReportType("unknown"); ok {
		t.Fatal("expected missing")
	}
}

func TestLearningActivityToCSV(t *testing.T) {
	var report learningActivityReport
	report.Summary.TotalEvents = 10
	report.Summary.UniqueUsers = 3
	report.ByDay = append(report.ByDay, struct {
		Day          string `json:"day"`
		CourseVisit  int64  `json:"courseVisit"`
		ContentOpen  int64  `json:"contentOpen"`
		ContentLeave int64  `json:"contentLeave"`
	}{Day: "2026-01-01", CourseVisit: 5, ContentOpen: 2, ContentLeave: 1})
	data, rows, err := learningActivityToCSV(report)
	if err != nil || rows < 4 {
		t.Fatalf("rows=%d err=%v data=%q", rows, err, string(data))
	}
	if !strings.Contains(string(data), "byDay") {
		t.Fatalf("csv = %q", string(data))
	}
}

func TestNormalizeExportFormat(t *testing.T) {
	if _, err := normalizeExportFormat("csv"); err != nil {
		t.Fatal(err)
	}
	if _, err := normalizeExportFormat("xml"); err == nil {
		t.Fatal("expected error")
	}
}

func TestReportsExport_RequiresYes(t *testing.T) {
	reportsCommonFlags.yes = false
	reportsCommonFlags.format = "csv"
	defer func() {
		reportsCommonFlags.yes = false
		reportsCommonFlags.format = "csv"
	}()
	err := reportsExportCmd.RunE(reportsExportCmd, []string{"learning-activity"})
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("err = %v", err)
	}
}

func TestReportsLearningActivity_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/reports/learning-activity" {
			now := time.Now().UTC()
			_ = json.NewEncoder(w).Encode(map[string]any{
				"range": map[string]any{"from": now.Add(-24 * time.Hour), "to": now},
				"summary": map[string]any{"totalEvents": 42, "uniqueUsers": 7, "uniqueCourses": 3},
				"byDay": []any{}, "byEventKind": []any{}, "topCourses": []any{},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	reportsLearningActivityCmd.SetOut(&out)
	if err := reportsLearningActivityCmd.RunE(reportsLearningActivityCmd, nil); err != nil {
		t.Fatalf("reports learning-activity: %v", err)
	}
	if !strings.Contains(out.String(), "events=42") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestReportsExportLearningActivity_ToDir(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/reports/learning-activity" {
			now := time.Now().UTC()
			_ = json.NewEncoder(w).Encode(map[string]any{
				"range": map[string]any{"from": now.Add(-24 * time.Hour), "to": now},
				"summary": map[string]any{"totalEvents": 1, "uniqueUsers": 1, "uniqueCourses": 1},
				"byDay": []any{}, "byEventKind": []any{}, "topCourses": []any{},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	dir := t.TempDir()
	reportsCommonFlags.yes = true
	reportsCommonFlags.format = "csv"
	reportsCommonFlags.out = dir
	defer func() {
		reportsCommonFlags.yes = false
		reportsCommonFlags.format = "csv"
		reportsCommonFlags.out = ""
	}()

	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	reportsExportCmd.SetOut(&out)
	if err := reportsExportCmd.RunE(reportsExportCmd, []string{"learning-activity"}); err != nil {
		t.Fatalf("reports export: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 {
		t.Fatalf("dir entries: %v", entries)
	}
	data, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if err != nil || len(data) == 0 {
		t.Fatalf("file read: %v", err)
	}
}

func TestAnalyticsCourse_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/analytics/engagement-overview") {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"students": []map[string]any{{
					"enrollmentId": "e1", "displayName": "Alex", "loginsLast7Days": 4,
					"avgTimeOnTaskMin": 12.5, "engagementScore": 55.0,
				}},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	globalFlags.jsonOut = true
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	analyticsCourseCmd.SetOut(&out)
	if err := analyticsCourseCmd.RunE(analyticsCourseCmd, []string{"CS101"}); err != nil {
		t.Fatalf("analytics course: %v", err)
	}
	if !strings.Contains(out.String(), "Alex") {
		t.Fatalf("output = %q", out.String())
	}
}