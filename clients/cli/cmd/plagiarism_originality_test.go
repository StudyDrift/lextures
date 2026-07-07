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

type plagiarismOriginalityServerConfig struct {
	plagiarismGetHandler    http.HandlerFunc
	plagiarismPatchHandler  http.HandlerFunc
	submissionsHandler      http.HandlerFunc
	originalitySummary      http.HandlerFunc
	originalityReports      http.HandlerFunc
	originalityEmbed        http.HandlerFunc
	originalityRetry        http.HandlerFunc
}

func newPlagiarismOriginalityServer(t *testing.T, cfg plagiarismOriginalityServerConfig) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/plagiarism-settings"):
			if cfg.plagiarismGetHandler != nil {
				cfg.plagiarismGetHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPatch && strings.HasSuffix(path, "/plagiarism-settings"):
			if cfg.plagiarismPatchHandler != nil {
				cfg.plagiarismPatchHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodGet && strings.Contains(path, "/assignments/") && strings.HasSuffix(path, "/submissions"):
			if cfg.submissionsHandler != nil {
				cfg.submissionsHandler(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/originality/summary"):
			if cfg.originalitySummary != nil {
				cfg.originalitySummary(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/originality/embed-url"):
			if cfg.originalityEmbed != nil {
				cfg.originalityEmbed(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/originality"):
			if cfg.originalityReports != nil {
				cfg.originalityReports(w, r)
			} else {
				http.NotFound(w, r)
			}
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/originality/retry"):
			if cfg.originalityRetry != nil {
				cfg.originalityRetry(w, r)
			} else {
				http.NotFound(w, r)
			}
		default:
			http.NotFound(w, r)
		}
	}))
}

func resetPlagiarismOriginalityFlags() {
	plagiarismSettingsSetFlags.file = ""
	originalityStatusFlags = struct {
		course string
		user   string
	}{}
	originalityGetFlags = struct {
		course string
		user   string
	}{}
	originalityListFlags = struct {
		course string
		page   int
		limit  int
	}{}
	originalitySubmitFlags = struct {
		course string
		user   string
	}{}
	originalityExportFlags = struct {
		course string
		out    string
		yes    bool
	}{}
}

func TestParsePlagiarismPolicyJSON_Valid(t *testing.T) {
	raw := []byte(`{
		"plagiarismChecksEnabled": true,
		"plagiarismProvider": "Turnitin",
		"plagiarismAlertThresholdPct": 25
	}`)
	patch, err := parsePlagiarismPolicyJSON(raw)
	if err != nil {
		t.Fatalf("parsePlagiarismPolicyJSON: %v", err)
	}
	if patch["plagiarismProvider"] != "turnitin" {
		t.Fatalf("provider = %v", patch["plagiarismProvider"])
	}
}

func TestParsePlagiarismPolicyJSON_RejectsUnknownField(t *testing.T) {
	_, err := parsePlagiarismPolicyJSON([]byte(`{"secretKey":"x"}`))
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("err = %v", err)
	}
}

func TestParsePlagiarismPolicyJSON_RejectsInvalidProvider(t *testing.T) {
	_, err := parsePlagiarismPolicyJSON([]byte(`{"plagiarismProvider":"proprietary"}`))
	if err == nil || !strings.Contains(err.Error(), "invalid plagiarismProvider") {
		t.Fatalf("err = %v", err)
	}
}

func TestConfirmOriginalityExport(t *testing.T) {
	if err := confirmOriginalityExport(false); err == nil || !strings.Contains(err.Error(), "FERPA") {
		t.Fatalf("expected FERPA error, got %v", err)
	}
	if err := confirmOriginalityExport(true); err != nil {
		t.Fatalf("expected nil with --yes, got %v", err)
	}
}

func TestPlagiarismSettingsGet_Success(t *testing.T) {
	srv := newPlagiarismOriginalityServer(t, plagiarismOriginalityServerConfig{
		plagiarismGetHandler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(plagiarismSettingsJSON{
				PlagiarismChecksEnabled:     true,
				PlagiarismAlertThresholdPct: 30,
			})
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetPlagiarismOriginalityFlags()

	var out bytes.Buffer
	plagiarismSettingsGetCmd.SetOut(&out)
	if err := plagiarismSettingsGetCmd.RunE(plagiarismSettingsGetCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !strings.Contains(out.String(), "Checks enabled: true") {
		t.Errorf("output = %q", out.String())
	}
}

func TestPlagiarismSettingsSet_Success(t *testing.T) {
	var gotPatch map[string]any
	srv := newPlagiarismOriginalityServer(t, plagiarismOriginalityServerConfig{
		plagiarismPatchHandler: func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&gotPatch)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(plagiarismSettingsJSON{
				PlagiarismChecksEnabled:     true,
				PlagiarismAlertThresholdPct: 20,
			})
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetPlagiarismOriginalityFlags()
	policyPath := filepath.Join(t.TempDir(), "policy.json")
	if err := os.WriteFile(policyPath, []byte(`{"plagiarismChecksEnabled":false,"plagiarismAlertThresholdPct":20}`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	plagiarismSettingsSetFlags.file = policyPath

	plagiarismSettingsSetCmd.SetOut(&bytes.Buffer{})
	if err := plagiarismSettingsSetCmd.RunE(plagiarismSettingsSetCmd, []string{"CS101"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if gotPatch["plagiarismChecksEnabled"] != false {
		t.Fatalf("patch = %v", gotPatch)
	}
}

func TestOriginalityGet_Success(t *testing.T) {
	sim := 12.5
	reportURL := "https://provider.example/report/1"
	srv := newPlagiarismOriginalityServer(t, plagiarismOriginalityServerConfig{
		submissionsHandler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(assignmentSubmissionsBody{
				Submissions: []assignmentSubmissionEntry{
					{ID: "sub-1", SubmittedBy: "u1", SubmittedAt: "2026-01-01T00:00:00Z"},
				},
			})
		},
		originalityReports: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(originalityReportsBody{
				Reports: []originalityReportJSON{
					{Provider: "turnitin", Status: "done", SimilarityPct: &sim, ReportURL: &reportURL},
				},
			})
		},
		originalityEmbed: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(originalityEmbedBody{
				Summary:  originalitySummaryJSON{Provider: "turnitin", SimilarityPct: &sim},
				EmbedURL: &reportURL,
			})
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetPlagiarismOriginalityFlags()
	originalityGetFlags.course = "CS101"
	originalityGetFlags.user = "u1"

	var out bytes.Buffer
	originalityGetCmd.SetOut(&out)
	if err := originalityGetCmd.RunE(originalityGetCmd, []string{"item-001"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !strings.Contains(out.String(), "12.5%") || !strings.Contains(out.String(), reportURL) {
		t.Errorf("output = %q", out.String())
	}
}

func TestOriginalityList_Success(t *testing.T) {
	sim := 8.0
	srv := newPlagiarismOriginalityServer(t, plagiarismOriginalityServerConfig{
		submissionsHandler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(assignmentSubmissionsBody{
				Submissions: []assignmentSubmissionEntry{
					{ID: "sub-1", SubmittedBy: "u1", SubmittedByDisplayName: "Ada"},
					{ID: "sub-2", SubmittedBy: "u2", SubmittedByDisplayName: "Bob"},
				},
			})
		},
		originalitySummary: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(originalitySummaryBody{
				Summary: originalitySummaryJSON{
					Provider:      "turnitin",
					SimilarityPct: &sim,
					DetectedAt:    "2026-01-02T00:00:00Z",
				},
			})
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetPlagiarismOriginalityFlags()
	originalityListFlags.course = "CS101"

	var out bytes.Buffer
	originalityListCmd.SetOut(&out)
	if err := originalityListCmd.RunE(originalityListCmd, []string{"item-001"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !strings.Contains(out.String(), "Ada") || !strings.Contains(out.String(), "8%") {
		t.Errorf("output = %q", out.String())
	}
}

func TestOriginalityExport_RequiresYes(t *testing.T) {
	setCfg("http://localhost:0", "test-key")
	resetPlagiarismOriginalityFlags()
	originalityExportFlags.course = "CS101"
	originalityExportFlags.out = t.TempDir() + "/scores.csv"

	err := originalityExportCmd.RunE(originalityExportCmd, []string{"item-001"})
	if err == nil {
		t.Fatal("expected FERPA refusal without --yes")
	}
	if !strings.Contains(err.Error(), "FERPA") {
		t.Errorf("err = %v", err)
	}
}

func TestOriginalitySubmit_Success(t *testing.T) {
	srv := newPlagiarismOriginalityServer(t, plagiarismOriginalityServerConfig{
		submissionsHandler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(assignmentSubmissionsBody{
				Submissions: []assignmentSubmissionEntry{
					{ID: "sub-1", SubmittedBy: "u1"},
				},
			})
		},
		originalityRetry: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"retried": 1})
		},
	})
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetPlagiarismOriginalityFlags()
	originalitySubmitFlags.course = "CS101"
	originalitySubmitFlags.user = "u1"

	var out bytes.Buffer
	originalitySubmitCmd.SetOut(&out)
	if err := originalitySubmitCmd.RunE(originalitySubmitCmd, []string{"item-001"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !strings.Contains(out.String(), "retried 1") {
		t.Errorf("output = %q", out.String())
	}
}

func TestPlagiarismOriginality_HasSubcommands(t *testing.T) {
	plagiarismNames := map[string]bool{}
	for _, sub := range plagiarismCmd.Commands() {
		plagiarismNames[sub.Name()] = true
	}
	if !plagiarismNames["settings"] {
		t.Error("plagiarism settings subcommand not registered")
	}

	originalityNames := map[string]bool{}
	for _, sub := range originalityCmd.Commands() {
		originalityNames[sub.Name()] = true
	}
	for _, want := range []string{"status", "get", "list", "submit", "export"} {
		if !originalityNames[want] {
			t.Errorf("originality subcommand %q not registered", want)
		}
	}
}

func TestPaginateSlice(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}
	page := paginateSlice(items, 2, 2)
	if len(page) != 2 || page[0] != 3 || page[1] != 4 {
		t.Fatalf("page = %v", page)
	}
}