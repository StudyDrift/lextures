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

func TestParseCredentialRecipientsCSV(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/recipients.csv"
	if err := os.WriteFile(path, []byte("email,course\na@example.com,CS101\na@example.com,CS101\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	items, err := parseCredentialRecipientsCSV(path)
	if err != nil || len(items) != 2 {
		t.Fatalf("items=%d err=%v", len(items), err)
	}
	deduped := dedupeCredentialRecipients(items)
	if len(deduped) != 1 {
		t.Fatalf("deduped=%d", len(deduped))
	}
}

func TestCredentialsList_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/me/credentials" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"credentials": []any{map[string]any{
					"id": "cred-1", "title": "Completion", "issuedAt": "2026-06-01T00:00:00Z", "revoked": false,
				}},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	var out bytes.Buffer
	credentialsListCmd.SetOut(&out)
	if err := credentialsListCmd.RunE(credentialsListCmd, nil); err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out.String(), "cred-1") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestCredentialsVerify_SkipAuth(t *testing.T) {
	if credentialsVerifyCmd.Annotations[SkipAuthAnnotation] != "true" {
		t.Fatalf("verify should skip auth")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/verify") {
			_ = json.NewEncoder(w).Encode(map[string]any{"valid": true, "status": "Valid", "title": "Cert"})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	setCfg(srv.URL, "")
	var out bytes.Buffer
	credentialsVerifyCmd.SetOut(&out)
	if err := credentialsVerifyCmd.RunE(credentialsVerifyCmd, []string{"cred-1"}); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !strings.Contains(out.String(), "Valid") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestCredentialsIssue_RequiresYes(t *testing.T) {
	credentialsIssueFlags.file = "recipients.csv"
	credentialsIssueFlags.yes = false
	defer func() {
		credentialsIssueFlags.file = ""
		credentialsIssueFlags.yes = false
	}()
	err := credentialsIssueCmd.RunE(credentialsIssueCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("err = %v", err)
	}
}

func TestTranscriptsExport_RequiresYes(t *testing.T) {
	transcriptsExportFlags.yes = false
	defer func() { transcriptsExportFlags.yes = false }()
	err := transcriptsExportCmd.RunE(transcriptsExportCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("err = %v", err)
	}
}

func TestAdvisingNotesList_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/advisor/students/stu-1/notes" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"notes": []any{map[string]any{
					"id": "n1", "content": "On track", "createdAt": "2026-06-01T00:00:00Z", "advisorEmail": "adv@example.com",
				}},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	advisingNotesListFlags.user = "stu-1"
	defer func() { advisingNotesListFlags.user = "" }()
	setCfg(srv.URL, "test-key")
	var out bytes.Buffer
	advisingNotesListCmd.SetOut(&out)
	if err := advisingNotesListCmd.RunE(advisingNotesListCmd, nil); err != nil {
		t.Fatalf("notes list: %v", err)
	}
	if !strings.Contains(out.String(), "On track") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestDegreeProgressGet_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/me/degree-progress" {
			_ = json.NewEncoder(w).Encode(map[string]any{"configured": true, "completionPercent": 72})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	globalFlags.jsonOut = true
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	degreeProgressGetCmd.SetOut(&out)
	if err := degreeProgressGetCmd.RunE(degreeProgressGetCmd, nil); err != nil {
		t.Fatalf("degree progress: %v", err)
	}
	if !strings.Contains(out.String(), "completionPercent") {
		t.Fatalf("output = %q", out.String())
	}
}
