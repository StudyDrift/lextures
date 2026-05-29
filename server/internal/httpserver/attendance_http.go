package httpserver

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/attendance"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/orgroles"
)

// handleSectionAttendance is GET/PUT /api/v1/sections/:sectionId/attendance/:date
func (d Deps) handleSectionAttendance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actorID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		sectionID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "sectionId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid section id.")
			return
		}
		dateStr := strings.TrimSpace(chi.URLParam(r, "date"))
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid date format; use YYYY-MM-DD.")
			return
		}

		// Verify caller has access: must be the section instructor or org admin.
		orgID, authOK := d.requireSectionAccess(w, r, actorID, sectionID)
		if !authOK {
			return
		}

		switch r.Method {
		case http.MethodGet:
			records, err := attendance.ListForSection(r.Context(), d.Pool, sectionID, date)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load attendance.")
				return
			}
			roster, err := attendance.ListRosterForSection(r.Context(), d.Pool, sectionID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load roster.")
				return
			}
			codes, err := attendance.ListCodes(r.Context(), d.Pool, orgID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load codes.")
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"records": recordsToJSON(records),
				"roster":  rosterToJSON(roster),
				"codes":   codesToJSON(codes),
			})

		case http.MethodPut:
			var body struct {
				Records []struct {
					StudentID string  `json:"studentId"`
					CodeID    string  `json:"codeId"`
					Period    *string `json:"period"`
					Note      *string `json:"note"`
					SchoolID  *string `json:"schoolId"`
				} `json:"records"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
			// Instructors can only edit within 5 days; org admins can always edit.
			isAdmin, _ := orgroles.UserHasRole(r.Context(), d.Pool, actorID, orgID, orgroles.RoleOrgAdmin)
			if !isAdmin && !attendance.IsWithinEditWindow(date) {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden,
					"Attendance edits are limited to 5 days; contact your administrator.")
				return
			}
			rows := make([]attendance.UpsertRow, 0, len(body.Records))
			for _, rec := range body.Records {
				studentID, err := uuid.Parse(strings.TrimSpace(rec.StudentID))
				if err != nil {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid studentId.")
					return
				}
				codeID, err := uuid.Parse(strings.TrimSpace(rec.CodeID))
				if err != nil {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid codeId.")
					return
				}
				var schoolID *uuid.UUID
				if rec.SchoolID != nil && strings.TrimSpace(*rec.SchoolID) != "" {
					sid, err := uuid.Parse(strings.TrimSpace(*rec.SchoolID))
					if err != nil {
						apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid schoolId.")
						return
					}
					schoolID = &sid
				}
				rows = append(rows, attendance.UpsertRow{
					StudentID:  studentID,
					SectionID:  sectionID,
					SchoolID:   schoolID,
					Date:       date,
					Period:     rec.Period,
					CodeID:     codeID,
					Note:       rec.Note,
					RecordedBy: &actorID,
				})
			}
			if err := attendance.BatchUpsert(r.Context(), d.Pool, rows); err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save attendance.")
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"saved":   len(rows),
				"message": "Attendance saved.",
			})

		default:
			w.Header().Set("Allow", http.MethodGet+","+http.MethodPut)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	}
}

// handleStudentAttendance is GET /api/v1/students/:studentId/attendance
func (d Deps) handleStudentAttendance() http.HandlerFunc {
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
		if actorID != studentID {
			// Check if actor is an org admin or is a course instructor with this student.
			orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, actorID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load org.")
				return
			}
			isAdmin, _ := orgroles.UserHasRole(r.Context(), d.Pool, actorID, orgID, orgroles.RoleOrgAdmin)
			if !isAdmin {
				// Allow teachers who share a course section with this student.
				var shared bool
				_ = d.Pool.QueryRow(r.Context(), `
SELECT EXISTS(
    SELECT 1 FROM course.course_enrollments teacher
    JOIN course.course_enrollments student
        ON student.course_id = teacher.course_id AND student.user_id = $2 AND student.active
    WHERE teacher.user_id = $1 AND teacher.active
      AND teacher.role IN ('teacher', 'instructor', 'owner', 'ta')
)`, actorID, studentID).Scan(&shared)
				if !shared {
					apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Access denied.")
					return
				}
			}
		}
		records, err := attendance.ListForStudent(r.Context(), d.Pool, studentID, 200)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load attendance.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"records": recordsToJSON(records)})
	}
}

// handleOrgUnitAttendanceDashboard is GET /api/v1/org-units/:unitId/attendance/dashboard
func (d Deps) handleOrgUnitAttendanceDashboard() http.HandlerFunc {
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
		unitID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "unitId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid org unit id.")
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, actorID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load org.")
			return
		}
		isAdmin, _ := orgroles.UserHasRole(r.Context(), d.Pool, actorID, orgID, orgroles.RoleOrgAdmin)
		if !isAdmin {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Admin access required.")
			return
		}
		var date time.Time
		dateStr := r.URL.Query().Get("date")
		if dateStr != "" {
			date, err = time.Parse("2006-01-02", strings.TrimSpace(dateStr))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid date; use YYYY-MM-DD.")
				return
			}
		} else {
			date = time.Now().UTC()
		}
		entries, err := attendance.DashboardForOrgUnit(r.Context(), d.Pool, unitID, date)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load dashboard.")
			return
		}
		out := make([]map[string]any, 0, len(entries))
		for _, e := range entries {
			out = append(out, map[string]any{
				"sectionId":     e.SectionID.String(),
				"sectionCode":   e.SectionCode,
				"courseName":    e.CourseName,
				"date":          e.Date.Format("2006-01-02"),
				"totalStudents": e.TotalStudents,
				"presentCount":  e.PresentCount,
				"absentCount":   e.AbsentCount,
				"tardyCount":    e.TardyCount,
				"notTaken":      e.NotTaken,
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"entries": out, "date": date.Format("2006-01-02")})
	}
}

// handleAdminAttendanceCodes is GET/POST /api/v1/admin/orgs/:orgId/attendance/codes
func (d Deps) handleAdminAttendanceCodes() http.HandlerFunc {
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
			codes, err := attendance.ListCodes(r.Context(), d.Pool, orgID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list codes.")
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(map[string]any{"codes": codesToJSON(codes)})
		case http.MethodPost:
			var body struct {
				Code         string  `json:"code"`
				Label        string  `json:"label"`
				StateCode    *string `json:"stateCode"`
				Category     string  `json:"category"`
				SeedDefaults bool    `json:"seedDefaults"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
			if body.SeedDefaults {
				if err := attendance.SeedDefaultCodes(r.Context(), d.Pool, orgID); err != nil {
					apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to seed codes.")
					return
				}
				codes, _ := attendance.ListCodes(r.Context(), d.Pool, orgID)
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				_ = json.NewEncoder(w).Encode(map[string]any{"codes": codesToJSON(codes)})
				return
			}
			code := strings.TrimSpace(body.Code)
			label := strings.TrimSpace(body.Label)
			category := strings.TrimSpace(body.Category)
			if code == "" || label == "" {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "code and label are required.")
				return
			}
			switch category {
			case "present", "absent", "tardy", "other":
			case "":
				category = "present"
			default:
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid category (present|absent|tardy|other).")
				return
			}
			c, err := attendance.UpsertCode(r.Context(), d.Pool, orgID, code, label, body.StateCode, category)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not create attendance code.")
				return
			}
			w.WriteHeader(http.StatusCreated)
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(codeToJSON(c))
		default:
			w.Header().Set("Allow", http.MethodGet+","+http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	}
}

// handleAdminAttendanceCodeDelete is DELETE /api/v1/admin/orgs/:orgId/attendance/codes/:codeId
func (d Deps) handleAdminAttendanceCodeDelete() http.HandlerFunc {
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
		codeID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "codeId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid code id.")
			return
		}
		deleted, err := attendance.DeleteCode(r.Context(), d.Pool, orgID, codeID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, err.Error())
			return
		}
		if !deleted {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Code not found.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleAdminAttendanceExport is POST /api/v1/admin/orgs/:orgId/attendance/export
// Returns a synchronous CSV download.
func (d Deps) handleAdminAttendanceExport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
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
		var body struct {
			StartDate string `json:"startDate"`
			EndDate   string `json:"endDate"`
			Format    string `json:"format"` // "calpads" or "csv" (default)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		startDate, err := time.Parse("2006-01-02", strings.TrimSpace(body.StartDate))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid startDate (YYYY-MM-DD).")
			return
		}
		endDate, err := time.Parse("2006-01-02", strings.TrimSpace(body.EndDate))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid endDate (YYYY-MM-DD).")
			return
		}
		if endDate.Before(startDate) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "endDate must be >= startDate.")
			return
		}

		rows, err := d.Pool.Query(r.Context(), `
SELECT ar.date, ar.period, ar.student_id, u.email, u.display_name,
       cs.section_code, c.title AS course_name, ac.code, ac.state_code, ac.category, ar.note
FROM course.attendance_records ar
JOIN course.attendance_codes ac ON ac.id = ar.code_id AND ac.org_id = $1
JOIN "user".users u ON u.id = ar.student_id
JOIN course.course_sections cs ON cs.id = ar.section_id
JOIN course.courses c ON c.id = cs.course_id
WHERE ar.date >= $2::date AND ar.date <= $3::date
ORDER BY ar.date ASC, cs.section_code ASC, u.email ASC
`, orgID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to query attendance records.")
			return
		}
		defer rows.Close()

		exportFormat := strings.ToLower(strings.TrimSpace(body.Format))
		filename := fmt.Sprintf("attendance_%s_%s.csv", startDate.Format("20060102"), endDate.Format("20060102"))
		if exportFormat == "calpads" {
			filename = fmt.Sprintf("calpads_attendance_%s_%s.csv", startDate.Format("20060102"), endDate.Format("20060102"))
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

		cw := csv.NewWriter(w)
		if exportFormat == "calpads" {
			_ = cw.Write([]string{"SSID", "LocalStudentID", "AttendanceDate", "Period", "AttendanceCode", "CALPADSCode", "Category"})
		} else {
			_ = cw.Write([]string{"Date", "Period", "StudentID", "StudentEmail", "StudentName", "SectionCode", "CourseName", "AttendanceCode", "Category", "Note"})
		}

		for rows.Next() {
			var (
				date        time.Time
				period      *string
				studentID   uuid.UUID
				email       string
				displayName *string
				sectionCode string
				courseName  string
				code        string
				stateCode   *string
				category    string
				note        *string
			)
			if err := rows.Scan(&date, &period, &studentID, &email, &displayName,
				&sectionCode, &courseName, &code, &stateCode, &category, &note); err != nil {
				continue
			}
			p := ""
			if period != nil {
				p = *period
			}
			n := ""
			if note != nil {
				n = *note
			}
			name := ""
			if displayName != nil {
				name = *displayName
			}
			sc := code
			if stateCode != nil && *stateCode != "" {
				sc = *stateCode
			}
			if exportFormat == "calpads" {
				_ = cw.Write([]string{
					studentID.String(), email,
					date.Format("2006-01-02"), p,
					code, sc, category,
				})
			} else {
				_ = cw.Write([]string{
					date.Format("2006-01-02"), p,
					studentID.String(), email, name,
					sectionCode, courseName,
					code, category, n,
				})
			}
		}
		cw.Flush()
	}
}

// handleParentStudentAttendance is GET /api/v1/parent/students/:sid/attendance
func (d Deps) handleParentStudentAttendance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		parentID, _, ok := d.requireParentViewer(w, r)
		if !ok {
			return
		}
		studentID, ok := d.parseStudentIDParam(w, r)
		if !ok {
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, parentID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load org.")
			return
		}
		if _, ok := d.requireParentLink(w, r, parentID, orgID, studentID); !ok {
			return
		}
		records, err := attendance.ListForStudent(r.Context(), d.Pool, studentID, 200)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load attendance.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"records": recordsToJSON(records)})
	}
}

// requireSectionAccess checks that actorID is the section instructor, a course teacher, or org admin.
func (d Deps) requireSectionAccess(w http.ResponseWriter, r *http.Request, actorID, sectionID uuid.UUID) (uuid.UUID, bool) {
	orgIDPtr, err := attendance.OrgIDForSection(r.Context(), d.Pool, sectionID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load section.")
		return uuid.UUID{}, false
	}
	if orgIDPtr == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Section not found.")
		return uuid.UUID{}, false
	}
	orgID := *orgIDPtr

	isAdmin, _ := orgroles.UserHasRole(r.Context(), d.Pool, actorID, orgID, orgroles.RoleOrgAdmin)
	if isAdmin {
		return orgID, true
	}

	// Check if caller is the section's assigned instructor.
	instID, err := attendance.InstructorForSection(r.Context(), d.Pool, sectionID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify instructor.")
		return uuid.UUID{}, false
	}
	if instID != nil && *instID == actorID {
		return orgID, true
	}

	// Check if caller has a teacher enrollment in this section or its course.
	var enrolled bool
	_ = d.Pool.QueryRow(r.Context(), `
SELECT EXISTS(
    SELECT 1 FROM course.course_enrollments ce
    JOIN course.course_sections cs ON (ce.section_id = cs.id OR ce.course_id = cs.course_id)
    WHERE cs.id = $1 AND ce.user_id = $2 AND ce.active
      AND ce.role IN ('teacher', 'instructor', 'owner', 'ta')
)`, sectionID, actorID).Scan(&enrolled)
	if !enrolled {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have access to this section's attendance.")
		return uuid.UUID{}, false
	}
	return orgID, true
}

// ─── JSON helpers ────────────────────────────────────────────────────────────

func codeToJSON(c *attendance.Code) map[string]any {
	if c == nil {
		return nil
	}
	m := map[string]any{
		"id":       c.ID.String(),
		"orgId":    c.OrgID.String(),
		"code":     c.Code,
		"label":    c.Label,
		"category": c.Category,
	}
	if c.StateCode != nil {
		m["stateCode"] = *c.StateCode
	}
	return m
}

func codesToJSON(codes []attendance.Code) []map[string]any {
	out := make([]map[string]any, 0, len(codes))
	for i := range codes {
		out = append(out, codeToJSON(&codes[i]))
	}
	return out
}

func recordToJSON(rec *attendance.Record) map[string]any {
	if rec == nil {
		return nil
	}
	m := map[string]any{
		"id":         rec.ID.String(),
		"studentId":  rec.StudentID.String(),
		"sectionId":  rec.SectionID.String(),
		"date":       rec.Date.Format("2006-01-02"),
		"codeId":     rec.CodeID.String(),
		"code":       rec.Code,
		"codeLabel":  rec.CodeLabel,
		"category":   rec.Category,
		"recordedAt": rec.RecordedAt.UTC().Format(time.RFC3339Nano),
		"updatedAt":  rec.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if rec.SchoolID != nil {
		m["schoolId"] = rec.SchoolID.String()
	}
	if rec.Period != nil {
		m["period"] = *rec.Period
	}
	if rec.Note != nil {
		m["note"] = *rec.Note
	}
	if rec.RecordedBy != nil {
		m["recordedBy"] = rec.RecordedBy.String()
	}
	return m
}

func recordsToJSON(records []attendance.Record) []map[string]any {
	out := make([]map[string]any, 0, len(records))
	for i := range records {
		out = append(out, recordToJSON(&records[i]))
	}
	return out
}

func rosterToJSON(roster []attendance.StudentRow) []map[string]any {
	out := make([]map[string]any, 0, len(roster))
	for _, s := range roster {
		m := map[string]any{
			"userId": s.UserID.String(),
			"email":  s.Email,
		}
		if s.DisplayName != nil {
			m["displayName"] = *s.DisplayName
		}
		out = append(out, m)
	}
	return out
}

func (d Deps) registerAttendanceRoutes(r chi.Router) {
	// Teacher roll-taking
	r.Method(http.MethodGet, "/api/v1/sections/{sectionId}/attendance/{date}", d.handleSectionAttendance())
	r.Method(http.MethodPut, "/api/v1/sections/{sectionId}/attendance/{date}", d.handleSectionAttendance())
	// Student history (self, teacher, parent)
	r.Get("/api/v1/students/{studentId}/attendance", d.handleStudentAttendance())
	// School dashboard (admin)
	r.Get("/api/v1/org-units/{unitId}/attendance/dashboard", d.handleOrgUnitAttendanceDashboard())
	// Admin codes management
	r.Method(http.MethodGet, "/api/v1/admin/orgs/{orgId}/attendance/codes", d.handleAdminAttendanceCodes())
	r.Method(http.MethodPost, "/api/v1/admin/orgs/{orgId}/attendance/codes", d.handleAdminAttendanceCodes())
	r.Delete("/api/v1/admin/orgs/{orgId}/attendance/codes/{codeId}", d.handleAdminAttendanceCodeDelete())
	// Export (synchronous CSV)
	r.Post("/api/v1/admin/orgs/{orgId}/attendance/export", d.handleAdminAttendanceExport())
	// Parent view
	r.Get("/api/v1/parent/students/{sid}/attendance", d.handleParentStudentAttendance())
}
