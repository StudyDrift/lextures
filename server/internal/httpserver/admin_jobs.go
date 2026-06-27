package httpserver

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/jobqueue"
)

func jobsMethodNotAllowed(w http.ResponseWriter, allow string) {
	w.Header().Set("Allow", allow)
	http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
}

// handleAdminJobsList returns queue stats plus a recent-jobs list for the admin
// jobs page (plan 17.3 §9 GET /admin/jobs).
func (d Deps) handleAdminJobsList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			jobsMethodNotAllowed(w, http.MethodGet)
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		stats, err := jobqueue.GetStats(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		jobs, err := jobqueue.ListJobs(r.Context(), d.Pool, 100)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"stats": stats,
			"jobs":  jobs,
		})
	}
}

// handleAdminJobsDeadLetters lists dead-letter jobs (plan 17.3 §9).
func (d Deps) handleAdminJobsDeadLetters() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			jobsMethodNotAllowed(w, http.MethodGet)
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		rows, err := jobqueue.ListDeadLetters(r.Context(), d.Pool, 100)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"deadLetters": rows})
	}
}

// handleAdminJobsRedrive re-enqueues a dead-letter job (plan 17.3 AC-5).
func (d Deps) handleAdminJobsRedrive() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			jobsMethodNotAllowed(w, http.MethodPost)
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid job id.")
			return
		}
		newID, err := jobqueue.Redrive(r.Context(), d.Pool, id, time.Now().UTC())
		if errors.Is(err, jobqueue.ErrNotFound) {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Dead-letter job not found.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"jobId": newID.String()})
	}
}

// handleAdminJobsCancel deletes a pending job (plan 17.3 §9 DELETE /admin/jobs/{id}).
func (d Deps) handleAdminJobsCancel() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			jobsMethodNotAllowed(w, http.MethodDelete)
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid job id.")
			return
		}
		err = jobqueue.Cancel(r.Context(), d.Pool, id)
		if errors.Is(err, jobqueue.ErrNotFound) {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Pending job not found.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

func (d Deps) registerAdminJobRoutes(r chi.Router) {
	r.Get("/api/v1/admin/jobs", d.handleAdminJobsList())
	r.Get("/api/v1/admin/jobs/dead-letters", d.handleAdminJobsDeadLetters())
	r.Post("/api/v1/admin/jobs/dead-letters/{id}/redrive", d.handleAdminJobsRedrive())
	r.Delete("/api/v1/admin/jobs/{id}", d.handleAdminJobsCancel())
}
