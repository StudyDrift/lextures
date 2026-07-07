package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/lextures/lextures/clients/cli/internal/config"
)

func TestResolveEmailTemplateSlot(t *testing.T) {
	if got := resolveEmailTemplateSlot("welcome", "es"); got != "welcome.es" {
		t.Fatalf("got %q", got)
	}
}

func TestValidateBannerWindow(t *testing.T) {
	from := time.Now().UTC().Format(time.RFC3339)
	until := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	if err := validateBannerWindow(from, until); err != nil {
		t.Fatalf("valid window: %v", err)
	}
	if err := validateBannerWindow(until, from); err == nil {
		t.Fatal("expected invalid window")
	}
}

func TestEmailTemplatesList_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/admin-console/email-templates" {
			_ = json.NewEncoder(w).Encode([]map[string]any{{
				"id": "welcome", "description": "Welcome email", "hasCustom": true,
			}})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	emailTemplatesListCmd.SetOut(&out)
	if err := emailTemplatesListCmd.RunE(emailTemplatesListCmd, nil); err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out.String(), "welcome") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestBannersCreate_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/admin/banners" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "b1", "scope": "org", "message": "Maintenance 2am", "severity": "warning", "isActive": true,
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	bannersWriteFlags.message = "Maintenance 2am"
	bannersWriteFlags.severity = "warning"
	defer func() {
		bannersWriteFlags.message = ""
		bannersWriteFlags.severity = "info"
	}()

	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	bannersCreateCmd.SetOut(&out)
	if err := bannersCreateCmd.RunE(bannersCreateCmd, nil); err != nil {
		t.Fatalf("create: %v", err)
	}
	if !strings.Contains(out.String(), "b1") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestEmailTemplatesSet_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/email-templates/welcome.es") {
			_ = json.NewEncoder(w).Encode(map[string]any{"slotId": "welcome.es"})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	path := t.TempDir() + "/welcome.html"
	if err := os.WriteFile(path, []byte("<p>Hola</p>"), 0o600); err != nil {
		t.Fatal(err)
	}
	emailTemplatesSetFlags.file = path
	emailTemplatesSetFlags.locale = "es"
	defer func() {
		emailTemplatesSetFlags.file = ""
		emailTemplatesSetFlags.locale = ""
	}()

	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	emailTemplatesSetCmd.SetOut(&out)
	if err := emailTemplatesSetCmd.RunE(emailTemplatesSetCmd, []string{"welcome"}); err != nil {
		t.Fatalf("set: %v", err)
	}
	if !strings.Contains(out.String(), "welcome.es") {
		t.Fatalf("output = %q", out.String())
	}
}