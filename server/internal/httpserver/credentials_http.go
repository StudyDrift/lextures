package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	repocred "github.com/lextures/lextures/server/internal/repos/credentials"
	credsvc "github.com/lextures/lextures/server/internal/service/credentials"
	svcPaths "github.com/lextures/lextures/server/internal/service/learningpaths"
	"github.com/lextures/lextures/server/internal/service/notifications"
)

func (d Deps) credentialsFeatureOff(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFCompletionCredentials {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Completion credentials are not enabled.")
		return true
	}
	return false
}

func (d Deps) registerCredentialRoutes(r chi.Router) {
	r.Get("/api/v1/me/credentials", d.handleListMyCredentials())
	r.Get("/api/v1/credentials/{id}/download", d.handleDownloadCredential())
	r.Get("/api/v1/credentials/{id}/json", d.handleCredentialJSON())
	r.Get("/api/v1/credentials/{id}/verify", d.handleVerifyCredential())
}

func (d Deps) credentialDeps() credsvc.IssueDeps {
	return credsvc.IssueDeps{
		Pool:    d.Pool,
		Cfg:     d.effectiveConfig(),
		Storage: d.Storage,
		Notify:  &notifications.Service{Pool: d.Pool, Config: d.effectiveConfig()},
	}
}

func (d Deps) handleListMyCredentials() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.credentialsFeatureOff(w) {
			return
		}
		items, err := repocred.ListForRecipient(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load credentials.")
			return
		}
		cfg := d.effectiveConfig()
		out := make([]map[string]any, 0, len(items))
		for _, item := range items {
			out = append(out, credentialListJSON(item, cfg.PublicWebOrigin))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"credentials": out})
	}
}

func (d Deps) handleDownloadCredential() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.credentialsFeatureOff(w) {
			return
		}
		id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid credential id.")
			return
		}
		row, err := repocred.GetByID(r.Context(), d.Pool, id)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load credential.")
			return
		}
		if row == nil || row.RecipientID != userID {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Credential not found.")
			return
		}
		if row.Revoked {
			apierr.WriteJSON(w, http.StatusGone, apierr.CodeNotFound, "Credential has been revoked.")
			return
		}

		var pdfBytes []byte
		if row.PDFKey != nil {
			pdfBytes, err = credsvc.ReadStoredPDF(r.Context(), d.Storage, *row.PDFKey)
		}
		if err != nil || len(pdfBytes) == 0 {
			learnerName, _ := d.learnerDisplayName(r, userID)
			pdfBytes, err = credsvc.BuildPDFBytes(d.effectiveConfig(), row, learnerName)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to render PDF.")
				return
			}
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="certificate-%s.pdf"`, id.String()))
		_, _ = w.Write(pdfBytes)
	}
}

func (d Deps) handleCredentialJSON() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.credentialsFeatureOff(w) {
			return
		}
		id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid credential id.")
			return
		}
		row, err := repocred.GetByID(r.Context(), d.Pool, id)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load credential.")
			return
		}
		if row == nil || row.RecipientID != userID {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Credential not found.")
			return
		}
		w.Header().Set("Content-Type", "application/ld+json; charset=utf-8")
		_, _ = w.Write(row.CredentialJSON)
	}
}

func (d Deps) handleVerifyCredential() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.credentialsFeatureOff(w) {
			return
		}
		id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid credential id.")
			return
		}
		result, err := credsvc.Verify(r.Context(), d.Pool, d.effectiveConfig(), id)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Verification failed.")
			return
		}
		if result == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Credential not found.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"valid":       result.Valid,
			"status":      result.Status,
			"revoked":     result.Revoked,
			"issuerName":  result.IssuerName,
			"learnerName": result.LearnerName,
			"achievement": result.Achievement,
			"issuedAt":    result.IssuedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
			"credential":  result.Credential,
		})
	}
}

func (d Deps) pathProgressOpts(r *http.Request, userID uuid.UUID) *svcPaths.ProgressOptions {
	if !d.effectiveConfig().FFCompletionCredentials {
		return nil
	}
	name, err := d.learnerDisplayName(r, userID)
	if err != nil {
		name = ""
	}
	deps := d.credentialDeps()
	return &svcPaths.ProgressOptions{CredDeps: &deps, LearnerName: name}
}

func (d Deps) tryIssueCourseCredential(r *http.Request, courseID, recipientID uuid.UUID) *string {
	if !d.effectiveConfig().FFCompletionCredentials {
		return nil
	}
	learnerName, err := d.learnerDisplayName(r, recipientID)
	if err != nil || strings.TrimSpace(learnerName) == "" {
		return nil
	}
	created, _, err := credsvc.IssueForCourseCompletion(r.Context(), d.credentialDeps(), courseID, recipientID, learnerName)
	if err != nil || created == nil {
		return nil
	}
	id := created.ID.String()
	return &id
}

func credentialListJSON(item repocred.ListItem, origin string) map[string]any {
	verifyURL := fmt.Sprintf("%s/verify/%s", strings.TrimRight(strings.TrimSpace(origin), "/"), item.ID.String())
	return map[string]any{
		"id":              item.ID.String(),
		"sourceType":      string(item.SourceType),
		"sourceId":        item.SourceID.String(),
		"title":           item.Title,
		"issuedAt":        item.IssuedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		"revoked":         item.Revoked,
		"hasPdf":          item.HasPDF,
		"verificationUrl": verifyURL,
	}
}