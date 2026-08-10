package httpserver

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/navprefs"
)

const navPrefsWriteLimitPerMinute = 60

type navPrefsRateEntry struct {
	count  int
	window time.Time
}

var (
	navPrefsRateMu     sync.Mutex
	navPrefsRateByUser = map[uuid.UUID]navPrefsRateEntry{}
)

type navPreferencesBody struct {
	Scope     string   `json:"scope"`
	Pinned    []string `json:"pinned"`
	Hidden    []string `json:"hidden"`
	Collapsed []string `json:"collapsed"`
}

func (d Deps) checkNavPrefsWriteRate(userID uuid.UUID) bool {
	navPrefsRateMu.Lock()
	defer navPrefsRateMu.Unlock()
	now := time.Now()
	e, ok := navPrefsRateByUser[userID]
	if !ok || now.Sub(e.window) >= time.Minute {
		navPrefsRateByUser[userID] = navPrefsRateEntry{count: 1, window: now}
		return true
	}
	if e.count >= navPrefsWriteLimitPerMinute {
		return false
	}
	e.count++
	navPrefsRateByUser[userID] = e
	return true
}

func encodeNavPreferences(w http.ResponseWriter, row navprefs.Row) {
	if row.Pinned == nil {
		row.Pinned = []string{}
	}
	if row.Hidden == nil {
		row.Hidden = []string{}
	}
	if row.Collapsed == nil {
		row.Collapsed = []string{}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(row)
}

func (d Deps) handleGetNavPreferences() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		scope := strings.TrimSpace(r.URL.Query().Get("scope"))
		if scope == "" {
			scope = "global"
		}
		if !navprefs.ValidScope(scope) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeValidation, "Invalid navigation preference scope.")
			return
		}
		row, err := navprefs.Get(r.Context(), d.Pool, userID, scope)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load navigation preferences.")
			return
		}
		encodeNavPreferences(w, row)
	}
}

func (d Deps) handlePutNavPreferences() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if !d.checkNavPrefsWriteRate(userID) {
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Navigation preferences write rate limit exceeded.")
			return
		}
		payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeValidation, "Could not read body.")
			return
		}
		var body navPreferencesBody
		if err := json.Unmarshal(payload, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeValidation, "Invalid JSON body.")
			return
		}
		scope := strings.TrimSpace(body.Scope)
		if scope == "" {
			scope = "global"
		}
		if !navprefs.ValidScope(scope) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeValidation, "Invalid navigation preference scope.")
			return
		}
		if body.Pinned == nil {
			body.Pinned = []string{}
		}
		if body.Hidden == nil {
			body.Hidden = []string{}
		}
		if body.Collapsed == nil {
			body.Collapsed = []string{}
		}
		row, err := navprefs.Upsert(r.Context(), d.Pool, userID, navprefs.Row{
			Scope:     scope,
			Pinned:    body.Pinned,
			Hidden:    body.Hidden,
			Collapsed: body.Collapsed,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not save navigation preferences.")
			return
		}
		encodeNavPreferences(w, row)
	}
}

func (d Deps) handleDeleteNavPreferences() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		scope := strings.TrimSpace(r.URL.Query().Get("scope"))
		if scope == "" {
			scope = "global"
		}
		if !navprefs.ValidScope(scope) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeValidation, "Invalid navigation preference scope.")
			return
		}
		if err := navprefs.Delete(r.Context(), d.Pool, userID, scope); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not reset navigation preferences.")
			return
		}
		encodeNavPreferences(w, navprefs.Default(scope))
	}
}

func (d Deps) registerNavPreferenceRoutes(r interface {
	Get(pattern string, h http.HandlerFunc)
	Put(pattern string, h http.HandlerFunc)
	Delete(pattern string, h http.HandlerFunc)
}) {
	r.Get("/api/v1/nav/preferences", d.handleGetNavPreferences())
	r.Put("/api/v1/nav/preferences", d.handlePutNavPreferences())
	r.Delete("/api/v1/nav/preferences", d.handleDeleteNavPreferences())
}
