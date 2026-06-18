package httpserver

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	userrepo "github.com/lextures/lextures/server/internal/repos/user"
	"github.com/lextures/lextures/server/internal/service/studyreminders"
)

func (d Deps) studyRemindersEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFStudyReminders {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Study reminders are not enabled.")
		return false
	}
	return true
}

func (d Deps) registerStudyReminderRoutes(r chi.Router) {
	r.Get("/api/v1/me/reminder-config", d.handleGetReminderConfig())
	r.Patch("/api/v1/me/reminder-config", d.handlePatchReminderConfig())
	r.Post("/api/v1/me/reminder-config/pause", d.handlePauseReminderConfig())
}

func (d Deps) studyReminderService() *studyreminders.Service {
	return &studyreminders.Service{Pool: d.Pool, Config: d.effectiveConfig()}
}

func (d Deps) handleGetReminderConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.studyRemindersEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		tz, _ := userrepo.GetTimezone(r.Context(), d.Pool, userID)
		cfg, err := d.studyReminderService().LoadAPIConfig(r.Context(), userID, time.Now().UTC(), tz)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load reminder settings.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(cfg)
	}
}

func (d Deps) handlePatchReminderConfig() http.HandlerFunc {
	type patchBody struct {
		DailyGoalMinutes *int      `json:"dailyGoalMinutes"`
		ReminderTime     *string   `json:"reminderTime"`
		ReminderChannels []string  `json:"reminderChannels"`
		WeeklySummary    *bool     `json:"weeklySummary"`
		Enabled          *bool     `json:"enabled"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			w.Header().Set("Allow", http.MethodPatch)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.studyRemindersEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not read body.")
			return
		}
		var body patchBody
		if err := json.Unmarshal(payload, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if body.DailyGoalMinutes != nil {
			if *body.DailyGoalMinutes < 5 || *body.DailyGoalMinutes > 480 {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Daily goal must be between 5 and 480 minutes.")
				return
			}
		}
		tz, _ := userrepo.GetTimezone(r.Context(), d.Pool, userID)
		current, err := d.studyReminderService().LoadAPIConfig(r.Context(), userID, time.Now().UTC(), tz)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load reminder settings.")
			return
		}
		if body.DailyGoalMinutes != nil {
			current.DailyGoalMinutes = *body.DailyGoalMinutes
		}
		if body.ReminderTime != nil {
			current.ReminderTime = *body.ReminderTime
		}
		if body.ReminderChannels != nil {
			current.ReminderChannels = body.ReminderChannels
		}
		if body.WeeklySummary != nil {
			current.WeeklySummary = *body.WeeklySummary
		}
		if body.Enabled != nil {
			current.Enabled = *body.Enabled
		}
		cfg, err := d.studyReminderService().SaveAPIConfig(r.Context(), userID, current)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not save reminder settings.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(cfg)
	}
}

func (d Deps) handlePauseReminderConfig() http.HandlerFunc {
	type pauseBody struct {
		Days int `json:"days"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.studyRemindersEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not read body.")
			return
		}
		var body pauseBody
		if err := json.Unmarshal(payload, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		tz, _ := userrepo.GetTimezone(r.Context(), d.Pool, userID)
		cfg, err := d.studyReminderService().Pause(r.Context(), userID, body.Days, time.Now().UTC(), tz)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not pause reminders.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(cfg)
	}
}
