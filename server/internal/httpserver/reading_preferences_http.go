package httpserver

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/readingprefs"
)

func (d Deps) speechToTextEnabled() bool {
	return d.effectiveConfig().SpeechToTextEnabled
}

func (d Deps) requireSpeechToText(w http.ResponseWriter) bool {
	if !d.speechToTextEnabled() {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Speech-to-text is not enabled.")
		return false
	}
	return true
}

func (d Deps) handleGetMyReadingPreferences() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		row, err := readingprefs.Get(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load reading preferences.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(row)
	}
}

func (d Deps) handlePatchMyReadingPreferences() http.HandlerFunc {
	type body struct {
		FontFace      *string `json:"fontFace"`
		LetterSpacing *string `json:"letterSpacing"`
		WordSpacing   *string `json:"wordSpacing"`
		LineHeight    *string `json:"lineHeight"`
		RulerEnabled  *bool   `json:"rulerEnabled"`
		RulerColor    *string `json:"rulerColor"`
		STTEnabled    *bool   `json:"sttEnabled"`
		STTLanguage   *string `json:"sttLanguage"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
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
		p := readingprefs.Patch{
			FontFace:      b.FontFace,
			LetterSpacing: b.LetterSpacing,
			WordSpacing:   b.WordSpacing,
			LineHeight:    b.LineHeight,
			RulerEnabled:  b.RulerEnabled,
			RulerColor:    b.RulerColor,
			STTEnabled:    b.STTEnabled,
			STTLanguage:   b.STTLanguage,
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
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(row)
	}
}
