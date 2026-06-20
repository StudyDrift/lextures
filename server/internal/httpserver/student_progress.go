package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	modelsp "github.com/lextures/lextures/server/internal/models/studentprogress"
	"github.com/lextures/lextures/server/internal/repos/assignmentoverrides"
	"github.com/lextures/lextures/server/internal/repos/coursegrants"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	stprog "github.com/lextures/lextures/server/internal/repos/studentprogress"
	"github.com/lextures/lextures/server/internal/repos/user"
)

// applyAssignToDueDates patches missing/assignment row due dates in place with each item's
// plan 2.15 effective (assign-to resolved) due date for this enrollment, so late detection
// and missing-item due dates agree with the dates shown on the student's dashboard/calendar.
func applyAssignToDueDates(ctx context.Context, pool *pgxpool.Pool, enrollmentID uuid.UUID, missing []stprog.MissingItemRow, assignRows []stprog.AssignmentRow) {
	ids := make([]uuid.UUID, 0, len(missing)+len(assignRows))
	bases := make(map[uuid.UUID]assignmentoverrides.BaseDates)
	for _, m := range missing {
		ids = append(ids, m.ItemID)
		bases[m.ItemID] = assignmentoverrides.BaseDates{DueAt: m.DueAt}
	}
	for _, a := range assignRows {
		ids = append(ids, a.ItemID)
		bases[a.ItemID] = assignmentoverrides.BaseDates{DueAt: a.DueAt}
	}
	if len(ids) == 0 {
		return
	}
	effMap, err := assignmentoverrides.EffectiveForStudentBatch(ctx, pool, enrollmentID, ids, bases)
	if err != nil {
		return
	}
	for i := range missing {
		if eff, ok := effMap[missing[i].ItemID]; ok {
			missing[i].DueAt = eff.DueAt
		}
	}
	for i := range assignRows {
		if eff, ok := effMap[assignRows[i].ItemID]; ok {
			assignRows[i].DueAt = eff.DueAt
		}
	}
}

func (d Deps) studentProgressEnabled() bool {
	return d.effectiveConfig().StudentProgressEnabled
}

func (d Deps) guardStudentProgress(w http.ResponseWriter) bool {
	if !d.studentProgressEnabled() {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
		return false
	}
	return true
}

func (d Deps) canViewEnrollmentProgress(ctx context.Context, w http.ResponseWriter, viewer uuid.UUID, en *enrollment.ByID) bool {
	if en.UserID == viewer {
		return true
	}
	code := en.CourseCode
	for _, perm := range []string{
		"course:" + code + ":gradebook:view",
		coursegrants.CourseEnrollmentsReadPermission(code),
	} {
		ok, err := courseroles.UserHasPermission(ctx, d.Pool, viewer, perm)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return false
		}
		if ok {
			return true
		}
	}
	apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Access denied.")
	return false
}

func (d Deps) canManageProgressNotes(ctx context.Context, w http.ResponseWriter, viewer uuid.UUID, en *enrollment.ByID) bool {
	if en.UserID == viewer {
		return false
	}
	code := en.CourseCode
	ok, err := courseroles.UserHasPermission(ctx, d.Pool, viewer, "course:"+code+":gradebook:view")
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return false
	}
	if !ok {
		ok, err = courseroles.UserHasPermission(ctx, d.Pool, viewer, coursegrants.CourseEnrollmentsReadPermission(code))
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return false
		}
	}
	if !ok {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Access denied.")
		return false
	}
	staff, err := enrollment.UserIsCourseStaff(ctx, d.Pool, code, viewer)
	if err != nil || !staff {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Access denied.")
		return false
	}
	return true
}

func (d Deps) loadEnrollmentForProgress(w http.ResponseWriter, r *http.Request, courseCode string, viewer uuid.UUID) (*enrollment.ByID, bool) {
	eid, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "enrollment_id")))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid enrollment id.")
		return nil, false
	}
	en, err := enrollment.GetByID(r.Context(), d.Pool, eid)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load enrollment.")
		return nil, false
	}
	if en == nil || !strings.EqualFold(en.CourseCode, courseCode) {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Enrollment not found.")
		return nil, false
	}
	if !d.canViewEnrollmentProgress(r.Context(), w, viewer, en) {
		return nil, false
	}
	return en, true
}

func (d Deps) logProgressAccess(courseCode string, studentID, viewer uuid.UUID, self bool) {
	role := "instructor"
	if self {
		role = "student"
	}
	slog.Info("progress.page_view",
		"course_id", courseCode,
		"student_id", studentID.String(),
		"viewer_role", role,
	)
}

func (d Deps) handleEnrollmentProgressGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.guardStudentProgress(w) {
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
		en, ok := d.loadEnrollmentForProgress(w, r, courseCode, viewer)
		if !ok {
			return
		}
		ctx := r.Context()
		d.logProgressAccess(courseCode, en.UserID, viewer, en.UserID == viewer)

		go func() {
			bg := contextWithoutCancel(ctx)
			if _, err := stprog.RefreshViewIfStale(bg, d.Pool); err != nil {
				slog.Warn("progress.refresh_failed", "err", err)
			}
		}()

		meta, err := stprog.GetRefreshMeta(ctx, d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load progress.")
			return
		}
		snap, err := stprog.GetSnapshot(ctx, d.Pool, en.ID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load progress.")
			return
		}
		if snap == nil {
			if err := stprog.RefreshViewNow(ctx, d.Pool); err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load progress.")
				return
			}
			meta, _ = stprog.GetRefreshMeta(ctx, d.Pool)
			snap, err = stprog.GetSnapshot(ctx, d.Pool, en.ID)
			if err != nil || snap == nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load progress.")
				return
			}
		}

		displayName := "Student"
		var avatarURL *string
		if u, _ := user.FindByID(ctx, d.Pool, en.UserID); u != nil {
			if u.DisplayName != nil && strings.TrimSpace(*u.DisplayName) != "" {
				displayName = strings.TrimSpace(*u.DisplayName)
			}
			if u.AvatarURL != nil && strings.TrimSpace(*u.AvatarURL) != "" {
				avatarURL = u.AvatarURL
			}
		}

		avgGrade, _ := stprog.AvgGradePercent(ctx, d.Pool, en.CourseID, en.UserID)
		missing, _ := stprog.ListMissing(ctx, d.Pool, en.CourseID, en.UserID, time.Now().UTC())
		assignRows, _ := stprog.ListAssignments(ctx, d.Pool, en.CourseID, en.UserID)
		quizRows, _ := stprog.ListQuizAttempts(ctx, d.Pool, en.CourseID, en.UserID)
		// Plan 2.15: late detection and missing-item due dates must use this student's
		// effective (assign-to resolved) due date, not the item's base due date.
		applyAssignToDueDates(ctx, d.Pool, en.ID, missing, assignRows)

		var avgQuiz *float64
		if snap.AvgQuizScore != nil {
			f := float64(*snap.AvgQuizScore)
			avgQuiz = &f
		}
		staleMin := int(time.Since(meta.RefreshedAt).Minutes())
		canNotes := false
		if en.UserID != viewer {
			staff, _ := enrollment.UserIsCourseStaff(ctx, d.Pool, en.CourseCode, viewer)
			gb, _ := courseroles.UserHasPermission(ctx, d.Pool, viewer, "course:"+en.CourseCode+":gradebook:view")
			ro, _ := courseroles.UserHasPermission(ctx, d.Pool, viewer, coursegrants.CourseEnrollmentsReadPermission(en.CourseCode))
			canNotes = staff && (gb || ro)
		}

		sum := modelsp.Summary{
			EnrollmentID:            en.ID.String(),
			CourseID:                en.CourseID.String(),
			StudentUserID:           en.UserID.String(),
			StudentDisplayName:      displayName,
			StudentAvatarURL:        avatarURL,
			AssignmentsSubmittedPct: stprog.Pct(snap.AssignmentsSubmitted, snap.AssignmentsTotal),
			ModulesViewedPct:        stprog.Pct(snap.ModuleViewsCount, snap.ModulesTotal),
			AvgQuizScore:            avgQuiz,
			AvgGradePercent:         avgGrade,
			LastActiveAt:            snap.LastActiveAt,
			MissingCount:            len(missing),
			DataAsOf:                meta.RefreshedAt,
			StaleMinutes:            staleMin,
			CanManageNotes:          canNotes,
		}

		outMissing := make([]modelsp.MissingItem, 0, len(missing))
		for _, m := range missing {
			mi := modelsp.MissingItem{
				ItemID:      m.ItemID.String(),
				Title:       m.Title,
				Kind:        m.Kind,
				DaysOverdue: m.DaysOverdue,
				GradeStatus: m.GradeStatus,
			}
			if m.DueAt != nil {
				s := m.DueAt.UTC().Format(time.RFC3339)
				mi.DueAt = &s
			}
			outMissing = append(outMissing, mi)
		}

		outAssign := make([]modelsp.AssignmentRow, 0, len(assignRows))
		for _, a := range assignRows {
			row := modelsp.AssignmentRow{
				ItemID: a.ItemID.String(),
				Title:  a.Title,
				Grade:  "—",
				Status: "missing",
			}
			if a.SubmittedAt != nil {
				row.Status = "submitted"
				s := a.SubmittedAt.UTC().Format(time.RFC3339)
				row.SubmittedAt = &s
			} else if a.DueAt != nil && a.DueAt.Before(time.Now()) {
				row.Status = "late"
			} else {
				row.Status = "pending"
			}
			if a.DueAt != nil {
				s := a.DueAt.UTC().Format(time.RFC3339)
				row.DueAt = &s
			}
			if a.Points != nil && a.PointsWorth != nil && *a.PointsWorth > 0 {
				row.Grade = formatProgressPoints(*a.Points, float64(*a.PointsWorth))
			} else if a.Points != nil {
				row.Grade = strconv.FormatFloat(*a.Points, 'f', -1, 64)
			}
			outAssign = append(outAssign, row)
		}

		outQuiz := make([]modelsp.QuizRow, 0, len(quizRows))
		for _, q := range quizRows {
			qr := modelsp.QuizRow{
				AttemptID:   q.AttemptID.String(),
				ItemID:      q.ItemID.String(),
				Title:       q.Title,
				SubmittedAt: q.SubmittedAt.UTC().Format(time.RFC3339),
			}
			if q.ScorePercent != nil {
				f := float64(*q.ScorePercent)
				qr.ScorePercent = &f
			}
			outQuiz = append(outQuiz, qr)
		}

		var notes []modelsp.Note
		if canNotes {
			noteRows, err := stprog.ListNotes(ctx, d.Pool, en.ID)
			if err == nil {
				for _, n := range noteRows {
					notes = append(notes, modelsp.Note{
						ID:        n.ID.String(),
						AuthorID:  n.AuthorID.String(),
						NoteText:  n.NoteText,
						CreatedAt: n.CreatedAt.UTC().Format(time.RFC3339),
						UpdatedAt: n.UpdatedAt.UTC().Format(time.RFC3339),
					})
				}
			}
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(modelsp.ProgressResponse{
			Summary:     sum,
			Missing:     outMissing,
			Assignments: outAssign,
			Quizzes:     outQuiz,
			Notes:       notes,
		})
	}
}

func formatProgressPoints(earned, max float64) string {
	if max <= 0 {
		return strconv.FormatFloat(earned, 'f', -1, 64)
	}
	pct := (earned / max) * 100
	return strconv.FormatFloat(earned, 'f', -1, 64) + " / " + strconv.FormatFloat(max, 'f', -1, 64) +
		" (" + strconv.FormatFloat(pct, 'f', 1, 64) + "%)"
}

func (d Deps) handleEnrollmentProgressActivity() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.guardStudentProgress(w) {
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
		en, ok := d.loadEnrollmentForProgress(w, r, courseCode, viewer)
		if !ok {
			return
		}
		ctx := r.Context()
		var cursor *time.Time
		if c := strings.TrimSpace(r.URL.Query().Get("cursor")); c != "" {
			t, err := time.Parse(time.RFC3339, c)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid cursor.")
				return
			}
			t = t.UTC()
			cursor = &t
		}
		rows, next, err := stprog.ListActivity(ctx, d.Pool, en.CourseID, en.UserID, cursor, 50)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load activity.")
			return
		}
		events := make([]modelsp.ActivityEvent, 0, len(rows))
		for _, row := range rows {
			events = append(events, modelsp.ActivityEvent{
				OccurredAt: row.OccurredAt.UTC().Format(time.RFC3339),
				Kind:       row.Kind,
				Label:      row.Label,
				Detail:     row.Detail,
			})
		}
		var nextCursor *string
		if next != nil {
			s := next.Format(time.RFC3339)
			nextCursor = &s
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(modelsp.ActivityPage{Events: events, NextCursor: nextCursor})
	}
}

func (d Deps) handleEnrollmentProgressNotePost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.guardStudentProgress(w) {
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
		en, ok := d.loadEnrollmentForProgress(w, r, courseCode, viewer)
		if !ok {
			return
		}
		if !d.canManageProgressNotes(r.Context(), w, viewer, en) {
			return
		}
		var body modelsp.CreateNoteRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		text := strings.TrimSpace(body.NoteText)
		if text == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Note text is required.")
			return
		}
		n, err := stprog.CreateNote(r.Context(), d.Pool, en.ID, viewer, text)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save note.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(modelsp.Note{
			ID:        n.ID.String(),
			AuthorID:  n.AuthorID.String(),
			NoteText:  n.NoteText,
			CreatedAt: n.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt: n.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}
}

func (d Deps) handleEnrollmentProgressNotePut() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.Header().Set("Allow", http.MethodPut)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.guardStudentProgress(w) {
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
		en, ok := d.loadEnrollmentForProgress(w, r, courseCode, viewer)
		if !ok {
			return
		}
		if !d.canManageProgressNotes(r.Context(), w, viewer, en) {
			return
		}
		nid, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "note_id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid note id.")
			return
		}
		var body modelsp.UpdateNoteRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		text := strings.TrimSpace(body.NoteText)
		if text == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Note text is required.")
			return
		}
		n, err := stprog.UpdateNote(r.Context(), d.Pool, nid, viewer, text)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Note not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update note.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(modelsp.Note{
			ID:        n.ID.String(),
			AuthorID:  n.AuthorID.String(),
			NoteText:  n.NoteText,
			CreatedAt: n.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt: n.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}
}

func (d Deps) handleEnrollmentProgressNoteDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.Header().Set("Allow", http.MethodDelete)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.guardStudentProgress(w) {
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
		en, ok := d.loadEnrollmentForProgress(w, r, courseCode, viewer)
		if !ok {
			return
		}
		if !d.canManageProgressNotes(r.Context(), w, viewer, en) {
			return
		}
		nid, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "note_id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid note id.")
			return
		}
		okDel, err := stprog.DeleteNote(r.Context(), d.Pool, nid, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to delete note.")
			return
		}
		if !okDel {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Note not found.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// contextWithoutCancel returns a background context (for async refresh after response).
func contextWithoutCancel(ctx context.Context) context.Context {
	return context.WithoutCancel(ctx)
}
