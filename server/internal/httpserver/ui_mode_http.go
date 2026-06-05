package httpserver

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	acmodel "github.com/lextures/lextures/server/internal/models/accommodations"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/readingprefs"
)

// handlePatchAdminUserUIMode is PATCH /api/v1/admin/users/{userId}/ui-mode.
// Requires global:user:accommodations:manage (same gate as the accommodations engine).
// Body: { "uiMode": "k2" | "elementary" | "standard" | null }
// Pass null or omit uiMode to clear the override and restore grade-level derivation.
func (d Deps) handlePatchAdminUserUIMode() http.HandlerFunc {
	type body struct {
		UIMode *string `json:"uiMode"`
	}
	type resp struct {
		StudentID      string  `json:"studentId"`
		UIModeOverride *string `json:"uiModeOverride"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			w.Header().Set("Allow", http.MethodPatch)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		// Auth before feature flag: unauthenticated requests always get 401.
		requesterID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if !d.uiModeEnabled() {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "UI mode is not enabled.")
			return
		}
		ctx := r.Context()
		hasPerm, err := rbac.UserHasPermission(ctx, d.Pool, requesterID, acmodel.PermManage)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		if !hasPerm {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return
		}
		rawID := chi.URLParam(r, "userId")
		studentID, err := uuid.Parse(rawID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid userId.")
			return
		}
		payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not read body.")
			return
		}
		var b body
		if err := json.Unmarshal(payload, &b); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		// nil or empty string clears the override.
		var mode *string
		if b.UIMode != nil && *b.UIMode != "" {
			if !readingprefs.ValidUIMode(*b.UIMode) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "uiMode must be one of: k2, elementary, standard.")
				return
			}
			mode = b.UIMode
		}
		if err := readingprefs.SetUIModeOverride(ctx, d.Pool, studentID, mode); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not save UI mode override.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp{StudentID: studentID.String(), UIModeOverride: mode})
	}
}

func (d Deps) registerUIModeRoutes(r chi.Router) {
	r.Patch("/api/v1/admin/users/{userId}/ui-mode", d.handlePatchAdminUserUIMode())
}
