package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lextures/lextures/clients/cli/internal/auth"
	"github.com/lextures/lextures/clients/cli/internal/config"
)

func TestBuildAuditLogQuery(t *testing.T) {
	q := buildAuditLogQuery(auditLogFilters{
		ActorID: "a1", EventType: "role_grant", From: "2026-01-01T00:00:00Z",
	})
	if !strings.Contains(q, "actorId=a1") || !strings.Contains(q, "eventType=role_grant") {
		t.Fatalf("query = %q", q)
	}
}

func TestMapAdminSearchType(t *testing.T) {
	if mapAdminSearchType("user") != "users" {
		t.Fatal("expected users")
	}
	if mapAdminSearchType("org") != "" {
		t.Fatal("org not supported")
	}
}

func TestIsImpersonationWriteBlock(t *testing.T) {
	err := fmt.Errorf("server error (403): writes_blocked_during_impersonation")
	if !isImpersonationWriteBlock(err) {
		t.Fatal("expected write block detection")
	}
}

func TestAuditLogList_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/compliance/audit-log" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"events": []map[string]any{{
					"eventId": "e1", "eventType": "login", "actorId": "u1", "timestamp": "2026-01-01T00:00:00Z",
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
	auditLogListCmd.SetOut(&out)
	if err := auditLogListCmd.RunE(auditLogListCmd, nil); err != nil {
		t.Fatalf("audit-log list: %v", err)
	}
	if !strings.Contains(out.String(), "e1") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestAdminSearch_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/admin/search" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"users": []map[string]any{{
					"id": "u1", "type": "user", "title": "Jane", "subtitle": "jane@example.com", "path": "/users/u1",
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
	adminSearchCmd.SetOut(&out)
	if err := adminSearchCmd.RunE(adminSearchCmd, []string{"jane"}); err != nil {
		t.Fatalf("admin search: %v", err)
	}
	if !strings.Contains(out.String(), "Jane") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestImpersonateWhoami_Impersonating(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/me" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "target-1", "email": "target@example.com",
				"impersonating": map[string]any{"adminId": "admin-1"},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: srv.URL, APIKey: "imp-token"}
	var out bytes.Buffer
	impersonateWhoamiCmd.SetOut(&out)
	if err := impersonateWhoamiCmd.RunE(impersonateWhoamiCmd, nil); err != nil {
		t.Fatalf("whoami: %v", err)
	}
	if !strings.Contains(out.String(), "IMPERSONATION ACTIVE") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestImpersonationStore_RoundTrip(t *testing.T) {
	path := t.TempDir() + "/imp.json"
	store := auth.NewImpersonationStoreAt(path)
	sess := &auth.ImpersonationSession{
		RealAccessToken: "real", ImpersonationToken: "imp", TargetUserID: "u1",
	}
	if err := store.Save("default", sess); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load("default")
	if err != nil || loaded.ImpersonationToken != "imp" {
		t.Fatalf("load = %+v err=%v", loaded, err)
	}
}