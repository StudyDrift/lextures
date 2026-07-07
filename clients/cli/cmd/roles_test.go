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

func TestComputeRoleApplyDiff(t *testing.T) {
	current := []rbacRole{{
		ID: "r1", Name: "Admin", Scope: "global",
		Permissions: []rbacPermission{{PermissionString: "rbac:manage"}, {PermissionString: "course:view"}},
	}}
	desired := rolesExportFile{Roles: []rolesExportEntry{{
		Name: "Admin", Scope: "global",
		Permissions: []string{"rbac:manage", "course:manage"},
	}, {
		Name: "Auditor", Scope: "global", Permissions: []string{"audit:read"},
	}}}
	diff := computeRoleApplyDiff(current, desired)
	if len(diff.Create) != 1 || diff.Create[0].Name != "Auditor" {
		t.Fatalf("create = %+v", diff.Create)
	}
	if len(diff.Perms) != 1 || len(diff.Perms[0].Add) != 1 || diff.Perms[0].Add[0] != "course:manage" {
		t.Fatalf("permission diff = %+v", diff.Perms)
	}
}

func TestCallerWouldLockOut(t *testing.T) {
	current := []rbacRole{{
		ID: "r1", Name: "Admin", Scope: "global",
		Permissions: []rbacPermission{{PermissionString: rbacManagePermission}},
	}}
	diff := roleApplyDiff{Perms: []rolePermissionDiff{{
		Role: "Admin", Remove: []string{rbacManagePermission},
	}}}
	roleUsers := map[string][]rbacUserBrief{"r1": {{ID: "user-1"}}}
	if !callerWouldLockOut(current, diff, "user-1", roleUsers) {
		t.Fatal("expected lockout detection")
	}
	if callerWouldLockOut(current, diff, "user-2", roleUsers) {
		t.Fatal("unexpected lockout for unrelated user")
	}
}

func TestRolesList_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/settings/roles" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"roles": []map[string]any{{
					"id": "r1", "name": "Admin", "scope": "global", "permissions": []any{},
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
	rolesListCmd.SetOut(&out)
	if err := rolesListCmd.RunE(rolesListCmd, nil); err != nil {
		t.Fatalf("roles list: %v", err)
	}
	if !strings.Contains(out.String(), "Admin") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestRolesApply_DryRun(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/settings/roles":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"roles": []map[string]any{{
					"id": "r1", "name": "Admin", "scope": "global",
					"permissions": []map[string]any{{"id": "p1", "permissionString": "rbac:manage"}},
				}},
			})
		case "/api/v1/settings/permissions":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"permissions": []map[string]any{{"id": "p1", "permissionString": "rbac:manage"}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	path := t.TempDir() + "/roles.json"
	raw, _ := json.Marshal(rolesExportFile{Roles: []rolesExportEntry{{
		Name: "Admin", Scope: "global", Permissions: []string{"rbac:manage"},
	}}})
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}

	rolesApplyFlags.file = path
	rolesApplyFlags.dryRun = true
	defer func() {
		rolesApplyFlags.file = "roles.json"
		rolesApplyFlags.dryRun = false
	}()

	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	rolesApplyCmd.SetOut(&out)
	if err := rolesApplyCmd.RunE(rolesApplyCmd, nil); err != nil {
		t.Fatalf("roles apply --dry-run: %v", err)
	}
	if !strings.Contains(out.String(), "dry-run") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestPermissionsCheck_Allowed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/me":
			_ = json.NewEncoder(w).Encode(map[string]string{"id": "user-1"})
		case "/api/v1/me/permissions":
			_ = json.NewEncoder(w).Encode(map[string]any{"permissionStrings": []string{"course:manage"}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	permissionsCheckFlags.user = ""
	permissionsCheckFlags.capability = "course:manage"
	defer func() {
		permissionsCheckFlags.capability = ""
	}()

	globalFlags.jsonOut = false
	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	var out bytes.Buffer
	permissionsCheckCmd.SetOut(&out)
	if err := permissionsCheckCmd.RunE(permissionsCheckCmd, nil); err != nil {
		t.Fatalf("permissions check: %v", err)
	}
	if !strings.Contains(out.String(), "allowed") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestRolesGrant_Forbidden(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/settings/roles" {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"message":"permission denied"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	rolesGrantFlags.user = "user-1"
	rolesGrantFlags.role = "Admin"
	defer func() {
		rolesGrantFlags.user = ""
		rolesGrantFlags.role = ""
	}()

	Cfg = &config.Config{Server: srv.URL, APIKey: "test-key"}
	if err := rolesGrantCmd.RunE(rolesGrantCmd, nil); err == nil {
		t.Fatal("expected permission error")
	} else if !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("err = %v", err)
	}
}