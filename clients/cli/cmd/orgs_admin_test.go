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

func TestOrgsUpdate_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch && strings.Contains(r.URL.Path, "/admin/orgs/") {
			_ = json.NewEncoder(w).Encode(sampleOrg("org-1", "west", "West", "active"))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	orgsUpdateFlags.name = "West Updated"
	defer func() { orgsUpdateFlags.name = "" }()

	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	orgsUpdateCmd.SetOut(&out)
	if err := orgsUpdateCmd.RunE(orgsUpdateCmd, []string{"org-1"}); err != nil {
		t.Fatalf("orgs update: %v", err)
	}
	if !strings.Contains(out.String(), "Updated organization") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestTermsCreate_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/terms") {
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(termPublic{
				ID: "term-1", Name: "Fall2026", StartDate: "2026-08-15", EndDate: "2026-12-15", Status: "active",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	termsCreateFlags.name = "Fall2026"
	termsCreateFlags.start = "2026-08-15"
	termsCreateFlags.end = "2026-12-15"
	defer func() {
		termsCreateFlags.name = ""
		termsCreateFlags.start = ""
		termsCreateFlags.end = ""
	}()

	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	termsCreateCmd.SetOut(&out)
	if err := termsCreateCmd.RunE(termsCreateCmd, []string{"org-1"}); err != nil {
		t.Fatalf("terms create: %v", err)
	}
	if !strings.Contains(out.String(), "Created term") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestOrgUnitsList_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/units") {
			_ = json.NewEncoder(w).Encode(orgUnitsListBody{
				Units: []orgUnitRow{{ID: "u1", Name: "District", UnitType: "district", Status: "active"}},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	orgUnitsListCmd.SetOut(&out)
	if err := orgUnitsListCmd.RunE(orgUnitsListCmd, []string{"org-1"}); err != nil {
		t.Fatalf("org-units list: %v", err)
	}
	if !strings.Contains(out.String(), "District") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestReadJSONSettingsFile(t *testing.T) {
	path := t.TempDir() + "/settings.json"
	if err := os.WriteFile(path, []byte(`{"timezone":"America/Chicago"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	out, err := readJSONSettingsFile(path)
	if err != nil {
		t.Fatalf("readJSONSettingsFile: %v", err)
	}
	if out["timezone"] != "America/Chicago" {
		t.Fatalf("timezone = %v", out["timezone"])
	}
}