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

func resetQuestionsExtendFlags() {
	questionsExportFlags.bank = ""
	questionsExportFlags.out = ""
	questionsExportFlags.qti = true
	questionsExportFlags.quiet = false
	questionsBanksListFlags.course = ""
	questionsBanksCreateFlags.course = ""
	questionsBanksCreateFlags.name = ""
	questionsBanksCreateFlags.description = ""
}

func TestQuestionsExport_WritesZip(t *testing.T) {
	zipBytes := []byte("PK\x03\x04" + strings.Repeat("x", 100))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/export") {
			if !strings.Contains(r.URL.Path, "bank-abc") {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(zipBytes)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	outPath := filepath.Join(t.TempDir(), "export.zip")
	setCfg(srv.URL, "test-key")
	resetQuestionsExtendFlags()
	questionsExportFlags.bank = "bank-abc"
	questionsExportFlags.out = outPath
	questionsExportFlags.quiet = true

	if err := questionsExportCmd.RunE(questionsExportCmd, nil); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	if len(data) != len(zipBytes) {
		t.Errorf("file size = %d, want %d", len(data), len(zipBytes))
	}
}

func TestQuestionsExport_JSONOutput(t *testing.T) {
	zipBytes := []byte("PK\x03\x04test")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(zipBytes)
	}))
	defer srv.Close()

	outPath := filepath.Join(t.TempDir(), "out.zip")
	setCfg(srv.URL, "test-key")
	resetQuestionsExtendFlags()
	questionsExportFlags.bank = "bank-1"
	questionsExportFlags.out = outPath
	questionsExportFlags.quiet = true
	globalFlags.jsonOut = true
	defer func() { globalFlags.jsonOut = false }()

	var out bytes.Buffer
	questionsExportCmd.SetOut(&out)
	if err := questionsExportCmd.RunE(questionsExportCmd, nil); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["bytes"] != float64(len(zipBytes)) {
		t.Errorf("bytes = %v", result["bytes"])
	}
}

func TestQuestionsBanksList_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/question-pools") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]questionBankPublic{{ID: "pool-1", Name: "Algebra"}})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetQuestionsExtendFlags()
	questionsBanksListFlags.course = "CS101"

	var out bytes.Buffer
	questionsBanksListCmd.SetOut(&out)
	if err := questionsBanksListCmd.RunE(questionsBanksListCmd, nil); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !strings.Contains(out.String(), "pool-1") {
		t.Errorf("output = %q", out.String())
	}
}

func TestQuestionsCmd_HasExportAndBanks(t *testing.T) {
	names := map[string]bool{}
	for _, sub := range questionsCmd.Commands() {
		names[sub.Name()] = true
	}
	for _, want := range []string{"list", "create", "import", "export", "banks"} {
		if !names[want] {
			t.Errorf("questions subcommand %q not registered", want)
		}
	}
}