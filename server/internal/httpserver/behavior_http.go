package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/behavior"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/orgroles"
)

// handleAdminBehaviorCategories is GET/POST /api/v1/admin/orgs/:orgId/behavior/categories
func (d Deps) handleAdminBehaviorCategories() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid org id.")
			return
		}
		if _, ok := d.orgRoleAccess(w, r, orgID, true); !ok {
			return
		}
		switch r.Method {
		case http.MethodGet:
			cats, err := behavior.ListCategories(r.Context(), d.Pool, orgID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list categories.")
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(map[string]any{"categories": categoriesToJSON(cats)})

		case http.MethodPost:
			var body struct {
				Name         string  `json:"name"`
				Type         string  `json:"type"`
				Color        *string `json:"color"`
				SeedDefaults bool    `json:"seedDefaults"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
			if body.SeedDefaults {
				if err := behavior.SeedDefaultCategories(r.Context(), d.Pool, orgID); err != nil {
					apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to seed categories.")
					return
				}
				cats, _ := behavior.ListCategories(r.Context(), d.Pool, orgID)
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				_ = json.NewEncoder(w).Encode(map[string]any{"categories": categoriesToJSON(cats)})
				return
			}
			name := strings.TrimSpace(body.Name)
			catType := strings.TrimSpace(body.Type)
			if name == "" || catType == "" {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "name and type are required.")
				return
			}
			switch catType {
			case "positive", "negative":
			default:
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "type must be positive or negative.")
				return
			}
			cat, err := behavior.UpsertCategory(r.Context(), d.Pool, orgID, name, catType, body.Color)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not create category.")
				return
			}
			w.WriteHeader(http.StatusCreated)
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(categoryToJSON(cat))

		default:
			w.Header().Set("Allow", http.MethodGet+","+http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	}
}

// handleAdminBehaviorCategoryDelete is DELETE /api/v1/admin/orgs/:orgId/behavior/categories/:categoryId
func (d Deps) handleAdminBehaviorCategoryDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.Header().Set("Allow", http.MethodDelete)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		orgID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid org id.")
			return
		}
		if _, ok := d.orgRoleAccess(w, r, orgID, true); !ok {
			return
		}
		catID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "categoryId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid category id.")
			return
		}
		deactivated, err := behavior.DeleteCategory(r.Context(), d.Pool, orgID, catID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, err.Error())
			return
		}
		if !deactivated {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Category not found.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handlePBISAwards is POST /api/v1/pbis/awards
func (d Deps) handlePBISAwards() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		actorID, ok := d.meUserID(w, r)
		if !ok {
			return
		}

		// Parse and validate body before the expensive authorization DB queries.
		var body struct {
			Awards []struct {
				StudentID  string  `json:"studentId"`
				CategoryID string  `json:"categoryId"`
				Points     int     `json:"points"`
				Note       *string `json:"note"`
			} `json:"awards"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if len(body.Awards) == 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "awards must not be empty.")
			return
		}

		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, actorID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load org.")
			return
		}
		isAdmin, _ := orgroles.UserHasRole(r.Context(), d.Pool, actorID, orgID, orgroles.RoleOrgAdmin)
		isTeacher := false
		if !isAdmin {
			_ = d.Pool.QueryRow(r.Context(), `
SELECT EXISTS(
    SELECT 1 FROM course.course_enrollments
    WHERE user_id = $1 AND active AND role IN ('teacher', 'instructor', 'owner', 'ta')
)`, actorID).Scan(&isTeacher)
		}
		if !isAdmin && !isTeacher {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Teacher or admin access required.")
			return
		}

		inputs := make([]behavior.AwardInput, 0, len(body.Awards))
		for _, a := range body.Awards {
			studentID, err := uuid.Parse(strings.TrimSpace(a.StudentID))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid studentId.")
				return
			}
			catID, err := uuid.Parse(strings.TrimSpace(a.CategoryID))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid categoryId.")
				return
			}
			points := a.Points
			if points <= 0 {
				points = 1
			}
			inputs = append(inputs, behavior.AwardInput{
				StudentID:  studentID,
				AwardedBy:  actorID,
				CategoryID: catID,
				OrgID:      orgID,
				Points:     points,
				Note:       a.Note,
			})
		}

		awards, err := behavior.BatchAwardPoints(r.Context(), d.Pool, inputs)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save awards.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"saved":   len(awards),
			"awards":  awardsToJSON(awards),
			"message": "Points awarded.",
		})
	}
}

// handleBehaviorReferrals is POST /api/v1/behavior/referrals
func (d Deps) handleBehaviorReferrals() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		actorID, ok := d.meUserID(w, r)
		if !ok {
			return
		}

		// Parse and validate body before the expensive authorization DB queries.
		var body struct {
			StudentID   string  `json:"studentId"`
			CategoryID  string  `json:"categoryId"`
			SchoolID    *string `json:"schoolId"`
			IncidentAt  string  `json:"incidentAt"`
			Location    *string `json:"location"`
			Description string  `json:"description"`
			Response    *string `json:"response"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		description := strings.TrimSpace(body.Description)
		if description == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "description is required.")
			return
		}

		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, actorID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load org.")
			return
		}
		isAdmin, _ := orgroles.UserHasRole(r.Context(), d.Pool, actorID, orgID, orgroles.RoleOrgAdmin)
		isTeacher := false
		if !isAdmin {
			_ = d.Pool.QueryRow(r.Context(), `
SELECT EXISTS(
    SELECT 1 FROM course.course_enrollments
    WHERE user_id = $1 AND active AND role IN ('teacher', 'instructor', 'owner', 'ta')
)`, actorID).Scan(&isTeacher)
		}
		if !isAdmin && !isTeacher {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Teacher or admin access required.")
			return
		}

		studentID, err := uuid.Parse(strings.TrimSpace(body.StudentID))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid studentId.")
			return
		}
		catID, err := uuid.Parse(strings.TrimSpace(body.CategoryID))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid categoryId.")
			return
		}
		incidentAt := time.Now().UTC()
		if body.IncidentAt != "" {
			incidentAt, err = time.Parse(time.RFC3339, strings.TrimSpace(body.IncidentAt))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid incidentAt; use RFC3339.")
				return
			}
		}
		var schoolID *uuid.UUID
		if body.SchoolID != nil && strings.TrimSpace(*body.SchoolID) != "" {
			sid, err := uuid.Parse(strings.TrimSpace(*body.SchoolID))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid schoolId.")
				return
			}
			schoolID = &sid
		}

		inp := behavior.ReferralInput{
			StudentID:   studentID,
			FiledBy:     actorID,
			OrgID:       orgID,
			SchoolID:    schoolID,
			CategoryID:  catID,
			IncidentAt:  incidentAt,
			Location:    body.Location,
			Description: description,
			Response:    body.Response,
		}
		ref, err := behavior.FileReferral(r.Context(), d.Pool, inp)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to file referral.")
			return
		}
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(referralToJSON(ref, true))
	}
}

// handleStudentBehavior is GET /api/v1/students/:studentId/behavior
func (d Deps) handleStudentBehavior() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		actorID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		studentID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "studentId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid student id.")
			return
		}

		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, actorID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load org.")
			return
		}

		isSelf := actorID == studentID
		isAdmin, _ := orgroles.UserHasRole(r.Context(), d.Pool, actorID, orgID, orgroles.RoleOrgAdmin)
		isTeacher := false
		if !isSelf && !isAdmin {
			_ = d.Pool.QueryRow(r.Context(), `
SELECT EXISTS(
    SELECT 1 FROM course.course_enrollments teacher
    JOIN course.course_enrollments student
        ON student.course_id = teacher.course_id AND student.user_id = $2 AND student.active
    WHERE teacher.user_id = $1 AND teacher.active
      AND teacher.role IN ('teacher', 'instructor', 'owner', 'ta')
)`, actorID, studentID).Scan(&isTeacher)
		}
		if !isSelf && !isAdmin && !isTeacher {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Access denied.")
			return
		}

		fullDesc := isAdmin || isTeacher
		awards, err := behavior.ListAwardsForStudent(r.Context(), d.Pool, studentID, 200)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load behavior awards.")
			return
		}
		referrals, err := behavior.ListReferralsForStudent(r.Context(), d.Pool, studentID, 200, fullDesc)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load behavior referrals.")
			return
		}

		totalPoints := 0
		for _, a := range awards {
			totalPoints += a.Points
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"studentId":   studentID.String(),
			"totalPoints": totalPoints,
			"awards":      awardsToJSON(awards),
			"referrals":   referralsToJSON(referrals, fullDesc),
		})
	}
}

// handleOrgBehaviorDashboard is GET /api/v1/admin/orgs/:orgId/behavior/dashboard
func (d Deps) handleOrgBehaviorDashboard() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		orgID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid org id.")
			return
		}
		if _, ok := d.orgRoleAccess(w, r, orgID, true); !ok {
			return
		}

		weekStart := monday(time.Now().UTC())
		if ws := r.URL.Query().Get("weekStart"); ws != "" {
			parsed, err := time.Parse("2006-01-02", strings.TrimSpace(ws))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid weekStart (YYYY-MM-DD).")
				return
			}
			weekStart = monday(parsed.UTC())
		}

		data, err := behavior.OrgDashboard(r.Context(), d.Pool, orgID, weekStart)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load behavior dashboard.")
			return
		}

		pointsByCat := make([]map[string]any, 0, len(data.PointsByCategory))
		for _, p := range data.PointsByCategory {
			pointsByCat = append(pointsByCat, map[string]any{
				"categoryId":   p.CategoryID.String(),
				"categoryName": p.CategoryName,
				"points":       p.Points,
			})
		}
		refsByCat := make([]map[string]any, 0, len(data.ReferralsByCategory))
		for _, r := range data.ReferralsByCategory {
			refsByCat = append(refsByCat, map[string]any{
				"categoryId":   r.CategoryID.String(),
				"categoryName": r.CategoryName,
				"count":        r.Count,
			})
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"weekStart":           weekStart.Format("2006-01-02"),
			"totalPoints":         data.TotalPoints,
			"totalReferrals":      data.TotalReferrals,
			"pointsByCategory":    pointsByCat,
			"referralsByCategory": refsByCat,
		})
	}
}

// handleParentStudentBehavior is GET /api/v1/parent/students/:sid/behavior
func (d Deps) handleParentStudentBehavior() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		parentID, orgID, ok := d.requireParentViewer(w, r)
		if !ok {
			return
		}
		studentID, ok := d.parseStudentIDParam(w, r)
		if !ok {
			return
		}
		if _, ok := d.requireParentLink(w, r, parentID, orgID, studentID); !ok {
			return
		}

		awards, err := behavior.ListAwardsForStudent(r.Context(), d.Pool, studentID, 200)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load behavior awards.")
			return
		}
		referrals, err := behavior.ListReferralsForStudent(r.Context(), d.Pool, studentID, 200, false)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load behavior referrals.")
			return
		}

		totalPoints := 0
		for _, a := range awards {
			totalPoints += a.Points
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"studentId":   studentID.String(),
			"totalPoints": totalPoints,
			"awards":      awardsToJSON(awards),
			"referrals":   referralsToJSON(referrals, false),
		})
	}
}

// monday returns the Monday of the week containing t.
func monday(t time.Time) time.Time {
	t = t.Truncate(24 * time.Hour)
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return t.AddDate(0, 0, -(weekday - 1))
}

func (d Deps) registerBehaviorRoutes(r chi.Router) {
	// Admin category management
	r.Method(http.MethodGet, "/api/v1/admin/orgs/{orgId}/behavior/categories", d.handleAdminBehaviorCategories())
	r.Method(http.MethodPost, "/api/v1/admin/orgs/{orgId}/behavior/categories", d.handleAdminBehaviorCategories())
	r.Delete("/api/v1/admin/orgs/{orgId}/behavior/categories/{categoryId}", d.handleAdminBehaviorCategoryDelete())
	// PBIS award (teacher/admin)
	r.Post("/api/v1/pbis/awards", d.handlePBISAwards())
	// Behavior referral (teacher/admin)
	r.Post("/api/v1/behavior/referrals", d.handleBehaviorReferrals())
	// Student behavior summary
	r.Get("/api/v1/students/{studentId}/behavior", d.handleStudentBehavior())
	// Admin dashboard
	r.Get("/api/v1/admin/orgs/{orgId}/behavior/dashboard", d.handleOrgBehaviorDashboard())
	// Parent view
	r.Get("/api/v1/parent/students/{sid}/behavior", d.handleParentStudentBehavior())
}

// ─── JSON helpers ─────────────────────────────────────────────────────────────

func categoryToJSON(c *behavior.Category) map[string]any {
	if c == nil {
		return nil
	}
	m := map[string]any{
		"id":     c.ID.String(),
		"orgId":  c.OrgID.String(),
		"name":   c.Name,
		"type":   c.Type,
		"active": c.Active,
	}
	if c.Color != nil {
		m["color"] = *c.Color
	}
	return m
}

func categoriesToJSON(cats []behavior.Category) []map[string]any {
	out := make([]map[string]any, 0, len(cats))
	for i := range cats {
		out = append(out, categoryToJSON(&cats[i]))
	}
	return out
}

func awardToJSON(a *behavior.Award) map[string]any {
	if a == nil {
		return nil
	}
	m := map[string]any{
		"id":           a.ID.String(),
		"studentId":    a.StudentID.String(),
		"awardedBy":    a.AwardedBy.String(),
		"categoryId":   a.CategoryID.String(),
		"categoryName": a.CategoryName,
		"orgId":        a.OrgID.String(),
		"points":       a.Points,
		"awardedAt":    a.AwardedAt.UTC().Format(time.RFC3339Nano),
	}
	if a.Note != nil {
		m["note"] = *a.Note
	}
	return m
}

func awardsToJSON(awards []behavior.Award) []map[string]any {
	out := make([]map[string]any, 0, len(awards))
	for i := range awards {
		out = append(out, awardToJSON(&awards[i]))
	}
	return out
}

func referralToJSON(r *behavior.Referral, showDescription bool) map[string]any {
	if r == nil {
		return nil
	}
	m := map[string]any{
		"id":           r.ID.String(),
		"studentId":    r.StudentID.String(),
		"filedBy":      r.FiledBy.String(),
		"orgId":        r.OrgID.String(),
		"categoryId":   r.CategoryID.String(),
		"categoryName": r.CategoryName,
		"incidentAt":   r.IncidentAt.UTC().Format(time.RFC3339Nano),
		"createdAt":    r.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	if r.SchoolID != nil {
		m["schoolId"] = r.SchoolID.String()
	}
	if r.Location != nil {
		m["location"] = *r.Location
	}
	if showDescription {
		m["description"] = r.Description
	}
	if r.Response != nil {
		m["response"] = *r.Response
	}
	return m
}

func referralsToJSON(refs []behavior.Referral, showDescription bool) []map[string]any {
	out := make([]map[string]any, 0, len(refs))
	for i := range refs {
		out = append(out, referralToJSON(&refs[i], showDescription))
	}
	return out
}
