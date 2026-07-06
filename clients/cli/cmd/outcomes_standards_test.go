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
)

func resetOutcomesStandardsFlags() {
	standardsListFlags.framework = ""
	standardsListFlags.grade = ""
	standardsListFlags.query = ""
	standardsListFlags.limit = 200
	standardsImportFlags.file = ""
	standardsImportFlags.org = ""
	outcomesCreateFlags.title = ""
	outcomesCreateFlags.description = ""
	outcomesCreateFlags.file = ""
	outcomesAlignFlags.file = ""
	outcomesReportFlags.section = ""
	outcomesReportFlags.group = ""
	outcomesMasteryFlags.user = ""
	outcomesMasteryFlags.period = ""
	outcomesMasteryFlags.method = ""
	sbgGetFlags.period = ""
	reportCardsListFlags.period = ""
	reportCardsGetFlags.period = ""
	reportCardsGetFlags.card = ""
	reportCardsGetFlags.user = ""
	reportCardsExportFlags.period = ""
	reportCardsExportFlags.section = ""
	reportCardsExportFlags.user = ""
	reportCardsExportFlags.format = "pdf"
	reportCardsExportFlags.out = ""
	reportCardsExportFlags.yes = false
}

func newOutcomesStandardsServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case r.Method == http.MethodGet && path == "/api/v1/standards":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"id":"std-1","code":"6.RP.1","description":"Ratios"}]`))
		case r.Method == http.MethodGet && strings.HasPrefix(path, "/api/v1/standards/"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"std-1","code":"6.RP.1","description":"Ratios","frameworkCode":"ccss-math","frameworkName":"CCSS Math","frameworkVersion":"2010"}`))
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/sbg/standards/import"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"domainsCreated":1,"standardsImported":2,"errors":[]}`))
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/analytics/outcomes"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"courseId":"course-1","masteryThreshold":70,"dataAsOf":"2026-01-01T00:00:00Z","staleMinutes":0,"outcomes":[{"outcomeId":"out-1","title":"Analyze data","sortOrder":1,"nStudents":2,"nAssessed":2,"meanScore":82.5,"pctMet":100,"pctNotMet":0,"threshold":70,"alignmentCount":1,"improvementNote":"","noAlignments":false}]}`))
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/outcomes"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"enrolledLearners":2,"outcomes":[{"id":"out-1","title":"Analyze data","links":[]}]}`))
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/outcomes"):
			w.WriteHeader(http.StatusCreated)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"out-new","title":"New outcome"}`))
		case r.Method == http.MethodPost && strings.Contains(path, "/outcomes/") && strings.HasSuffix(path, "/links"):
			w.WriteHeader(http.StatusCreated)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"link-1"}`))
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/sbg/standards"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"standards":[{"id":"s-1","code":"MATH.1","description":"Add"}],"courseCode":"CS101"}`))
		case r.Method == http.MethodGet && strings.Contains(path, "/sbg/heatmap/"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"cells":[{"studentId":"u-1","standardId":"s-1","scoreValue":3}],"courseCode":"CS101","period":"Q1"}`))
		case r.Method == http.MethodGet && strings.Contains(path, "/students/") && strings.Contains(path, "/sbg/"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"studentId":"u-1","period":"Q1","method":"most_recent","scores":[{"standardId":"s-1","scoreValue":3}]}`))
		case r.Method == http.MethodGet && strings.Contains(path, "/report-cards/"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"reportCards":[{"id":"card-1","studentId":"u-1","gradingPeriod":"Q1","status":"approved","finalGradePct":91.2,"letterGrade":"A-"}],"period":"Q1"}`))
		case r.Method == http.MethodGet && path == "/api/v1/courses/CS101/enrollments":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"enrollments":[{"id":"e-1","userId":"u-1","role":"student","sectionId":"sec-1"}]}`))
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/generate-pdf"):
			w.Header().Set("Content-Type", "application/pdf")
			_, _ = w.Write([]byte("%PDF-1.4 test"))
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestFrameworkImportToCSV_FromDomains(t *testing.T) {
	raw := []byte(`{
	  "domains": [{
	    "code": "RP",
	    "name": "Ratios",
	    "gradeLevel": "6",
	    "standards": [{"code": "6.RP.1", "description": "Understand ratio concepts"}]
	  }]
	}`)
	csvBytes, err := frameworkImportToCSV(raw)
	if err != nil {
		t.Fatalf("frameworkImportToCSV: %v", err)
	}
	if !strings.Contains(string(csvBytes), "6.RP.1") || !strings.Contains(string(csvBytes), "domain_code") {
		t.Fatalf("unexpected CSV: %s", string(csvBytes))
	}
}

func TestParseOutcomeAlignFile_JSON(t *testing.T) {
	raw := []byte(`{"links":[{"outcomeId":"out-1","structureItemId":"item-1","targetKind":"assignment"}]}`)
	rows, err := parseOutcomeAlignFile(raw)
	if err != nil {
		t.Fatalf("parseOutcomeAlignFile: %v", err)
	}
	if len(rows) != 1 || rows[0].OutcomeID != "out-1" {
		t.Fatalf("unexpected rows: %+v", rows)
	}
}

func TestReportCardsToCSV(t *testing.T) {
	csvBytes, err := reportCardsToCSV([]reportCardRecord{
		{"id": "card-1", "studentId": "u-1", "gradingPeriod": "Q1", "status": "approved", "finalGradePct": 90.5},
	})
	if err != nil {
		t.Fatalf("reportCardsToCSV: %v", err)
	}
	if !strings.Contains(string(csvBytes), "card-1") || !strings.Contains(string(csvBytes), "90.5") {
		t.Fatalf("unexpected CSV: %s", string(csvBytes))
	}
}

func TestStandardsList_Table(t *testing.T) {
	srv := newOutcomesStandardsServer(t)
	defer srv.Close()
	setCfg(srv.URL, "test-key")
	resetOutcomesStandardsFlags()
	standardsListFlags.framework = "ccss-math"

	var buf bytes.Buffer
	standardsListCmd.SetOut(&buf)
	if err := standardsListCmd.RunE(standardsListCmd, nil); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !strings.Contains(buf.String(), "6.RP.1") {
		t.Fatalf("output = %q", buf.String())
	}
}

func TestStandardsImport_JSONFramework(t *testing.T) {
	srv := newOutcomesStandardsServer(t)
	defer srv.Close()
	setCfg(srv.URL, "test-key")
	resetOutcomesStandardsFlags()

	tmp := t.TempDir()
	path := filepath.Join(tmp, "framework.json")
	if err := os.WriteFile(path, []byte(`{"standards":[{"code":"6.RP.1","description":"Ratios","domainCode":"RP","domainName":"Ratios","gradeLevel":"6"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	standardsImportFlags.file = path
	standardsImportFlags.org = "org-1"

	var buf bytes.Buffer
	standardsImportCmd.SetOut(&buf)
	if err := standardsImportCmd.RunE(standardsImportCmd, nil); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !strings.Contains(buf.String(), "Imported") {
		t.Fatalf("output = %q", buf.String())
	}
}

func TestOutcomesAlign_Summary(t *testing.T) {
	srv := newOutcomesStandardsServer(t)
	defer srv.Close()
	setCfg(srv.URL, "test-key")
	resetOutcomesStandardsFlags()

	tmp := t.TempDir()
	alignPath := filepath.Join(tmp, "align.json")
	if err := os.WriteFile(alignPath, []byte(`{"links":[{"outcomeId":"out-1","structureItemId":"item-1","targetKind":"assignment"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	outcomesAlignFlags.file = alignPath

	var buf bytes.Buffer
	outcomesAlignCmd.SetOut(&buf)
	if err := outcomesAlignCmd.RunE(outcomesAlignCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !strings.Contains(buf.String(), "Aligned 1 link") {
		t.Fatalf("output = %q", buf.String())
	}
}

func TestOutcomesReport_JSON(t *testing.T) {
	srv := newOutcomesStandardsServer(t)
	defer srv.Close()
	setCfg(srv.URL, "test-key")
	resetOutcomesStandardsFlags()
	globalFlags.jsonOut = true
	defer func() { globalFlags.jsonOut = false }()

	var buf bytes.Buffer
	outcomesReportCmd.SetOut(&buf)
	if err := outcomesReportCmd.RunE(outcomesReportCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	var report map[string]any
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("json: %v body=%s", err, buf.String())
	}
	if report["masteryThreshold"] == nil {
		t.Fatalf("missing masteryThreshold: %v", report)
	}
}

func TestSbgGet_JSON(t *testing.T) {
	srv := newOutcomesStandardsServer(t)
	defer srv.Close()
	setCfg(srv.URL, "test-key")
	resetOutcomesStandardsFlags()
	globalFlags.jsonOut = true
	defer func() { globalFlags.jsonOut = false }()
	sbgGetFlags.period = "Q1"

	var buf bytes.Buffer
	sbgGetCmd.SetOut(&buf)
	if err := sbgGetCmd.RunE(sbgGetCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("json: %v", err)
	}
	if out["period"] != "Q1" {
		t.Fatalf("period = %v", out["period"])
	}
}

func TestReportCardsExport_PDFManifest(t *testing.T) {
	srv := newOutcomesStandardsServer(t)
	defer srv.Close()
	setCfg(srv.URL, "test-key")
	resetOutcomesStandardsFlags()

	outDir := t.TempDir()
	reportCardsExportFlags.period = "Q1"
	reportCardsExportFlags.format = "pdf"
	reportCardsExportFlags.out = outDir
	reportCardsExportFlags.yes = true

	var buf bytes.Buffer
	reportCardsExportCmd.SetOut(&buf)
	if err := reportCardsExportCmd.RunE(reportCardsExportCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	manifestPath := filepath.Join(outDir, "manifest.csv")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("reading manifest: %v", err)
	}
	if !strings.Contains(string(data), "card-1") {
		t.Fatalf("manifest = %s", string(data))
	}
}

func TestReportCardsExport_RequiresYes(t *testing.T) {
	setCfg("http://unused", "test-key")
	resetOutcomesStandardsFlags()
	reportCardsExportFlags.period = "Q1"
	reportCardsExportFlags.format = "json"
	reportCardsExportFlags.yes = false

	err := reportCardsExportCmd.RunE(reportCardsExportCmd, []string{"CS101"})
	if err == nil || !strings.Contains(err.Error(), "FERPA") {
		t.Fatalf("expected FERPA error, got %v", err)
	}
}

func TestOutcomesMastery_ResolvesUser(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/users/"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"u-1","email":"stu@school.edu","name":"Student"}`))
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/students/"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"studentId":"u-1","period":"Q1","method":"most_recent","scores":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	setCfg(srv.URL, "test-key")
	resetOutcomesStandardsFlags()
	outcomesMasteryFlags.user = "stu@school.edu"
	outcomesMasteryFlags.period = "Q1"

	var buf bytes.Buffer
	outcomesMasteryCmd.SetOut(&buf)
	if err := outcomesMasteryCmd.RunE(outcomesMasteryCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
}