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

func TestRedactSettingsSecrets(t *testing.T) {
	m := map[string]any{
		"smtpPassword": "secret-value",
		"nested": map[string]any{"apiKey": "abc"},
		"count": 3,
	}
	redactSettingsSecrets(m)
	if m["smtpPassword"] != secretPlaceholder {
		t.Fatalf("password = %v", m["smtpPassword"])
	}
	nested := m["nested"].(map[string]any)
	if nested["apiKey"] != secretPlaceholder {
		t.Fatalf("apiKey = %v", nested["apiKey"])
	}
}

func TestComputeSettingsApplyDiff(t *testing.T) {
	current := settingsExportFile{
		Locale: map[string]any{"locale": "en"},
	}
	desired := settingsExportFile{
		Locale: map[string]any{"locale": "es"},
		Platform: map[string]any{"ltiEnabled": true},
	}
	diff := computeSettingsApplyDiff(current, desired)
	if !diff.Locale || !diff.Platform {
		t.Fatalf("diff = %+v", diff)
	}
}

func TestSettingsApply_DryRun(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/settings/platform":
			_ = json.NewEncoder(w).Encode(map[string]any{"ltiEnabled": false})
		case "/api/v1/settings/locale":
			_ = json.NewEncoder(w).Encode(map[string]any{"locale": "en"})
		case "/api/v1/settings/timezone":
			_ = json.NewEncoder(w).Encode(map[string]any{"timezone": "UTC"})
		case "/api/v1/settings/system-prompts":
			_ = json.NewEncoder(w).Encode(map[string]any{"prompts": []any{}})
		case "/api/v1/admin/password-policy":
			_ = json.NewEncoder(w).Encode(map[string]any{"minLength": 12})
		case "/api/v1/admin/ai-settings":
			_ = json.NewEncoder(w).Encode(map[string]any{"provider": "openrouter", "byokConfigured": false})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	path := t.TempDir() + "/settings.json"
	raw, _ := json.Marshal(settingsExportFile{
		Version: 1,
		Locale:  map[string]any{"locale": "es"},
	})
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}

	settingsApplyFlags.file = path
	settingsApplyFlags.dryRun = true
	defer func() {
		settingsApplyFlags.file = "settings.json"
		settingsApplyFlags.dryRun = false
	}()

	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	settingsApplyCmd.SetOut(&out)
	if err := settingsApplyCmd.RunE(settingsApplyCmd, nil); err != nil {
		t.Fatalf("apply dry-run: %v", err)
	}
	if !strings.Contains(out.String(), "dry-run") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestFilesUsage_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/courses/cs101/storage-usage" {
			limit := int64(1000)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"used_bytes": 500, "limit_bytes": limit, "percent_used": 50.0,
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	filesUsageFlags.course = "cs101"
	defer func() { filesUsageFlags.course = "" }()

	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	filesUsageCmd.SetOut(&out)
	if err := filesUsageCmd.RunE(filesUsageCmd, nil); err != nil {
		t.Fatalf("files usage: %v", err)
	}
	if !strings.Contains(out.String(), "50.0%") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestStorageQuotasList_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/admin/storage-quotas" {
			limit := int64(1024)
			_ = json.NewEncoder(w).Encode([]map[string]any{{
				"scope": "course", "scope_id": "c1", "used_bytes": 512, "limit_bytes": limit, "percent_used": 50.0,
			}})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	storageQuotasListCmd.SetOut(&out)
	if err := storageQuotasListCmd.RunE(storageQuotasListCmd, nil); err != nil {
		t.Fatalf("storage quotas list: %v", err)
	}
	if !strings.Contains(out.String(), "course") {
		t.Fatalf("output = %q", out.String())
	}
}