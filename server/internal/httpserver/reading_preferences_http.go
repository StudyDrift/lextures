package httpserver

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/readingprefs"
)

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
		HighContrast  *bool   `json:"highContrast"`
		ReduceMotion  *bool   `json:"reduceMotion"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
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
		p := readingprefs.Patch{
			FontFace:      b.FontFace,
			LetterSpacing: b.LetterSpacing,
			WordSpacing:   b.WordSpacing,
			LineHeight:    b.LineHeight,
			RulerEnabled:  b.RulerEnabled,
			RulerColor:    b.RulerColor,
			HighContrast:  b.HighContrast,
			ReduceMotion:  b.ReduceMotion,
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
