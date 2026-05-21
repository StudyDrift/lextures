package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/trustcenter"
)

type trustSubscribeBody struct {
	Email string `json:"email"`
}

// POST /api/v1/trust/sub-processor-updates/subscribe
func (d Deps) handleTrustSubProcessorSubscribe() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body trustSubscribeBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		email := strings.TrimSpace(body.Email)
		if email == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "email is required.")
			return
		}
		if d.Pool == nil {
			// No DB in tests without pool — return 503.
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Service unavailable.")
			return
		}
		if err := trustcenter.Subscribe(r.Context(), d.Pool, email); err != nil {
			if strings.Contains(err.Error(), "invalid email") {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid email address.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to subscribe.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) registerTrustRoutes(r chi.Router) {
	r.Post("/api/v1/trust/sub-processor-updates/subscribe", d.handleTrustSubProcessorSubscribe())
}
