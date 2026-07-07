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

func TestFilterCanvasCourses(t *testing.T) {
	courses := []canvasCourseItem{
		{ID: 1, Name: "Algebra I", CourseCode: "MATH-101"},
		{ID: 2, Name: "Biology", CourseCode: "BIO-201"},
	}
	got := filterCanvasCourses(courses, "bio")
	if len(got) != 1 || got[0].ID != 2 {
		t.Fatalf("filter = %+v", got)
	}
}

func TestIncludeForArtifact(t *testing.T) {
	inc := includeForArtifact("grades")
	if !inc.Grades || inc.Modules {
		t.Fatalf("grades include = %+v", inc)
	}
}

func TestCanvasCatalogList_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/integrations/canvas/courses" && r.Method == http.MethodPost {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"courses": []map[string]any{{"id": 42, "name": "Intro", "courseCode": "INTRO-1", "workflowState": "available"}},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	dir := t.TempDir()
	tokenPath := dir + "/token.txt"
	if err := os.WriteFile(tokenPath, []byte("canvas-token\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	canvasCatalogCommonFlags.canvasBase = "https://canvas.example"
	canvasCatalogCommonFlags.tokenFile = tokenPath
	defer func() {
		canvasCatalogCommonFlags.canvasBase = ""
		canvasCatalogCommonFlags.tokenFile = ""
	}()

	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	canvasCatalogListCmd.SetOut(&out)
	if err := canvasCatalogListCmd.RunE(canvasCatalogListCmd, nil); err != nil {
		t.Fatalf("canvas catalog list: %v", err)
	}
	if !strings.Contains(out.String(), "Intro") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestContentImportStatus_Success(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/imports/") && strings.HasSuffix(r.URL.Path, "/status") {
			calls++
			status := "running"
			if calls >= 2 {
				status = "done"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": status, "totalItems": 10, "processedItems": 10,
				"succeededItems": 9, "failedItems": 1, "skippedItems": 0,
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	importsStatusFlags.content = true
	importsStatusFlags.wait = true
	importsStatusFlags.timeout = 5 * jobPollInterval
	defer func() {
		importsStatusFlags.content = false
		importsStatusFlags.wait = false
	}()

	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	importsStatusCmd.SetOut(&out)
	if err := importsStatusCmd.RunE(importsStatusCmd, []string{"job-1"}); err != nil {
		t.Fatalf("imports status --content --wait: %v", err)
	}
	if !strings.Contains(out.String(), "succeeded=9") {
		t.Fatalf("output = %q", out.String())
	}
}