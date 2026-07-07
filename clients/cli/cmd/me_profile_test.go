package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMeGet_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/me" {
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "u1", "email": "me@example.com"})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	setCfg(srv.URL, "tok")
	var out strings.Builder
	meGetCmd.SetOut(&out)
	if err := meGetCmd.RunE(meGetCmd, nil); err != nil {
		t.Fatalf("me get: %v", err)
	}
	if !strings.Contains(out.String(), "me@example.com") {
		t.Fatalf("out=%q", out.String())
	}
}

func TestMeSessionsRevoke_RequiresYes(t *testing.T) {
	meSessionsRevokeFlags.all = true
	meSessionsRevokeFlags.yes = false
	defer func() {
		meSessionsRevokeFlags.all = false
		meSessionsRevokeFlags.yes = false
	}()
	err := meSessionsRevokeCmd.RunE(meSessionsRevokeCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("err=%v", err)
	}
}

func TestValidateParentChildID(t *testing.T) {
	children := []parentChildRow{{ID: "c1", DisplayName: "Kid"}}
	if err := validateParentChildID(children, "c1"); err != nil {
		t.Fatalf("err=%v", err)
	}
	if err := validateParentChildID(children, "other"); err == nil {
		t.Fatal("expected error for non-linked child")
	}
}

func TestRevokeAllSessionsExceptCurrent(t *testing.T) {
	sessions := []sessionRow{
		{ID: "s1", Current: true},
		{ID: "s2", Current: false},
	}
	ids := revokeAllSessionsExceptCurrent(sessions, false)
	if len(ids) != 1 || ids[0] != "s2" {
		t.Fatalf("ids=%v", ids)
	}
}