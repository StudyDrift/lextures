package httpserver

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/httpserver/kernel"
	"github.com/lextures/lextures/server/internal/repos/course"
)

// kernelAccess adapts Deps to kernel.Access so guards wrap the existing
// meUserID / requireCourseAccess family (TD.7 FR-5) without changing messages.
func (d Deps) kernelAccess() kernel.Access {
	return depsAccess{d: d}
}

type depsAccess struct{ d Deps }

func (a depsAccess) Authenticate(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	return a.d.meUserID(w, r)
}

func (a depsAccess) RequireCourseAccess(w http.ResponseWriter, r *http.Request) (string, uuid.UUID, bool) {
	return a.d.requireCourseAccess(w, r)
}

func (a depsAccess) UserHasPermission(ctx context.Context, userID uuid.UUID, perm string) (bool, error) {
	return courseroles.UserHasPermission(ctx, a.d.Pool, userID, perm)
}

func (a depsAccess) LookupCourseID(ctx context.Context, courseCode string) (*uuid.UUID, error) {
	return course.GetIDByCourseCode(ctx, a.d.Pool, courseCode)
}
