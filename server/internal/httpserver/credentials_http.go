package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/logging"
	"github.com/lextures/lextures/server/internal/notificationevents"
	credrepo "github.com/lextures/lextures/server/internal/repos/credentials"
	"github.com/lextures/lextures/server/internal/repos/useraudit"
	credsvc "github.com/lextures/lextures/server/internal/service/credentials"
	"github.com/lextures/lextures/server/internal/service/notifications"
)

func (d Deps) credentialsFeatureOff(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFCompletionCredentials {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Completion credentials are not enabled.")
		return true
	}
	return false
}

func (d Deps) registerCredentialsRoutes(r chi.Router) {
	r.Get("/api/v1/me/credentials", d.handleListMyCredentials())
	r.Get("/api/v1/credentials/{id}/download", d.handleDownloadCredential())
	r.Get("/api/v1/credentials/{id}/json", d.handleCredentialJSON())
	r.Get("/api/v1/credentials/{id}/badge-export", d.handleBadgeExport())
	r.Get("/api/v1/credentials/{id}/badge-export/download", d.handleBadgeExportDownload())
	r.Get("/api/v1/credentials/{id}/linkedin-params", d.handleLinkedInParams())
	r.Post("/api/v1/credentials/{id}/share", d.handleCredentialShare())
	r.Get("/api/v1/credentials/{id}/verify", d.handleVerifyCredential())
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
		rows, err := credrepo.ListByRecipient(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load credentials.")
			return
		}
		cfg := d.effectiveConfig()
		out := make([]map[string]any, 0, len(rows))
		for i := range rows {
			out = append(out, credentialSummaryJSON(&rows[i], cfg.PublicWebOrigin))
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
		cred, ok := d.loadOwnedCredential(w, r, userID)
		if !ok {
			return
		}
		learnerName, err := d.learnerDisplayName(r, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load profile.")
			return
		}
		cfg := d.effectiveConfig()
		institution := strings.TrimSpace(cfg.CCRInstitutionName)
		if institution == "" {
			institution = "Lextures"
		}
		pdfBytes, err := credsvc.BuildPDF(credsvc.PDFInput{
			InstitutionName: institution,
			LearnerName:     learnerName,
			CredentialName:  cred.Title,
			IssuedAt:        cred.IssuedAt,
			VerificationURL: credsvc.VerificationURL(cfg.PublicWebOrigin, cred.ID),
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to render PDF.")
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-certificate.pdf"`, sanitizeFilename(cred.Title)))
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
		cred, ok := d.loadOwnedCredential(w, r, userID)
		if !ok {
			return
		}
		body, err := credsvc.FullCredentialJSON(cred)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load credential JSON.")
			return
		}
		w.Header().Set("Content-Type", "application/ld+json; charset=utf-8")
		_, _ = w.Write(body)
	}
}

func (d Deps) handleBadgeExport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.credentialsFeatureOff(w) {
			return
		}
		cred, ok := d.loadOwnedCredential(w, r, userID)
		if !ok {
			return
		}
		cfg := d.effectiveConfig()
		token, expires, err := credsvc.BadgeExportToken(cfg, cred.ID, time.Now().UTC())
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create download URL.")
			return
		}
		base := strings.TrimRight(strings.TrimSpace(cfg.PublicWebOrigin), "/")
		downloadURL := fmt.Sprintf("%s/api/v1/credentials/%s/badge-export/download?token=%s", base, cred.ID, token)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"downloadUrl": downloadURL,
			"expiresAt":   expires.UTC().Format(time.RFC3339),
		})
	}
}

func (d Deps) handleBadgeExportDownload() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.credentialsFeatureOff(w) {
			return
		}
		credID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid credential id.")
			return
		}
		token := strings.TrimSpace(r.URL.Query().Get("token"))
		if token == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "token is required.")
			return
		}
		cfg := d.effectiveConfig()
		parsedID, err := credsvc.VerifyBadgeExportToken(cfg, token, time.Now().UTC())
		if err != nil || parsedID != credID {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Invalid or expired download token.")
			return
		}
		cred, err := credrepo.GetByID(r.Context(), d.Pool, credID)
		if err != nil || cred == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Credential not found.")
			return
		}
		body, err := credsvc.FullCredentialJSON(cred)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load credential JSON.")
			return
		}
		w.Header().Set("Content-Type", "application/ld+json; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-badge.json"`, sanitizeFilename(cred.Title)))
		_, _ = w.Write(body)
	}
}

func (d Deps) handleLinkedInParams() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.credentialsFeatureOff(w) {
			return
		}
		cred, ok := d.loadOwnedCredential(w, r, userID)
		if !ok {
			return
		}
		params := credsvc.LinkedInParamsForCredential(d.effectiveConfig(), cred)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(params)
	}
}

func (d Deps) handleCredentialShare() http.HandlerFunc {
	type body struct {
		Channel string `json:"channel"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.credentialsFeatureOff(w) {
			return
		}
		cred, ok := d.loadOwnedCredential(w, r, userID)
		if !ok {
			return
		}
		var req body
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		channel := strings.TrimSpace(req.Channel)
		if channel != "linkedin" && channel != "badge_export" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "channel must be linkedin or badge_export.")
			return
		}
		courseID := cred.SourceID
		if cred.SourceType != credrepo.SourceCourse {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Share tracking requires a course credential.")
			return
		}
		if err := useraudit.InsertCredentialShare(r.Context(), d.Pool, userID, courseID, cred.ID, channel); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to record share event.")
			return
		}
		logging.GlobalCredentialsMetrics.IncShares()
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) handleVerifyCredential() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.credentialsFeatureOff(w) {
			return
		}
		credID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid credential id.")
			return
		}
		cred, err := credrepo.GetByID(r.Context(), d.Pool, credID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify credential.")
			return
		}
		if cred == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Credential not found.")
			return
		}
		if cred.Revoked {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"valid":      false,
				"status":     "Revoked",
				"issuerName": issuerNameFromConfig(d.effectiveConfig()),
				"issuedAt":   cred.IssuedAt.UTC().Format(time.RFC3339),
				"credential": json.RawMessage(cred.Proof),
				"title":      cred.Title,
			})
			return
		}
		cfg := d.effectiveConfig()
		valid, err := credsvc.VerifyCredential(cfg, cred.Proof)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Verification failed.")
			return
		}
		logging.GlobalCredentialsMetrics.IncVerifications()
		status := "Invalid"
		if valid {
			status = "Valid"
		}
		learnerName := extractLearnerName(cred.Proof)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"valid":        valid,
			"status":       status,
			"issuerName":   issuerNameFromConfig(cfg),
			"issuedAt":     cred.IssuedAt.UTC().Format(time.RFC3339),
			"credential":   json.RawMessage(cred.Proof),
			"title":        cred.Title,
			"learnerName":  learnerName,
			"verifyType":   "completion_credential",
		})
	}
}

func (d Deps) loadOwnedCredential(w http.ResponseWriter, r *http.Request, userID uuid.UUID) (*credrepo.IssuedCredential, bool) {
	credID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid credential id.")
		return nil, false
	}
	cred, err := credrepo.GetByID(r.Context(), d.Pool, credID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load credential.")
		return nil, false
	}
	if cred == nil || cred.RecipientID != userID {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Credential not found.")
		return nil, false
	}
	return cred, true
}

func (d Deps) notifyCertificateIssued(r *http.Request, userID uuid.UUID, cred *credrepo.IssuedCredential) {
	cfg := d.effectiveConfig()
	if !cfg.EmailNotificationsEnabled || d.Pool == nil {
		return
	}
	verifyURL := credsvc.VerificationURL(cfg.PublicWebOrigin, cred.ID)
	linkedIn := credsvc.LinkedInParamsForCredential(cfg, cred)
	svc := notifications.Service{Pool: d.Pool, Config: cfg}
	_ = svc.EnqueueEmail(r.Context(), userID, notificationevents.CertificateIssued, "certificate_issued", map[string]string{
		"credentialName": cred.Title,
		"verifyUrl":      verifyURL,
		"linkedInUrl":    linkedIn.URL,
		"credentialsUrl": cfg.PublicWebOrigin + "/me/credentials",
	}, nil)
}

func credentialSummaryJSON(cred *credrepo.IssuedCredential, origin string) map[string]any {
	return map[string]any{
		"id":              cred.ID.String(),
		"title":           cred.Title,
		"sourceType":      string(cred.SourceType),
		"sourceId":        cred.SourceID.String(),
		"issuedAt":        cred.IssuedAt.UTC().Format(time.RFC3339),
		"verificationUrl": credsvc.VerificationURL(origin, cred.ID),
		"revoked":         cred.Revoked,
	}
}

func issuerNameFromConfig(cfg config.Config) string {
	if strings.TrimSpace(cfg.CCRInstitutionName) != "" {
		return strings.TrimSpace(cfg.CCRInstitutionName)
	}
	return "Lextures"
}

func extractLearnerName(proof json.RawMessage) string {
	var vc map[string]any
	if err := json.Unmarshal(proof, &vc); err != nil {
		return ""
	}
	subject, _ := vc["credentialSubject"].(map[string]any)
	if subject == nil {
		return ""
	}
	if name, ok := subject["name"].(string); ok {
		return name
	}
	return ""
}

func sanitizeFilename(s string) string {
	out := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, strings.TrimSpace(s))
	if out == "" {
		return "credential"
	}
	return out
}