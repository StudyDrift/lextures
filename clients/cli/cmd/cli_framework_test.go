package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCompletion_Bash(t *testing.T) {
	var out bytes.Buffer
	completionCmd.SetOut(&out)
	if err := completionCmd.RunE(completionCmd, []string{"bash"}); err != nil {
		t.Fatalf("completion: %v", err)
	}
	if !strings.Contains(out.String(), "lextures") {
		t.Fatalf("output missing lextures: %q", out.String())
	}
}

func TestWhoami_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/me" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "u1", "email": "a@example.com", "displayName": "Ada",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	setCfg(srv.URL, "tok")
	var out bytes.Buffer
	whoamiCmd.SetOut(&out)
	if err := whoamiCmd.RunE(whoamiCmd, nil); err != nil {
		t.Fatalf("whoami: %v", err)
	}
	if !strings.Contains(out.String(), "a@example.com") {
		t.Fatalf("out=%q", out.String())
	}
}

func TestSessionRevokeConfirmMessage(t *testing.T) {
	msg := sessionRevokeConfirmMessage(true, false)
	if !strings.Contains(msg, "--yes") {
		t.Fatalf("msg=%q", msg)
	}
}