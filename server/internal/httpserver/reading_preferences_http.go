package httpserver

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/readingprefs"
	acsvc "github.com/lextures/lextures/server/internal/service/accommodations"
)

type readingPreferencesJSON struct {
	STTEnabled             bool `json:"sttEnabled"`
	STTLanguage            string `json:"sttLanguage"`
	TTSEnabled             bool `json:"ttsEnabled"`
	DyslexiaDisplayEnabled bool `json:"dyslexiaDisplayEnabled"`
	HighContrastEnabled    bool `json:"highContrastEnabled"`
	ReducedMotionEnabled   bool `json:"reducedMotionEnabled"`
}

type readingPreferencesResponse struct {
	readingPreferencesJSON
	AccommodationOverrides readingprefs.AccommodationOverrides `json:"accommodationOverrides"`
}

func (d Deps) speechToTextEnabled() bool {
	return d.effectiveConfig().SpeechToTextEnabled
}

func (d Deps) readingPreferencesEnabled() bool {
	cfg := d.effectiveConfig()
	return cfg.SpeechToTextEnabled || cfg.AccommodationsEngineEnabled
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

func rowToReadingPreferencesJSON(r readingprefs.Row) readingPreferencesJSON {
	return readingPreferencesJSON{
		STTEnabled:             r.STTEnabled,
		STTLanguage:            r.STTLanguage,
		TTSEnabled:             r.TTSEnabled,
		DyslexiaDisplayEnabled: r.DyslexiaDisplayEnabled,
		HighContrastEnabled:    r.HighContrastEnabled,
		ReducedMotionEnabled:   r.ReducedMotionEnabled,
	}
}

func (d Deps) handleGetMyReadingPreferences() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.requireReadingPreferences(w) {
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
		overrides := readingprefs.AccommodationOverrides{}
		if d.accommodationsEngineFeatureEnabled() {
			eff := acsvc.ResolveEffectiveGlobal(r.Context(), d.Pool, userID)
			row, overrides = readingprefs.MergeAccommodationOverrides(row, eff)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(readingPreferencesResponse{
			readingPreferencesJSON: rowToReadingPreferencesJSON(row),
			AccommodationOverrides: overrides,
		})
	}
}

func (d Deps) handlePatchMyReadingPreferences() http.HandlerFunc {
	type body struct {
		STTEnabled             *bool   `json:"sttEnabled"`
		STTLanguage            *string `json:"sttLanguage"`
		TTSEnabled             *bool   `json:"ttsEnabled"`
		DyslexiaDisplayEnabled *bool   `json:"dyslexiaDisplayEnabled"`
		HighContrastEnabled    *bool   `json:"highContrastEnabled"`
		ReducedMotionEnabled   *bool   `json:"reducedMotionEnabled"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			w.Header().Set("Allow", http.MethodPatch)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.requireReadingPreferences(w) {
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
		var b body
		if err := json.Unmarshal(payload, &b); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
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
		row, err := readingprefs.Patch(r.Context(), d.Pool, userID,
			b.STTEnabled, b.STTLanguage, b.TTSEnabled, b.DyslexiaDisplayEnabled, b.HighContrastEnabled, b.ReducedMotionEnabled)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not save reading preferences.")
			return
		}
		overrides := readingprefs.AccommodationOverrides{}
		if d.accommodationsEngineFeatureEnabled() {
			eff := acsvc.ResolveEffectiveGlobal(r.Context(), d.Pool, userID)
			row, overrides = readingprefs.MergeAccommodationOverrides(row, eff)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(readingPreferencesResponse{
			readingPreferencesJSON: rowToReadingPreferencesJSON(row),
			AccommodationOverrides: overrides,
		})
	}
}
