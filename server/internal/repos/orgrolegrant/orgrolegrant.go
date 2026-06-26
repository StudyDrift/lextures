// Package orgrolegrant stores and queries user.org_role_grants (plan 5.8 org role hierarchy).
package orgrolegrant

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Org-scoped role names stored in user.org_role_grants.role.
const (
	RoleOrgAdmin     = "org_admin"
	RoleOrgUnitAdmin = "org_unit_admin"
	RoleOrgViewer    = "org_viewer"
)

// Row is one grant row for APIs.
type Row struct {
	ID        uuid.UUID
	OrgID     uuid.UUID
	UserID    uuid.UUID
	OrgUnitID *uuid.UUID
	Role      string
	GrantedBy *uuid.UUID
	GrantedAt time.Time
	ExpiresAt *time.Time
}

const activeGrantSQL = `(g.expires_at IS NULL OR g.expires_at > NOW())`

// HasActiveOrgAdmin is true when the user holds an active org_admin grant for orgID (org-wide).
func HasActiveOrgAdmin(ctx context.Context, pool *pgxpool.Pool, userID, orgID uuid.UUID) (bool, error) {
	var ok bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS(
  SELECT 1 FROM "user".org_role_grants g
  WHERE g.org_id = $1 AND g.user_id = $2 AND g.role = $3 AND g.org_unit_id IS NULL AND `+activeGrantSQL+`
)
`, orgID, userID, RoleOrgAdmin).Scan(&ok)
	if err != nil {
		return false, err
	}
	return ok, nil
}

// CanManageOrgRoleGrants is true for global platform check OR active org_admin grant on orgID.
func CanManageOrgRoleGrants(ctx context.Context, pool *pgxpool.Pool, userID, orgID uuid.UUID, isGlobalAdmin bool) (bool, error) {
	if isGlobalAdmin {
		return true, nil
	}
	return HasActiveOrgAdmin(ctx, pool, userID, orgID)
}

// OrgCourseAccess describes how org catalog courses may be listed for a user.
type OrgCourseAccess int

const (
	OrgCourseAccessNone OrgCourseAccess = iota
	OrgCourseAccessAllInOrg
	OrgCourseAccessSubtree
)

// ResolveOrgCourseAccess returns how much of orgID's course catalog userID may see via org grants (not enrollments).
func ResolveOrgCourseAccess(ctx context.Context, pool *pgxpool.Pool, userID, orgID uuid.UUID, isGlobalAdmin bool) (OrgCourseAccess, error) {
	if isGlobalAdmin {
		return OrgCourseAccessAllInOrg, nil
	}
	var adminOrViewer bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS(
  SELECT 1 FROM "user".org_role_grants g
  WHERE g.org_id = $1 AND g.user_id = $2
    AND g.role IN ($3, $4)
    AND g.org_unit_id IS NULL
    AND `+activeGrantSQL+`
)
`, orgID, userID, RoleOrgAdmin, RoleOrgViewer).Scan(&adminOrViewer)
	if err != nil {
		return OrgCourseAccessNone, err
	}
	if adminOrViewer {
		return OrgCourseAccessAllInOrg, nil
	}
	var unitScoped bool
	err = pool.QueryRow(ctx, `
SELECT EXISTS(
  SELECT 1 FROM "user".org_role_grants g
  WHERE g.org_id = $1 AND g.user_id = $2 AND g.role = $3 AND g.org_unit_id IS NOT NULL AND `+activeGrantSQL+`
)
`, orgID, userID, RoleOrgUnitAdmin).Scan(&unitScoped)
	if err != nil {
		return OrgCourseAccessNone, err
	}
	if unitScoped {
		return OrgCourseAccessSubtree, nil
	}
	return OrgCourseAccessNone, nil
}

// ListOrgUnitAdminRootUnitIDs returns org_unit_id roots from org_role_grants for org_unit_admin in this org.
func ListOrgUnitAdminRootUnitIDs(ctx context.Context, pool *pgxpool.Pool, userID, orgID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := pool.Query(ctx, `
SELECT DISTINCT g.org_unit_id
FROM "user".org_role_grants g
WHERE g.org_id = $1 AND g.user_id = $2 AND g.role = $3
  AND g.org_unit_id IS NOT NULL
  AND `+activeGrantSQL+`
`, orgID, userID, RoleOrgUnitAdmin)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
