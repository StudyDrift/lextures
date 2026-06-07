// Package demographics implements access control and audit logging for student demographic data (plan 13.13).
package demographics

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	repo "github.com/lextures/lextures/server/internal/repos/demographics"
	"github.com/lextures/lextures/server/internal/repos/orgrolegrant"
	"github.com/lextures/lextures/server/internal/repos/orgroles"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/service/ferpa"
)

const (
	ReadPermission   = "compliance:demographics:read:*"
	WritePermission  = "compliance:demographics:write:*"
	ReportPermission = "compliance:demographics:report:*"

	bulkAccessThreshold = 100
	bulkAccessWindow    = 60 * time.Second
)

// CanReadIndividual returns true when the actor may view individual demographic records.
func CanReadIndividual(ctx context.Context, pool *pgxpool.Pool, actorID, orgID, studentID uuid.UUID) (bool, error) {
	if ok, err := rbac.UserHasPermission(ctx, pool, actorID, ReadPermission); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}
	if ok, err := rbac.UserHasPermission(ctx, pool, actorID, permGlobalRBACManage); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}
	if ok, err := orgroles.UserHasRole(ctx, pool, actorID, orgID, orgroles.RoleOrgAdmin); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}
	if ok, err := orgroles.UserHasRole(ctx, pool, actorID, orgID, orgroles.Role("data_analyst")); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}
	// Principal (org_unit_admin): must be scoped to the student's school subtree.
	hasUnitAdmin, err := orgroles.UserHasRole(ctx, pool, actorID, orgID, orgroles.RoleOrgUnitAdmin)
	if err != nil || !hasUnitAdmin {
		return false, err
	}
	roots, err := orgrolegrant.ListOrgUnitAdminRootUnitIDs(ctx, pool, actorID, orgID)
	if err != nil {
		return false, err
	}
	for _, rootID := range roots {
		inSubtree, err := repo.StudentInSchoolSubtree(ctx, pool, studentID, rootID)
		if err != nil {
			return false, err
		}
		if inSubtree {
			return true, nil
		}
	}
	return false, nil
}

// CanWrite returns true when the actor may manually update demographic records.
func CanWrite(ctx context.Context, pool *pgxpool.Pool, actorID, orgID uuid.UUID) (bool, error) {
	if ok, err := rbac.UserHasPermission(ctx, pool, actorID, WritePermission); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}
	if ok, err := rbac.UserHasPermission(ctx, pool, actorID, permGlobalRBACManage); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}
	return orgroles.UserHasRole(ctx, pool, actorID, orgID, orgroles.RoleOrgAdmin)
}

// CanRunReports returns true when the actor may access aggregate demographic reports.
func CanRunReports(ctx context.Context, pool *pgxpool.Pool, actorID, orgID uuid.UUID) (bool, error) {
	if ok, err := rbac.UserHasPermission(ctx, pool, actorID, ReportPermission); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}
	if ok, err := rbac.UserHasPermission(ctx, pool, actorID, permGlobalRBACManage); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}
	if ok, err := orgroles.UserHasRole(ctx, pool, actorID, orgID, orgroles.RoleOrgAdmin); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}
	if ok, err := orgroles.UserHasRole(ctx, pool, actorID, orgID, orgroles.Role("data_analyst")); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}
	return orgroles.UserHasRole(ctx, pool, actorID, orgID, orgroles.RoleOrgViewer)
}

// LogView records a FERPA disclosure for individual demographic access and checks bulk-access threshold.
func LogView(ctx context.Context, pool *pgxpool.Pool, orgID, accessorID, studentID uuid.UUID) error {
	if err := ferpa.LogDisclosure(ctx, pool, orgID, accessorID, studentID, "demographics", "school_official", nil); err != nil {
		return err
	}
	n, err := repo.CountRecentDisclosureViews(ctx, pool, accessorID, bulkAccessWindow)
	if err != nil {
		return err
	}
	if n > bulkAccessThreshold {
		slog.Warn("demographics bulk access alert",
			"accessor_id", accessorID,
			"views_in_window", n,
			"threshold", bulkAccessThreshold,
			"window_seconds", bulkAccessWindow.Seconds(),
		)
	}
	return nil
}

const permGlobalRBACManage = "global:app:rbac:manage"
