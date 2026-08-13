package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	mcrepo "github.com/lextures/lextures/server/internal/repos/marketingcontent"
	mcservice "github.com/lextures/lextures/server/internal/service/marketingcontent"
)

func writeMarketingError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		apierr.WriteJSON(w, 404, apierr.CodeNotFound, "Marketing content resource not found.")
	case errors.Is(err, mcrepo.ErrDuplicateSlug):
		apierr.WriteJSON(w, 409, "duplicate_slug", "An article or redirect already uses that path.")
	case errors.Is(err, mcservice.ErrInvalidTransition), errors.Is(err, mcservice.ErrScheduledInPast), errors.Is(err, mcservice.ErrReviewNoteTooShort), errors.Is(err, mcservice.ErrReviewerRequired), errors.Is(err, mcservice.ErrOverrideJustification):
		apierr.WriteJSON(w, 422, apierr.CodeUnprocessableEntity, err.Error())
	case errors.Is(err, mcservice.ErrLintBlocked):
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(422)
		payload := map[string]any{
			"error": map[string]string{
				"code":    "content_validation_failed",
				"message": "Publishing is blocked by content validation errors.",
			},
		}
		var blocked *mcservice.LintBlockedError
		if errors.As(err, &blocked) {
			payload["score"] = blocked.Report.Score
			payload["findings"] = blocked.Report.Findings
		}
		_ = json.NewEncoder(w).Encode(payload)
	default:
		apierr.WriteInternal(w, r, "Marketing content operation failed.", err)
	}
}

func (d Deps) writeMarketingConflict(w http.ResponseWriter, r *http.Request, id uuid.UUID, err error) {
	if !errors.Is(err, mcrepo.ErrRevisionConflict) {
		writeMarketingError(w, r, err)
		return
	}
	a, getErr := mcrepo.GetArticleByID(r.Context(), d.Pool, id)
	if getErr != nil {
		writeMarketingError(w, r, getErr)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(409)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": "CONFLICT", "message": "A newer revision has already been saved."}, "currentRevisionNo": a.RevisionNo, "updatedBy": a.UpdatedBy, "updatedAt": a.UpdatedAt})
}
