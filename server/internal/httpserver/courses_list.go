package httpserver

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/repos/orgunit"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/rbac"
)

type coursesListResponse struct {
	Courses []course.CoursePublic `json:"courses"`
}

func (d Deps) handleListCourses() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if d.serveIfPublicAPIToken(w, r, func(pctx *publicAPIContext, tok *auth.APITokenAuth) {
			d.publicAPIListCourses(w, r, pctx, tok)
		}) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		ctx := r.Context()
		termIDStr := strings.TrimSpace(r.URL.Query().Get("term_id"))
		if termIDStr == "" {
			termIDStr = strings.TrimSpace(r.URL.Query().Get("termId"))
		}
		var termFilter *uuid.UUID
		if termIDStr != "" {
			tid, err := uuid.Parse(termIDStr)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid term_id query parameter.")
				return
			}
			termFilter = &tid
		}
		if termFilter != nil {
			slog.Info("course list filtered by term", "term_id", termFilter.String())
		}
		var gradeLevelFilter *string
		if glStr := strings.TrimSpace(r.URL.Query().Get("grade_level")); glStr != "" {
			if !course.ValidGradeLevel(glStr) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid grade_level query parameter.")
				return
			}
			gradeLevelFilter = &glStr
		}
		courses, err := course.ListForEnrolledUser(ctx, d.Pool, userID, gradeLevelFilter)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list courses.")
			return
		}
		ga, err := rbac.UserHasPermission(ctx, d.Pool, userID, permGlobalRBACManage)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list courses.")
			return
		}
		if !ga {
			orgID, err := organization.OrgIDForUser(ctx, d.Pool, userID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list courses.")
				return
			}
			subtrees, err := orgunit.ListSubtreeIDsForUserOrgUnitAdmin(ctx, d.Pool, userID, orgID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list courses.")
				return
			}
			if len(subtrees) > 0 {
				if termFilter != nil {
					unitScoped, err := course.ListForEnrolledUserInOrgUnitsByTerm(ctx, d.Pool, userID, subtrees, *termFilter, gradeLevelFilter)
					if err != nil {
						apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list courses.")
						return
					}
					courses = unitScoped
				} else {
					unitScoped, err := course.ListForEnrolledUserInOrgUnits(ctx, d.Pool, userID, subtrees, gradeLevelFilter)
					if err != nil {
						apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list courses.")
						return
					}
					courses = unitScoped
				}
			} else if termFilter != nil {
				filtered, err := course.ListForEnrolledUserByTerm(ctx, d.Pool, userID, *termFilter, gradeLevelFilter)
				if err != nil {
					apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list courses.")
					return
				}
				courses = filtered
			}
		} else if termFilter != nil {
			filtered, err := course.ListForEnrolledUserByTerm(ctx, d.Pool, userID, *termFilter, gradeLevelFilter)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list courses.")
				return
			}
			courses = filtered
		}
		if courses == nil {
			courses = []course.CoursePublic{}
		}
		if tok, ok := auth.APITokenFromContext(ctx); ok && len(tok.CourseIDs) > 0 {
			allowed := make(map[string]struct{}, len(tok.CourseIDs))
			for _, id := range tok.CourseIDs {
				allowed[id.String()] = struct{}{}
			}
			filtered := make([]course.CoursePublic, 0, len(courses))
			for _, c := range courses {
				if _, ok := allowed[c.ID]; ok {
					filtered = append(filtered, c)
				}
			}
			courses = filtered
		}
		if d.effectiveConfig().FFEnrollmentStateMachine {
			for i := range courses {
				st, changedAt, err := enrollment.ViewerStudentState(ctx, d.Pool, courses[i].CourseCode, userID)
				if err != nil {
					continue
				}
				s := string(st)
				courses[i].ViewerEnrollmentState = &s
				if changedAt != nil {
					t := changedAt.UTC()
					courses[i].ViewerEnrollmentStateChangedAt = &t
				}
			}
		}
		for i := range courses {
			inv, err := enrollment.ViewerInvitationForCourse(ctx, d.Pool, courses[i].CourseCode, userID)
			if err != nil || !inv.Pending || inv.EnrollmentID == nil {
				continue
			}
			courses[i].ViewerEnrollmentInvitationPending = true
			s := inv.EnrollmentID.String()
			courses[i].ViewerPendingEnrollmentID = &s
		}
		if err := course.AttachUserCatalogMeta(ctx, d.Pool, userID, courses); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list courses.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(coursesListResponse{Courses: courses})
	}
}
