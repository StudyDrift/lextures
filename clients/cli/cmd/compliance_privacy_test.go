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

func TestFilterDSARsBySubject(t *testing.T) {
	reqs := []dsarRequest{
		{ID: "1", UserID: "u1"},
		{ID: "2", UserID: "u2"},
	}
	got := filterDSARsBySubject(reqs, "u1")
	if len(got) != 1 || got[0].ID != "1" {
		t.Fatalf("filter = %+v", got)
	}
}

func TestGDPRErase_RequiresConfirmSubject(t *testing.T) {
	gdprEraseFlags.subject = "user-1"
	gdprEraseFlags.confirmSubject = ""
	gdprEraseFlags.yes = true
	defer func() {
		gdprEraseFlags.subject = ""
		gdprEraseFlags.confirmSubject = ""
		gdprEraseFlags.yes = false
	}()
	err := gdprEraseCmd.RunE(gdprEraseCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "--confirm-subject") {
		t.Fatalf("err = %v", err)
	}
}

func TestGDPRExport_RequiresYes(t *testing.T) {
	gdprExportFlags.yes = false
	defer func() { gdprExportFlags.yes = false }()
	err := gdprExportCmd.RunE(gdprExportCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("err = %v", err)
	}
}

func TestSOC2EvidenceExport_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/internal/compliance/soc2/evidence-summary" {
			_ = json.NewEncoder(w).Encode(map[string]any{"controls": 12, "evidenceItems": 48})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	dir := t.TempDir()
	soc2EvidenceExportFlags.yes = true
	soc2EvidenceExportFlags.out = dir
	defer func() {
		soc2EvidenceExportFlags.yes = false
		soc2EvidenceExportFlags.out = "."
	}()

	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	if err := soc2EvidenceExportCmd.RunE(soc2EvidenceExportCmd, nil); err != nil {
		t.Fatalf("soc2 export: %v", err)
	}
	data, err := os.ReadFile(dir + "/soc2-evidence.json")
	if err != nil || !strings.Contains(string(data), "evidenceItems") {
		t.Fatalf("file err=%v data=%q", err, string(data))
	}
}

func TestGDPRStatus_QueueSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/compliance/gdpr/dsar" && r.URL.Query().Get("queue") == "true" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"requests": []any{map[string]any{
					"id": "r1", "userId": "u1", "requestType": "access", "status": "pending",
					"requestedAt": "2026-01-01T00:00:00Z", "dueAt": "2026-01-15T00:00:00Z",
				}},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	gdprStatusFlags.queue = true
	defer func() { gdprStatusFlags.queue = false }()

	globalFlags.jsonOut = true
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	gdprStatusCmd.SetOut(&out)
	if err := gdprStatusCmd.RunE(gdprStatusCmd, nil); err != nil {
		t.Fatalf("gdpr status: %v", err)
	}
	if !strings.Contains(out.String(), "r1") {
		t.Fatalf("output = %q", out.String())
	}
}