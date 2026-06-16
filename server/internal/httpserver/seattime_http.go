package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/logging"
	"github.com/lextures/lextures/server/internal/notificationevents"
	"github.com/lextures/lextures/server/internal/repos/course"
	reposeattime "github.com/lextures/lextures/server/internal/repos/seattime"
	"github.com/lextures/lextures/server/internal/service/notifications"
	seattimesvc "github.com/lextures/lextures/server/internal/service/seattime"
)

func (d Deps) ceuTrackingFeatureOff(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFCEUTracking {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "CEU tracking is not enabled.")
		return true
	}
	return false
}

func (d Deps) registerSeatTimeRoutes(r chi.Router) {
	r.Post("/api/v1/seat-time/heartbeat", d.handleSeatTimeHeartbeat())
	r.Get("/api/v1/me/seat-time", d.handleGetMySeatTime())
	r.Get("/api/v1/me/ce-transcript", d.handleGetMyCETranscript())
	r.Get("/api/v1/courses/{course_code}/seat-time-report", d.handleGetCourseSeatTimeReport())
	r.Post("/api/v1/admin/courses/{course_code}/ceu-config", d.handlePostAdminCEUConfig())
}

func (d Deps) seatTimeBuffer() *seattimesvc.Buffer {
	if seattimesvc.GlobalBuffer == nil && d.Pool != nil {
		seattimesvc.InitGlobalBuffer(d.Pool)
	}
	return seattimesvc.GlobalBuffer
}

func (d Deps) handleSeatTimeHeartbeat() http.HandlerFunc {
	type body struct {
		ContentItemID string `json:"contentItemId"`
		SessionToken  string `json:"sessionToken"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.ceuTrackingFeatureOff(w) {
			return
		}
		var req body
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		itemID, err := uuid.Parse(strings.TrimSpace(req.ContentItemID))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid content item id.")
			return
		}
		token := strings.TrimSpace(req.SessionToken)
		if token == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "sessionToken is required.")
			return
		}

		buf := d.seatTimeBuffer()
		if buf == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Seat-time buffer unavailable.")
			return
		}

		now := time.Now().UTC()
		result, err := buf.ProcessHeartbeat(r.Context(), userID, itemID, token, now)
		if err != nil {
			if errors.Is(err, seattimesvc.ErrNotEnrolled) {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You are not enrolled in this course.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to process heartbeat.")
			return
		}

		logging.GlobalSeatTimeMetrics.IncHeartbeats()
		if result.AnomalyFlag {
			logging.GlobalSeatTimeMetrics.IncAnomalies()
		}
		if result.Counted {
			logging.GlobalSeatTimeMetrics.AddMinutes(1)
		}

		meta, _ := reposeattime.ResolveContentItemCourse(r.Context(), d.Pool, itemID)
		if meta != nil && result.Counted {
			award, created, err := seattimesvc.MaybeIssueCEUAward(r.Context(), d.Pool, userID, meta.CourseID, now)
			if err == nil && created && award != nil {
				logging.GlobalSeatTimeMetrics.IncAwards()
				d.notifyCEUAward(r, userID, meta.CourseID, award)
			}
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"minutesActive": result.MinutesActive,
			"counted":       result.Counted,
			"anomalyFlag":   result.AnomalyFlag,
		})
	}
}

func (d Deps) notifyCEUAward(r *http.Request, userID, courseID uuid.UUID, award *reposeattime.CEUAward) {
	cfg := d.effectiveConfig()
	if !cfg.EmailNotificationsEnabled || d.Pool == nil {
		return
	}
	title, _ := reposeattime.CourseTitle(r.Context(), d.Pool, courseID)
	svc := notifications.Service{Pool: d.Pool, Config: cfg}
	_ = svc.EnqueueEmail(r.Context(), userID, notificationevents.CEUAwarded, "ceu_awarded", map[string]string{
		"subject":     "CEU certificate earned",
		"courseTitle": title,
		"ceuCredit":   formatCEU(award.CEUCredit),
		"link":        cfg.PublicWebOrigin + "/me/ce-transcript",
	}, nil)
}

func formatCEU(v float64) string {
	return strings.TrimRight(strings.TrimRight(strings.TrimSpace(
		strings.Replace(jsonNumber(v), ",", ".", 1),
	), "0"), ".")
}

func jsonNumber(v float64) string {
	b, _ := json.Marshal(v)
	return strings.Trim(string(b), "\"")
}

func (d Deps) handleGetMySeatTime() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.ceuTrackingFeatureOff(w) {
			return
		}
		courseIDStr := strings.TrimSpace(r.URL.Query().Get("courseId"))
		if courseIDStr == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "courseId query parameter is required.")
			return
		}
		courseID, err := uuid.Parse(courseIDStr)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid course id.")
			return
		}
		progress, err := seattimesvc.LoadProgress(r.Context(), d.Pool, userID, courseID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load seat time.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"totalMinutes":  progress.TotalMinutes,
			"requiredHours": progress.RequiredHours,
			"ceuEarned":     progress.CEUEarned,
			"progressPct":   progress.ProgressPct,
			"awarded":       progress.Awarded,
		})
	}
}

func (d Deps) handleGetMyCETranscript() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.ceuTrackingFeatureOff(w) {
			return
		}
		ctx := r.Context()
		awards, err := reposeattime.ListCEUAwardsForUser(ctx, d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load CE transcript.")
			return
		}

		titles := make(map[string]string, len(awards))
		rows := make([]seattimesvc.TranscriptRow, 0, len(awards))
		for _, a := range awards {
			title, _ := reposeattime.CourseTitle(ctx, d.Pool, a.CourseID)
			titles[a.CourseID.String()] = title
			rows = append(rows, seattimesvc.TranscriptRow{
				CourseTitle:  title,
				CEUCredit:    a.CEUCredit,
				ContactHours: a.ContactHours,
				CompletedAt:  a.IssuedAt,
			})
		}

		format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
		if format == "pdf" {
			learnerName, err := d.learnerDisplayName(r, userID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load profile.")
				return
			}
			cfg := d.effectiveConfig()
			institution := strings.TrimSpace(cfg.CCRInstitutionName)
			if institution == "" {
				institution = "Lextures"
			}
			pdfBytes, err := seattimesvc.BuildTranscriptPDF(seattimesvc.TranscriptInput{
				InstitutionName: institution,
				LearnerName:     learnerName,
				GeneratedAt:     time.Now().UTC(),
				Rows:            rows,
			})
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to render transcript PDF.")
				return
			}
			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("Content-Disposition", `attachment; filename="ce-transcript.pdf"`)
			_, _ = w.Write(pdfBytes)
			return
		}

		out := make([]map[string]any, 0, len(awards))
		for _, row := range rows {
			out = append(out, map[string]any{
				"courseTitle":  row.CourseTitle,
				"ceuCredit":    row.CEUCredit,
				"contactHours": row.ContactHours,
				"completedAt":  row.CompletedAt.UTC().Format(time.RFC3339),
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"awards": out})
	}
}

func (d Deps) handleGetCourseSeatTimeReport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.ceuTrackingFeatureOff(w) {
			return
		}
		has, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":gradebook:view")
		if err != nil || !has {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to view seat-time reports.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		rows, err := reposeattime.CourseSeatTimeReport(r.Context(), d.Pool, *cid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load seat-time report.")
			return
		}
		cfg, _ := reposeattime.GetCEUConfig(r.Context(), d.Pool, *cid)
		requiredHours := 0.0
		if cfg != nil {
			requiredHours = cfg.RequiredHours
		}
		students := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			students = append(students, map[string]any{
				"userId":       row.UserID.String(),
				"displayName":  row.DisplayName,
				"totalMinutes": row.TotalMinutes,
				"contactHours": float64(row.TotalMinutes) / 60.0,
				"ceuEarned":    row.CEUEarned,
				"requiredHours": requiredHours,
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"students": students})
	}
}

func (d Deps) handlePostAdminCEUConfig() http.HandlerFunc {
	type body struct {
		RequiredHours       float64 `json:"requiredHours"`
		CEUCredit           float64 `json:"ceuCredit"`
		CertificateTemplate *string `json:"certificateTemplate"`
		Enabled             *bool   `json:"enabled"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		if d.ceuTrackingFeatureOff(w) {
			return
		}
		courseCode := strings.TrimSpace(chi.URLParam(r, "course_code"))
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		var req body
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if req.RequiredHours <= 0 || req.CEUCredit <= 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "requiredHours and ceuCredit must be positive.")
			return
		}
		enabled := true
		if req.Enabled != nil {
			enabled = *req.Enabled
		}
		cfg, err := reposeattime.UpsertCEUConfig(r.Context(), d.Pool, *cid, req.RequiredHours, req.CEUCredit, req.CertificateTemplate, enabled)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save CEU configuration.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"courseId":      cfg.CourseID.String(),
			"requiredHours": cfg.RequiredHours,
			"ceuCredit":     cfg.CEUCredit,
			"enabled":       cfg.Enabled,
		})
	}
}
