package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/courseevaluations"
	"github.com/lextures/lextures/server/internal/repos/organization"
)

func templateToJSON(t *courseevaluations.Template) map[string]any {
	out := map[string]any{
		"id":        t.ID.String(),
		"orgId":     t.OrgID.String(),
		"name":      t.Name,
		"questions": t.Questions,
		"createdAt": t.CreatedAt.UTC().Format(time.RFC3339Nano),
		"updatedAt": t.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if t.CreatedBy != nil {
		out["createdBy"] = t.CreatedBy.String()
	}
	return out
}

// handleAdminListEvaluationTemplates returns all templates for the caller's org.
func (d Deps) handleAdminListEvaluationTemplates() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.evaluationsFeatureOff(w, r) {
			return
		}
		userID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}

		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to determine organization.")
			return
		}

		templates, err := courseevaluations.ListTemplates(r.Context(), d.Pool, orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load evaluation templates.")
			return
		}

		out := make([]map[string]any, 0, len(templates))
		for i := range templates {
			out = append(out, templateToJSON(&templates[i]))
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"templates": out})
	}
}

// handleAdminCreateEvaluationTemplate creates a new evaluation template.
func (d Deps) handleAdminCreateEvaluationTemplate() http.HandlerFunc {
	type body struct {
		Name      string          `json:"name"`
		Questions json.RawMessage `json:"questions"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if d.evaluationsFeatureOff(w, r) {
			return
		}
		userID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}

		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to determine organization.")
			return
		}

		raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Failed to read request.")
			return
		}
		var b body
		if err := json.Unmarshal(raw, &b); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid request body.")
			return
		}
		if b.Name == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Template name is required.")
			return
		}

		tmpl, err := courseevaluations.CreateTemplate(r.Context(), d.Pool, courseevaluations.CreateTemplateInput{
			OrgID:     orgID,
			Name:      b.Name,
			Questions: b.Questions,
			CreatedBy: &userID,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create evaluation template.")
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(templateToJSON(tmpl))
	}
}

// handleAdminEvaluationTemplateItem handles GET, PATCH, DELETE for a single template.
func (d Deps) handleAdminEvaluationTemplateItem() http.HandlerFunc {
	type patchBody struct {
		Name      string          `json:"name"`
		Questions json.RawMessage `json:"questions"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if d.evaluationsFeatureOff(w, r) {
			return
		}
		_, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}

		tmplIDStr := chi.URLParam(r, "template_id")
		tmplID, err := uuid.Parse(tmplIDStr)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid template ID.")
			return
		}

		switch r.Method {
		case http.MethodGet:
			tmpl, err := courseevaluations.GetTemplate(r.Context(), d.Pool, tmplID)
			if errors.Is(err, courseevaluations.ErrTemplateNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Template not found.")
				return
			}
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load template.")
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(templateToJSON(tmpl))

		case http.MethodPatch:
			raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Failed to read request.")
				return
			}
			var b patchBody
			if err := json.Unmarshal(raw, &b); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid request body.")
				return
			}
			tmpl, err := courseevaluations.UpdateTemplate(r.Context(), d.Pool, tmplID, courseevaluations.UpdateTemplateInput{
				Name:      b.Name,
				Questions: b.Questions,
			})
			if errors.Is(err, courseevaluations.ErrTemplateNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Template not found.")
				return
			}
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update template.")
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(templateToJSON(tmpl))

		case http.MethodDelete:
			if err := courseevaluations.DeleteTemplate(r.Context(), d.Pool, tmplID); err != nil {
				if errors.Is(err, courseevaluations.ErrTemplateNotFound) {
					apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Template not found.")
					return
				}
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to delete template.")
				return
			}
			w.WriteHeader(http.StatusNoContent)

		default:
			w.Header().Set("Allow", "GET, PATCH, DELETE")
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	}
}

// handleAdminEvaluationReport returns a cross-section evaluation completion report for the caller's org.
func (d Deps) handleAdminEvaluationReport() http.HandlerFunc {
	type rowResp struct {
		CourseID      string   `json:"courseId"`
		CourseCode    string   `json:"courseCode"`
		CourseTitle   string   `json:"courseTitle"`
		WindowID      string   `json:"windowId"`
		OpensAt       string   `json:"opensAt"`
		ClosesAt      string   `json:"closesAt"`
		EnrolledCount int      `json:"enrolledCount"`
		ResponseCount int      `json:"responseCount"`
		CompletionPct float64  `json:"completionPct"`
		AverageRating *float64 `json:"averageRating,omitempty"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if d.evaluationsFeatureOff(w, r) {
			return
		}
		userID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}

		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to determine organization.")
			return
		}

		closedOnly := r.URL.Query().Get("closed_only") == "true"
		rows, err := courseevaluations.ListAdminReport(r.Context(), d.Pool, orgID, closedOnly)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load evaluation report.")
			return
		}

		out := make([]rowResp, 0, len(rows))
		for _, row := range rows {
			out = append(out, rowResp{
				CourseID:      row.CourseID.String(),
				CourseCode:    row.CourseCode,
				CourseTitle:   row.CourseTitle,
				WindowID:      row.WindowID.String(),
				OpensAt:       row.OpensAt.UTC().Format(time.RFC3339Nano),
				ClosesAt:      row.ClosesAt.UTC().Format(time.RFC3339Nano),
				EnrolledCount: row.EnrolledCount,
				ResponseCount: row.ResponseCount,
				CompletionPct: row.CompletionPct,
			})
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"rows": out})
	}
}
