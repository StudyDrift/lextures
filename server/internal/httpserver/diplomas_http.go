package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/background"
	diplomasrepo "github.com/lextures/lextures/server/internal/repos/diplomas"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/service/diplomaissue"
)

func (d Deps) registerDiplomasRoutes(r chi.Router) {
	r.Get("/api/v1/admin/credentials/templates", d.handleAdminListDiplomaTemplates())
	r.Post("/api/v1/admin/credentials/templates", d.handleAdminCreateDiplomaTemplate())
	r.Put("/api/v1/admin/credentials/templates/{id}", d.handleAdminUpdateDiplomaTemplate())
	r.Post("/api/v1/admin/credentials/issue", d.handleAdminIssueDiploma())
	r.Post("/api/v1/admin/credentials/issue/batch", d.handleAdminIssueDiplomaBatch())
	r.Get("/api/v1/admin/credentials/batches/{id}", d.handleAdminGetDiplomaBatch())
	r.Post("/api/v1/admin/credentials/{id}/revoke", d.handleAdminRevokeDiploma())
	r.Post("/api/v1/admin/credentials/{id}/unrevoke", d.handleAdminUnrevokeDiploma())
	// /api/v1/me/credentials is reserved for completion credentials (15.5); T11 uses /me/diplomas.
	r.Get("/api/v1/me/diplomas", d.handleMeListDiplomas())
	r.Get("/api/v1/me/diplomas/{id}/download", d.handleMeDownloadDiploma())
}

func (d Deps) diplomasFeatureOff(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFDiplomas {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Diplomas are not enabled.")
		return true
	}
	return false
}

type diplomaTemplateJSON struct {
	ID            string          `json:"id"`
	OrgID         string          `json:"orgId"`
	Kind          string          `json:"kind"`
	Name          string          `json:"name"`
	Title         string          `json:"title"`
	Program       *string         `json:"program,omitempty"`
	ConferralText *string         `json:"conferralText,omitempty"`
	Layout        json.RawMessage `json:"layout"`
	Active        bool            `json:"active"`
	CreatedAt     string          `json:"createdAt"`
	UpdatedAt     string          `json:"updatedAt"`
}

func diplomaTemplateToJSON(t *diplomasrepo.Template) diplomaTemplateJSON {
	return diplomaTemplateJSON{
		ID:            t.ID.String(),
		OrgID:         t.OrgID.String(),
		Kind:          string(t.Kind),
		Name:          t.Name,
		Title:         t.Title,
		Program:       t.Program,
		ConferralText: t.ConferralText,
		Layout:        t.Layout,
		Active:        t.Active,
		CreatedAt:     t.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:     t.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

type diplomaJSON struct {
	ID              string          `json:"id"`
	UserID          string          `json:"userId"`
	OrgID           string          `json:"orgId"`
	TemplateID      *string         `json:"templateId,omitempty"`
	Kind            string          `json:"kind"`
	CredentialTitle string          `json:"credentialTitle"`
	Program         *string         `json:"program,omitempty"`
	Honors          *string         `json:"honors,omitempty"`
	ConferredAt     string          `json:"conferredAt"`
	Version         int             `json:"version"`
	ReplacesID      *string         `json:"replacesId,omitempty"`
	ContentHash     string          `json:"contentHash"`
	VerifyToken     *string         `json:"verifyToken,omitempty"`
	RevokedAt       *string         `json:"revokedAt,omitempty"`
	RevokeReason    *string         `json:"revokeReason,omitempty"`
	IssuedAt        string          `json:"issuedAt"`
	ProgramRef      *string         `json:"programRef,omitempty"`
	HasPDF          bool            `json:"hasPdf"`
	HasVC           bool            `json:"hasVc"`
	Canonical       json.RawMessage `json:"canonical,omitempty"`
}

func diplomaToJSON(d *diplomasrepo.Diploma) diplomaJSON {
	out := diplomaJSON{
		ID:              d.ID.String(),
		UserID:          d.UserID.String(),
		OrgID:           d.OrgID.String(),
		Kind:            string(d.Kind),
		CredentialTitle: d.CredentialTitle,
		Program:         d.Program,
		Honors:          d.Honors,
		ConferredAt:     d.ConferredAt.UTC().Format(time.RFC3339),
		Version:         d.Version,
		ContentHash:     d.ContentHash,
		VerifyToken:     d.VerifyToken,
		RevokeReason:    d.RevokeReason,
		IssuedAt:        d.IssuedAt.UTC().Format(time.RFC3339),
		HasPDF:          len(d.PDFBytes) > 0 || (d.PDFKey != nil && *d.PDFKey != ""),
		HasVC:           len(d.VCProof) > 0,
		Canonical:       d.Canonical,
	}
	if d.TemplateID != nil {
		s := d.TemplateID.String()
		out.TemplateID = &s
	}
	if d.ReplacesID != nil {
		s := d.ReplacesID.String()
		out.ReplacesID = &s
	}
	if d.RevokedAt != nil {
		s := d.RevokedAt.UTC().Format(time.RFC3339)
		out.RevokedAt = &s
	}
	if d.ProgramRef != nil {
		s := d.ProgramRef.String()
		out.ProgramRef = &s
	}
	return out
}

type diplomaBatchJSON struct {
	ID           string  `json:"id"`
	OrgID        string  `json:"orgId"`
	TemplateID   string  `json:"templateId"`
	ProgramRef   *string `json:"programRef,omitempty"`
	Program      *string `json:"program,omitempty"`
	Honors       *string `json:"honors,omitempty"`
	ConferredAt  string  `json:"conferredAt"`
	Status       string  `json:"status"`
	TotalCount   int     `json:"totalCount"`
	SuccessCount int     `json:"successCount"`
	FailCount    int     `json:"failCount"`
	SkipCount    int     `json:"skipCount"`
	ErrorSummary *string `json:"errorSummary,omitempty"`
	CreatedAt    string  `json:"createdAt"`
	StartedAt    *string `json:"startedAt,omitempty"`
	FinishedAt   *string `json:"finishedAt,omitempty"`
}

func batchToJSON(b *diplomasrepo.Batch) diplomaBatchJSON {
	out := diplomaBatchJSON{
		ID:           b.ID.String(),
		OrgID:        b.OrgID.String(),
		TemplateID:   b.TemplateID.String(),
		Program:      b.Program,
		Honors:       b.Honors,
		ConferredAt:  b.ConferredAt.UTC().Format(time.RFC3339),
		Status:       b.Status,
		TotalCount:   b.TotalCount,
		SuccessCount: b.SuccessCount,
		FailCount:    b.FailCount,
		SkipCount:    b.SkipCount,
		ErrorSummary: b.ErrorSummary,
		CreatedAt:    b.CreatedAt.UTC().Format(time.RFC3339),
	}
	if b.ProgramRef != nil {
		s := b.ProgramRef.String()
		out.ProgramRef = &s
	}
	if b.StartedAt != nil {
		s := b.StartedAt.UTC().Format(time.RFC3339)
		out.StartedAt = &s
	}
	if b.FinishedAt != nil {
		s := b.FinishedAt.UTC().Format(time.RFC3339)
		out.FinishedAt = &s
	}
	return out
}

func (d Deps) handleAdminListDiplomaTemplates() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.diplomasFeatureOff(w) {
			return
		}
		userID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID)
		if err != nil || orgID == uuid.Nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Organization required.")
			return
		}
		activeOnly := strings.EqualFold(r.URL.Query().Get("active"), "true")
		list, err := diplomasrepo.ListTemplates(r.Context(), d.Pool, orgID, activeOnly)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list templates.")
			return
		}
		out := make([]diplomaTemplateJSON, 0, len(list))
		for i := range list {
			out = append(out, diplomaTemplateToJSON(&list[i]))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"templates": out})
	}
}

type createTemplateBody struct {
	Kind          string          `json:"kind"`
	Name          string          `json:"name"`
	Title         string          `json:"title"`
	Program       *string         `json:"program"`
	ConferralText *string         `json:"conferralText"`
	Layout        json.RawMessage `json:"layout"`
}

func (d Deps) handleAdminCreateDiplomaTemplate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.diplomasFeatureOff(w) {
			return
		}
		userID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID)
		if err != nil || orgID == uuid.Nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Organization required.")
			return
		}
		var body createTemplateBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		tmpl, err := diplomasrepo.CreateTemplate(r.Context(), d.Pool, diplomasrepo.CreateTemplateInput{
			OrgID:         orgID,
			Kind:          diplomasrepo.Kind(body.Kind),
			Name:          body.Name,
			Title:         body.Title,
			Program:       body.Program,
			ConferralText: body.ConferralText,
			Layout:        body.Layout,
			CreatedBy:     &userID,
		})
		if err != nil {
			if errors.Is(err, diplomasrepo.ErrInvalidKind) || errors.Is(err, diplomasrepo.ErrInvalidInput) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create template.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"template": diplomaTemplateToJSON(tmpl)})
	}
}

type updateTemplateBody struct {
	Name          *string         `json:"name"`
	Title         *string         `json:"title"`
	Program       *string         `json:"program"`
	ConferralText *string         `json:"conferralText"`
	Layout        json.RawMessage `json:"layout"`
	Active        *bool           `json:"active"`
	ClearProgram  bool            `json:"clearProgram"`
	ClearText     bool            `json:"clearConferralText"`
}

func (d Deps) handleAdminUpdateDiplomaTemplate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.diplomasFeatureOff(w) {
			return
		}
		userID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid template id.")
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID)
		if err != nil || orgID == uuid.Nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Organization required.")
			return
		}
		cur, err := diplomasrepo.GetTemplateByID(r.Context(), d.Pool, id)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load template.")
			return
		}
		if cur == nil || cur.OrgID != orgID {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Template not found.")
			return
		}
		var body updateTemplateBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		tmpl, err := diplomasrepo.UpdateTemplate(r.Context(), d.Pool, id, diplomasrepo.UpdateTemplateInput{
			Name:          body.Name,
			Title:         body.Title,
			Program:       body.Program,
			ConferralText: body.ConferralText,
			Layout:        body.Layout,
			Active:        body.Active,
			ClearProgram:  body.ClearProgram,
			ClearText:     body.ClearText,
		})
		if err != nil {
			if errors.Is(err, diplomasrepo.ErrNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Template not found.")
				return
			}
			if errors.Is(err, diplomasrepo.ErrInvalidInput) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update template.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"template": diplomaTemplateToJSON(tmpl)})
	}
}

type issueBody struct {
	UserID       string  `json:"userId"`
	TemplateID   string  `json:"templateId"`
	LearnerName  string  `json:"learnerName"`
	Program      *string `json:"program"`
	Honors       *string `json:"honors"`
	ConferredAt  string  `json:"conferredAt"`
	ProgramRef   *string `json:"programRef"`
	CorrectPrior bool    `json:"correctPrior"`
}

func (d Deps) handleAdminIssueDiploma() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.diplomasFeatureOff(w) {
			return
		}
		adminID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, adminID)
		if err != nil || orgID == uuid.Nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Organization required.")
			return
		}
		var body issueBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		userID, err := uuid.Parse(strings.TrimSpace(body.UserID))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid userId.")
			return
		}
		templateID, err := uuid.Parse(strings.TrimSpace(body.TemplateID))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid templateId.")
			return
		}
		conferredAt := time.Now().UTC()
		if s := strings.TrimSpace(body.ConferredAt); s != "" {
			t, err := time.Parse(time.RFC3339, s)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid conferredAt (RFC3339).")
				return
			}
			conferredAt = t.UTC()
		}
		var programRef *uuid.UUID
		if body.ProgramRef != nil && strings.TrimSpace(*body.ProgramRef) != "" {
			id, err := uuid.Parse(strings.TrimSpace(*body.ProgramRef))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid programRef.")
				return
			}
			programRef = &id
		}
		res, err := diplomaissue.Issue(r.Context(), d.Pool, d.effectiveConfig(), diplomaissue.IssueParams{
			OrgID:        orgID,
			TemplateID:   templateID,
			UserID:       userID,
			LearnerName:  body.LearnerName,
			Program:      body.Program,
			Honors:       body.Honors,
			ConferredAt:  conferredAt,
			ProgramRef:   programRef,
			IssuedBy:     &adminID,
			CorrectPrior: body.CorrectPrior,
		})
		if err != nil {
			writeDiplomaIssueErr(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"diploma": diplomaToJSON(res.Diploma),
			"skipped": res.Skipped,
			"reason":  res.Reason,
		})
	}
}

type batchIssueBody struct {
	TemplateID  string   `json:"templateId"`
	UserIDs     []string `json:"userIds"`
	Program     *string  `json:"program"`
	Honors      *string  `json:"honors"`
	ConferredAt string   `json:"conferredAt"`
	ProgramRef  *string  `json:"programRef"`
}

func (d Deps) handleAdminIssueDiplomaBatch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.diplomasFeatureOff(w) {
			return
		}
		adminID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, adminID)
		if err != nil || orgID == uuid.Nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Organization required.")
			return
		}
		var body batchIssueBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		templateID, err := uuid.Parse(strings.TrimSpace(body.TemplateID))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid templateId.")
			return
		}
		tmpl, err := diplomasrepo.GetTemplateByID(r.Context(), d.Pool, templateID)
		if err != nil || tmpl == nil || tmpl.OrgID != orgID {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Template not found.")
			return
		}
		conferredAt := time.Now().UTC()
		if s := strings.TrimSpace(body.ConferredAt); s != "" {
			t, err := time.Parse(time.RFC3339, s)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid conferredAt (RFC3339).")
				return
			}
			conferredAt = t.UTC()
		}
		var programRef *uuid.UUID
		if body.ProgramRef != nil && strings.TrimSpace(*body.ProgramRef) != "" {
			id, err := uuid.Parse(strings.TrimSpace(*body.ProgramRef))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid programRef.")
				return
			}
			programRef = &id
		}
		userIDs := make([]uuid.UUID, 0, len(body.UserIDs))
		for _, s := range body.UserIDs {
			id, err := uuid.Parse(strings.TrimSpace(s))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid userIds entry.")
				return
			}
			userIDs = append(userIDs, id)
		}
		if len(userIDs) == 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "userIds required.")
			return
		}
		batch, err := diplomasrepo.CreateBatch(r.Context(), d.Pool, orgID, templateID, programRef, body.Program, body.Honors, conferredAt, &adminID, userIDs)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create batch.")
			return
		}
		if _, err := background.EnqueueDiplomaBatch(r.Context(), d.Pool, batch.ID); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to enqueue batch.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{"batch": batchToJSON(batch)})
	}
}

func (d Deps) handleAdminGetDiplomaBatch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.diplomasFeatureOff(w) {
			return
		}
		userID, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid batch id.")
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID)
		if err != nil || orgID == uuid.Nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Organization required.")
			return
		}
		batch, err := diplomasrepo.GetBatch(r.Context(), d.Pool, id)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load batch.")
			return
		}
		if batch == nil || batch.OrgID != orgID {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Batch not found.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"batch": batchToJSON(batch)})
	}
}

func (d Deps) handleAdminRevokeDiploma() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.diplomasFeatureOff(w) {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid credential id.")
			return
		}
		var body revokeBody
		_ = json.NewDecoder(r.Body).Decode(&body)
		dip, err := diplomaissue.Revoke(r.Context(), d.Pool, d.effectiveConfig(), id, body.Reason)
		if err != nil {
			writeDiplomaIssueErr(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"diploma": diplomaToJSON(dip)})
	}
}

func (d Deps) handleAdminUnrevokeDiploma() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.diplomasFeatureOff(w) {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid credential id.")
			return
		}
		dip, err := diplomaissue.Unrevoke(r.Context(), d.Pool, d.effectiveConfig(), id)
		if err != nil {
			writeDiplomaIssueErr(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"diploma": diplomaToJSON(dip)})
	}
}

func (d Deps) handleMeListDiplomas() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.diplomasFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		list, err := diplomasrepo.ListByUser(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list credentials.")
			return
		}
		out := make([]diplomaJSON, 0, len(list))
		for i := range list {
			out = append(out, diplomaToJSON(&list[i]))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"credentials": out})
	}
}

func (d Deps) handleMeDownloadDiploma() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.diplomasFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}
		id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid credential id.")
			return
		}
		dip, err := diplomasrepo.GetByID(r.Context(), d.Pool, id)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load credential.")
			return
		}
		if dip == nil || dip.UserID != userID {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Credential not found.")
			return
		}
		if len(dip.PDFBytes) == 0 {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "PDF not available.")
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", `attachment; filename="credential.pdf"`)
		_, _ = w.Write(dip.PDFBytes)
	}
}

func writeDiplomaIssueErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, diplomaissue.ErrFeatureDisabled):
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Diplomas are not enabled.")
	case errors.Is(err, diplomaissue.ErrNotFound), errors.Is(err, diplomasrepo.ErrNotFound):
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
	case errors.Is(err, diplomaissue.ErrInvalidInput), errors.Is(err, diplomasrepo.ErrInvalidInput), errors.Is(err, diplomasrepo.ErrInvalidKind):
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
	case errors.Is(err, diplomaissue.ErrTemplateInactive):
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Template is inactive.")
	default:
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Request failed.")
	}
}
