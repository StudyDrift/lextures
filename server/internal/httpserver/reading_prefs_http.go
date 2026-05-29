package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/readingprefs"
)

func (d Deps) highContrastReducedMotionEnabled() bool {
	return d.effectiveConfig().FFHighContrastReducedMotion
}

func (d Deps) requireHighContrastReducedMotion(w http.ResponseWriter) bool {
	if !d.highContrastReducedMotionEnabled() {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "High-contrast and reduced-motion preferences are not enabled.")
		return false
	}
	return true
}

func (d Deps) handleGetMyReadingPreferences() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireHighContrastReducedMotion(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		row, err := readingprefs.Get(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load reading preferences.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"highContrast": row.HighContrast,
			"reduceMotion": row.ReduceMotion,
		})
	}
}

func (d Deps) handlePatchMyReadingPreferences() http.HandlerFunc {
	type body struct {
		HighContrast *bool `json:"highContrast"`
		ReduceMotion *bool `json:"reduceMotion"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireHighContrastReducedMotion(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		var req body
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		current, err := readingprefs.Get(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load reading preferences.")
			return
		}
		hc := current.HighContrast
		rm := current.ReduceMotion
		if req.HighContrast != nil {
			hc = *req.HighContrast
		}
		if req.ReduceMotion != nil {
			rm = *req.ReduceMotion
		}
		row, err := readingprefs.Upsert(r.Context(), d.Pool, userID, hc, rm)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not save reading preferences.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"highContrast": row.HighContrast,
			"reduceMotion": row.ReduceMotion,
		})
	}
}
