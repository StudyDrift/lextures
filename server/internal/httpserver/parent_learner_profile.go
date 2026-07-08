package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	lpsvc "github.com/lextures/lextures/server/internal/service/learnerprofile"
)

func (d Deps) parentLearnerProfileEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().LearnerProfileEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Learner profile is not enabled.")
		return false
	}
	return true
}

func (d Deps) handleParentStudentLearnerProfile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.parentLearnerProfileEnabled(w) {
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
		svc := d.learnerProfileService()
		if svc == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Learner profile service unavailable.")
			return
		}
		profile, err := svc.Get(r.Context(), studentID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load learner profile.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"studentUserId": studentID.String(),
			"profile":       learnerProfileToJSON(profile),
		})
	}
}

func (d Deps) handleParentStudentLearnerProfileControl(action string, fn func(*lpsvc.Service, uuid.UUID, *http.Request) error, responseStatus string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.parentLearnerProfileEnabled(w) {
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
		if !d.checkLearnerProfileControlRateLimit(parentID) {
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Too many profile control requests. Try again later.")
			return
		}
		svc := d.learnerProfileService()
		if svc == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Learner profile service unavailable.")
			return
		}
		if err := fn(svc, studentID, r); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not update learner profile.")
			return
		}
		d.recordLearnerProfileControlAudit(r, parentID, studentID, action)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"studentUserId": studentID.String(),
			"status":        responseStatus,
		})
	}
}

func (d Deps) handleParentStudentLearnerProfilePause() http.HandlerFunc {
	return d.handleParentStudentLearnerProfileControl("pause", func(svc *lpsvc.Service, studentID uuid.UUID, r *http.Request) error {
		return svc.Pause(r.Context(), studentID)
	}, "paused")
}

func (d Deps) handleParentStudentLearnerProfileResume() http.HandlerFunc {
	return d.handleParentStudentLearnerProfileControl("resume", func(svc *lpsvc.Service, studentID uuid.UUID, r *http.Request) error {
		return svc.Resume(r.Context(), studentID)
	}, "active")
}

func (d Deps) handleParentStudentLearnerProfileReset() http.HandlerFunc {
	return d.handleParentStudentLearnerProfileControl("reset", func(svc *lpsvc.Service, studentID uuid.UUID, r *http.Request) error {
		return svc.Reset(r.Context(), studentID)
	}, "reset")
}

func (d Deps) handleParentStudentLearnerProfileExport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.parentLearnerProfileEnabled(w) {
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
		if !d.checkLearnerProfileControlRateLimit(parentID) {
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Too many profile control requests. Try again later.")
			return
		}
		svc := d.learnerProfileService()
		if svc == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Learner profile service unavailable.")
			return
		}
		doc, err := svc.Export(r.Context(), studentID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not export learner profile.")
			return
		}
		d.recordLearnerProfileControlAudit(r, parentID, studentID, "export")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="learner-profile-export.json"`)
		_ = json.NewEncoder(w).Encode(doc)
	}
}

func (d Deps) registerParentLearnerProfileRoutes(r chi.Router) {
	r.Get("/api/v1/parent/students/{sid}/learner-profile", d.handleParentStudentLearnerProfile())
	r.Post("/api/v1/parent/students/{sid}/learner-profile/pause", d.handleParentStudentLearnerProfilePause())
	r.Post("/api/v1/parent/students/{sid}/learner-profile/resume", d.handleParentStudentLearnerProfileResume())
	r.Post("/api/v1/parent/students/{sid}/learner-profile/reset", d.handleParentStudentLearnerProfileReset())
	r.Get("/api/v1/parent/students/{sid}/learner-profile/export", d.handleParentStudentLearnerProfileExport())
}