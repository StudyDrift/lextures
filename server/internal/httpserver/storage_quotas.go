package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/service/storagequota"
)

// CodeQuotaExceeded is the JSON error code returned when a storage quota is breached.
const CodeQuotaExceeded = "QUOTA_EXCEEDED"

func (d Deps) registerStorageQuotaRoutes(r chi.Router) {
	r.Get("/api/v1/courses/{course_code}/storage-usage", d.handleGetCourseStorageUsage())
}

// handleGetCourseStorageUsage handles GET /api/v1/courses/{course_code}/storage-usage.
// Returns { used_bytes, limit_bytes, percent_used } for the course.
func (d Deps) handleGetCourseStorageUsage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.StorageQuota == nil {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented,
				"Storage quotas are not enabled on this server.")
			return
		}
		if _, ok := d.meUserID(w, r); !ok {
			return
		}
		courseCode := chi.URLParam(r, "course_code")
		ctx := r.Context()

		courseID, err := course.GetIDByCourseCode(ctx, d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if courseID == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}

		info, err := d.StorageQuota.GetCourseUsage(ctx, *courseID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load usage.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(info)
	}
}

// handleAdminStorageQuotasList handles GET /api/v1/admin/storage-quotas.
func (d Deps) handleAdminStorageQuotasList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.StorageQuota == nil {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented,
				"Storage quotas are not enabled on this server.")
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		entries, err := d.StorageQuota.ListQuotas(r.Context())
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list quotas.")
			return
		}
		if entries == nil {
			entries = []storagequota.QuotaEntry{}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(entries)
	}
}

// handleAdminStorageQuotasPut handles PUT /api/v1/admin/storage-quotas/{scope}/{scope_id}.
// Body: { "limit_bytes": 1073741824 } or { "limit_bytes": null } to remove the limit.
func (d Deps) handleAdminStorageQuotasPut() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.StorageQuota == nil {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented,
				"Storage quotas are not enabled on this server.")
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}

		scopeName := chi.URLParam(r, "scope")
		if scopeName != "tenant" && scopeName != "course" && scopeName != "user" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput,
				"scope must be 'tenant', 'course', or 'user'.")
			return
		}
		scopeID, err := uuid.Parse(chi.URLParam(r, "scope_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid scope_id UUID.")
			return
		}

		var body struct {
			LimitBytes *int64 `json:"limit_bytes"`
		}
		if err = json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if body.LimitBytes != nil && *body.LimitBytes < 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "limit_bytes must be >= 0.")
			return
		}

		if err = d.StorageQuota.SetQuota(r.Context(), scopeName, scopeID, body.LimitBytes); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to set quota.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleAdminStorageQuotasReconcile handles POST /api/v1/admin/storage-quotas/reconcile.
func (d Deps) handleAdminStorageQuotasReconcile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.StorageQuota == nil {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented,
				"Storage quotas are not enabled on this server.")
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		if err := d.StorageQuota.Reconcile(r.Context()); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Reconcile failed.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
