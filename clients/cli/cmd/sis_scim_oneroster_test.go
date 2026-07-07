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

func TestRedactSISConnectionsList(t *testing.T) {
	raw := []byte(`{"connections":[{"id":"1","clientSecretRef":"sekret","vendor":"powerschool"}]}`)
	out := redactSISConnectionsList(raw)
	if !strings.Contains(string(out), secretPlaceholder) {
		t.Fatalf("expected redacted secret, got %s", out)
	}
	if strings.Contains(string(out), "sekret") {
		t.Fatal("secret leaked")
	}
}

func TestValidateOneRosterCSVFiles(t *testing.T) {
	dir := t.TempDir()
	users := dir + "/users.csv"
	if err := writeTestCSV(users, []string{"sourcedId", "status"}, [][]string{{"u1", "active"}}); err != nil {
		t.Fatal(err)
	}
	if err := validateOneRosterCSVFiles([]string{users}); err != nil {
		t.Fatalf("validate: %v", err)
	}
	bad := dir + "/users2.csv"
	if err := writeTestCSV(bad, []string{"status"}, [][]string{{"active"}}); err != nil {
		t.Fatal(err)
	}
	if err := validateOneRosterCSVFiles([]string{bad}); err == nil {
		t.Fatal("expected missing sourcedId error")
	}
}

func writeTestCSV(path string, header []string, rows [][]string) error {
	var buf bytes.Buffer
	buf.WriteString(strings.Join(header, ",") + "\n")
	for _, row := range rows {
		buf.WriteString(strings.Join(row, ",") + "\n")
	}
	return os.WriteFile(path, buf.Bytes(), 0o600)
}

func TestSISConfigTest_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/sis/connections"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"connections": []map[string]any{{"id": "conn-1", "vendor": "powerschool", "baseUrl": "https://sis.example", "active": true}},
			})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/test"):
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "message": "Connection test succeeded.", "vendor": "powerschool"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	sisCommonFlags.org = "00000000-0000-0000-0000-000000000001"
	sisConfigTestFlags.connection = "powerschool"
	defer func() {
		sisCommonFlags.org = ""
		sisConfigTestFlags.connection = ""
	}()

	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	sisConfigTestCmd.SetOut(&out)
	if err := sisConfigTestCmd.RunE(sisConfigTestCmd, nil); err != nil {
		t.Fatalf("sis config test: %v", err)
	}
	if !strings.Contains(out.String(), "succeeded") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestScimUsersList_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/scim/v2/Users" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"totalResults": 1,
				"Resources":    []map[string]any{{"id": "u1", "userName": "alice@school.edu", "active": true}},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	scimCommonFlags.token = "scim-token"
	defer func() { scimCommonFlags.token = "" }()

	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	scimUsersListCmd.SetOut(&out)
	if err := scimUsersListCmd.RunE(scimUsersListCmd, nil); err != nil {
		t.Fatalf("scim users list: %v", err)
	}
	if !strings.Contains(out.String(), "alice@school.edu") {
		t.Fatalf("output = %q", out.String())
	}
}