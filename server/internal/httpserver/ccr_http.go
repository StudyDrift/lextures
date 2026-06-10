package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/user"
	repo "github.com/lextures/lextures/server/internal/repos/ccr"
	ccrsvc "github.com/lextures/lextures/server/internal/service/ccr"
	"github.com/lextures/lextures/server/internal/service/vc_signing"
)

func (d Deps) requireCoCurricularTranscript(w http.ResponseWriter) bool {
	cfg := d.effectiveConfig()
	if !cfg.FFCoCurricularTranscript && !d.Config.FFCoCurricularTranscript {
		apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Co-curricular transcript is not enabled.")
		return false
	}
	return true
}

func (d Deps) ccrService() *ccrsvc.Service {
	cfg := d.effectiveConfig()
	return &ccrsvc.Service{
		Pool:            d.Pool,
		Config:          cfg,
		SecretsKey:      cfg.PlatformSecretsKey,
		PublicWebOrigin: cfg.PublicWebOrigin,
	}
}

func achievementToJSON(a repo.Achievement) map[string]any {
	out := map[string]any{
		"id":              a.ID.String(),
		"achievementType": string(a.AchievementType),
		"title":           a.Title,
		"issuedAt":        a.IssuedAt.UTC().Format(time.RFC3339),
		"outcomeTags":     a.OutcomeTags,
	}
	if a.SourceID != nil {
		out["sourceId"] = a.SourceID.String()
	}
	if a.Description != nil {
		out["description"] = *a.Description
	}
	if a.EvidenceURL != nil {
		out["evidenceUrl"] = *a.EvidenceURL
	}
	if a.AddedBy != nil {
		out["addedBy"] = a.AddedBy.String()
	}
	return out
}

func documentSummaryJSON(doc *repo.Document) map[string]any {
	out := map[string]any{
		"id":          doc.ID.String(),
		"generatedAt": doc.GeneratedAt.UTC().Format(time.RFC3339),
		"hasShareLink": doc.ShareToken != nil && doc.ConsentedAt != nil,
	}
	if doc.ShareToken != nil {
		out["shareToken"] = *doc.ShareToken
	}
	if doc.ConsentedAt != nil {
		out["consentedAt"] = doc.ConsentedAt.UTC().Format(time.RFC3339)
	}
	return out
}

func (d Deps) handleGetMyCCR() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireCoCurricularTranscript(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		svc := d.ccrService()
		achievements, err := svc.ListAchievements(r.Context(), userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load achievements.")
			return
		}
		docs, err := svc.ListDocuments(r.Context(), userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load CCR documents.")
			return
		}
		achOut := make([]map[string]any, 0, len(achievements))
		for _, a := range achievements {
			achOut = append(achOut, achievementToJSON(a))
		}
		docOut := make([]map[string]any, 0, len(docs))
		for i := range docs {
			docOut = append(docOut, documentSummaryJSON(&docs[i]))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"achievements": achOut,
			"documents":    docOut,
		})
	}
}

func (d Deps) handlePostMyCCRGenerate() http.HandlerFunc {
	type reqBody struct {
		ConsentToShare bool `json:"consentToShare"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireCoCurricularTranscript(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		var body reqBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		row, err := user.FindByID(r.Context(), d.Pool, userID)
		if err != nil || row == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "User not found.")
			return
		}
		studentName := row.Email
		if row.DisplayName != nil && strings.TrimSpace(*row.DisplayName) != "" {
			studentName = strings.TrimSpace(*row.DisplayName)
		}
		institutionName, _ := organizationNameForUser(r.Context(), d.Pool, userID)
		cfg := d.effectiveConfig()
		result, err := d.ccrService().Generate(r.Context(), ccrsvc.GenerateParams{
			UserID:              userID,
			StudentName:         studentName,
			InstitutionName:     institutionName,
			ConsentToShare:      body.ConsentToShare,
			VerificationBaseURL: cfg.PublicWebOrigin,
		})
		if err != nil {
			switch {
			case errors.Is(err, ccrsvc.ErrNoAchievements):
				apierr.WriteJSON(w, http.StatusUnprocessableEntity, apierr.CodeInvalidInput, err.Error())
			default:
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to generate CCR.")
			}
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"document": documentSummaryJSON(result.Document),
		})
	}
}

func (d Deps) handleGetMyCCRDownload() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireCoCurricularTranscript(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		docID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid document id.")
			return
		}
		doc, err := d.ccrService().GetDocument(r.Context(), userID, docID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load CCR document.")
			return
		}
		if doc == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "CCR document not found.")
			return
		}
		format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
		if format == "" {
			format = "json"
		}
		switch format {
		case "json":
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Content-Disposition", `attachment; filename="ccr.json"`)
			_, _ = w.Write(doc.CLRJSON)
		case "pdf":
			row, uerr := user.FindByID(r.Context(), d.Pool, userID)
			if uerr != nil || row == nil {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "User not found.")
				return
			}
			studentName := row.Email
			if row.DisplayName != nil {
				studentName = strings.TrimSpace(*row.DisplayName)
			}
			institutionName, _ := organizationNameForUser(r.Context(), d.Pool, userID)
			achievements, aerr := d.ccrService().ListAchievements(r.Context(), userID)
			if aerr != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load achievements.")
				return
			}
			verifyURL := ""
			if doc.ShareToken != nil && strings.TrimSpace(d.effectiveConfig().PublicWebOrigin) != "" {
				verifyURL = strings.TrimRight(d.effectiveConfig().PublicWebOrigin, "/") + "/verify/" + *doc.ShareToken
			}
			pdfBytes, perr := ccrsvc.BuildPDF(ccrsvc.BuildPDFInput{
				InstitutionName: institutionName,
				StudentName:     studentName,
				GeneratedAt:     doc.GeneratedAt,
				Achievements:    achievements,
				VerificationURL: verifyURL,
			})
			if perr != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to render CCR PDF.")
				return
			}
			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("Content-Disposition", `attachment; filename="ccr.pdf"`)
			_, _ = w.Write(pdfBytes)
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid format (use json or pdf).")
		}
	}
}

func (d Deps) handleVerifyCCRShareToken() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireCoCurricularTranscript(w) {
			return
		}
		token := strings.TrimSpace(chi.URLParam(r, "share_token"))
		if token == "" {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Verification link not found.")
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
			return
		}
		doc, err := repo.GetDocumentByShareToken(r.Context(), d.Pool, token)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify credential.")
			return
		}
		if doc == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Verification link not found.")
			return
		}
		valid, issuerName, issuedAt, achievements, err := d.ccrService().VerifyShareToken(r.Context(), token)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify credential.")
			return
		}
		achOut := make([]map[string]any, 0, len(achievements))
		for _, a := range achievements {
			achOut = append(achOut, achievementToJSON(a))
		}
		status := "Invalid"
		if valid {
			status = "Valid"
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"valid":        valid,
			"status":       status,
			"issuerName":   issuerName,
			"issuedAt":     issuedAt.UTC().Format(time.RFC3339),
			"achievements": achOut,
		})
	}
}

func (d Deps) handlePostAdminStudentCCRAchievement() http.HandlerFunc {
	type reqBody struct {
		Title        string   `json:"title"`
		Description  *string  `json:"description"`
		IssuedAt     string   `json:"issuedAt"`
		EvidenceURL  *string  `json:"evidenceUrl"`
		OutcomeTags  []string `json:"outcomeTags"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireCoCurricularTranscript(w) {
			return
		}
		viewer, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		isAdmin, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, permGlobalRBACManage)
		if err != nil || !isAdmin {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Admin access required.")
			return
		}
		studentID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "uid")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid student id.")
			return
		}
		var body reqBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		title := strings.TrimSpace(body.Title)
		if title == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Title is required.")
			return
		}
		issuedAt := time.Now().UTC()
		if strings.TrimSpace(body.IssuedAt) != "" {
			parsed, perr := time.Parse(time.RFC3339, strings.TrimSpace(body.IssuedAt))
			if perr != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid issuedAt (use RFC3339).")
				return
			}
			issuedAt = parsed.UTC()
		}
		rec, err := d.ccrService().AddManualAchievement(r.Context(), repo.UpsertAchievementParams{
			UserID:          studentID,
			AchievementType: repo.TypeExtracurricular,
			Title:           title,
			Description:     body.Description,
			IssuedAt:        issuedAt,
			EvidenceURL:     body.EvidenceURL,
			OutcomeTags:     body.OutcomeTags,
			AddedBy:         &viewer,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to add extracurricular record.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"achievement": achievementToJSON(*rec)})
	}
}

func (d Deps) handlePublicDIDDocument() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireCoCurricularTranscript(w) {
			return
		}
		km, err := d.ccrService().SigningKeyMaterial(r.Context())
		if err != nil || km == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Signing key not configured.")
			return
		}
		doc, err := vc_signing.BuildDIDDocument(km)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to build DID document.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(doc)
	}
}

func (d Deps) registerCCRRoutes(r chi.Router) {
	r.Get("/api/v1/me/ccr", d.handleGetMyCCR())
	r.Post("/api/v1/me/ccr/generate", d.handlePostMyCCRGenerate())
	r.Get("/api/v1/me/ccr/{id}/download", d.handleGetMyCCRDownload())
	r.Get("/api/v1/verify/{share_token}", d.handleVerifyCCRShareToken())
	r.Post("/api/v1/admin/students/{uid}/ccr/achievements", d.handlePostAdminStudentCCRAchievement())
}

func organizationNameForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (string, error) {
	var name string
	err := pool.QueryRow(ctx, `
SELECT o.name
FROM "user".users u
JOIN tenant.organizations o ON o.id = u.org_id
WHERE u.id = $1
`, userID).Scan(&name)
	if err != nil {
		return "Institution", err
	}
	if strings.TrimSpace(name) == "" {
		return "Institution", nil
	}
	return name, nil
}
