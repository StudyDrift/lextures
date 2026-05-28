package httpserver

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/user"
)

var localeBCP47Pattern = regexp.MustCompile(`^[a-z]{2}(-[A-Z]{2})?$`)

var supportedLocales = map[string]struct{}{
	"en": {}, "es": {}, "fr": {},
	"en-US": {}, "en-GB": {}, "es-ES": {}, "es-MX": {}, "fr-FR": {}, "fr-CA": {},
}

type localeResponse struct {
	Locale string `json:"locale"`
}

type patchLocaleBody struct {
	Locale string `json:"locale"`
}

func normalizeLocaleInput(raw string) (string, error) {
	t := strings.TrimSpace(raw)
	if t == "" {
		return "", apierrValidationError{msg: "Locale is required."}
	}
	if !localeBCP47Pattern.MatchString(t) {
		return "", apierrValidationError{msg: "Locale must be a valid BCP 47 tag (e.g. en, es, fr, en-US)."}
	}
	if _, ok := supportedLocales[t]; !ok {
		return "", apierrValidationError{msg: "Locale is not supported. Supported: en, es, fr (and common regional variants)."}
	}
	return t, nil
}

func (d Deps) handleGetSettingsLocale() http.HandlerFunc {
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
		row, err := user.FindByID(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load locale.")
			return
		}
		if row == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "User not found.")
			return
		}
		loc := row.Locale
		if loc == "" {
			loc = "en"
		}
		writeJSON(w, http.StatusOK, localeResponse{Locale: loc})
	}
}

func (d Deps) handlePutSettingsLocale() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.Header().Set("Allow", http.MethodPut)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		var req patchLocaleBody
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		locale, err := normalizeLocaleInput(req.Locale)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		row, err := user.UpdateLocale(r.Context(), d.Pool, userID, locale)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update locale.")
			return
		}
		if row == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "User not found.")
			return
		}
		writeJSON(w, http.StatusOK, localeResponse{Locale: row.Locale})
	}
}
