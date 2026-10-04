package httpserver

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"unicode"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/gradingdisplay"
	"github.com/lextures/lextures/server/internal/repos/attendancesessions"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursegrades"
	"github.com/lextures/lextures/server/internal/repos/coursemoduleassignments"
	"github.com/lextures/lextures/server/internal/repos/coursestructure"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/repos/gradingschemes"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/service/grading"
	"github.com/lextures/lextures/server/internal/service/notifications"
)

// handleExportCourseGradebookCSV is GET /api/v1/courses/{course_code}/gradebook.csv (plan 3.11).
func (d Deps) handleExportCourseGradebookCSV() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		if !auth.RequireAccessKeyScope(w, r.Context(), "grades:read") {
			return
		}
		if !d.requireGradebookView(w, r, courseCode, viewer) {
			return
		}
		if d.rejectGradebookCSVDisabled(w) {
			return
		}
		book, courseID, err := d.loadGradebookCSVBook(r.Context(), courseCode, viewer)
		if err != nil {
			slog.Error("gradebook_export", "course_code", courseCode, "err", err)
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to export gradebook.")
			return
		}
		raw, err := grading.BuildGradebookCSV(book)
		if err != nil {
			slog.Error("gradebook_export", "course_code", courseCode, "err", err)
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to export gradebook.")
			return
		}
		slog.Info("gradebook_export", "course_id", courseID.String(), "course_code", courseCode, "row_count", len(book.Students))
		filename := safeGradebookFilename(courseCode)
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(raw)
	}
}

// handleGradebookImportValidate is POST /api/v1/courses/{course_code}/gradebook/import/validate.
func (d Deps) handleGradebookImportValidate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, courseID, ok := d.requireGradebookCSVEdit(w, r)
		if !ok {
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 8<<20))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not read CSV.")
			return
		}
		book, _, err := d.loadGradebookCSVBook(r.Context(), courseCode, viewer)
		if err != nil {
			slog.Error("gradebook_import", "course_code", courseCode, "err", err)
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to validate gradebook CSV.")
			return
		}
		res, err := grading.ValidateGradebookCSV(string(body), book)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		if res.Preview.Confirmable {
			tok := uuid.New()
			res.Preview.Token = &tok
			grading.DefaultImportSessions.Save(grading.ImportSession{
				Token:      tok,
				CourseID:   courseID,
				ActorID:    viewer,
				Grades:     res.Pending,
				RequireAck: res.RequireAck,
			})
		}
		writeJSON(w, http.StatusOK, res.Preview)
	}
}

// handleGradebookImportConfirm is POST /api/v1/courses/{course_code}/gradebook/import/confirm.
func (d Deps) handleGradebookImportConfirm() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, courseID, ok := d.requireGradebookCSVEdit(w, r)
		if !ok {
			return
		}
		var body struct {
			Token                      uuid.UUID `json:"token"`
			AcknowledgeBlindManualHold *bool     `json:"acknowledgeBlindManualHold"`
		}
		if err := decodeJSON(w, r, &body, 1<<20); err != nil {
			return
		}
		if body.Token == uuid.Nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Missing import token.")
			return
		}
		sess, ok := grading.DefaultImportSessions.Take(body.Token, courseID, viewer)
		if !ok {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Import session expired or not found.")
			return
		}
		if sess.RequireAck && (body.AcknowledgeBlindManualHold == nil || !*body.AcknowledgeBlindManualHold) {
			// Put the session back so the instructor can acknowledge and retry.
			grading.DefaultImportSessions.Save(sess)
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Confirm the blind-grading manual-hold acknowledgement to continue.")
			return
		}
		if err := coursegrades.ApplyGradebookBulkImport(r.Context(), d.Pool, courseID, viewer, sess.Grades); err != nil {
			grading.DefaultImportSessions.Save(sess)
			slog.Error("gradebook_import", "course_code", courseCode, "err", err)
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not apply grade import.")
			return
		}
		notifications.NotifyAutoPostedFromGradebookPut(r.Context(), d.Pool, d.effectiveConfig(), courseID, sess.Grades, d.SmsNotificationQueue)
		n := 0
		for _, row := range sess.Grades {
			n += len(row)
		}
		slog.Info("gradebook_import", "course_id", courseID.String(), "course_code", courseCode, "row_count", n)
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleGradebookImportDelete is DELETE /api/v1/courses/{course_code}/gradebook/import/{token}.
func (d Deps) handleGradebookImportDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, viewer, courseID, ok := d.requireGradebookCSVEdit(w, r)
		if !ok {
			return
		}
		token, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "token")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid import token.")
			return
		}
		grading.DefaultImportSessions.Delete(token, courseID, viewer)
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) rejectGradebookCSVDisabled(w http.ResponseWriter) bool {
	if d.effectiveConfig().GradebookCSVEnabled {
		return false
	}
	apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Gradebook CSV is not enabled.")
	return true
}

func (d Deps) requireGradebookView(w http.ResponseWriter, r *http.Request, courseCode string, viewer uuid.UUID) bool {
	ok, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":gradebook:view")
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return false
	}
	if !ok {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to view the gradebook.")
		return false
	}
	return true
}

// requireGradebookCSVEdit checks course access, item-create permission, and the CSV flag.
// It returns false after writing the error response.
func (d Deps) requireGradebookCSVEdit(w http.ResponseWriter, r *http.Request) (courseCode string, viewer, courseID uuid.UUID, ok bool) {
	courseCode, viewer, ok = d.requireCourseAccess(w, r)
	if !ok {
		return "", uuid.Nil, uuid.Nil, false
	}
	if d.Pool == nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
		return "", uuid.Nil, uuid.Nil, false
	}
	canEdit, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return "", uuid.Nil, uuid.Nil, false
	}
	if !canEdit {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to edit grades.")
		return "", uuid.Nil, uuid.Nil, false
	}
	if d.rejectGradebookCSVDisabled(w) {
		return "", uuid.Nil, uuid.Nil, false
	}
	cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
	if err != nil || cid == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
		return "", uuid.Nil, uuid.Nil, false
	}
	return courseCode, viewer, *cid, true
}

func (d Deps) loadGradebookCSVBook(ctx context.Context, courseCode string, viewer uuid.UUID) (grading.CSVBook, uuid.UUID, error) {
	cid, err := course.GetIDByCourseCode(ctx, d.Pool, courseCode)
	if err != nil {
		return grading.CSVBook{}, uuid.Nil, err
	}
	if cid == nil {
		return grading.CSVBook{}, uuid.Nil, errString("course not found")
	}
	courseID := *cid
	sectionFilter, err := enrollment.GradebookStudentSectionFilter(ctx, d.Pool, courseID, courseCode, viewer, false)
	if err != nil {
		return grading.CSVBook{}, uuid.Nil, err
	}
	cfg := d.effectiveConfig()
	type rosterRow struct {
		id   uuid.UUID
		name string
	}
	var roster []rosterRow
	if cfg.FFEnrollmentStateMachine {
		rows, err := enrollment.ListGradebookStudents(ctx, d.Pool, courseCode, sectionFilter, true)
		if err != nil {
			return grading.CSVBook{}, uuid.Nil, err
		}
		for _, s := range rows {
			roster = append(roster, rosterRow{id: s.UserID, name: s.DisplayName})
		}
	} else {
		legacy, err := enrollment.ListStudentUsersForCourseCode(ctx, d.Pool, courseCode, sectionFilter)
		if err != nil {
			return grading.CSVBook{}, uuid.Nil, err
		}
		for _, s := range legacy {
			roster = append(roster, rosterRow{id: s.UserID, name: s.DisplayName})
		}
	}
	ids := make([]uuid.UUID, len(roster))
	for i, s := range roster {
		ids[i] = s.id
	}
	emails, err := enrollment.EmailsByUserIDs(ctx, d.Pool, ids)
	if err != nil {
		return grading.CSVBook{}, uuid.Nil, err
	}

	items, err := coursestructure.ListForCourseWithEnrichment(ctx, d.Pool, courseID, true)
	if err != nil {
		return grading.CSVBook{}, uuid.Nil, err
	}
	var assignIDs, attendanceIDs []uuid.UUID
	for i := range items {
		id, e := uuid.Parse(items[i].ID)
		if e != nil {
			continue
		}
		switch items[i].Kind {
		case "assignment":
			assignIDs = append(assignIDs, id)
		case "attendance":
			attendanceIDs = append(attendanceIDs, id)
		}
	}
	posting, err := coursemoduleassignments.PostingByItemID(ctx, d.Pool, courseID, assignIDs)
	if err != nil {
		return grading.CSVBook{}, uuid.Nil, err
	}
	typeMap, err := coursemoduleassignments.GradingTypeByItemID(ctx, d.Pool, courseID, assignIDs)
	if err != nil {
		return grading.CSVBook{}, uuid.Nil, err
	}
	blindHold, err := coursemoduleassignments.BlindManualHoldByItemID(ctx, d.Pool, courseID, assignIDs)
	if err != nil {
		return grading.CSVBook{}, uuid.Nil, err
	}
	attendancePoints, err := attendancesessions.LoadPointsByStructureItemIDs(ctx, d.Pool, attendanceIDs)
	if err != nil {
		return grading.CSVBook{}, uuid.Nil, err
	}

	schemeRow, err := gradingschemes.GetActiveForCourse(ctx, d.Pool, courseID)
	if err != nil {
		return grading.CSVBook{}, uuid.Nil, err
	}
	var courseKind *gradingdisplay.Kind
	var parsed *gradingdisplay.ParsedScale
	if schemeRow != nil {
		k, ok := gradingdisplay.ParseKind(schemeRow.GradingDisplayType)
		if !ok {
			k = gradingdisplay.Points
		}
		courseKind = &k
		ps, err := gradingdisplay.ParseScale(k, schemeRow.ScaleJSON)
		if err != nil {
			return grading.CSVBook{}, uuid.Nil, err
		}
		parsed = &ps
	} else {
		p := gradingdisplay.ParsedScale{Kind: gradingdisplay.Points}
		parsed = &p
	}

	var gridCols []gradebookGridColumn
	var csvCols []grading.CSVColumn
	for i := range items {
		kind := items[i].Kind
		if kind != "assignment" && kind != "quiz" && kind != "h5p" && kind != "scorm" && kind != "attendance" {
			continue
		}
		itemID, err := uuid.Parse(items[i].ID)
		if err != nil {
			continue
		}
		mp := gradebookMaxPoints(&items[i])
		if kind == "attendance" {
			if pts, ok := attendancePoints[itemID]; ok {
				v := pts
				mp = &v
			}
		}
		gc := gradebookGridColumn{
			ID:        items[i].ID,
			Kind:      kind,
			Title:     items[i].Title,
			MaxPoints: mp,
		}
		if kind == "assignment" {
			if gt, ok := typeMap[itemID]; ok {
				gc.AssignmentGradingType = gt
			}
			if p, ok := posting[itemID]; ok {
				pp := p.Policy
				gc.PostingPolicy = &pp
			}
		}
		eff := gradingdisplay.ResolveEffective(courseKind, gc.AssignmentGradingType)
		gc.EffectiveDisplayType = eff.String()
		gridCols = append(gridCols, gc)
		csvCols = append(csvCols, grading.CSVColumn{
			ID:              itemID,
			Title:           items[i].Title,
			MaxPoints:       mp,
			ManualHoldBlind: cfg.BlindGradingEnabled && blindHold[itemID],
		})
	}

	grades, _, _, excused, err := coursegrades.ListForCourse(ctx, d.Pool, courseID)
	if err != nil {
		return grading.CSVBook{}, uuid.Nil, err
	}
	display := gridDisplayGrades(grades, gridCols, parsed, excused)

	book := grading.CSVBook{
		Columns: csvCols,
		Grades:  map[uuid.UUID]map[uuid.UUID]string{},
		Display: map[uuid.UUID]map[uuid.UUID]string{},
		Excused: map[uuid.UUID]map[uuid.UUID]bool{},
	}
	for _, s := range roster {
		book.Students = append(book.Students, grading.CSVStudent{ID: s.id, Name: s.name, Email: emails[s.id]})
	}
	copyNestedStrings(book.Grades, grades)
	copyNestedStrings(book.Display, display)
	for su, row := range excused {
		sid, err := uuid.Parse(su)
		if err != nil {
			continue
		}
		if book.Excused[sid] == nil {
			book.Excused[sid] = map[uuid.UUID]bool{}
		}
		for iu, on := range row {
			iid, err := uuid.Parse(iu)
			if err != nil || !on {
				continue
			}
			book.Excused[sid][iid] = true
		}
	}
	book.FinalScore, book.FinalGrade = grading.StraightFinals(book)
	return book, courseID, nil
}

func copyNestedStrings(dst map[uuid.UUID]map[uuid.UUID]string, src map[string]map[string]string) {
	for su, row := range src {
		sid, err := uuid.Parse(su)
		if err != nil {
			continue
		}
		if dst[sid] == nil {
			dst[sid] = map[uuid.UUID]string{}
		}
		for iu, v := range row {
			iid, err := uuid.Parse(iu)
			if err != nil {
				continue
			}
			dst[sid][iid] = v
		}
	}
}

func safeGradebookFilename(courseCode string) string {
	var b strings.Builder
	for _, r := range courseCode {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	slug := b.String()
	if slug == "" {
		slug = "course"
	}
	return slug + "-gradebook.csv"
}

type errString string

func (e errString) Error() string { return string(e) }

// decodeJSON reads a JSON body. On failure it writes 400 and returns the error.
func decodeJSON(w http.ResponseWriter, r *http.Request, dest any, limit int64) error {
	if limit <= 0 {
		limit = 1 << 20
	}
	payload, err := io.ReadAll(io.LimitReader(r.Body, limit))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not read body.")
		return err
	}
	if err := json.Unmarshal(payload, dest); err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
		return err
	}
	return nil
}
