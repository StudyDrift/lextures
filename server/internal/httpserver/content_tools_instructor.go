package httpserver

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	ctmodel "github.com/lextures/lextures/server/internal/models/contenttools"
	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	ctsvc "github.com/lextures/lextures/server/internal/service/contenttools"
)

func (d Deps) registerContentToolsInstructorRoutes(r chi.Router) {
	r.Get("/api/v1/courses/{course_code}/content-tools/instances/{instance_id}/states", d.handleContentToolsRosterList())
	r.Get("/api/v1/courses/{course_code}/content-tools/instances/{instance_id}/states/export", d.handleContentToolsRosterExport())
	r.Get("/api/v1/courses/{course_code}/content-tools/instances/{instance_id}/states/{enrollment_id}", d.handleContentToolsStateDetail())
	r.Post("/api/v1/courses/{course_code}/content-tools/state-resets", d.handleContentToolsStateReset())
	r.Get("/api/v1/courses/{course_code}/content-tools/state-resets", d.handleContentToolsStateResetsList())
	r.Post("/api/v1/courses/{course_code}/content-tools/state-resets/{reset_id}/restore", d.handleContentToolsStateResetRestore())
	r.Get("/api/v1/courses/{course_code}/content-tools/reset-jobs/{job_id}", d.handleContentToolsResetJobGet())
	r.Post("/api/v1/courses/{course_code}/content-tools/instances/{instance_id}/self-reset", d.handleContentToolsSelfReset())
}

func (d Deps) viewerCanGradeContentTools(ctx context.Context, courseCode string, viewer uuid.UUID) (bool, error) {
	return courseroles.UserHasPermission(ctx, d.Pool, viewer, "course:"+courseCode+":gradebook:view")
}

func (d Deps) contentToolsTASectionFilter(ctx context.Context, courseID uuid.UUID, courseCode string, viewer uuid.UUID) ([]uuid.UUID, error) {
	return enrollment.GradebookStudentSectionFilter(ctx, d.Pool, courseID, courseCode, viewer, true)
}

func (d Deps) requireContentToolsGradeRead(w http.ResponseWriter, r *http.Request) (courseCode string, viewer uuid.UUID, courseID uuid.UUID, ok bool) {
	courseCode, viewer, courseID, ok = d.requireContentToolsCourse(w, r)
	if !ok {
		return "", uuid.Nil, uuid.Nil, false
	}
	can, err := d.viewerCanGradeContentTools(r.Context(), courseCode, viewer)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return "", uuid.Nil, uuid.Nil, false
	}
	if !can {
		// Authors with item:create may also manage tools.
		canEdit, err := d.viewerCanEditContentTools(r.Context(), courseCode, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return "", uuid.Nil, uuid.Nil, false
		}
		if !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return "", uuid.Nil, uuid.Nil, false
		}
	}
	return courseCode, viewer, courseID, true
}

func (d Deps) handleContentToolsRosterList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, courseID, ok := d.requireContentToolsGradeRead(w, r)
		if !ok {
			return
		}
		instanceID, err := uuid.Parse(chi.URLParam(r, "instance_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid instance id.")
			return
		}
		inst, err := ctrepo.GetInstance(r.Context(), d.Pool, courseID, instanceID)
		if err != nil || inst == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Instance not found.")
			return
		}
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
		pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
		if pageSize <= 0 {
			pageSize = 50
		}
		var sectionID *uuid.UUID
		if s := strings.TrimSpace(r.URL.Query().Get("sectionId")); s != "" {
			id, err := uuid.Parse(s)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid sectionId.")
				return
			}
			sectionID = &id
		}
		taSections, err := d.contentToolsTASectionFilter(r.Context(), courseID, courseCode, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to resolve section scope.")
			return
		}
		rows, total, err := ctrepo.ListInstanceRoster(r.Context(), d.Pool, ctrepo.RosterListParams{
			InstanceID: instanceID,
			CourseID:   courseID,
			Status:     r.URL.Query().Get("status"),
			SectionID:  sectionID,
			SectionIDs: taSections,
			Page:       page,
			PageSize:   pageSize,
		})
		if err != nil {
			if strings.Contains(err.Error(), "invalid status") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid status filter.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load roster.")
			return
		}
		items := make([]ctmodel.RosterStateRow, 0, len(rows))
		for _, row := range rows {
			item := ctmodel.RosterStateRow{
				EnrollmentID:     row.EnrollmentID,
				DisplayName:      row.DisplayName,
				Status:           row.Status,
				InteractionCount: row.InteractionCount,
				LastInteractedAt: row.LastInteractedAt,
				ResetCount:       row.ResetCount,
			}
			if row.ScoreRaw != nil && row.ScoreMax != nil {
				item.Score = &ctmodel.ToolScore{Raw: *row.ScoreRaw, Max: *row.ScoreMax}
			}
			items = append(items, item)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(ctmodel.RosterStatesResponse{
			Items:      items,
			Page:       page,
			PageSize:   pageSize,
			TotalCount: total,
		})
	}
}

func (d Deps) handleContentToolsStateDetail() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, courseID, ok := d.requireContentToolsGradeRead(w, r)
		if !ok {
			return
		}
		instanceID, err := uuid.Parse(chi.URLParam(r, "instance_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid instance id.")
			return
		}
		enrollmentID, err := uuid.Parse(chi.URLParam(r, "enrollment_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid enrollment id.")
			return
		}
		inst, err := ctrepo.GetInstance(r.Context(), d.Pool, courseID, instanceID)
		if err != nil || inst == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Instance not found.")
			return
		}
		taSections, err := d.contentToolsTASectionFilter(r.Context(), courseID, courseCode, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to resolve section scope.")
			return
		}
		if len(taSections) > 0 {
			okSec, err := enrollmentInSections(r.Context(), d.Pool, enrollmentID, taSections)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify section scope.")
				return
			}
			if !okSec {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Enrollment is outside your section scope.")
				return
			}
		}
		displayName, studentID, err := ctrepo.EnrollmentDisplayName(r.Context(), d.Pool, enrollmentID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Enrollment not found.")
			return
		}
		st, err := ctrepo.GetInstanceStateDetail(r.Context(), d.Pool, instanceID, enrollmentID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load state.")
			return
		}
		orgID, _ := course.CourseOrgID(r.Context(), d.Pool, courseCode)
		if orgID != nil && studentID != viewer {
			if err := ctsvc.LogStateDetailAccess(r.Context(), d.Pool, *orgID, viewer, studentID); err != nil {
				slog.Warn("content_tools.ferpa_log", "err", err)
			}
		}
		env := contentToolsStateEnvelope(instanceID, st)
		var scoreRaw, scoreMax *float64
		if st != nil {
			scoreRaw, scoreMax = st.ScoreRaw, st.ScoreMax
		}
		summary := ctsvc.SummarizeState(inst.ToolID, env.State, env.Status, scoreRaw, scoreMax)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(ctmodel.StateDetailResponse{
			EnrollmentID: enrollmentID,
			DisplayName:  displayName,
			Summary:      summary,
			State:        env,
		})
	}
}

func (d Deps) handleContentToolsRosterExport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, courseID, ok := d.requireContentToolsGradeRead(w, r)
		if !ok {
			return
		}
		if !d.contentToolsRateLimit(w, r, viewer, "ct_export", 5) {
			return
		}
		instanceID, err := uuid.Parse(chi.URLParam(r, "instance_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid instance id.")
			return
		}
		inst, err := ctrepo.GetInstance(r.Context(), d.Pool, courseID, instanceID)
		if err != nil || inst == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Instance not found.")
			return
		}
		taSections, err := d.contentToolsTASectionFilter(r.Context(), courseID, courseCode, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to resolve section scope.")
			return
		}
		rows, _, err := ctrepo.ListInstanceRoster(r.Context(), d.Pool, ctrepo.RosterListParams{
			InstanceID: instanceID,
			CourseID:   courseID,
			SectionIDs: taSections,
			Page:       1,
			PageSize:   10000,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to export roster.")
			return
		}
		format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
		if format == "" {
			format = "csv"
		}
		switch format {
		case "json":
			items := make([]map[string]any, 0, len(rows))
			for _, row := range rows {
				m := map[string]any{
					"enrollmentId":     row.EnrollmentID.String(),
					"displayName":      row.DisplayName,
					"status":           row.Status,
					"interactionCount": row.InteractionCount,
					"resetCount":       row.ResetCount,
					"summary":          ctsvc.SummarizeState(inst.ToolID, nil, row.Status, row.ScoreRaw, row.ScoreMax),
				}
				if row.ScoreRaw != nil {
					m["scoreRaw"] = *row.ScoreRaw
				}
				if row.ScoreMax != nil {
					m["scoreMax"] = *row.ScoreMax
				}
				if row.LastInteractedAt != nil {
					m["lastInteractedAt"] = row.LastInteractedAt.UTC().Format(time.RFC3339)
				}
				items = append(items, m)
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Content-Disposition", `attachment; filename="content-tool-states.json"`)
			_ = json.NewEncoder(w).Encode(map[string]any{"items": items})
		case "csv":
			w.Header().Set("Content-Type", "text/csv; charset=utf-8")
			w.Header().Set("Content-Disposition", `attachment; filename="content-tool-states.csv"`)
			cw := csv.NewWriter(w)
			_ = cw.Write([]string{"enrollmentId", "displayName", "status", "scoreRaw", "scoreMax", "interactionCount", "lastInteractedAt", "resetCount", "summary"})
			for _, row := range rows {
				scoreRaw, scoreMax, last := "", "", ""
				if row.ScoreRaw != nil {
					scoreRaw = strconv.FormatFloat(*row.ScoreRaw, 'f', -1, 64)
				}
				if row.ScoreMax != nil {
					scoreMax = strconv.FormatFloat(*row.ScoreMax, 'f', -1, 64)
				}
				if row.LastInteractedAt != nil {
					last = row.LastInteractedAt.UTC().Format(time.RFC3339)
				}
				_ = cw.Write([]string{
					row.EnrollmentID.String(),
					row.DisplayName,
					row.Status,
					scoreRaw,
					scoreMax,
					strconv.Itoa(row.InteractionCount),
					last,
					strconv.Itoa(row.ResetCount),
					ctsvc.SummarizeState(inst.ToolID, nil, row.Status, row.ScoreRaw, row.ScoreMax),
				})
			}
			cw.Flush()
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "format must be csv or json.")
		}
	}
}
