package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/logging"
	repoBilling "github.com/lextures/lextures/server/internal/repos/billing"
	credrepo "github.com/lextures/lextures/server/internal/repos/credentials"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/repos/learnerprogress"
	credsvc "github.com/lextures/lextures/server/internal/service/credentials"
	"github.com/lextures/lextures/server/internal/service/gamification"
	"github.com/lextures/lextures/server/internal/service/selfpaced"
)

// selfPacedCourse holds the self-paced configuration needed to gate enrollment and progress.
type selfPacedCourse struct {
	ID             uuid.UUID
	CourseMode     string
	OpenEnrollment bool
	GatingEnabled  bool
	PriceCents     int
	CatalogSlug    *string
}

// loadSelfPacedCourse fetches the self-paced columns for a course, or nil when not found.
func (d Deps) loadSelfPacedCourse(r *http.Request, courseCode string) (*selfPacedCourse, error) {
	var c selfPacedCourse
	err := d.Pool.QueryRow(r.Context(), `
SELECT id, course_mode, open_enrollment, module_gating_enabled, price_cents, catalog_slug
FROM course.courses
WHERE course_code = $1
`, courseCode).Scan(&c.ID, &c.CourseMode, &c.OpenEnrollment, &c.GatingEnabled, &c.PriceCents, &c.CatalogSlug)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

// handleCourseSelfEnroll enrolls the viewer as a student in a self-paced, open-enrollment
// course without any instructor action (plan 15.2 FR-2, AC-1).
func (d Deps) handleCourseSelfEnroll() http.HandlerFunc {
	type respBody struct {
		Enrolled     bool    `json:"enrolled"`
		EnrollmentID string  `json:"enrollmentId"`
		FirstItemID  *string `json:"firstItemId,omitempty"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.effectiveConfig().FFSelfPacedMode {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Self-paced enrollment is not enabled.")
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		courseCode, ok := chiCourseCode(w, r)
		if !ok {
			return
		}
		c, err := d.loadSelfPacedCourse(r, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if c == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		if c.CourseMode != "self_paced" || !c.OpenEnrollment {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "This course is not open for self-enrollment.")
			return
		}
		// Paid self-paced courses require an active entitlement (plan MKT4 FR-6).
		hasAccess, accessErr := repoBilling.HasCourseAccess(r.Context(), d.Pool, viewer, c.ID)
		if accessErr != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify purchase.")
			return
		}
		if !hasAccess {
			hint := "/marketplace/" + courseCode
			if c.CatalogSlug != nil && strings.TrimSpace(*c.CatalogSlug) != "" {
				hint = "/marketplace/" + strings.TrimSpace(*c.CatalogSlug)
			}
			apierr.WritePaymentRequired(w, "Purchase required.", hint)
			return
		}

		ctx := r.Context()
		tx, err := d.Pool.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to start transaction.")
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()
		tag, err := tx.Exec(ctx, `
INSERT INTO course.course_enrollments (course_id, user_id, role)
VALUES ($1, $2, 'student')
ON CONFLICT (course_id, user_id, role) DO NOTHING
`, c.ID, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to enroll.")
			return
		}
		if err := courseroles.RefreshManagedGrantsForCourseUser(ctx, tx, viewer, c.ID, courseCode); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to sync course permissions.")
			return
		}
		if err := tx.Commit(ctx); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save enrollment.")
			return
		}
		if tag.RowsAffected() > 0 {
			d.notifyCourses(viewer)
		}
		eid, err := enrollment.GetStudentEnrollmentID(ctx, d.Pool, c.ID, viewer)
		if err != nil || eid == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load enrollment.")
			return
		}
		first, err := learnerprogress.FirstItem(ctx, d.Pool, c.ID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course content.")
			return
		}
		resp := respBody{Enrolled: true, EnrollmentID: eid.String()}
		if first != nil {
			s := first.String()
			resp.FirstItemID = &s
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// progressResponse is the learner-facing progress snapshot.
type progressResponse struct {
	selfpaced.Summary
	EnrollmentID string  `json:"enrollmentId"`
	ResumeItemID *string `json:"resumeItemId,omitempty"`
	JustComplete  bool    `json:"justCompleted,omitempty"`
	CredentialID  *string `json:"credentialId,omitempty"`
}

// handleCourseMyProgress returns the viewer's self-paced progress for a course (FR-4, FR-5, AC-2).
func (d Deps) handleCourseMyProgress() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		courseCode, ok := chiCourseCode(w, r)
		if !ok {
			return
		}
		c, err := d.loadSelfPacedCourse(r, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if c == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		eid, err := enrollment.GetStudentEnrollmentID(r.Context(), d.Pool, c.ID, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load enrollment.")
			return
		}
		if eid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "You are not enrolled in this course.")
			return
		}
		summary, err := selfpaced.LoadSummary(r.Context(), d.Pool, c.ID, *eid, c.GatingEnabled)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to compute progress.")
			return
		}
		resume, err := learnerprogress.FirstIncompleteItem(r.Context(), d.Pool, c.ID, *eid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to compute resume position.")
			return
		}
		resp := progressResponse{Summary: summary, EnrollmentID: eid.String()}
		if resume != nil {
			s := resume.String()
			resp.ResumeItemID = &s
		}
		if summary.Completed && d.effectiveConfig().FFCompletionCredentials {
			if cred, err := credrepo.GetByRecipientAndSource(r.Context(), d.Pool, viewer, credrepo.SourceCourse, c.ID); err == nil && cred != nil {
				id := cred.ID.String()
				resp.CredentialID = &id
			}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// handleCourseItemComplete marks an item complete for the viewer's enrollment, enforcing
// module gating and triggering the completion flow on the final item (FR-3, FR-6, FR-8, AC-3, AC-5).
func (d Deps) handleCourseItemComplete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		courseCode, ok := chiCourseCode(w, r)
		if !ok {
			return
		}
		itemID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "item_id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}
		c, err := d.loadSelfPacedCourse(r, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if c == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		// The viewer may only mark their own enrollment's items complete (security: no cross-learner writes).
		eid, err := enrollment.GetStudentEnrollmentID(r.Context(), d.Pool, c.ID, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load enrollment.")
			return
		}
		if eid == nil {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You are not enrolled in this course.")
			return
		}
		belongs, err := learnerprogress.ItemBelongsToCourse(r.Context(), d.Pool, c.ID, itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify item.")
			return
		}
		if !belongs {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Item not found in this course.")
			return
		}
		if c.GatingEnabled {
			moduleID, err := learnerprogress.ModuleForItem(r.Context(), d.Pool, c.ID, itemID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify module gate.")
				return
			}
			if moduleID != nil {
				modules, err := learnerprogress.ModuleProgressForEnrollment(r.Context(), d.Pool, c.ID, *eid)
				if err != nil {
					apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify module gate.")
					return
				}
				if selfpaced.ModuleIsLocked(modules, true, *moduleID) {
					apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Complete the previous module to unlock this one.")
					return
				}
			}
		}
		changed, err := learnerprogress.MarkCompleted(r.Context(), d.Pool, *eid, itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to mark item complete.")
			return
		}
		if changed {
			cfg := d.effectiveConfig()
			gamification.EmitModuleItemCompleted(d.Pool, cfg, viewer, c.ID, itemID)
		}
		d.recordConditionalReleaseProgress(r, c.ID, viewer, itemID)
		summary, err := selfpaced.LoadSummary(r.Context(), d.Pool, c.ID, *eid, c.GatingEnabled)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to compute progress.")
			return
		}
		resume, err := learnerprogress.FirstIncompleteItem(r.Context(), d.Pool, c.ID, *eid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to compute resume position.")
			return
		}
		resp := progressResponse{Summary: summary, EnrollmentID: eid.String()}
		if resume != nil {
			s := resume.String()
			resp.ResumeItemID = &s
		}
		resp.JustComplete = changed && summary.Completed
		if resp.JustComplete {
			cfg := d.effectiveConfig()
			gamification.EmitCourseCompleted(d.Pool, cfg, viewer, c.ID)
		}
		if resp.JustComplete && d.effectiveConfig().FFCompletionCredentials {
			learnerName, nameErr := d.learnerDisplayName(r, viewer)
			if nameErr == nil {
				cfg := d.effectiveConfig()
				cred, issueErr := credsvc.IssueCourseCompletion(r.Context(), d.Pool, cfg, credsvc.IssueCourseParams{
					RecipientID: viewer,
					LearnerName: learnerName,
					CourseID:    c.ID,
				})
				if issueErr == nil && cred != nil {
					logging.GlobalCredentialsMetrics.IncIssued()
					d.notifyCertificateIssued(r, viewer, cred)
					id := cred.ID.String()
					resp.CredentialID = &id
				}
			}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// selfPacedEnrollmentRow is one self-paced course the viewer is enrolled in.
type selfPacedEnrollmentRow struct {
	CourseCode      string  `json:"courseCode"`
	Title           string  `json:"title"`
	EnrollmentID    string  `json:"enrollmentId"`
	ProgressPercent int     `json:"progressPercent"`
	TotalItems      int     `json:"totalItems"`
	CompletedItems  int     `json:"completedItems"`
	Completed       bool    `json:"completed"`
	ResumeItemID    *string `json:"resumeItemId,omitempty"`
}

// handleMySelfPacedEnrollments lists the viewer's self-paced enrollments with progress (FR-4, FR-5).
func (d Deps) handleMySelfPacedEnrollments() http.HandlerFunc {
	type respBody struct {
		Enrollments []selfPacedEnrollmentRow `json:"enrollments"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		// Only the self_paced mode is supported by this endpoint; the query param documents intent.
		if mode := strings.TrimSpace(r.URL.Query().Get("mode")); mode != "" && mode != "self_paced" {
			_ = json.NewEncoder(w).Encode(respBody{Enrollments: []selfPacedEnrollmentRow{}})
			return
		}
		rows, err := d.Pool.Query(r.Context(), `
SELECT c.id, c.course_code, c.title, ce.id
FROM course.course_enrollments ce
INNER JOIN course.courses c ON c.id = ce.course_id
INNER JOIN course.enrollment_roles er ON er.role_key = ce.role AND er.is_student_equivalent = true
WHERE ce.user_id = $1 AND ce.active AND c.course_mode = 'self_paced' AND NOT c.archived
ORDER BY c.title
`, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load enrollments.")
			return
		}
		defer rows.Close()
		type courseRef struct {
			id    uuid.UUID
			code  string
			title string
			eid   uuid.UUID
		}
		var refs []courseRef
		for rows.Next() {
			var cr courseRef
			if err := rows.Scan(&cr.id, &cr.code, &cr.title, &cr.eid); err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to read enrollments.")
				return
			}
			refs = append(refs, cr)
		}
		if err := rows.Err(); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to read enrollments.")
			return
		}
		out := make([]selfPacedEnrollmentRow, 0, len(refs))
		for _, cr := range refs {
			totals, err := learnerprogress.CourseProgress(r.Context(), d.Pool, cr.id, cr.eid)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to compute progress.")
				return
			}
			resume, err := learnerprogress.FirstIncompleteItem(r.Context(), d.Pool, cr.id, cr.eid)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to compute resume position.")
				return
			}
			row := selfPacedEnrollmentRow{
				CourseCode:      cr.code,
				Title:           cr.title,
				EnrollmentID:    cr.eid.String(),
				ProgressPercent: selfpaced.ProgressPercent(totals.CompletedItems, totals.TotalItems),
				TotalItems:      totals.TotalItems,
				CompletedItems:  totals.CompletedItems,
				Completed:       selfpaced.IsCourseComplete(totals.CompletedItems, totals.TotalItems),
			}
			if resume != nil {
				s := resume.String()
				row.ResumeItemID = &s
			}
			out = append(out, row)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(respBody{Enrollments: out})
	}
}
