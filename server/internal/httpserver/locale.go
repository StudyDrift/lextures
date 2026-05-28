package httpserver

import (
	"net/http"
	"strings"
)

func detectBrowserLocale(acceptLanguage string) string {
	if strings.TrimSpace(acceptLanguage) == "" {
		return "en"
	}
	for _, part := range strings.Split(acceptLanguage, ",") {
		tag := strings.TrimSpace(strings.Split(part, ";")[0])
		if tag == "" || tag == "*" {
			continue
		}
		if _, err := normalizeLocaleInput(tag); err == nil {
			return tag
		}
	}
	return "en"
}

func (d Deps) handleGetPublicLocaleDefaults() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		locale := detectBrowserLocale(r.Header.Get("Accept-Language"))
		writeJSON(w, http.StatusOK, map[string]string{
			"locale": locale,
		})
	}
}
