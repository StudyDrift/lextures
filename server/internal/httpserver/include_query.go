package httpserver

import (
	"net/http"
	"strings"
)

func wantsInclude(r *http.Request, name string) bool {
	raw := strings.TrimSpace(r.URL.Query().Get("include"))
	if raw == "" {
		return false
	}
	for _, part := range strings.Split(raw, ",") {
		if strings.EqualFold(strings.TrimSpace(part), name) {
			return true
		}
	}
	return false
}
