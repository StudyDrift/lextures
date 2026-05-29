package httpserver

import (
	"encoding/json"
	"io"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/readingpreferences"
	ttssvc "github.com/lextures/lextures/server/internal/service/tts"
)

const ttsRateLimitPerMinute = 60

type ttsRateEntry struct {
	count  int
	window time.Time
}

var (
	ttsRateMu     sync.Mutex
	ttsRateByUser = map[uuid.UUID]ttsRateEntry{}
)

func (d Deps) readAloudEnabled() bool {
	cfg := d.effectiveConfig()
	return cfg.ReadAloudEnabled && cfg.FFReadAloud
}

func (d Deps) requireReadAloud(w http.ResponseWriter) bool {
	if !d.readAloudEnabled() {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Read aloud is not enabled.")
		return false
	}
	return true
}

func (d Deps) registerReadingPreferencesRoutes(r chi.Router) {
	r.Get("/api/v1/me/reading-preferences", d.handleGetMyReadingPreferences())
	r.Patch("/api/v1/me/reading-preferences", d.handlePatchMyReadingPreferences())
}

func (d Deps) registerTTSRoutes(r chi.Router) {
	r.Post("/api/v1/tts/synthesize", d.handlePostTTSSynthesize())
}

type readingPreferencesJSON struct {
	TTSEnabled   bool     `json:"ttsEnabled"`
	TTSSpeed     float64  `json:"ttsSpeed"`
	TTSVoiceName *string  `json:"ttsVoiceName,omitempty"`
	UpdatedAt    *string  `json:"updatedAt,omitempty"`
}

func rowToReadingPrefsJSON(r readingpreferences.Row) readingPreferencesJSON {
	out := readingPreferencesJSON{
		TTSEnabled:   r.TTSEnabled,
		TTSSpeed:     r.TTSSpeed,
		TTSVoiceName: r.TTSVoiceName,
	}
	if !r.UpdatedAt.IsZero() {
		s := r.UpdatedAt.UTC().Format(time.RFC3339)
		out.UpdatedAt = &s
	}
	return out
}

func (d Deps) handleGetMyReadingPreferences() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		row, err := readingpreferences.Get(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load reading preferences.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(rowToReadingPrefsJSON(row))
	}
}

func (d Deps) handlePatchMyReadingPreferences() http.HandlerFunc {
	type body struct {
		TTSEnabled   *bool    `json:"ttsEnabled"`
		TTSSpeed     *float64 `json:"ttsSpeed"`
		TTSVoiceName *string  `json:"ttsVoiceName"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			w.Header().Set("Allow", http.MethodPatch)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not read body.")
			return
		}
		var b body
		if err := json.Unmarshal(payload, &b); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if b.TTSSpeed != nil {
			s := *b.TTSSpeed
			if s < 0.75 || s > 2.0 {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "ttsSpeed must be between 0.75 and 2.0.")
				return
			}
			rounded := math.Round(s*100) / 100
			b.TTSSpeed = &rounded
		}
		var voicePtr **string
		if b.TTSVoiceName != nil {
			v := b.TTSVoiceName
			voicePtr = &v
		}
		row, err := readingpreferences.Patch(r.Context(), d.Pool, userID, b.TTSEnabled, b.TTSSpeed, voicePtr)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not save reading preferences.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(rowToReadingPrefsJSON(row))
	}
}

func (d Deps) checkTTSRateLimit(userID uuid.UUID) bool {
	ttsRateMu.Lock()
	defer ttsRateMu.Unlock()
	now := time.Now()
	e, ok := ttsRateByUser[userID]
	if !ok || now.Sub(e.window) >= time.Minute {
		ttsRateByUser[userID] = ttsRateEntry{count: 1, window: now}
		return true
	}
	if e.count >= ttsRateLimitPerMinute {
		return false
	}
	e.count++
	ttsRateByUser[userID] = e
	return true
}

type ttsSynthesizeRequest struct {
	Text  string  `json:"text"`
	Lang  string  `json:"lang"`
	Speed float64 `json:"speed"`
}

func (d Deps) handlePostTTSSynthesize() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.requireReadAloud(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if !d.checkTTSRateLimit(userID) {
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "TTS rate limit exceeded.")
			return
		}
		payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not read body.")
			return
		}
		var req ttsSynthesizeRequest
		if err := json.Unmarshal(payload, &req); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		text := strings.TrimSpace(req.Text)
		if text == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "text is required.")
			return
		}
		if len(text) > 5000 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "text exceeds 5000 characters.")
			return
		}
		speed := req.Speed
		if speed <= 0 {
			speed = 1
		}
		if speed < 0.75 || speed > 2.0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "speed must be between 0.75 and 2.0.")
			return
		}
		_ = req.Lang // reserved for phase 2 voice locale
		wav := ttssvc.StubWAV(text, speed)
		w.Header().Set("Content-Type", "audio/wav")
		w.Header().Set("Cache-Control", "private, no-store")
		_, _ = w.Write(wav)
	}
}
