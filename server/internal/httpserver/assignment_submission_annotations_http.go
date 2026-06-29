package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/originalityreports"
	"github.com/lextures/lextures/server/internal/repos/submissionannotations"
)

var validAnnotationTools = map[string]bool{
	"highlight": true,
	"draw":      true,
	"text":      true,
	"pin":       true,
}

func annotationToJSON(a submissionannotations.AnnotationRow) map[string]any {
	coords := json.RawMessage(a.CoordsJSON)
	if len(coords) == 0 {
		coords = json.RawMessage(`{}`)
	}
	return map[string]any{
		"id":           a.ID.String(),
		"submissionId": a.SubmissionID.String(),
		"annotatorId":  a.AnnotatorID.String(),
		"clientId":     a.ClientID,
		"page":         a.Page,
		"toolType":     a.ToolType,
		"colour":       a.Colour,
		"coordsJson":   coords,
		"body":         a.Body,
		"createdAt":    a.CreatedAt.UTC().Format(time.RFC3339),
		"updatedAt":    a.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

// requireAnnotationWorkflow returns false (writing a 404) when the annotation feature is disabled.
func (d Deps) requireAnnotationWorkflow(w http.ResponseWriter) bool {
	if !d.effectiveConfig().AnnotationEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
		return false
	}
	return true
}

// loadAnnotationSubmission resolves the submission and viewer for an annotation request and
// reports whether the viewer may grade (write) the submission. Students who own the submission
// may read but not write; everyone else needs gradebook access.
func (d Deps) loadAnnotationSubmission(w http.ResponseWriter, r *http.Request) (sc *originalityreports.SubmissionContext, courseCode string, viewer uuid.UUID, canGrade bool, ok bool) {
	if !d.requireAnnotationWorkflow(w) {
		return nil, "", uuid.Nil, false, false
	}
	viewer, ok = d.meUserID(w, r)
	if !ok {
		return nil, "", uuid.Nil, false, false
	}
	courseCode, ok = chiCourseCode(w, r)
	if !ok {
		return nil, "", uuid.Nil, false, false
	}
	submissionID, ok := d.parseSubmissionID(w, r)
	if !ok {
		return nil, "", uuid.Nil, false, false
	}
	if d.Pool == nil {
		apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
		return nil, "", uuid.Nil, false, false
	}
	sc, err := originalityreports.GetSubmissionContext(r.Context(), d.Pool, courseCode, submissionID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load submission.")
		return nil, "", uuid.Nil, false, false
	}
	if sc == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Submission not found.")
		return nil, "", uuid.Nil, false, false
	}
	canGrade, err = courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":gradebook:view")
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return nil, "", uuid.Nil, false, false
	}
	return sc, courseCode, viewer, canGrade, true
}

// handleListSubmissionAnnotations is GET .../submissions/{submission_id}/annotations
func (d Deps) handleListSubmissionAnnotations() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sc, _, viewer, canGrade, ok := d.loadAnnotationSubmission(w, r)
		if !ok {
			return
		}
		// Staff may read all; the owning student may read feedback on their own submission.
		if !canGrade && sc.SubmittedBy != viewer {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Forbidden.")
			return
		}
		rows, err := submissionannotations.ListBySubmission(r.Context(), d.Pool, sc.SubmissionID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load annotations.")
			return
		}
		items := make([]map[string]any, 0, len(rows))
		for _, a := range rows {
			items = append(items, annotationToJSON(a))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"annotations": items})
	}
}

type postAnnotationBody struct {
	ClientID   string          `json:"clientId"`
	Page       int32           `json:"page"`
	ToolType   string          `json:"toolType"`
	Colour     string          `json:"colour"`
	CoordsJSON json.RawMessage `json:"coordsJson"`
	Body       *string         `json:"body"`
}

// handlePostSubmissionAnnotation is POST .../submissions/{submission_id}/annotations
func (d Deps) handlePostSubmissionAnnotation() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sc, _, viewer, canGrade, ok := d.loadAnnotationSubmission(w, r)
		if !ok {
			return
		}
		if !canGrade {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Only graders can annotate submissions.")
			return
		}
		var body postAnnotationBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		clientID := strings.TrimSpace(body.ClientID)
		if clientID == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "clientId is required.")
			return
		}
		if !validAnnotationTools[body.ToolType] {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Unsupported annotation tool.")
			return
		}
		if len(body.CoordsJSON) == 0 || !json.Valid(body.CoordsJSON) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "coordsJson is required.")
			return
		}
		page := body.Page
		if page < 1 {
			page = 1
		}
		colour := strings.TrimSpace(body.Colour)
		if colour == "" {
			colour = "#FFFF00"
		}
		var note *string
		if body.Body != nil {
			if trimmed := strings.TrimSpace(*body.Body); trimmed != "" {
				if len(trimmed) > 10<<10 {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Comment is too long.")
					return
				}
				note = &trimmed
			}
		}
		row, err := submissionannotations.Upsert(r.Context(), d.Pool, submissionannotations.AnnotationUpsertWrite{
			SubmissionID: sc.SubmissionID,
			AnnotatorID:  viewer,
			ClientID:     clientID,
			Page:         page,
			ToolType:     body.ToolType,
			Colour:       colour,
			CoordsJSON:   body.CoordsJSON,
			Body:         note,
		})
		if err != nil || row == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save annotation.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"annotation": annotationToJSON(*row)})
	}
}

// handleDeleteSubmissionAnnotation is DELETE .../submissions/{submission_id}/annotations/{annotation_id}
func (d Deps) handleDeleteSubmissionAnnotation() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sc, _, _, canGrade, ok := d.loadAnnotationSubmission(w, r)
		if !ok {
			return
		}
		if !canGrade {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Only graders can delete annotations.")
			return
		}
		annotationID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "annotation_id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid annotation id.")
			return
		}
		deleted, err := submissionannotations.SoftDelete(r.Context(), d.Pool, sc.SubmissionID, annotationID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to delete annotation.")
			return
		}
		if !deleted {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Annotation not found.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
