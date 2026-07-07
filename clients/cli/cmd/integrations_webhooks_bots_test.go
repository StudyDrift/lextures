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

func TestParseWebhookEventTypes(t *testing.T) {
	got := parseWebhookEventTypes([]string{"grade.posted,enrollment.created"})
	if len(got) != 2 {
		t.Fatalf("comma-split got %v", got)
	}
	got = parseWebhookEventTypes([]string{"grade.posted", "assignment.due"})
	if len(got) != 2 {
		t.Fatalf("multi-flag got %v", got)
	}
}

func TestRedactTokenSecret(t *testing.T) {
	m := map[string]any{"token": "secret-value"}
	redactTokenSecret(m)
	if m["token"] != secretPlaceholder {
		t.Fatalf("got %v", m["token"])
	}
}

func TestTestCloudProvider(t *testing.T) {
	if err := testCloudProvider(cloudProviderRow{Provider: "google_drive", Enabled: true, ClientID: "id"}); err != nil {
		t.Fatalf("expected ok: %v", err)
	}
	if err := testCloudProvider(cloudProviderRow{Provider: "google_drive", Enabled: false}); err == nil {
		t.Fatal("expected disabled error")
	}
}

func TestTokensList_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/admin/tokens" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"tokens": []map[string]any{{
					"id": "t1", "label": "ci", "tokenMask": "lx_…abc", "scopes": []string{"courses:read"},
					"isServiceToken": true, "serviceAccountName": "ci-bot",
				}},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	tokensListCmd.SetOut(&out)
	if err := tokensListCmd.RunE(tokensListCmd, nil); err != nil {
		t.Fatalf("tokens list: %v", err)
	}
	if !strings.Contains(out.String(), "ci-bot") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestTokensCreate_RedactsJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/admin/tokens" {
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "t2", "token": "one-time-secret"})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	dir := t.TempDir()
	cfg := dir + "/token.json"
	if err := os.WriteFile(cfg, []byte(`{"label":"ci","serviceAccountName":"bot","scopes":["courses:read"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	tokensCreateFlags.file = cfg
	defer func() { tokensCreateFlags.file = "" }()

	globalFlags.jsonOut = true
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	tokensCreateCmd.SetOut(&out)
	if err := tokensCreateCmd.RunE(tokensCreateCmd, nil); err != nil {
		t.Fatalf("tokens create: %v", err)
	}
	if !strings.Contains(out.String(), secretPlaceholder) || strings.Contains(out.String(), "one-time-secret") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestWebhooksTest_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/test") {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"delivery": map[string]any{"id": 1, "eventType": "grade.posted", "status": "delivered", "attemptCount": 1},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	webhooksTestCmd.SetOut(&out)
	if err := webhooksTestCmd.RunE(webhooksTestCmd, []string{"wh-1"}); err != nil {
		t.Fatalf("webhooks test: %v", err)
	}
	if !strings.Contains(out.String(), "delivered") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestAccessKeysCreate_RequiresFile(t *testing.T) {
	accessKeysCreateFlags.file = ""
	err := accessKeysCreateCmd.RunE(accessKeysCreateCmd, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}