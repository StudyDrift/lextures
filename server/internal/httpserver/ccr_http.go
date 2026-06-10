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
	"github.com/lextures/lextures/server/internal/logging"
	acmodel "github.com/lextures/lextures/server/internal/models/accommodations"
	ccrrepo "github.com/lextures/lextures/server/internal/repos/ccr"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/user"
	ccrsvc "github.com/lextures/lextures/server/internal/service/ccr"
	"github.com/lextures/lextures/server/internal/service/vc_signing"
)

func (d Deps) ccrFeatureOff(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFCoCurricularTranscript {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Co-curricular transcript is not enabled.")
		return true
	}
	return false
}

func (d Deps) registerCCRRoutes(r chi.Router) {
	r.Get("/api/v1/me/ccr", d.handleListMyCCR())
	r.Post("/api/v1/me/ccr/generate", d.handleGenerateMyCCR())
	r.Get("/api/v1/me/ccr/{id}/download", d.handleDownloadMyCCR())
	r.Get("/api/v1/verify/{shareToken}", d.handleVerifyCCR())
	r.Get("/.well-known/did.json", d.handleInstitutionDID())
	r.Post("/api/v1/admin/students/{uid}/ccr/achievements", d.handleAdminAddCCRAchievement())
}

func (d Deps) handleListMyCCR() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.ccrFeatureOff(w) {
			return
		}
		achievements, err := ccrsvc.AggregateAchievements(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load achievements.")
			return
		}
		docs, err := ccrrepo.ListDocuments(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load CCR documents.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"achievements": achievementListJSON(achievements),
			"documents":    documentListJSON(docs, d.effectiveConfig().PublicWebOrigin),
		})
	}
}

func (d Deps) handleGenerateMyCCR() http.HandlerFunc {
	type body struct {
		SharePublicly bool `json:"sharePublicly"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.ccrFeatureOff(w) {
			return
		}
		var req body
		_ = json.NewDecoder(r.Body).Decode(&req)

		learnerName, err := d.learnerDisplayName(r, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load profile.")
			return
		}

		cfg := d.effectiveConfig()
		result, err := ccrsvc.Generate(r.Context(), d.Pool, cfg, ccrsvc.GenerateParams{
			UserID:          userID,
			LearnerName:     learnerName,
			SharePublicly:   req.SharePublicly,
			InstitutionName: cfg.CCRInstitutionName,
			APIOrigin:       cfg.PublicWebOrigin,
			SigningSeedB64:  cfg.CCRSigningSeedB64,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to generate CCR.")
			return
		}
		logging.GlobalCCRMetrics.IncGenerated()

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"document":     documentJSON(result.Document, cfg.PublicWebOrigin),
			"achievements": achievementListJSON(result.Achievements),
			"verificationUrl": result.Verification,
		})
	}
}

func (d Deps) handleDownloadMyCCR() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.ccrFeatureOff(w) {
			return
		}
		docID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid document id.")
			return
		}
		doc, err := ccrrepo.GetDocumentByID(r.Context(), d.Pool, userID, docID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load document.")
			return
		}
		if doc == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Document not found.")
			return
		}

		format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
		if format == "pdf" {
			learnerName, err := d.learnerDisplayName(r, userID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load profile.")
				return
			}
			achievements, err := ccrsvc.AggregateAchievements(r.Context(), d.Pool, userID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load achievements.")
				return
			}
			verifyURL := ""
			if doc.ShareToken != nil {
				verifyURL = verificationURL(d.effectiveConfig().PublicWebOrigin, *doc.ShareToken)
			}
			cfg := d.effectiveConfig()
			institution := strings.TrimSpace(cfg.CCRInstitutionName)
			if institution == "" {
				institution = "Lextures"
			}
			pdfBytes, err := ccrsvc.BuildPDF(ccrsvc.PDFInput{
				InstitutionName: institution,
				StudentName:     learnerName,
				GeneratedAt:     doc.GeneratedAt,
				VerificationURL: verifyURL,
				Achievements:    achievements,
			})
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to render PDF.")
				return
			}
			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("Content-Disposition", `attachment; filename="ccr.pdf"`)
			_, _ = w.Write(pdfBytes)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(doc.VCProof)
	}
}

func (d Deps) handleVerifyCCR() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.ccrFeatureOff(w) {
			return
		}
		token := strings.TrimSpace(chi.URLParam(r, "shareToken"))
		if token == "" {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		doc, err := ccrrepo.GetDocumentByShareToken(r.Context(), d.Pool, token)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify credential.")
			return
		}
		if doc == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}

		var vc map[string]any
		if err := json.Unmarshal(doc.VCProof, &vc); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Invalid credential.")
			return
		}

		cfg := d.effectiveConfig()
		key, err := ccrsvc.ResolveSigningKey(cfg, cfg.PublicWebOrigin, cfg.CCRSigningSeedB64)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Signing key unavailable.")
			return
		}
		valid, err := vcsigning.VerifyCredential(vc, key.PublicKey)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Verification failed.")
			return
		}
		logging.GlobalCCRMetrics.IncVerifications()

		issuerName := "Lextures"
		if strings.TrimSpace(cfg.CCRInstitutionName) != "" {
			issuerName = cfg.CCRInstitutionName
		}
		status := "Invalid"
		if valid {
			status = "Valid"
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"valid":      valid,
			"status":     status,
			"issuerName": issuerName,
			"issuedAt":   doc.GeneratedAt.UTC().Format(time.RFC3339),
			"credential": vc,
		})
	}
}

func (d Deps) handleInstitutionDID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.ccrFeatureOff(w) {
			return
		}
		cfg := d.effectiveConfig()
		key, err := ccrsvc.ResolveSigningKey(cfg, cfg.PublicWebOrigin, cfg.CCRSigningSeedB64)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Signing key unavailable.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(key.DIDDocument())
	}
}

func (d Deps) handleAdminAddCCRAchievement() http.HandlerFunc {
	type body struct {
		Title       string   `json:"title"`
		Description *string  `json:"description"`
		IssuedAt    *string  `json:"issuedAt"`
		EvidenceURL *string  `json:"evidenceUrl"`
		OutcomeTags []string `json:"outcomeTags"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		actorID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		if d.ccrFeatureOff(w) {
			return
		}
		if !d.userHasAccommodationsManage(w, r, actorID) {
			return
		}
		studentID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "uid")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid student id.")
			return
		}
		var req body
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		title := strings.TrimSpace(req.Title)
		if title == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "title is required.")
			return
		}
		issuedAt := time.Now().UTC()
		if req.IssuedAt != nil && strings.TrimSpace(*req.IssuedAt) != "" {
			parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.IssuedAt))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "issuedAt must be RFC3339.")
				return
			}
			issuedAt = parsed.UTC()
		}
		created, err := ccrrepo.CreateAchievement(r.Context(), d.Pool, ccrrepo.Achievement{
			UserID:          studentID,
			AchievementType: ccrrepo.TypeExtracurricular,
			Title:           title,
			Description:     req.Description,
			IssuedAt:        issuedAt,
			EvidenceURL:     req.EvidenceURL,
			OutcomeTags:     req.OutcomeTags,
			AddedBy:         &actorID,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create achievement.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(achievementRowJSON(created))
	}
}

func (d Deps) learnerDisplayName(r *http.Request, userID uuid.UUID) (string, error) {
	row, err := user.FindByID(r.Context(), d.Pool, userID)
	if err != nil || row == nil {
		return "", err
	}
	if row.DisplayName != nil && strings.TrimSpace(*row.DisplayName) != "" {
		return strings.TrimSpace(*row.DisplayName), nil
	}
	return row.Email, nil
}

func (d Deps) userHasAccommodationsManage(w http.ResponseWriter, r *http.Request, userID uuid.UUID) bool {
	hasPerm, err := rbac.UserHasPermission(r.Context(), d.Pool, userID, acmodel.PermManage)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return false
	}
	if !hasPerm {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
		return false
	}
	return true
}

func achievementListJSON(items []ccrsvc.AggregatedAchievement) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, a := range items {
		out = append(out, map[string]any{
			"id":          a.ID,
			"type":        string(a.Type),
			"title":       a.Title,
			"description": a.Description,
			"issuedAt":    a.IssuedAt.UTC().Format(time.RFC3339),
			"evidenceUrl": a.EvidenceURL,
			"outcomeTags": a.OutcomeTags,
		})
	}
	return out
}

func achievementRowJSON(a *ccrrepo.Achievement) map[string]any {
	desc := ""
	if a.Description != nil {
		desc = *a.Description
	}
	evidence := ""
	if a.EvidenceURL != nil {
		evidence = *a.EvidenceURL
	}
	return map[string]any{
		"id":          a.ID.String(),
		"type":        string(a.AchievementType),
		"title":       a.Title,
		"description": desc,
		"issuedAt":    a.IssuedAt.UTC().Format(time.RFC3339),
		"evidenceUrl": evidence,
		"outcomeTags": a.OutcomeTags,
	}
}

func documentListJSON(docs []ccrrepo.Document, origin string) []map[string]any {
	out := make([]map[string]any, 0, len(docs))
	for i := range docs {
		out = append(out, documentJSON(&docs[i], origin))
	}
	return out
}

func documentJSON(doc *ccrrepo.Document, origin string) map[string]any {
	row := map[string]any{
		"id":          doc.ID.String(),
		"generatedAt": doc.GeneratedAt.UTC().Format(time.RFC3339),
		"shareable":   doc.ShareToken != nil,
	}
	if doc.ShareToken != nil {
		row["verificationUrl"] = verificationURL(origin, *doc.ShareToken)
	}
	return row
}

func verificationURL(origin, token string) string {
	return fmt.Sprintf("%s/verify/%s", strings.TrimRight(strings.TrimSpace(origin), "/"), token)
}
