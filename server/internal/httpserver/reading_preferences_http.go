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
	"github.com/lextures/lextures/server/internal/repos/readingprefs"
	acsvc "github.com/lextures/lextures/server/internal/service/accommodations"
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

type readingPreferencesResponse struct {
	readingprefs.Row
	AccommodationOverrides readingprefs.AccommodationOverrides `json:"accommodationOverrides,omitempty"`
}

func (d Deps) speechToTextEnabled() bool {
	return d.effectiveConfig().SpeechToTextEnabled
}

func (d Deps) readingPreferencesEnabled() bool {
	cfg := d.effectiveConfig()
	return cfg.SpeechToTextEnabled ||
		cfg.AccommodationsEngineEnabled ||
		(cfg.ReadAloudEnabled && cfg.FFReadAloud) ||
		cfg.FFReadingPreferences
}

func (d Deps) requireReadingPreferences(w http.ResponseWriter) bool {
	if !d.readingPreferencesEnabled() {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Reading preferences are not enabled.")
		return false
	}
	return true
}

func (d Deps) requireSpeechToText(w http.ResponseWriter) bool {
	if !d.speechToTextEnabled() {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Speech-to-text is not enabled.")
		return false
	}
	return true
}

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

func (d Deps) registerTTSRoutes(r chi.Router) {
	r.Post("/api/v1/tts/synthesize", d.handlePostTTSSynthesize())
}

func (d Deps) encodeReadingPreferences(w http.ResponseWriter, row *readingprefs.Row, overrides readingprefs.AccommodationOverrides) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if d.accommodationsEngineFeatureEnabled() {
		_ = json.NewEncoder(w).Encode(readingPreferencesResponse{
			Row:                    *row,
			AccommodationOverrides: overrides,
		})
		return
	}
	_ = json.NewEncoder(w).Encode(row)
}

func (d Deps) handleGetMyReadingPreferences() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if !d.requireReadingPreferences(w) {
			return
		}
		row, err := readingprefs.Get(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load reading preferences.")
			return
		}
		overrides := readingprefs.AccommodationOverrides{}
		if d.accommodationsEngineFeatureEnabled() {
			eff := acsvc.ResolveEffectiveGlobal(r.Context(), d.Pool, userID)
			row, overrides = readingprefs.MergeAccommodationOverrides(row, eff)
		}
		d.encodeReadingPreferences(w, row, overrides)
	}
}

func (d Deps) handlePatchMyReadingPreferences() http.HandlerFunc {
	type body struct {
		FontFace               *string  `json:"fontFace"`
		LetterSpacing          *string  `json:"letterSpacing"`
		WordSpacing            *string  `json:"wordSpacing"`
		LineHeight             *string  `json:"lineHeight"`
		RulerEnabled           *bool    `json:"rulerEnabled"`
		RulerColor             *string  `json:"rulerColor"`
		TTSEnabled             *bool    `json:"ttsEnabled"`
		TTSSpeed               *float64 `json:"ttsSpeed"`
		TTSVoiceName           *string  `json:"ttsVoiceName"`
		STTEnabled             *bool    `json:"sttEnabled"`
		STTLanguage            *string  `json:"sttLanguage"`
		DyslexiaDisplayEnabled *bool    `json:"dyslexiaDisplayEnabled"`
		HighContrastEnabled    *bool    `json:"highContrastEnabled"`
		ReducedMotionEnabled   *bool    `json:"reducedMotionEnabled"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if !d.requireReadingPreferences(w) {
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
		if b.STTEnabled != nil || b.STTLanguage != nil {
			if !d.speechToTextEnabled() {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Speech-to-text is not enabled.")
				return
			}
			if b.STTLanguage != nil {
				lang := strings.TrimSpace(*b.STTLanguage)
				if lang != "" && len(lang) > 20 {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "sttLanguage is too long.")
					return
				}
				if lang != "" {
					b.STTLanguage = &lang
				}
			}
		}
		if b.TTSSpeed != nil {
			rounded := math.Round(*b.TTSSpeed*100) / 100
			b.TTSSpeed = &rounded
		}
		var voicePtr **string
		if b.TTSVoiceName != nil {
			v := b.TTSVoiceName
			voicePtr = &v
		}
		p := readingprefs.Patch{
			FontFace:               b.FontFace,
			LetterSpacing:          b.LetterSpacing,
			WordSpacing:            b.WordSpacing,
			LineHeight:             b.LineHeight,
			RulerEnabled:           b.RulerEnabled,
			RulerColor:             b.RulerColor,
			TTSEnabled:             b.TTSEnabled,
			TTSSpeed:               b.TTSSpeed,
			TTSVoiceName:           voicePtr,
			STTEnabled:             b.STTEnabled,
			STTLanguage:            b.STTLanguage,
			DyslexiaDisplayEnabled: b.DyslexiaDisplayEnabled,
			HighContrastEnabled:    b.HighContrastEnabled,
			ReducedMotionEnabled:   b.ReducedMotionEnabled,
		}
		if err := p.Validate(); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		row, err := readingprefs.Upsert(r.Context(), d.Pool, userID, p)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not save reading preferences.")
			return
		}
		overrides := readingprefs.AccommodationOverrides{}
		if d.accommodationsEngineFeatureEnabled() {
			eff := acsvc.ResolveEffectiveGlobal(r.Context(), d.Pool, userID)
			row, overrides = readingprefs.MergeAccommodationOverrides(row, eff)
		}
		d.encodeReadingPreferences(w, row, overrides)
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
