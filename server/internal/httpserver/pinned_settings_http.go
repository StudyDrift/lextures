package httpserver

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/pinnedsettings"
	"github.com/lextures/lextures/server/internal/telemetry"
)

const pinnedSettingsWriteLimitPerMinute = 60

type pinnedSettingsRateEntry struct {
	count  int
	window time.Time
}

var (
	pinnedSettingsRateMu     sync.Mutex
	pinnedSettingsRateByUser = map[uuid.UUID]pinnedSettingsRateEntry{}
)

type pinnedSettingsResponse struct {
	Surfaces pinnedsettings.All `json:"surfaces"`
}

type putPinnedSettingsBody struct {
	SettingKeys []string `json:"settingKeys"`
}

func (d Deps) requirePinnedSettings(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFPinnedSettings {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Pinned settings are not enabled.")
		return false
	}
	return true
}

func (d Deps) checkPinnedSettingsWriteRate(userID uuid.UUID) bool {
	pinnedSettingsRateMu.Lock()
	defer pinnedSettingsRateMu.Unlock()
	now := time.Now()
	e, ok := pinnedSettingsRateByUser[userID]
	if !ok || now.Sub(e.window) >= time.Minute {
		pinnedSettingsRateByUser[userID] = pinnedSettingsRateEntry{count: 1, window: now}
		return true
	}
	if e.count >= pinnedSettingsWriteLimitPerMinute {
		return false
	}
	e.count++
	pinnedSettingsRateByUser[userID] = e
	return true
}

func encodePinnedSettings(w http.ResponseWriter, all pinnedsettings.All) {
	// Ensure non-null arrays in JSON.
	if all.Assignment == nil {
		all.Assignment = []string{}
	}
	if all.Quiz == nil {
		all.Quiz = []string{}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(pinnedSettingsResponse{Surfaces: all})
}

func (d Deps) handleGetMyPinnedSettings() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if !d.requirePinnedSettings(w) {
			return
		}
		all, err := pinnedsettings.GetAll(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load pinned settings.")
			return
		}
		encodePinnedSettings(w, all)
	}
}

func (d Deps) handlePutMyPinnedSettings() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if !d.requirePinnedSettings(w) {
			return
		}
		surface := chi.URLParam(r, "surface")
		if !pinnedsettings.ValidSurface(surface) {
			slog.Info("pinned_settings.reject",
				"user_id", userID.String(),
				"surface", surface,
				"reason", string(pinnedsettings.ReasonBadSurface),
				"key_count", 0,
			)
			telemetry.RecordPinnedSettingsReject(string(pinnedsettings.ReasonBadSurface))
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeValidation, "surface must be assignment or quiz.")
			return
		}
		if !d.checkPinnedSettingsWriteRate(userID) {
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Pinned settings write rate limit exceeded.")
			return
		}
		payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeValidation, "Could not read body.")
			return
		}
		var body putPinnedSettingsBody
		if err := json.Unmarshal(payload, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeValidation, "Invalid JSON body.")
			return
		}
		// Null settingKeys is treated as empty clear (not a nested object / wrong type).
		if body.SettingKeys == nil {
			// Distinguish explicit null vs missing: both clear is fine; wrong types fail Unmarshal into []string.
			body.SettingKeys = []string{}
		}
		keys, reason, verr := pinnedsettings.ValidateKeys(body.SettingKeys)
		if verr != nil {
			slog.Info("pinned_settings.reject",
				"user_id", userID.String(),
				"surface", surface,
				"reason", string(reason),
				"key_count", len(body.SettingKeys),
			)
			telemetry.RecordPinnedSettingsReject(string(reason))
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeValidation, verr.Error())
			return
		}
		all, err := pinnedsettings.Upsert(r.Context(), d.Pool, userID, surface, keys)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not save pinned settings.")
			return
		}
		telemetry.RecordPinnedSettingsWrite(surface)
		telemetry.ObservePinnedSettingsPinCount(len(keys))
		encodePinnedSettings(w, all)
	}
}
