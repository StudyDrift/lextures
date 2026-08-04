package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/ratelimit"
	"github.com/lextures/lextures/server/internal/repos/course"
	ccrepo "github.com/lextures/lextures/server/internal/repos/coursechecklist"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/repos/orgroles"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/service/coursechecklist"
)

func (d Deps) registerCourseChecklistRoutes(r chi.Router) {
	r.Get("/api/v1/courses/{course_code}/checklist", d.handleCourseChecklistGet())
	r.Get("/api/v1/courses/{course_code}/checklist/summary", d.handleCourseChecklistSummary())
	r.Post("/api/v1/courses/{course_code}/checklist/refresh", d.handleCourseChecklistRefresh())
	r.Get("/api/v1/courses/{course_code}/checklist/history", d.handleCourseChecklistHistory())
	r.Post("/api/v1/courses/{course_code}/checklist/items/{item_id}/dismiss", d.handleCourseChecklistDismiss())
	r.Post("/api/v1/courses/{course_code}/checklist/items/{item_id}/restore", d.handleCourseChecklistRestore())
	r.Post("/api/v1/courses/{course_code}/checklist/items/{item_id}/recheck", d.handleCourseChecklistRecheck())
}

// requireCourseChecklistAccess enforces FR-1 / FR-2 / FR-3 for every checklist route.
func (d Deps) requireCourseChecklistAccess(w http.ResponseWriter, r *http.Request) (courseCode string, userID uuid.UUID, ok bool) {
	userID, ok = d.meUserID(w, r)
	if !ok {
		return "", uuid.Nil, false
	}
	courseCode = strings.TrimSpace(chi.URLParam(r, "course_code"))
	if courseCode == "" {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Missing course code.")
		return "", uuid.Nil, false
	}
	ctx := r.Context()
	if d.Pool == nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
		return "", uuid.Nil, false
	}
	courseID, err := course.GetIDByCourseCode(ctx, d.Pool, courseCode)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
		return "", uuid.Nil, false
	}
	if courseID == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
		return "", uuid.Nil, false
	}
	if !auth.AccessKeyAllowsCourse(ctx, *courseID) {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
		return "", uuid.Nil, false
	}

	ga, err := rbac.UserHasPermission(ctx, d.Pool, userID, permGlobalRBACManage)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return "", uuid.Nil, false
	}
	if ga {
		return courseCode, userID, true
	}

	orgID, err := course.CourseOrgID(ctx, d.Pool, courseCode)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
		return "", uuid.Nil, false
	}
	if orgID != nil {
		isAdmin, err := orgroles.UserHasRole(ctx, d.Pool, userID, *orgID, orgroles.RoleOrgAdmin)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return "", uuid.Nil, false
		}
		if isAdmin {
			return courseCode, userID, true
		}
	}

	canEdit, err := courseroles.UserHasPermission(ctx, d.Pool, userID, "course:"+courseCode+":item:create")
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return "", uuid.Nil, false
	}
	if canEdit {
		return courseCode, userID, true
	}

	hasAccess, err := enrollment.UserHasAccess(ctx, d.Pool, courseCode, userID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify course access.")
		return "", uuid.Nil, false
	}
	if hasAccess {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Forbidden.")
		return "", uuid.Nil, false
	}
	apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
	return "", uuid.Nil, false
}

func (d Deps) checklistService() *coursechecklist.Service {
	ttl := d.effectiveConfig().ChecklistSnapshotTTL
	return coursechecklist.NewService(d.Pool, ttl)
}

func (d Deps) checklistRateLimited(w http.ResponseWriter, r *http.Request, key string, limit int) bool {
	if limit <= 0 {
		return false
	}
	limiter := d.buildRateLimiter()
	rule := config.RateLimitRule{Limit: limit, Window: time.Minute}
	dec := limiter.Allow(r.Context(), key, rule, ratelimit.LimitTypeToken)
	if dec.Allowed {
		return false
	}
	ratelimit.RecordExceeded("course_checklist", ratelimit.LimitTypeToken)
	w.Header().Set("Retry-After", strconv.Itoa(dec.RetryAfter))
	apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Too many checklist requests. Try again later.")
	return true
}

func writeChecklistJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeChecklistServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, coursechecklist.ErrCourseNotFound):
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
	case errors.Is(err, coursechecklist.ErrItemNotFound):
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Checklist item not found.")
	case errors.Is(err, ccrepo.ErrInvalidReason):
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid dismiss reason.")
	case errors.Is(err, ccrepo.ErrNoteTooLong):
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Dismiss note must be 500 characters or fewer.")
	default:
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to process checklist request.")
	}
}
