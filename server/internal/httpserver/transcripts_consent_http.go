package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/organization"
	transcriptsrepo "github.com/lextures/lextures/server/internal/repos/transcripts"
	ferpaservice "github.com/lextures/lextures/server/internal/service/ferpa"
)

func (d Deps) registerTranscriptConsentRoutes(r chi.Router) {
	r.Get("/api/v1/transcripts/orders/{id}/consent/preview", d.handleGetTranscriptConsentPreview())
	r.Post("/api/v1/transcripts/orders/{id}/consent", d.handlePostTranscriptConsent())
	r.Post("/api/v1/transcripts/orders/{id}/consent/revoke", d.handlePostTranscriptConsentRevoke())
	r.Get("/api/v1/transcripts/orders/{id}/consent/export", d.handleGetTranscriptConsentExport())
	r.Post("/api/v1/parent/transcripts/orders/{id}/consent", d.handlePostParentTranscriptConsent())
}

type consentSummaryJSON struct {
	ID                   string  `json:"id"`
	SignerID             string  `json:"signerId"`
	SignerRole           string  `json:"signerRole"`
	GuardianRelationship *string `json:"guardianRelationship,omitempty"`
	TextVersion          string  `json:"textVersion"`
	Locale               string  `json:"locale"`
	SignatureMethod      string  `json:"signatureMethod"`
	PayloadHash          string  `json:"payloadHash"`
	SignedAt             string  `json:"signedAt"`
	RevokedAt            *string `json:"revokedAt,omitempty"`
	ExpiresAt            *string `json:"expiresAt,omitempty"`
}

type consentPreviewJSON struct {
	OrderID            string                         `json:"orderId"`
	Status             string                         `json:"status"`
	TextVersion        string                         `json:"textVersion"`
	Locale             string                         `json:"locale"`
	AuthorizationText  string                         `json:"authorizationText"`
	Scope              string                         `json:"scope"`
	Purpose            string                         `json:"purpose"`
	Recipients         []consentRecipientSnapshotJSON `json:"recipients"`
	RequiresConsent    bool                           `json:"requiresConsent"`
	SelfDisclosureOnly bool                           `json:"selfDisclosureOnly"`
	RequiresGuardian   bool                           `json:"requiresGuardian"`
	IsMinor            bool                           `json:"isMinor"`
	ConsentRequired    bool                           `json:"consentRequired"`
	ActiveConsent      *consentSummaryJSON            `json:"activeConsent,omitempty"`
}

type consentRecipientSnapshotJSON struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
}

func consentToSummaryJSON(c *transcriptsrepo.Consent) *consentSummaryJSON {
	if c == nil {
		return nil
	}
	out := &consentSummaryJSON{
		ID:              c.ID.String(),
		SignerID:        c.SignerID.String(),
		SignerRole:      string(c.SignerRole),
		TextVersion:     c.TextVersion,
		Locale:          c.Locale,
		SignatureMethod: string(c.SignatureMethod),
		PayloadHash:     c.PayloadHash,
		SignedAt:        c.SignedAt.UTC().Format(time.RFC3339),
	}
	out.GuardianRelationship = c.GuardianRelationship
	if c.RevokedAt != nil {
		s := c.RevokedAt.UTC().Format(time.RFC3339)
		out.RevokedAt = &s
	}
	if c.ExpiresAt != nil {
		s := c.ExpiresAt.UTC().Format(time.RFC3339)
		out.ExpiresAt = &s
	}
	return out
}

func consentPreviewToJSON(p *transcriptsrepo.ConsentPreview) consentPreviewJSON {
	recs := make([]consentRecipientSnapshotJSON, 0, len(p.Recipients))
	for _, r := range p.Recipients {
		recs = append(recs, consentRecipientSnapshotJSON{ID: r.ID, Type: r.Type, Name: r.Name})
	}
	return consentPreviewJSON{
		OrderID:            p.OrderID.String(),
		Status:             string(p.Status),
		TextVersion:        p.TextVersion,
		Locale:             p.Locale,
		AuthorizationText:  p.AuthorizationText,
		Scope:              p.Scope,
		Purpose:            p.Purpose,
		Recipients:         recs,
		RequiresConsent:    p.RequiresConsent,
		SelfDisclosureOnly: p.SelfDisclosureOnly,
		RequiresGuardian:   p.RequiresGuardian,
		IsMinor:            p.IsMinor,
		ConsentRequired:    p.ConsentRequired,
		ActiveConsent:      consentToSummaryJSON(p.ActiveConsent),
	}
}

func writeConsentRepoError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, transcriptsrepo.ErrOrderNotFound):
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Order not found.")
	case errors.Is(err, transcriptsrepo.ErrConsentNotFound):
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Consent not found.")
	case errors.Is(err, transcriptsrepo.ErrConsentAlreadySigned):
		apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Order already has an active authorization.")
	case errors.Is(err, transcriptsrepo.ErrConsentNotRequired):
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Consent is not required for this order.")
	case errors.Is(err, transcriptsrepo.ErrConsentNotAgreed):
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "You must agree to authorize the release.")
	case errors.Is(err, transcriptsrepo.ErrConsentInvalidSignature):
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Signature is missing or invalid.")
	case errors.Is(err, transcriptsrepo.ErrConsentWrongState):
		apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Order is not awaiting consent.")
	case errors.Is(err, transcriptsrepo.ErrConsentAlreadyDelivered):
		apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Cannot revoke after delivery has started.")
	case errors.Is(err, transcriptsrepo.ErrConsentStudentIsMinor):
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "A linked parent or guardian must authorize this release.")
	case errors.Is(err, transcriptsrepo.ErrConsentGuardianRequired):
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Guardian authorization is required for this student.")
	case errors.Is(err, transcriptsrepo.ErrConsentNotGuardian):
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "No active link to this student.")
	case errors.Is(err, transcriptsrepo.ErrOrderEmpty):
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Order must have at least one item.")
	default:
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Consent request failed.")
	}
}

func (d Deps) handleGetTranscriptConsentPreview() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		orderID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid order id.")
			return
		}
		o, err := transcriptsrepo.GetOrderForUser(r.Context(), d.Pool, orderID, userID)
		if err != nil {
			writeConsentRepoError(w, err)
			return
		}
		cfg, err := transcriptsrepo.GetConfig(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load config.")
			return
		}
		locale := strings.TrimSpace(r.URL.Query().Get("locale"))
		preview, err := transcriptsrepo.BuildConsentPreview(r.Context(), d.Pool, cfg, o, locale)
		if err != nil {
			writeConsentRepoError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"preview": consentPreviewToJSON(preview)})
	}
}

type postTranscriptConsentBody struct {
	Method        string `json:"method"`
	SignatureData string `json:"signatureData"`
	Agree         bool   `json:"agree"`
	Locale        string `json:"locale"`
	Purpose       string `json:"purpose"`
}

func (d Deps) handlePostTranscriptConsent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		orderID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid order id.")
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body postTranscriptConsentBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		method := transcriptsrepo.SignatureMethod(strings.ToLower(strings.TrimSpace(body.Method)))
		if method != transcriptsrepo.SignatureTyped && method != transcriptsrepo.SignatureDrawn {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "method must be typed or drawn.")
			return
		}
		cfg, err := transcriptsrepo.GetConfig(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load config.")
			return
		}
		consent, order, err := transcriptsrepo.SignConsent(r.Context(), d.Pool, cfg, transcriptsrepo.SignConsentInput{
			OrderID:       orderID,
			SignerID:      userID,
			SignerRole:    transcriptsrepo.SignerRoleStudent,
			Method:        method,
			SignatureData: body.SignatureData,
			Agree:         body.Agree,
			Locale:        body.Locale,
			Purpose:       body.Purpose,
			IP:            clientIP(r),
			UserAgent:     r.UserAgent(),
		})
		if err != nil {
			writeConsentRepoError(w, err)
			return
		}
		d.logTranscriptConsentDisclosure(r, order, consent, "transcript_consent_signed")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"consent": consentToSummaryJSON(consent),
			"order":   orderToJSON(order),
		})
	}
}

func (d Deps) handlePostTranscriptConsentRevoke() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		orderID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid order id.")
			return
		}
		consent, order, err := transcriptsrepo.RevokeConsent(r.Context(), d.Pool, orderID, userID)
		if err != nil {
			writeConsentRepoError(w, err)
			return
		}
		d.logTranscriptConsentDisclosure(r, order, consent, "transcript_consent_revoked")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"consent": consentToSummaryJSON(consent),
			"order":   orderToJSON(order),
		})
	}
}

func (d Deps) handleGetTranscriptConsentExport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		orderID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid order id.")
			return
		}
		export, err := transcriptsrepo.ExportConsentJSON(r.Context(), d.Pool, orderID, userID)
		if err != nil {
			writeConsentRepoError(w, err)
			return
		}
		format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
		if format == "pdf" {
			pdf := transcriptsrepo.ExportConsentPDFBytes(export)
			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("Content-Disposition", `attachment; filename="transcript-consent.pdf"`)
			_, _ = w.Write(pdf)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"export": export})
	}
}

func (d Deps) handlePostParentTranscriptConsent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.transcriptsFeatureOff(w) {
			return
		}
		parentID, orgID, ok := d.requireParentViewer(w, r)
		if !ok {
			return
		}
		orderID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid order id.")
			return
		}
		o, err := transcriptsrepo.GetOrderByID(r.Context(), d.Pool, orderID)
		if err != nil {
			writeConsentRepoError(w, err)
			return
		}
		if o.OrgID == nil || *o.OrgID != orgID {
			// Still allow when student org matches parent org via OrgIDForUser fallback.
			studentOrg, oerr := organization.OrgIDForUser(r.Context(), d.Pool, o.UserID)
			if oerr != nil || studentOrg != orgID {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "No active link to this student.")
				return
			}
		}
		link, ok := d.requireParentLink(w, r, parentID, orgID, o.UserID)
		if !ok {
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body postTranscriptConsentBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		method := transcriptsrepo.SignatureMethod(strings.ToLower(strings.TrimSpace(body.Method)))
		if method != transcriptsrepo.SignatureTyped && method != transcriptsrepo.SignatureDrawn {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "method must be typed or drawn.")
			return
		}
		cfg, err := transcriptsrepo.GetConfig(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load config.")
			return
		}
		rel := link.Relationship
		if rel == "" {
			rel = "guardian"
		}
		consent, order, err := transcriptsrepo.SignConsent(r.Context(), d.Pool, cfg, transcriptsrepo.SignConsentInput{
			OrderID:       orderID,
			SignerID:      parentID,
			SignerRole:    transcriptsrepo.SignerRoleGuardian,
			GuardianRel:   &rel,
			Method:        method,
			SignatureData: body.SignatureData,
			Agree:         body.Agree,
			Locale:        body.Locale,
			Purpose:       body.Purpose,
			IP:            clientIP(r),
			UserAgent:     r.UserAgent(),
		})
		if err != nil {
			writeConsentRepoError(w, err)
			return
		}
		d.logTranscriptConsentDisclosure(r, order, consent, "transcript_consent_signed_guardian")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"consent": consentToSummaryJSON(consent),
			"order":   orderToJSON(order),
		})
	}
}

func (d Deps) logTranscriptConsentDisclosure(r *http.Request, order *transcriptsrepo.Order, consent *transcriptsrepo.Consent, dataType string) {
	if d.Pool == nil || order == nil || consent == nil || order.OrgID == nil {
		return
	}
	recipient := "transcript_order:" + order.ID.String()
	_ = ferpaservice.LogDisclosure(r.Context(), d.Pool, *order.OrgID, consent.SignerID, order.UserID, dataType, "consent", &recipient)
}
