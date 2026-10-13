package httpserver

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/repos/orgroles"
	"github.com/lextures/lextures/server/internal/repos/rbac"
)

// userCanOpenCourse is true when the user is enrolled, is a Global Admin, or holds an
// org_admin / org_viewer grant on the course's organization. Org admins must be able to
// open any course in their org (including drafts they are not enrolled in) — see #789.
func userCanOpenCourse(ctx context.Context, pool *pgxpool.Pool, courseCode string, userID uuid.UUID) (bool, error) {
	has, err := enrollment.UserHasAccess(ctx, pool, courseCode, userID)
	if err != nil || has {
		return has, err
	}
	return userHasOrgAdminCourseAccess(ctx, pool, courseCode, userID)
}

// userHasOrgAdminCourseAccess reports whether the user may open the course via org/global admin.
func userHasOrgAdminCourseAccess(ctx context.Context, pool *pgxpool.Pool, courseCode string, userID uuid.UUID) (bool, error) {
	ga, err := rbac.UserHasPermission(ctx, pool, userID, permGlobalRBACManage)
	if err != nil {
		return false, err
	}
	if ga {
		return true, nil
	}
	orgID, err := course.CourseOrgID(ctx, pool, courseCode)
	if err != nil || orgID == nil {
		return false, err
	}
	admin, err := orgroles.UserHasRole(ctx, pool, userID, *orgID, orgroles.RoleOrgAdmin)
	if err != nil || admin {
		return admin, err
	}
	return orgroles.UserHasRole(ctx, pool, userID, *orgID, orgroles.RoleOrgViewer)
}
