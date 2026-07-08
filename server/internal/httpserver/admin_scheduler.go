package httpserver

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/scheduler"
)

// handleAdminSchedulerList returns every configured scheduled job with its
// enabled state, last run, last status and next run (plan 17.4 §9 GET
// /admin/scheduler).
func (d Deps) handleAdminSchedulerList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			jobsMethodNotAllowed(w, http.MethodGet)
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		if d.Scheduler == nil || d.Pool == nil {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeInternal, "Scheduler not configured.")
			return
		}
		jobs, err := d.Scheduler.Summary(r.Context(), time.Now().UTC())
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		cfg := d.effectiveConfig()
		writeJSON(w, http.StatusOK, map[string]any{
			"jobs":                    jobs,
			"backgroundJobsEnabled":   cfg.BackgroundJobsEnabled,
			"schedulerTickEnabled":    cfg.SchedulerEnabled && cfg.BackgroundJobsEnabled,
		})
	}
}

// handleAdminSchedulerHistory returns the recent trigger history for one job
// (plan 17.4 §10, AC-4).
func (d Deps) handleAdminSchedulerHistory() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			jobsMethodNotAllowed(w, http.MethodGet)
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		if d.Scheduler == nil || d.Pool == nil {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeInternal, "Scheduler not configured.")
			return
		}
		name := chi.URLParam(r, "name")
		rows, err := scheduler.ListHistory(r.Context(), d.Pool, name, 10)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"history": rows})
	}
}

// handleAdminSchedulerSetEnabled enables or disables a scheduled job without a
// deploy (plan 17.4 FR-6, §9 enable/disable). enabled is fixed per route.
func (d Deps) handleAdminSchedulerSetEnabled(enabled bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			jobsMethodNotAllowed(w, http.MethodPost)
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		if d.Scheduler == nil || d.Pool == nil {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeInternal, "Scheduler not configured.")
			return
		}
		name := chi.URLParam(r, "name")
		if !d.schedulerJobExists(name) {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Scheduled job not found.")
			return
		}
		if err := scheduler.SetEnabled(r.Context(), d.Pool, name, enabled, time.Now().UTC()); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"name": name, "enabled": enabled})
	}
}

// handleAdminSchedulerTrigger manually fires a job now for testing (plan 17.4
// §9 POST .../trigger). It enqueues onto the durable queue like a normal trigger.
func (d Deps) handleAdminSchedulerTrigger() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			jobsMethodNotAllowed(w, http.MethodPost)
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		if d.Scheduler == nil || d.Pool == nil {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeInternal, "Scheduler not configured.")
			return
		}
		name := chi.URLParam(r, "name")
		if !d.effectiveConfig().BackgroundJobsEnabled {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal,
				"Background job worker is disabled. Set BACKGROUND_JOBS_ENABLED=1 and restart the API, or use APP_ENV=local for the default dev worker.")
			return
		}
		jobID, err := d.Scheduler.Trigger(r.Context(), name, time.Now().UTC())
		if errors.Is(err, scheduler.ErrUnknownJob) {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Scheduled job not found.")
			return
		}
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"jobId": jobID.String()})
	}
}

func (d Deps) schedulerJobExists(name string) bool {
	for _, j := range d.Scheduler.Jobs() {
		if j.Name == name {
			return true
		}
	}
	return false
}

func (d Deps) registerAdminSchedulerRoutes(r chi.Router) {
	r.Get("/api/v1/admin/scheduler", d.handleAdminSchedulerList())
	r.Get("/api/v1/admin/scheduler/{name}/history", d.handleAdminSchedulerHistory())
	r.Post("/api/v1/admin/scheduler/{name}/enable", d.handleAdminSchedulerSetEnabled(true))
	r.Post("/api/v1/admin/scheduler/{name}/disable", d.handleAdminSchedulerSetEnabled(false))
	r.Post("/api/v1/admin/scheduler/{name}/trigger", d.handleAdminSchedulerTrigger())
}
