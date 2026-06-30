package httpserver

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/repos/orgroles"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	cfsvc "github.com/lextures/lextures/server/internal/service/customfields"
)

func (d Deps) customFieldsViewerLevel(ctx context.Context, userID, orgID uuid.UUID) (cfsvc.ViewerLevel, error) {
	if d.Pool == nil {
		return cfsvc.ViewerStudent, nil
	}
	return customFieldsViewerLevel(ctx, d.Pool, userID, orgID)
}

func customFieldsViewerLevel(ctx context.Context, pool *pgxpool.Pool, userID, orgID uuid.UUID) (cfsvc.ViewerLevel, error) {
	ga, err := rbac.UserHasPermission(ctx, pool, userID, permGlobalRBACManage)
	if err != nil {
		return cfsvc.ViewerStudent, err
	}
	if ga {
		return cfsvc.ViewerAdmin, nil
	}
	admin, err := orgroles.UserHasRole(ctx, pool, userID, orgID, orgroles.RoleOrgAdmin)
	if err != nil {
		return cfsvc.ViewerStudent, err
	}
	if admin {
		return cfsvc.ViewerAdmin, nil
	}
	viewer, err := orgroles.UserHasRole(ctx, pool, userID, orgID, orgroles.RoleOrgViewer)
	if err != nil {
		return cfsvc.ViewerStudent, err
	}
	if viewer {
		return cfsvc.ViewerAdmin, nil
	}
	var teacher bool
	err = pool.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM "user".user_app_roles uar
  INNER JOIN "user".app_roles ar ON ar.id = uar.role_id
  WHERE uar.user_id = $1 AND ar.name IN ('Teacher', 'Instructor')
)
`, userID).Scan(&teacher)
	if err != nil {
		return cfsvc.ViewerStudent, err
	}
	if teacher {
		return cfsvc.ViewerInstructor, nil
	}
	return cfsvc.ViewerStudent, nil
}
