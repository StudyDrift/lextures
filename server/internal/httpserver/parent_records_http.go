package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/homeschoolrecords"
	"github.com/lextures/lextures/server/internal/repos/user"
	hsvc "github.com/lextures/lextures/server/internal/service/homeschoolrecords"
)

func (d Deps) registerParentRecordsRoutes(r chi.Router) {
	r.Get("/api/v1/parent/records/settings", d.handleGetParentRecordsSettings())
	r.Put("/api/v1/parent/records/settings", d.handlePutParentRecordsSettings())
	r.Get("/api/v1/parent/students/{sid}/records/hours", d.handleParentStudentHoursLog())
	r.Post("/api/v1/parent/students/{sid}/records/hours", d.handleParentStudentHoursCreate())
	r.Patch("/api/v1/parent/students/{sid}/records/hours/{entryId}", d.handleParentStudentHoursUpdate())
	r.Delete("/api/v1/parent/students/{sid}/records/hours/{entryId}", d.handleParentStudentHoursDelete())
	r.Get("/api/v1/parent/students/{sid}/records/hours/export.csv", d.handleParentStudentHoursExportCSV())
	r.Get("/api/v1/parent/students/{sid}/records/hours/export.pdf", d.handleParentStudentHoursExportPDF())
	r.Get("/api/v1/parent/students/{sid}/records/transcript.pdf", d.handleParentStudentTranscriptPDF())
	r.Get("/api/v1/parent/students/{sid}/records/submissions", d.handleParentStudentRecordSubmissions())
	r.Post("/api/v1/parent/students/{sid}/records/portfolio", d.handleParentStudentPortfolioExport())
}

func (d Deps) parentRecordsAuth(w http.ResponseWriter, r *http.Request) (parentID, orgID, studentID uuid.UUID, ok bool) {
	parentID, orgID, ok = d.requireParentViewer(w, r)
	if !ok {
		return
	}
	studentID, ok = d.parseStudentIDParam(w, r)
	if !ok {
		return
	}
	if _, ok = d.requireParentLink(w, r, parentID, orgID, studentID); !ok {
		return
	}
	return parentID, orgID, studentID, true
}

func (d Deps) resolveSchoolYearWindow(w http.ResponseWriter, r *http.Request, parentID uuid.UUID) (settings homeschoolrecords.Settings, win hsvc.SchoolYearWindow, ok bool) {
	settings, err := homeschoolrecords.GetSettings(r.Context(), d.Pool, parentID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load records settings.")
		return settings, win, false
	}
	q := strings.TrimSpace(r.URL.Query().Get("schoolYear"))
	fromQ := strings.TrimSpace(r.URL.Query().Get("from"))
	toQ := strings.TrimSpace(r.URL.Query().Get("to"))
	if fromQ != "" || toQ != "" {
		from, err1 := time.Parse("2006-01-02", fromQ)
		to, err2 := time.Parse("2006-01-02", toQ)
		if err1 != nil || err2 != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "from and to must be YYYY-MM-DD.")
			return settings, win, false
		}
		if to.Before(from) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "to must be on or after from.")
			return settings, win, false
		}
		win = hsvc.SchoolYearWindow{Label: q, From: from, To: to}
		return settings, win, true
	}
	if q != "" {
		startYear, parsed := hsvc.ParseSchoolYearLabel(q)
		if !parsed {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "schoolYear must look like 2025-2026.")
			return settings, win, false
		}
		win = hsvc.WindowForYear(settings, startYear)
		return settings, win, true
	}
	win = hsvc.CurrentSchoolYear(settings, time.Now().UTC())
	return settings, win, true
}

func (d Deps) loadStudentDisplayName(r *http.Request, studentID uuid.UUID) string {
	row, err := user.FindByID(r.Context(), d.Pool, studentID)
	if err != nil || row == nil {
		return studentID.String()
	}
	if row.DisplayName != nil && strings.TrimSpace(*row.DisplayName) != "" {
		return strings.TrimSpace(*row.DisplayName)
	}
	return row.Email
}

func hoursEntryJSON(e *homeschoolrecords.HoursEntry) map[string]any {
	out := map[string]any{
		"id":        e.ID.String(),
		"date":      e.EntryDate.UTC().Format("2006-01-02"),
		"subject":   e.Subject,
		"hours":     e.Hours,
		"notes":     e.Notes,
		"source":    e.Source,
		"editable":  true,
		"createdAt": e.CreatedAt.UTC().Format(time.RFC3339Nano),
		"updatedAt": e.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if e.CourseID != nil {
		out["courseId"] = e.CourseID.String()
	}
	return out
}

func (d Deps) handleGetParentRecordsSettings() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parentID, _, ok := d.requireParentViewer(w, r)
		if !ok {
			return
		}
		s, err := homeschoolrecords.GetSettings(r.Context(), d.Pool, parentID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load records settings.")
			return
		}
		win := hsvc.CurrentSchoolYear(s, time.Now().UTC())
		writeJSON(w, http.StatusOK, map[string]any{
			"schoolName":           s.SchoolName,
			"schoolYearStartMonth": s.SchoolYearStartMonth,
			"schoolYearStartDay":   s.SchoolYearStartDay,
			"currentSchoolYear":    win.Label,
			"currentFrom":          win.From.Format("2006-01-02"),
			"currentTo":            win.To.Format("2006-01-02"),
		})
	}
}

func (d Deps) handlePutParentRecordsSettings() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parentID, _, ok := d.requireParentViewer(w, r)
		if !ok {
			return
		}
		var body struct {
			SchoolName           *string `json:"schoolName"`
			SchoolYearStartMonth *int    `json:"schoolYearStartMonth"`
			SchoolYearStartDay   *int    `json:"schoolYearStartDay"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		cur, err := homeschoolrecords.GetSettings(r.Context(), d.Pool, parentID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load records settings.")
			return
		}
		name := cur.SchoolName
		if body.SchoolName != nil {
			name = *body.SchoolName
		}
		month := cur.SchoolYearStartMonth
		if body.SchoolYearStartMonth != nil {
			month = *body.SchoolYearStartMonth
		}
		day := cur.SchoolYearStartDay
		if body.SchoolYearStartDay != nil {
			day = *body.SchoolYearStartDay
		}
		if month < 1 || month > 12 || day < 1 || day > 31 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid school year start month/day.")
			return
		}
		s, err := homeschoolrecords.UpsertSettings(r.Context(), d.Pool, parentID, name, month, day)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save records settings.")
			return
		}
		win := hsvc.CurrentSchoolYear(s, time.Now().UTC())
		writeJSON(w, http.StatusOK, map[string]any{
			"schoolName":           s.SchoolName,
			"schoolYearStartMonth": s.SchoolYearStartMonth,
			"schoolYearStartDay":   s.SchoolYearStartDay,
			"currentSchoolYear":    win.Label,
			"currentFrom":          win.From.Format("2006-01-02"),
			"currentTo":            win.To.Format("2006-01-02"),
		})
	}
}

func (d Deps) buildStudentHoursLog(w http.ResponseWriter, r *http.Request, parentID, studentID uuid.UUID) (homeschoolrecords.Settings, hsvc.SchoolYearWindow, hsvc.HoursLog, bool) {
	settings, win, ok := d.resolveSchoolYearWindow(w, r, parentID)
	if !ok {
		return settings, win, hsvc.HoursLog{}, false
	}
	auto, err := homeschoolrecords.ListAutoHours(r.Context(), d.Pool, studentID, win.From, win.To)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load activity hours.")
		return settings, win, hsvc.HoursLog{}, false
	}
	manual, err := homeschoolrecords.ListManualHours(r.Context(), d.Pool, studentID, win.From, win.To)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load hours entries.")
		return settings, win, hsvc.HoursLog{}, false
	}
	log := hsvc.BuildHoursLog(win.From, win.To, win.Label, auto, manual)
	return settings, win, log, true
}

func (d Deps) handleParentStudentHoursLog() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parentID, _, studentID, ok := d.parentRecordsAuth(w, r)
		if !ok {
			return
		}
		_, win, log, ok := d.buildStudentHoursLog(w, r, parentID, studentID)
		if !ok {
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"schoolYear":      win.Label,
			"from":            log.From,
			"to":              log.To,
			"entries":         log.Entries,
			"totalsBySubject": log.TotalsBySubject,
			"totalHours":      log.TotalHours,
		})
	}
}

func (d Deps) handleParentStudentHoursCreate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parentID, _, studentID, ok := d.parentRecordsAuth(w, r)
		if !ok {
			return
		}
		var body struct {
			Date     string  `json:"date"`
			Subject  string  `json:"subject"`
			CourseID string  `json:"courseId"`
			Hours    float64 `json:"hours"`
			Notes    string  `json:"notes"`
			Source   string  `json:"source"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		day, err := time.Parse("2006-01-02", strings.TrimSpace(body.Date))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "date must be YYYY-MM-DD.")
			return
		}
		if body.Hours < 0 || body.Hours > 24 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "hours must be between 0 and 24.")
			return
		}
		courseID, err := hsvc.ParseOptionalCourseID(body.CourseID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid courseId.")
			return
		}
		entry, err := homeschoolrecords.CreateHours(r.Context(), d.Pool, homeschoolrecords.CreateHoursInput{
			ParentUserID: parentID, StudentUserID: studentID,
			EntryDate: day, Subject: body.Subject, CourseID: courseID,
			Hours: body.Hours, Notes: body.Notes, Source: body.Source,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create hours entry.")
			return
		}
		writeJSON(w, http.StatusCreated, hoursEntryJSON(entry))
	}
}

func (d Deps) handleParentStudentHoursUpdate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parentID, _, studentID, ok := d.parentRecordsAuth(w, r)
		if !ok {
			return
		}
		entryID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "entryId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid entry id.")
			return
		}
		var body struct {
			Date     *string  `json:"date"`
			Subject  *string  `json:"subject"`
			CourseID *string  `json:"courseId"`
			Hours    *float64 `json:"hours"`
			Notes    *string  `json:"notes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		in := homeschoolrecords.UpdateHoursInput{
			Subject: body.Subject, Hours: body.Hours, Notes: body.Notes,
		}
		if body.Date != nil {
			day, err := time.Parse("2006-01-02", strings.TrimSpace(*body.Date))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "date must be YYYY-MM-DD.")
				return
			}
			in.EntryDate = &day
		}
		if body.CourseID != nil {
			cid, err := hsvc.ParseOptionalCourseID(*body.CourseID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid courseId.")
				return
			}
			in.CourseID = &cid
		}
		if body.Hours != nil && (*body.Hours < 0 || *body.Hours > 24) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "hours must be between 0 and 24.")
			return
		}
		entry, err := homeschoolrecords.UpdateHours(r.Context(), d.Pool, parentID, studentID, entryID, in)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update hours entry.")
			return
		}
		if entry == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Hours entry not found.")
			return
		}
		writeJSON(w, http.StatusOK, hoursEntryJSON(entry))
	}
}

func (d Deps) handleParentStudentHoursDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parentID, _, studentID, ok := d.parentRecordsAuth(w, r)
		if !ok {
			return
		}
		entryID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "entryId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid entry id.")
			return
		}
		okDel, err := homeschoolrecords.DeleteHours(r.Context(), d.Pool, parentID, studentID, entryID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to delete hours entry.")
			return
		}
		if !okDel {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Hours entry not found.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) handleParentStudentHoursExportCSV() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parentID, _, studentID, ok := d.parentRecordsAuth(w, r)
		if !ok {
			return
		}
		_, _, log, ok := d.buildStudentHoursLog(w, r, parentID, studentID)
		if !ok {
			return
		}
		b, err := hsvc.BuildHoursLogCSV(log)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to build CSV.")
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="hours-log.csv"`)
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(b)
	}
}

func (d Deps) handleParentStudentHoursExportPDF() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parentID, _, studentID, ok := d.parentRecordsAuth(w, r)
		if !ok {
			return
		}
		settings, win, log, ok := d.buildStudentHoursLog(w, r, parentID, studentID)
		if !ok {
			return
		}
		b, err := hsvc.BuildHoursLogPDF(hsvc.HoursLogPDFInput{
			SchoolName: settings.SchoolName, StudentName: d.loadStudentDisplayName(r, studentID),
			SchoolYear: win.Label, From: log.From, To: log.To, GeneratedAt: time.Now().UTC(), Log: log,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to build PDF.")
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", `attachment; filename="hours-log.pdf"`)
		w.Header().Set("Content-Length", strconv.Itoa(len(b)))
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(b)
	}
}

func (d Deps) handleParentStudentTranscriptPDF() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parentID, _, studentID, ok := d.parentRecordsAuth(w, r)
		if !ok {
			return
		}
		settings, win, ok := d.resolveSchoolYearWindow(w, r, parentID)
		if !ok {
			return
		}
		courses, err := homeschoolrecords.ListTranscriptCourses(r.Context(), d.Pool, studentID, win.From, win.To)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load transcript courses.")
			return
		}
		b, err := hsvc.BuildTranscriptPDF(hsvc.TranscriptPDFInput{
			SchoolName: settings.SchoolName, StudentName: d.loadStudentDisplayName(r, studentID),
			SchoolYear: win.Label, GeneratedAt: time.Now().UTC(), Courses: courses,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to build transcript PDF.")
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="transcript-%s.pdf"`, win.Label))
		w.Header().Set("Content-Length", strconv.Itoa(len(b)))
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(b)
	}
}

func (d Deps) handleParentStudentRecordSubmissions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parentID, _, studentID, ok := d.parentRecordsAuth(w, r)
		if !ok {
			return
		}
		_, win, ok := d.resolveSchoolYearWindow(w, r, parentID)
		if !ok {
			return
		}
		samples, err := homeschoolrecords.ListSubmissionSamples(r.Context(), d.Pool, studentID, win.From, win.To)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list submissions.")
			return
		}
		out := make([]map[string]any, 0, len(samples))
		for _, s := range samples {
			row := map[string]any{
				"submissionId":      s.SubmissionID.String(),
				"courseId":          s.CourseID.String(),
				"courseCode":        s.CourseCode,
				"courseTitle":       s.CourseTitle,
				"itemId":            s.ItemID.String(),
				"itemTitle":         s.ItemTitle,
				"submittedAt":       s.SubmittedAt.UTC().Format(time.RFC3339Nano),
				"bodyPreview":       s.BodyPreview,
				"instructorComment": s.InstructorComment,
			}
			if s.PointsEarned != nil {
				row["pointsEarned"] = *s.PointsEarned
			}
			out = append(out, row)
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"schoolYear":  win.Label,
			"from":        win.From.Format("2006-01-02"),
			"to":          win.To.Format("2006-01-02"),
			"submissions": out,
		})
	}
}

func (d Deps) handleParentStudentPortfolioExport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parentID, _, studentID, ok := d.parentRecordsAuth(w, r)
		if !ok {
			return
		}
		var body struct {
			SchoolYear    string   `json:"schoolYear"`
			SubmissionIDs []string `json:"submissionIds"`
			Format        string   `json:"format"` // pdf | zip
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		// Stash schoolYear into query for resolve helper.
		if strings.TrimSpace(body.SchoolYear) != "" {
			q := r.URL.Query()
			q.Set("schoolYear", body.SchoolYear)
			r.URL.RawQuery = q.Encode()
		}
		settings, win, ok := d.resolveSchoolYearWindow(w, r, parentID)
		if !ok {
			return
		}
		ids := make([]uuid.UUID, 0, len(body.SubmissionIDs))
		for _, s := range body.SubmissionIDs {
			id, err := uuid.Parse(strings.TrimSpace(s))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid submission id.")
				return
			}
			ids = append(ids, id)
		}
		if len(ids) > 50 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Select at most 50 submissions.")
			return
		}
		samples, err := homeschoolrecords.GetSubmissionSamplesByIDs(r.Context(), d.Pool, studentID, ids)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load submissions.")
			return
		}
		_, _, log, ok := d.buildStudentHoursLog(w, r, parentID, studentID)
		if !ok {
			return
		}
		pdfBytes, err := hsvc.BuildPortfolioPDF(hsvc.PortfolioPDFInput{
			SchoolName: settings.SchoolName, StudentName: d.loadStudentDisplayName(r, studentID),
			SchoolYear: win.Label, GeneratedAt: time.Now().UTC(),
			Samples: samples, HoursTotals: log.TotalsBySubject, TotalHours: log.TotalHours,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to build portfolio PDF.")
			return
		}
		format := strings.ToLower(strings.TrimSpace(body.Format))
		if format == "" {
			format = "pdf"
		}
		switch format {
		case "pdf":
			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("Content-Disposition", `attachment; filename="portfolio.pdf"`)
			w.Header().Set("Content-Length", strconv.Itoa(len(pdfBytes)))
			w.Header().Set("Cache-Control", "no-store")
			_, _ = w.Write(pdfBytes)
		case "zip":
			slug := strings.ReplaceAll(strings.ToLower(d.loadStudentDisplayName(r, studentID)), " ", "-")
			if slug == "" {
				slug = "student"
			}
			zipBytes, err := hsvc.BuildPortfolioZIP(pdfBytes, log, slug)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to build portfolio ZIP.")
				return
			}
			w.Header().Set("Content-Type", "application/zip")
			w.Header().Set("Content-Disposition", `attachment; filename="portfolio.zip"`)
			w.Header().Set("Content-Length", strconv.Itoa(len(zipBytes)))
			w.Header().Set("Cache-Control", "no-store")
			_, _ = w.Write(zipBytes)
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "format must be pdf or zip.")
		}
	}
}
