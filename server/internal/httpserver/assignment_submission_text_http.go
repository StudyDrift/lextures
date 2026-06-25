package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/moduleassignmentsubmissions"
	webhooksvc "github.com/lextures/lextures/server/internal/service/webhooks"
)

type postAssignmentSubmissionTextBody struct {
	Text string `json:"text"`
}

// handlePostAssignmentSubmissionText is POST .../assignments/{item_id}/submissions/text
func (d Deps) handlePostAssignmentSubmissionText() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid assignment id.")
			return
		}
		cid, assignRow, ok := d.loadAssignmentForSubmissions(w, r, courseCode, itemID)
		if !ok || assignRow == nil || cid == nil {
			return
		}
		if !d.enforceConditionalReleaseForLearner(w, r, courseCode, *cid, viewer, itemID) {
			return
		}
		if !assignRow.SubmissionAllowText {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "This assignment does not accept text submissions.")
			return
		}
		var body postAssignmentSubmissionTextBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		text := strings.TrimSpace(body.Text)
		if text == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Submission text is required.")
			return
		}
		if len(text) > 512<<10 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Submission text is too long.")
			return
		}
		subRow, err := moduleassignmentsubmissions.UpsertBodyText(r.Context(), d.Pool, *cid, itemID, viewer, text)
		if err != nil || subRow == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save submission.")
			return
		}
		d.maybeEnqueueAutoGrade(r, courseCode, *cid, itemID, subRow.ID)
		webhooksvc.EmitAssignmentSubmittedEvent(r.Context(), d.Pool, d.effectiveConfig(), *cid, courseCode, itemID, subRow.ID, viewer)
		out := d.submissionToJSON(r.Context(), courseCode, *subRow, false, 0, "")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"submission": out})
	}
}