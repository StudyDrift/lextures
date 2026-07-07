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

func TestParseUserImportCSV(t *testing.T) {
	path := writeUsersCSV(t, "email,name,role\na@example.com,Alice,student\nb@example.com,Bob,instructor\n")
	rows, err := parseUserImportCSV(path)
	if err != nil {
		t.Fatalf("parseUserImportCSV: %v", err)
	}
	if len(rows) != 2 || rows[0].Email != "a@example.com" || rows[1].Role != "instructor" {
		t.Fatalf("unexpected rows: %+v", rows)
	}
}

func TestRedactSecretsFromJSON(t *testing.T) {
	raw := []byte(`{"email":"a@example.com","temporaryPassword":"secret123","nested":{"token":"abc"}}`)
	out := string(redactSecretsFromJSON(raw))
	if strings.Contains(out, "secret123") || strings.Contains(out, `"abc"`) {
		t.Fatalf("secrets not redacted: %s", out)
	}
	if !strings.Contains(out, "[REDACTED]") {
		t.Fatalf("expected redaction marker: %s", out)
	}
}

func TestUsersImport_DryRun(t *testing.T) {
	path := writeUsersCSV(t, "email,name\na@example.com,Alice\n")
	usersImportFlags.file = path
	usersImportFlags.dryRun = true
	defer func() {
		usersImportFlags.file = ""
		usersImportFlags.dryRun = false
	}()

	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: "http://127.0.0.1:9", APIKey: "test-key"}
	var out bytes.Buffer
	usersImportCmd.SetOut(&out)
	if err := usersImportCmd.RunE(usersImportCmd, nil); err != nil {
		t.Fatalf("users import dry-run: %v", err)
	}
	if !strings.Contains(out.String(), "[dry-run]") || !strings.Contains(out.String(), "no changes") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestUsersSuspend_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/users/"):
			_ = json.NewEncoder(w).Encode(userPublic{ID: "user-1", Email: "a@example.com", Name: "Alice", Role: "student"})
		case r.Method == http.MethodPatch && strings.Contains(r.URL.Path, "/admin-console/users/"):
			_ = json.NewEncoder(w).Encode(adminConsoleUser{ID: "user-1", Email: "a@example.com", Active: false})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	usersSuspendCmd.SetOut(&out)
	if err := usersSuspendCmd.RunE(usersSuspendCmd, []string{"a@example.com"}); err != nil {
		t.Fatalf("users suspend: %v", err)
	}
	if !strings.Contains(out.String(), "Suspended user") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestImportsStatus_WaitComplete(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		status := "running"
		if calls >= 2 {
			status = "complete"
		}
		_ = json.NewEncoder(w).Encode(importJobStatus{
			JobID: "job-1", Status: status, ProcessedRows: 10, CreatedCount: 10,
		})
	}))
	defer srv.Close()

	importsStatusFlags.wait = true
	importsStatusFlags.timeout = 5 * importJobPollInterval
	defer func() {
		importsStatusFlags.wait = false
	}()

	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	importsStatusCmd.SetOut(&out)
	if err := importsStatusCmd.RunE(importsStatusCmd, []string{"job-1"}); err != nil {
		t.Fatalf("imports status --wait: %v", err)
	}
	if !strings.Contains(out.String(), "finished") {
		t.Fatalf("output = %q", out.String())
	}
}

func writeUsersCSV(t *testing.T, content string) string {
	t.Helper()
	path := t.TempDir() + "/users.csv"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}