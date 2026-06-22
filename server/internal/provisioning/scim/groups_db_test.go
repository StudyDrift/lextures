package scim

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	serverdata "github.com/lextures/lextures/server"
	"github.com/lextures/lextures/server/internal/db"
	"github.com/lextures/lextures/server/internal/migrate"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/service/authservice"
)

func TestSCIMGroups_CRUDAndRoleGrants_Pg(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	dsn := os.Getenv("DATABASE_URL")
	if err := migrate.RunWithFS(ctx, serverdata.Migrations, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	institutionID := organization.SeedDefaultOrgID
	baseURL := "http://localhost:8080"
	groupName := "scim-teachers-" + time.Now().Format("150405")

	_, err = pool.Exec(ctx, `
INSERT INTO provisioning.scim_group_mappings (institution_id, display_name, mapping_kind, app_role_name)
VALUES ($1, $2, 'app_role', 'Teacher')
ON CONFLICT DO NOTHING
`, institutionID, groupName)
	if err != nil {
		t.Fatalf("mapping: %v", err)
	}

	email := "scim-group-" + time.Now().Format("20060102150405") + "@example.com"
	ph, err := authservice.PlaceholderPasswordHash()
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	orgID, err := organization.ResolveOrgIDForProvisioning(ctx, pool, institutionID)
	if err != nil {
		t.Fatalf("org: %v", err)
	}
	var uid uuid.UUID
	err = pool.QueryRow(ctx, `
INSERT INTO "user".users (email, password_hash, display_name, org_id)
VALUES ($1, $2, $3, $4)
RETURNING id
`, email, ph, email, orgID).Scan(&uid)
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO provisioning.scim_user_bindings (institution_id, user_id) VALUES ($1, $2)
ON CONFLICT DO NOTHING
`, institutionID, uid); err != nil {
		t.Fatalf("binding: %v", err)
	}

	created, err := CreateGroup(ctx, pool, institutionID, &GroupResource{
		Schemas:     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
		DisplayName: groupName,
	}, baseURL)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	gid, err := uuid.Parse(created.ID)
	if err != nil {
		t.Fatal(err)
	}

	list, err := ListGroups(ctx, pool, institutionID, `displayName eq "`+groupName+`"`, baseURL)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list.Resources) != 1 || list.Resources[0].DisplayName != groupName {
		t.Fatalf("list: %+v", list.Resources)
	}

	patchBody := []byte(`{
		"schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
		"Operations": [{"op": "add", "path": "members", "value": [{"value": "` + uid.String() + `"}]}]
	}`)
	patched, err := PatchGroup(ctx, pool, institutionID, gid.String(), patchBody, baseURL)
	if err != nil {
		t.Fatalf("patch add member: %v", err)
	}
	if len(patched.Members) != 1 || patched.Members[0].Value != uid.String() {
		t.Fatalf("members: %+v", patched.Members)
	}

	hasTeacher, err := userHasAppRole(ctx, pool, uid, "Teacher")
	if err != nil {
		t.Fatalf("role check: %v", err)
	}
	if !hasTeacher {
		t.Fatal("expected Teacher app role after group membership")
	}

	removeBody := []byte(`{
		"schemas": ["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
		"Operations": [{"op": "remove", "path": "members[value eq \"` + uid.String() + `\"]"}]
	}`)
	if _, err := PatchGroup(ctx, pool, institutionID, gid.String(), removeBody, baseURL); err != nil {
		t.Fatalf("patch remove member: %v", err)
	}
	hasTeacher, err = userHasAppRole(ctx, pool, uid, "Teacher")
	if err != nil {
		t.Fatalf("role check after remove: %v", err)
	}
	if hasTeacher {
		t.Fatal("expected Teacher app role revoked after group removal")
	}

	if err := DeleteGroup(ctx, pool, institutionID, gid.String()); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := GetGroup(ctx, pool, institutionID, gid.String(), baseURL); err != ErrNotFound {
		t.Fatalf("expected not found after delete, got %v", err)
	}
}

func userHasAppRole(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, roleName string) (bool, error) {
	var ok bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1
  FROM "user".user_app_roles uar
  INNER JOIN "user".app_roles ar ON ar.id = uar.role_id
  WHERE uar.user_id = $1 AND ar.name = $2
)
`, userID, roleName).Scan(&ok)
	return ok, err
}