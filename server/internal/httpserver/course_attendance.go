package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/attendancesessions"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
)

func (d Deps) attendanceFeatureOff(w http.ResponseWriter, r *http.Request, courseCode string) bool {
	enabled, err := attendancesessions.AttendanceEnabledForCourseCode(r.Context(), d.Pool, courseCode)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
		return true
	}
	if !enabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Attendance is not enabled for this course.")
		return true
	}
	return false
}

func (d Deps) requireAttendanceManage(w http.ResponseWriter, r *http.Request, courseCode string, viewer uuid.UUID) bool {
	ok, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":attendance:manage")
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return false
	}
	if !ok {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to manage attendance.")
		return false
	}
	return true
}

func sessionToJSON(s *attendancesessions.Session) map[string]any {
	out := map[string]any{
		"id":               s.ID.String(),
		"title":            s.Title,
		"collectionMethod": s.CollectionMethod,
		"sessionDate":      attendancesessions.FormatSessionDate(s.SessionDate),
		"status":           s.Status,
		"gradebookEnabled": s.GradebookEnabled,
		"createdAt":        s.CreatedAt.UTC().Format(time.RFC3339Nano),
		"updatedAt":        s.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if s.SectionID != nil {
		out["sectionId"] = s.SectionID.String()
	}
	if s.StructureItemID != nil {
		out["structureItemId"] = s.StructureItemID.String()
	}
	if s.OpensAt != nil {
		out["opensAt"] = s.OpensAt.UTC().Format(time.RFC3339Nano)
	}
	if s.ClosesAt != nil {
		out["closesAt"] = s.ClosesAt.UTC().Format(time.RFC3339Nano)
	}
	if s.PointsPossible != nil {
		out["pointsPossible"] = *s.PointsPossible
	}
	if s.ClosedAt != nil {
		out["closedAt"] = s.ClosedAt.UTC().Format(time.RFC3339Nano)
	}
	return out
}

func sessionRecordToJSON(r attendancesessions.RecordRow, includeDetails bool) map[string]any {
	out := map[string]any{
		"studentUserId": r.StudentUserID.String(),
		"status":        r.Status,
	}
	if includeDetails {
		out["displayName"] = r.DisplayName
		out["source"] = r.Source
		if !r.RecordedAt.IsZero() {
			out["recordedAt"] = r.RecordedAt.UTC().Format(time.RFC3339Nano)
		}
		if r.RecordedBy != nil {
			out["recordedBy"] = r.RecordedBy.String()
		}
	}
	return out
}

func (d Deps) loadSessionRoster(ctx context.Context, courseCode string, courseID uuid.UUID, sectionID *uuid.UUID) ([]struct {
	UserID      uuid.UUID
	DisplayName string
}, error) {
	var filter []uuid.UUID
	if sectionID != nil {
		filter = []uuid.UUID{*sectionID}
	}
	rows, err := enrollment.ListStudentUsersForCourseCode(ctx, d.Pool, courseCode, filter)
	if err != nil {
		return nil, err
	}
	out := make([]struct {
		UserID      uuid.UUID
		DisplayName string
	}, len(rows))
	for i, r := range rows {
		out[i].UserID = r.UserID
		out[i].DisplayName = r.DisplayName
	}
	return out, nil
}

func (d Deps) handleCourseAttendanceSessionsList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.attendanceFeatureOff(w, r, courseCode) {
			return
		}
		canManage, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":attendance:manage")
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return
		}
		if !canManage {
			enrolled, err := enrollment.UserHasStudentEquivalentEnrollment(r.Context(), d.Pool, courseCode, viewer)
			if err != nil || !enrolled {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have access to attendance.")
				return
			}
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		sessions, err := attendancesessions.ListSessions(r.Context(), d.Pool, *cid, 100)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load sessions.")
			return
		}
		out := make([]map[string]any, 0, len(sessions))
		for i := range sessions {
			out = append(out, sessionToJSON(&sessions[i]))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"sessions": out})
	}
}

func (d Deps) handleCourseAttendanceSessionsPost() http.HandlerFunc {
	type body struct {
		CollectionMethod string  `json:"collectionMethod"`
		Title            string  `json:"title"`
		SessionDate      string  `json:"sessionDate"`
		SectionID        *string `json:"sectionId"`
		GradebookEnabled bool    `json:"gradebookEnabled"`
		PointsPossible   *int    `json:"pointsPossible"`
		OpensAt          *string `json:"opensAt"`
		ClosesAt         *string `json:"closesAt"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.attendanceFeatureOff(w, r, courseCode) {
			return
		}
		if !d.requireAttendanceManage(w, r, courseCode, viewer) {
			return
		}
		var req body
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		dateStr := strings.TrimSpace(req.SessionDate)
		if dateStr == "" {
			dateStr = time.Now().UTC().Format("2006-01-02")
		}
		sessionDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid sessionDate; use YYYY-MM-DD.")
			return
		}
		title := strings.TrimSpace(req.Title)
		if title == "" {
			title = "Attendance — " + dateStr
		}
		var sectionID *uuid.UUID
		if req.SectionID != nil && strings.TrimSpace(*req.SectionID) != "" {
			sid, err := uuid.Parse(strings.TrimSpace(*req.SectionID))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid sectionId.")
				return
			}
			okSec, err := attendancesessions.ValidateSectionInCourse(r.Context(), d.Pool, *cid, sid)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to validate section.")
				return
			}
			if !okSec {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Section not found in course.")
				return
			}
			sectionID = &sid
		}
		var opensAt, closesAt *time.Time
		if req.CollectionMethod == attendancesessions.CollectionSelfReport {
			now := time.Now().UTC()
			o, c := attendancesessions.DefaultSelfReportWindow(now)
			opensAt, closesAt = &o, &c
			if req.OpensAt != nil && strings.TrimSpace(*req.OpensAt) != "" {
				t, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.OpensAt))
				if err != nil {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid opensAt.")
					return
				}
				opensAt = &t
			}
			if req.ClosesAt != nil && strings.TrimSpace(*req.ClosesAt) != "" {
				t, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.ClosesAt))
				if err != nil {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid closesAt.")
					return
				}
				closesAt = &t
			}
		}
		sess, err := attendancesessions.CreateSession(r.Context(), d.Pool, attendancesessions.CreateInput{
			CourseID:         *cid,
			SectionID:        sectionID,
			Title:            title,
			CollectionMethod: req.CollectionMethod,
			SessionDate:      sessionDate,
			OpensAt:          opensAt,
			ClosesAt:         closesAt,
			GradebookEnabled: req.GradebookEnabled,
			PointsPossible:   req.PointsPossible,
			CreatedBy:        viewer,
		})
		if err != nil {
			if errors.Is(err, attendancesessions.ErrInvalidCollection) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid collectionMethod.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create session.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(sessionToJSON(sess))
	}
}

func (d Deps) handleCourseAttendanceSessionGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.attendanceFeatureOff(w, r, courseCode) {
			return
		}
		sessionID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "session_id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid session id.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		canManage, _ := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":attendance:manage")
		if !canManage {
			enrolled, err := enrollment.UserHasStudentEquivalentEnrollment(r.Context(), d.Pool, courseCode, viewer)
			if err != nil || !enrolled {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have access to this session.")
				return
			}
		}
		sess, err := attendancesessions.GetSession(r.Context(), d.Pool, *cid, sessionID)
		if err != nil {
			if errors.Is(err, attendancesessions.ErrSessionNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Session not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load session.")
			return
		}
		out := sessionToJSON(sess)
		if canManage {
			roster, err := d.loadSessionRoster(r.Context(), courseCode, *cid, sess.SectionID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load roster.")
				return
			}
			recMap, err := attendancesessions.ListRecordsForSession(r.Context(), d.Pool, sessionID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load records.")
				return
			}
			merged := attendancesessions.MergeRosterWithRecords(roster, recMap)
			recs := make([]map[string]any, 0, len(merged))
			for _, rec := range merged {
				recs = append(recs, sessionRecordToJSON(rec, true))
			}
			out["records"] = recs
		} else {
			recMap, err := attendancesessions.ListRecordsForSession(r.Context(), d.Pool, sessionID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load records.")
				return
			}
			if rec, ok := recMap[viewer]; ok {
				out["myRecord"] = sessionRecordToJSON(rec, false)
			}
			out["canSelfReport"] = attendancesessions.IsSelfReportWindowOpen(*sess, time.Now().UTC())
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

func (d Deps) handleCourseAttendanceSessionRecordsPut() http.HandlerFunc {
	type row struct {
		StudentUserID string `json:"studentUserId"`
		Status        string `json:"status"`
		Source        string `json:"source"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.Header().Set("Allow", http.MethodPut)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.attendanceFeatureOff(w, r, courseCode) {
			return
		}
		if !d.requireAttendanceManage(w, r, courseCode, viewer) {
			return
		}
		sessionID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "session_id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid session id.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		if _, err := attendancesessions.GetSession(r.Context(), d.Pool, *cid, sessionID); err != nil {
			if errors.Is(err, attendancesessions.ErrSessionNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Session not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load session.")
			return
		}
		var body struct {
			Records []row `json:"records"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		upserts := make([]attendancesessions.RecordUpsert, 0, len(body.Records))
		for _, rec := range body.Records {
			suid, err := uuid.Parse(strings.TrimSpace(rec.StudentUserID))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid studentUserId.")
				return
			}
			src := strings.TrimSpace(rec.Source)
			if src == "" {
				src = "instructor"
			}
			if src == "override" {
				src = "override"
			}
			upserts = append(upserts, attendancesessions.RecordUpsert{
				StudentUserID: suid,
				Status:        rec.Status,
				Source:        src,
				RecordedBy:    viewer,
			})
		}
		sess, _ := attendancesessions.GetSession(r.Context(), d.Pool, *cid, sessionID)
		allowClosed := sess != nil && sess.Status == attendancesessions.StatusClosed
		if err := attendancesessions.BatchUpsertRecords(r.Context(), d.Pool, sessionID, upserts, allowClosed); err != nil {
			if errors.Is(err, attendancesessions.ErrSessionClosed) {
				apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Session is closed.")
				return
			}
			if errors.Is(err, attendancesessions.ErrInvalidStatus) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid status.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save records.")
			return
		}
		if allowClosed && sess != nil && sess.GradebookEnabled {
			_ = attendancesessions.SyncGradebook(r.Context(), d.Pool, sessionID)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"saved":   len(upserts),
			"message": "Attendance saved.",
		})
	}
}

func (d Deps) handleCourseAttendanceSessionSelfReport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.attendanceFeatureOff(w, r, courseCode) {
			return
		}
		enrolled, err := enrollment.UserHasStudentEquivalentEnrollment(r.Context(), d.Pool, courseCode, viewer)
		if err != nil || !enrolled {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Only enrolled students can self-report.")
			return
		}
		sessionID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "session_id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid session id.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		if _, err := attendancesessions.GetSession(r.Context(), d.Pool, *cid, sessionID); err != nil {
			if errors.Is(err, attendancesessions.ErrSessionNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Session not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load session.")
			return
		}
		var body struct {
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		status := strings.TrimSpace(body.Status)
		if status == "" {
			status = "present"
		}
		if err := attendancesessions.SelfReport(r.Context(), d.Pool, sessionID, viewer, status); err != nil {
			switch {
			case errors.Is(err, attendancesessions.ErrSelfReportClosed):
				apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Self-report window is closed.")
			case errors.Is(err, attendancesessions.ErrAlreadySubmitted):
				apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "You have already checked in.")
			case errors.Is(err, attendancesessions.ErrInvalidStatus):
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid status.")
			default:
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to submit check-in.")
			}
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": status, "message": "Check-in recorded."})
	}
}

func (d Deps) handleCourseAttendanceSessionClose() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.attendanceFeatureOff(w, r, courseCode) {
			return
		}
		if !d.requireAttendanceManage(w, r, courseCode, viewer) {
			return
		}
		sessionID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "session_id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid session id.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		var body struct {
			FinalizeMissingAsAbsent bool `json:"finalizeMissingAsAbsent"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		sess, err := attendancesessions.CloseSession(r.Context(), d.Pool, *cid, sessionID, body.FinalizeMissingAsAbsent)
		if err != nil {
			if errors.Is(err, attendancesessions.ErrSessionNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Session not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to close session.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(sessionToJSON(sess))
	}
}
