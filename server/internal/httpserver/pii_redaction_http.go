package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	logredactionsvc "github.com/lextures/lextures/server/internal/service/logredaction"
)

func (d Deps) requireRedactionOpsRead(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	uid, ok := d.meUserID(w, r)
	if !ok {
		return uuid.UUID{}, false
	}
	can, err := logredactionsvc.CheckRead(r.Context(), d.Pool, uid)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Permission check failed.")
		return uuid.UUID{}, false
	}
	if !can {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
		return uuid.UUID{}, false
	}
	return uid, true
}

func (d Deps) registerPIIRedactionRoutes(r chi.Router) {
	r.Get("/api/v1/internal/ops/redaction-status", d.handleGetRedactionStatus())
}

// GET /api/v1/internal/ops/redaction-status (plan 10.14 §9)
func (d Deps) handleGetRedactionStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.requireRedactionOpsRead(w, r); !ok {
			return
		}
		cfg := d.effectiveConfig()
		status := logredactionsvc.BuildStatus(cfg.DisablePIIRedaction, cfg.AppEnv, cfg.PIIRedactFields)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(status)
	}
}
