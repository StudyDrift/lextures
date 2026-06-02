package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/orgroles"
	"github.com/lextures/lextures/server/internal/repos/sbgreport"
	"github.com/lextures/lextures/server/internal/service/sbgaggregation"
)

func (d Deps) registerSBGReportRoutes(r chi.Router) {
	// Org admin: manage standard domains, mastery scale, CSV import
	r.Get("/api/v1/admin/orgs/{orgId}/sbg/standard-domains", d.handleListStandardDomains())
	r.Post("/api/v1/admin/orgs/{orgId}/sbg/standard-domains", d.handleCreateStandardDomain())
	r.Get("/api/v1/admin/orgs/{orgId}/sbg/mastery-scale", d.handleGetMasteryScale())
	r.Put("/api/v1/admin/orgs/{orgId}/sbg/mastery-scale", d.handlePutMasteryScale())
	r.Post("/api/v1/admin/orgs/{orgId}/sbg/standards/import", d.handleImportStandards())

	// Instructor: course standards list and mastery score recording
	r.Get("/api/v1/courses/{course_code}/sbg/standards", d.handleListCourseStandards())
	r.Post("/api/v1/sbg/mastery-scores", d.handleRecordMasteryScore())

	// Instructor: mastery heatmap for a course+period
	r.Get("/api/v1/courses/{course_code}/sbg/heatmap/{period}", d.handleSBGHeatmap())

	// Instructor/admin/parent: per-student SBG report for a period
	r.Get("/api/v1/students/{studentId}/sbg/{period}", d.handleStudentSBGReport())
}

// ─── Admin: standard domains ──────────────────────────────────────────────────

func (d Deps) handleListStandardDomains() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid org id.")
			return
		}
		if _, ok := d.orgRoleAccess(w, r, orgID, true); !ok {
			return
		}
		domains, err := sbgreport.ListStandardDomains(r.Context(), d.Pool, orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list standard domains.")
			return
		}
		out := make([]map[string]any, 0, len(domains))
		for i := range domains {
			out = append(out, standardDomainToJSON(&domains[i]))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"domains": out})
	}
}

func (d Deps) handleCreateStandardDomain() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid org id.")
			return
		}
		if _, ok := d.orgRoleAccess(w, r, orgID, true); !ok {
			return
		}
		var body struct {
			Code       string  `json:"code"`
			Name       string  `json:"name"`
			GradeLevel *string `json:"gradeLevel"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if strings.TrimSpace(body.Code) == "" || strings.TrimSpace(body.Name) == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "code and name are required.")
			return
		}
		domain, err := sbgreport.CreateStandardDomain(r.Context(), d.Pool, orgID, strings.TrimSpace(body.Code), strings.TrimSpace(body.Name), body.GradeLevel)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create standard domain.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(standardDomainToJSON(domain))
	}
}

// ─── Admin: mastery scale ─────────────────────────────────────────────────────

func (d Deps) handleGetMasteryScale() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid org id.")
			return
		}
		if _, ok := d.orgRoleAccess(w, r, orgID, true); !ok {
			return
		}
		scales, err := sbgreport.ListMasteryScales(r.Context(), d.Pool, orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load mastery scale.")
			return
		}
		out := make([]map[string]any, 0, len(scales))
		for i := range scales {
			out = append(out, masteryScaleToJSON(&scales[i]))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"scale": out})
	}
}

func (d Deps) handlePutMasteryScale() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid org id.")
			return
		}
		if _, ok := d.orgRoleAccess(w, r, orgID, true); !ok {
			return
		}
		var body struct {
			Scale []struct {
				Label string  `json:"label"`
				Value int     `json:"value"`
				Color *string `json:"color"`
			} `json:"scale"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if len(body.Scale) == 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "scale must have at least one entry.")
			return
		}
		entries := make([]sbgreport.MasteryScaleEntry, 0, len(body.Scale))
		for _, e := range body.Scale {
			label := strings.TrimSpace(e.Label)
			if label == "" || e.Value < 1 {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Each scale entry must have label and value >= 1.")
				return
			}
			entries = append(entries, sbgreport.MasteryScaleEntry{Label: label, Value: e.Value, Color: e.Color})
		}
		scales, err := sbgreport.ReplaceMasteryScale(r.Context(), d.Pool, orgID, entries)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save mastery scale.")
			return
		}
		out := make([]map[string]any, 0, len(scales))
		for i := range scales {
			out = append(out, masteryScaleToJSON(&scales[i]))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"scale": out})
	}
}

// ─── Admin: CSV import ────────────────────────────────────────────────────────

func (d Deps) handleImportStandards() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid org id.")
			return
		}
		if _, ok := d.orgRoleAccess(w, r, orgID, true); !ok {
			return
		}

		// Accept both multipart/form-data (file upload) and text/csv direct body.
		var csvReader = r.Body
		ct := r.Header.Get("Content-Type")
		if strings.HasPrefix(ct, "multipart/") {
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Failed to parse multipart form.")
				return
			}
			f, _, err := r.FormFile("file")
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "file field is required.")
				return
			}
			defer f.Close()
			csvReader = f
		}

		result, err := sbgreport.ImportStandardsCSV(r.Context(), d.Pool, orgID, csvReader)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		errs := result.Errors
		if errs == nil {
			errs = []string{}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"domainsCreated":    result.DomainsCreated,
			"standardsImported": result.StandardsImported,
			"errors":            errs,
		})
	}
}

// ─── Instructor: course standards ─────────────────────────────────────────────

func (d Deps) handleListCourseStandards() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		perm := "course:" + courseCode + ":gradebook:view"
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, perm)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Gradebook access required.")
			return
		}
		// Resolve org from course code.
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load org.")
			return
		}
		standards, err := sbgreport.ListStandardsForOrg(r.Context(), d.Pool, orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list standards.")
			return
		}
		out := make([]map[string]any, 0, len(standards))
		for i := range standards {
			out = append(out, standardToJSON(&standards[i]))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"standards": out, "courseCode": courseCode})
	}
}

// ─── Instructor: record mastery score ─────────────────────────────────────────

func (d Deps) handleRecordMasteryScore() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actorID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		var body struct {
			StudentID     string  `json:"studentId"`
			StandardID    string  `json:"standardId"`
			CourseCode    string  `json:"courseCode"`
			GradingPeriod string  `json:"gradingPeriod"`
			ScoreValue    int     `json:"scoreValue"`
			Source        string  `json:"source"`
			SourceID      *string `json:"sourceId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}

		studentID, err := uuid.Parse(strings.TrimSpace(body.StudentID))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid studentId.")
			return
		}
		standardID, err := uuid.Parse(strings.TrimSpace(body.StandardID))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid standardId.")
			return
		}
		if strings.TrimSpace(body.GradingPeriod) == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "gradingPeriod is required.")
			return
		}
		if body.ScoreValue < 1 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "scoreValue must be >= 1.")
			return
		}
		src := strings.TrimSpace(body.Source)
		if src == "" {
			src = "observation"
		}
		switch src {
		case "assignment", "quiz", "observation":
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "source must be assignment, quiz, or observation.")
			return
		}

		courseCode := strings.TrimSpace(body.CourseCode)
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}

		// Verify actor is instructor/admin for the course.
		if !d.canManageReportCard(w, r, actorID, *cid) {
			return
		}

		// Verify the standard belongs to the org.
		orgID, err := sbgreport.GetDomainOrgID(r.Context(), d.Pool, standardID)
		if err != nil || orgID == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Standard not found.")
			return
		}
		actorOrgID, err := organization.OrgIDForUser(r.Context(), d.Pool, actorID)
		if err != nil || actorOrgID != *orgID {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Standard does not belong to your org.")
			return
		}

		var sourceID *uuid.UUID
		if body.SourceID != nil && strings.TrimSpace(*body.SourceID) != "" {
			sid, err := uuid.Parse(strings.TrimSpace(*body.SourceID))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid sourceId.")
				return
			}
			sourceID = &sid
		}

		score, err := sbgreport.RecordMasteryScore(
			r.Context(), d.Pool,
			studentID, standardID, *cid,
			body.GradingPeriod, body.ScoreValue,
			&actorID, src, sourceID,
		)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to record mastery score.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(masteryScoreToJSON(score))
	}
}

// ─── Instructor: mastery heatmap ──────────────────────────────────────────────

func (d Deps) handleSBGHeatmap() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		perm := "course:" + courseCode + ":gradebook:view"
		hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, perm)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Gradebook access required.")
			return
		}
		period := strings.TrimSpace(chi.URLParam(r, "period"))
		if period == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "period is required.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		cells, err := sbgreport.GetHeatmap(r.Context(), d.Pool, *cid, period)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load heatmap.")
			return
		}
		out := make([]map[string]any, 0, len(cells))
		for _, c := range cells {
			out = append(out, map[string]any{
				"studentId":  c.StudentID.String(),
				"standardId": c.StandardID.String(),
				"scoreValue": c.ScoreValue,
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"cells":      out,
			"courseCode": courseCode,
			"period":     period,
		})
	}
}

// ─── Student SBG report ───────────────────────────────────────────────────────

func (d Deps) handleStudentSBGReport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actorID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		studentID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "studentId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid studentId.")
			return
		}
		period := strings.TrimSpace(chi.URLParam(r, "period"))
		if period == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "period is required.")
			return
		}

		// Access control: student themselves, parent, or org admin / instructor.
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, actorID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load org.")
			return
		}
		isAdmin, _ := orgroles.UserHasRole(r.Context(), d.Pool, actorID, orgID, orgroles.RoleOrgAdmin)
		isSelf := actorID == studentID

		if !isAdmin && !isSelf {
			// Allow parent with active link.
			var isParent bool
			_ = d.Pool.QueryRow(r.Context(), `
SELECT EXISTS(
    SELECT 1 FROM "user".parent_links
    WHERE parent_id = $1 AND student_id = $2 AND status = 'active'
)`, actorID, studentID).Scan(&isParent)
			if !isParent {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Access denied.")
				return
			}
		}

		// Fetch all mastery scores for this student across all their courses in the period.
		scores, err := sbgreport.ListMasteryScoresForStudentPeriod(r.Context(), d.Pool, studentID, uuid.Nil, period)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load mastery scores.")
			return
		}

		// Aggregate using most_recent by default; callers may pass ?method=.
		method := sbgaggregation.ParseMethod(r.URL.Query().Get("method"))
		aggregated := sbgaggregation.AggregateForReport(scores, method)

		// Build flat output: per course_id → per standard_id → score.
		type studentStdScore struct {
			CourseID   string `json:"courseId"`
			StandardID string `json:"standardId"`
			ScoreValue int    `json:"scoreValue"`
		}
		var out []studentStdScore
		for sid, stds := range aggregated {
			for stdID, val := range stds {
				out = append(out, studentStdScore{
					CourseID:   sid.String(), // note: sid is StudentID here, courseID tracked separately
					StandardID: stdID.String(),
					ScoreValue: val,
				})
			}
		}
		// Build a simpler flat list keyed by standardId.
		type flatScore struct {
			StandardID string `json:"standardId"`
			ScoreValue int    `json:"scoreValue"`
		}
		flatOut := make([]flatScore, 0)
		for stdID, val := range aggregated[studentID] {
			flatOut = append(flatOut, flatScore{StandardID: stdID.String(), ScoreValue: val})
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"studentId": studentID.String(),
			"period":    period,
			"method":    string(method),
			"scores":    flatOut,
		})
	}
}

// ─── JSON helpers ─────────────────────────────────────────────────────────────

func standardDomainToJSON(d *sbgreport.StandardDomain) map[string]any {
	m := map[string]any{
		"id":        d.ID.String(),
		"orgId":     d.OrgID.String(),
		"code":      d.Code,
		"name":      d.Name,
		"createdAt": d.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if d.GradeLevel != nil {
		m["gradeLevel"] = *d.GradeLevel
	}
	return m
}

func standardToJSON(s *sbgreport.Standard) map[string]any {
	return map[string]any{
		"id":          s.ID.String(),
		"domainId":    s.DomainID.String(),
		"code":        s.Code,
		"description": s.Description,
		"createdAt":   s.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

func masteryScaleToJSON(m *sbgreport.MasteryScale) map[string]any {
	out := map[string]any{
		"id":        m.ID.String(),
		"orgId":     m.OrgID.String(),
		"label":     m.Label,
		"value":     m.Value,
		"createdAt": m.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if m.Color != nil {
		out["color"] = *m.Color
	}
	return out
}

func masteryScoreToJSON(m *sbgreport.MasteryScore) map[string]any {
	out := map[string]any{
		"id":            m.ID.String(),
		"studentId":     m.StudentID.String(),
		"standardId":    m.StandardID.String(),
		"courseId":      m.CourseID.String(),
		"gradingPeriod": m.GradingPeriod,
		"scoreValue":    m.ScoreValue,
		"source":        m.Source,
		"assessedAt":    m.AssessedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if m.AssessedBy != nil {
		out["assessedBy"] = m.AssessedBy.String()
	}
	if m.SourceID != nil {
		out["sourceId"] = m.SourceID.String()
	}
	return out
}
