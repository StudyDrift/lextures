package httpserver

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/service/kidmessaging"
)

// learnerSettingsOut is the simplified Settings payload for a managed learner (#775).
// It never includes email, password, privacy, or AI provider fields.
type learnerSettingsOut struct {
	DisplayName string  `json:"displayName"`
	AvatarURL   *string `json:"avatarUrl"`
	UITheme     string  `json:"uiTheme"`
}

func learnerSettingsFromRow(row *user.Row) learnerSettingsOut {
	out := learnerSettingsOut{AvatarURL: row.AvatarURL, UITheme: row.UITheme}
	if row.DisplayName != nil {
		out.DisplayName = *row.DisplayName
	}
	if out.UITheme == "" {
		out.UITheme = "light"
	}
	return out
}

func (d Deps) registerLearnerSettingsRoutes(r chi.Router) {
	r.Get("/api/v1/me/learner-settings", d.handleGetLearnerSettings())
	r.Patch("/api/v1/me/learner-settings", d.handlePatchLearnerSettings())
}

func (d Deps) requireManagedSelf(w http.ResponseWriter, r *http.Request) (uuid.UUID, *user.Row, bool) {
	userID, ok := d.meUserID(w, r)
	if !ok {
		return uuid.Nil, nil, false
	}
	row, err := user.FindByID(r.Context(), d.Pool, userID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load settings.")
		return uuid.Nil, nil, false
	}
	if !kidmessaging.IsManaged(row) {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Learner settings are only for managed learners.")
		return uuid.Nil, nil, false
	}
	return userID, row, true
}

// handleGetLearnerSettings is GET /api/v1/me/learner-settings.
func (d Deps) handleGetLearnerSettings() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, row, ok := d.requireManagedSelf(w, r)
		if !ok {
			return
		}
		writeJSON(w, http.StatusOK, learnerSettingsFromRow(row))
	}
}

// normalizeLearnerDisplayName trims and validates a learner-chosen display name.
func normalizeLearnerDisplayName(s *string) (*string, string) {
	if s == nil {
		return nil, ""
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil, "Display name is required."
	}
	if utf8.RuneCountInString(t) > 80 {
		return nil, "Display name is too long."
	}
	return &t, ""
}

// handlePatchLearnerSettings is PATCH /api/v1/me/learner-settings with
// { displayName?, avatarUrl? ("" clears), uiTheme? }.
func (d Deps) handlePatchLearnerSettings() http.HandlerFunc {
	type in struct {
		DisplayName *string `json:"displayName"`
		AvatarURL   *string `json:"avatarUrl"`
		UITheme     *string `json:"uiTheme"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _, ok := d.requireManagedSelf(w, r)
		if !ok {
			return
		}
		raw, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid body.")
			return
		}
		var body in
		if err := json.Unmarshal(raw, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		displayName, msg := normalizeLearnerDisplayName(body.DisplayName)
		if msg != "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, msg)
			return
		}
		clearAvatar := body.AvatarURL != nil && strings.TrimSpace(*body.AvatarURL) == ""
		avatarURL, err := normalizeAvatarURL(body.AvatarURL)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		uiTheme, err := normalizeTheme(body.UITheme)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		updated, err := user.UpdateManagedLearnerProfile(r.Context(), d.Pool, userID, displayName, avatarURL, uiTheme, clearAvatar)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update settings.")
			return
		}
		if updated == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "User not found.")
			return
		}
		writeJSON(w, http.StatusOK, learnerSettingsFromRow(updated))
	}
}
