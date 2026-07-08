package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/organization"
	icrepo "github.com/lextures/lextures/server/internal/repos/introcourse"
	auditservice "github.com/lextures/lextures/server/internal/service/adminaudit"
	introcourseservice "github.com/lextures/lextures/server/internal/service/introcourse"
)

func (d Deps) introCourseService() *introcourseservice.Service {
	if d.IntroCourseService != nil {
		return d.IntroCourseService
	}
	if d.Pool == nil {
		return nil
	}
	return introcourseservice.New(d.Pool)
}

func (d Deps) registerIntroCourseMeRoutes(r chi.Router) {
	r.Get("/api/v1/me/intro-course", d.handleMeIntroCourse())
	r.Put("/api/v1/me/intro-course/welcome-banner-dismissed", d.handleMeIntroCourseWelcomeBannerDismissed())
	r.Put("/api/v1/me/intro-course/celebration-seen", d.handleMeIntroCourseCelebrationSeen())
}

func (d Deps) registerIntroCourseAdminRoutes(r chi.Router) {
	r.Get("/api/v1/admin/intro-course", d.handleAdminIntroCourseStatus())
	r.Post("/api/v1/admin/intro-course/resync", d.handleAdminIntroCourseResync())
	r.Post("/api/v1/admin/intro-course/backfill", d.handleAdminIntroCourseBackfillStart())
	r.Get("/api/v1/admin/intro-course/backfill", d.handleAdminIntroCourseBackfillStatus())
	r.Get("/api/v1/admin/intro-course/analytics", d.handleAdminIntroCourseAnalytics())
}

func (d Deps) recordIntroCourseAdminAudit(r *http.Request, actorID uuid.UUID, action string, after map[string]any) {
	introcourseservice.RecordAdminAction(action)
	if !d.effectiveConfig().AdminAuditLogEnabled || d.Pool == nil {
		return
	}
	orgID, _ := organization.OrgIDForUser(r.Context(), d.Pool, actorID)
	var oid *uuid.UUID
	if orgID != uuid.Nil {
		oid = &orgID
	}
	tt := "intro_course"
	payload, _ := json.Marshal(after)
	ip := clientIP(r)
	ua := r.UserAgent()
	_, _ = auditservice.Record(r.Context(), d.Pool, auditservice.RecordParams{
		OrgID:      oid,
		EventType:  auditservice.EventIntroCourseAdmin,
		ActorID:    actorID,
		ActorIP:    &ip,
		UserAgent:  &ua,
		TargetType: &tt,
		AfterValue: payload,
	})
}

// handleAdminIntroCourseStatus is GET /api/v1/admin/intro-course (IC08).
func (d Deps) handleAdminIntroCourseStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		cfg := d.effectiveConfig()
		svc := d.introCourseService()
		if svc == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		status, err := introcourseservice.LoadAdminStatus(r.Context(), d.Pool, svc, cfg)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load intro course status.")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(status)
	}
}

// handleMeIntroCourse is GET /api/v1/me/intro-course (IC05).
func (d Deps) handleMeIntroCourse() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		cfg := d.effectiveConfig()
		if !introcourseservice.Enabled(cfg) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"enrolled":        false,
				"modulesComplete": 0,
				"modulesTotal":    7,
				"percent":         0,
				"courseCode":      introcourseservice.CourseCode,
			})
			return
		}
		svc := d.introCourseService()
		if svc == nil || d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		courseID, found, err := svc.CourseID(r.Context())
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load intro course.")
			return
		}
		if !found || courseID == uuid.Nil {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"enrolled":        false,
				"modulesComplete": 0,
				"modulesTotal":    7,
				"percent":         0,
				"courseCode":      introcourseservice.CourseCode,
			})
			return
		}
		prog, err := introcourseservice.LoadProgress(r.Context(), d.Pool, cfg, courseID, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load intro course progress.")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(prog)
	}
}

// handleAdminIntroCourseAnalytics is GET /api/v1/admin/intro-course/analytics (IC05/IC08).
func (d Deps) handleAdminIntroCourseAnalytics() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		cfg := d.effectiveConfig()
		svc := d.introCourseService()
		if svc == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		courseID, found, err := svc.CourseID(r.Context())
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load intro course.")
			return
		}
		if !found || courseID == uuid.Nil || !introcourseservice.Enabled(cfg) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(introcourseservice.Analytics{})
			return
		}
		analytics, err := introcourseservice.LoadAnalytics(r.Context(), d.Pool, courseID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load intro course analytics.")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(analytics)
	}
}

// handleAdminIntroCourseResync is POST /api/v1/admin/intro-course/resync (IC01/IC08).
func (d Deps) handleAdminIntroCourseResync() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		actorID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		svc := d.introCourseService()
		if svc == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		if _, err := introcourseservice.RunValidation(r.Context(), d.Pool); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Intro course content validation failed.")
			return
		}
		course, err := svc.EnsureProvisioned(r.Context(), d.effectiveConfig())
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Intro course provisioning failed.")
			return
		}
		if course.ID == uuid.Nil {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeInvalidInput, "Intro course is disabled and has not been provisioned.")
			return
		}
		contentReport, contentErr := svc.SyncContentForCourse(r.Context(), d.effectiveConfig(), course.ID)
		if contentErr != nil {
			_ = introcourseservice.RecordSyncStatus(r.Context(), d.Pool, contentReport, contentErr)
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Intro course content sync failed.")
			return
		}
		status := "reconciled"
		if course.Created {
			status = "created"
		}
		d.recordIntroCourseAdminAudit(r, actorID, "resync", map[string]any{
			"action":         "resync",
			"courseId":       course.ID.String(),
			"status":         status,
			"contentVersion": contentReport.ContentVersion,
			"contentSkipped": contentReport.Skipped,
			"modulesSynced":  contentReport.Modules,
			"itemsArchived":  contentReport.Archived,
		})
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"courseId":        course.ID,
			"status":          status,
			"contentVersion":  contentReport.ContentVersion,
			"contentSkipped":  contentReport.Skipped,
			"modulesSynced":   contentReport.Modules,
			"pagesSynced":     contentReport.Pages,
			"quizzesSynced":   contentReport.Quizzes,
			"itemsArchived":   contentReport.Archived,
		})
	}
}

// handleAdminIntroCourseBackfillStart is POST /api/v1/admin/intro-course/backfill (IC02/IC08).
func (d Deps) handleAdminIntroCourseBackfillStart() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		actorID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		cfg := d.effectiveConfig()
		if !introcourseservice.Enabled(cfg) {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeInvalidInput, "Intro course is disabled.")
			return
		}
		_, err := introcourseservice.EnqueueBackfillIfNeeded(r.Context(), d.Pool, cfg)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to queue intro course backfill.")
			return
		}
		svc := d.introCourseService()
		st, err := svc.BackfillStatus(r.Context(), cfg)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load backfill status.")
			return
		}
		d.recordIntroCourseAdminAudit(r, actorID, "backfill", map[string]any{
			"action":    "backfill",
			"remaining": st.Remaining,
			"startedAt": formatOptionalTime(st.StartedAt),
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"startedAt": formatOptionalTime(st.StartedAt),
			"remaining": st.Remaining,
		})
	}
}

// handleAdminIntroCourseBackfillStatus is GET /api/v1/admin/intro-course/backfill (IC02).
func (d Deps) handleAdminIntroCourseBackfillStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		svc := d.introCourseService()
		if svc == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		st, err := svc.BackfillStatus(r.Context(), d.effectiveConfig())
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load backfill status.")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"startedAt":     formatOptionalTime(st.StartedAt),
			"completedAt":   formatOptionalTime(st.CompletedAt),
			"enrolledCount": st.EnrolledCount,
			"remaining":     st.Remaining,
		})
	}
}

func formatOptionalTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}

// handleMeIntroCourseWelcomeBannerDismissed is PUT /api/v1/me/intro-course/welcome-banner-dismissed (IC06).
func (d Deps) handleMeIntroCourseWelcomeBannerDismissed() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.Header().Set("Allow", http.MethodPut)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		cfg := d.effectiveConfig()
		if !introcourseservice.Enabled(cfg) || d.Pool == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if err := icrepo.SetWelcomeBannerDismissed(r.Context(), d.Pool, userID, time.Now().UTC()); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save banner dismissal.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleMeIntroCourseCelebrationSeen is PUT /api/v1/me/intro-course/celebration-seen (IC06).
func (d Deps) handleMeIntroCourseCelebrationSeen() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.Header().Set("Allow", http.MethodPut)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		cfg := d.effectiveConfig()
		if !introcourseservice.Enabled(cfg) || d.Pool == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if err := icrepo.SetCelebrationSeen(r.Context(), d.Pool, userID, time.Now().UTC()); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save celebration state.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}