package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lextures/lextures/clients/cli/internal/config"
)

func TestRedactDeveloperAppSecret(t *testing.T) {
	m := map[string]any{"clientSecret": "mcs_secret", "token": "tok_secret"}
	redactDeveloperAppSecret(m)
	if m["clientSecret"] != secretPlaceholder || m["token"] != secretPlaceholder {
		t.Fatalf("secrets not redacted: %+v", m)
	}
}

func TestLTIPlatformConfig_JSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/lti/provider/jwks":
			_, _ = w.Write([]byte(`{"keys":[]}`))
		case "/api/v1/admin/lti/registrations":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"parentPlatforms": []map[string]any{{"deploymentIds": []string{"dep-1"}}},
				"externalTools":   []any{},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	globalFlags.jsonOut = true
	defer func() { globalFlags.jsonOut = false }()
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	ltiPlatformConfigCmd.SetOut(&out)
	if err := ltiPlatformConfigCmd.RunE(ltiPlatformConfigCmd, nil); err != nil {
		t.Fatalf("lti platform-config: %v", err)
	}
	if !strings.Contains(out.String(), "jwksUrl") || !strings.Contains(out.String(), "dep-1") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestDevKeysRotate_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/rotate") {
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "key-2", "token": "new-token-value"})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	devKeysRotateCmd.SetOut(&out)
	if err := devKeysRotateCmd.RunE(devKeysRotateCmd, []string{"key-1"}); err != nil {
		t.Fatalf("dev-keys rotate: %v", err)
	}
	if !strings.Contains(out.String(), "new-token-value") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestDevKeysList_AccessKeys(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/me/access-keys" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"tokens": []map[string]any{{"id": "k1", "label": "ci", "tokenMask": "lex_abc", "scopes": []string{"course:read"}}},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	devKeysListFlags.oauth = false
	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	devKeysListCmd.SetOut(&out)
	if err := devKeysListCmd.RunE(devKeysListCmd, nil); err != nil {
		t.Fatalf("dev-keys list: %v", err)
	}
	if !strings.Contains(out.String(), "ci") {
		t.Fatalf("output = %q", out.String())
	}
}