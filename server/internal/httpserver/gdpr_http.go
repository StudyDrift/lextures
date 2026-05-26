package httpserver

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/gdpr"
	"github.com/lextures/lextures/server/internal/repos/organization"
	gdprservice "github.com/lextures/lextures/server/internal/service/gdpr"
)

func (d Deps) gdprEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().GDPRModuleEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "GDPR module is not enabled.")
		return false
	}
	return true
}

func (d Deps) requireGDPRAdmin(w http.ResponseWriter, r *http.Request) (userID uuid.UUID, orgID uuid.UUID, ok bool) {
	uid, ok := d.meUserID(w, r)
	if !ok {
		return uuid.UUID{}, uuid.UUID{}, false
	}
	oid, err := organization.OrgIDForUser(r.Context(), d.Pool, uid)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load organization.")
		return uuid.UUID{}, uuid.UUID{}, false
	}
	isAdmin, err := gdprservice.CheckAdmin(r.Context(), d.Pool, uid)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Permission check failed.")
		return uuid.UUID{}, uuid.UUID{}, false
	}
	if !isAdmin {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
		return uuid.UUID{}, uuid.UUID{}, false
	}
	return uid, oid, true
}

func (d Deps) registerGDPRRoutes(r chi.Router) {
	r.Post("/api/v1/compliance/gdpr/dsar", d.handlePostDSAR())
	r.Get("/api/v1/compliance/gdpr/dsar", d.handleGetDSARList())
	r.Get("/api/v1/compliance/gdpr/dsar/{id}/download", d.handleGetDSARDownload())
	r.Patch("/api/v1/compliance/gdpr/dsar/{id}", d.handlePatchDSAR())
	r.Post("/api/v1/compliance/gdpr/consents", d.handlePostGDPRConsent())
	r.Get("/api/v1/compliance/gdpr/consents", d.handleGetGDPRConsents())
	r.Delete("/api/v1/compliance/gdpr/consents/{id}", d.handleDeleteGDPRConsent())
	r.Get("/api/v1/compliance/gdpr/ropa", d.handleGetRoPA())
	r.Post("/api/v1/compliance/gdpr/ropa", d.handlePostRoPAEntry())
	r.Delete("/api/v1/compliance/gdpr/ropa/{id}", d.handleDeleteRoPAEntry())
	r.Get("/api/v1/compliance/gdpr/dpa-template", d.handleGetDPATemplate())
}

type postDSARBody struct {
	RequestType string `json:"requestType"`
}

// POST /api/v1/compliance/gdpr/dsar
func (d Deps) handlePostDSAR() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.gdprEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body postDSARBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		validTypes := map[string]bool{
			"access": true, "erasure": true, "portability": true,
			"rectification": true, "restriction": true, "objection": true,
		}
		if !validTypes[body.RequestType] {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid requestType.")
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load organization.")
			return
		}
		id, err := gdprservice.SubmitDSAR(r.Context(), d.Pool, &orgID, userID, body.RequestType)
		if err != nil {
			if errors.Is(err, gdprservice.ErrAlreadyExists) {
				apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Your request is already in progress. Please wait for the current one to complete.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not submit DSAR.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": id.String()})
	}
}

// GET /api/v1/compliance/gdpr/dsar
func (d Deps) handleGetDSARList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.gdprEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}

		// Admins can see all pending requests via ?queue=true
		if r.URL.Query().Get("queue") == "true" {
			isAdmin, err := gdprservice.CheckAdmin(r.Context(), d.Pool, userID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Permission check failed.")
				return
			}
			if !isAdmin {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
				return
			}
			requests, err := gdprservice.ListPendingDSARs(r.Context(), d.Pool)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load DSAR queue.")
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(map[string]any{"requests": dsarsToJSON(requests)})
			return
		}

		requests, err := gdprservice.ListDSARsForUser(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load DSAR requests.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"requests": dsarsToJSON(requests)})
	}
}

// GET /api/v1/compliance/gdpr/dsar/{id}/download
func (d Deps) handleGetDSARDownload() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.gdprEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid DSAR id.")
			return
		}
		req, err := gdprservice.GetDSARForUser(r.Context(), d.Pool, id, userID)
		if err != nil {
			if errors.Is(err, gdprservice.ErrNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "DSAR request not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load DSAR request.")
			return
		}
		if req.Status != "completed" {
			apierr.WriteJSON(w, http.StatusUnprocessableEntity, apierr.CodeUnprocessableEntity, "DSAR is not yet completed.")
			return
		}
		if req.ArchiveURL == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Archive not available.")
			return
		}
		if req.ArchiveExpiresAt != nil && req.ArchiveExpiresAt.Before(time.Now().UTC()) {
			apierr.WriteJSON(w, http.StatusGone, apierr.CodeNotFound, "Download link has expired.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="dsar-export.json"`)
		_, _ = w.Write([]byte(*req.ArchiveURL))
	}
}

type patchDSARBody struct {
	Status          string  `json:"status"`
	RejectionReason *string `json:"rejectionReason"`
}

// PATCH /api/v1/compliance/gdpr/dsar/{id}
func (d Deps) handlePatchDSAR() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.gdprEnabled(w) {
			return
		}
		adminID, _, ok := d.requireGDPRAdmin(w, r)
		if !ok {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid DSAR id.")
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body patchDSARBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		switch body.Status {
		case "approved":
			if err := gdprservice.ApproveDSAR(r.Context(), d.Pool, id, adminID); err != nil {
				if errors.Is(err, gdprservice.ErrNotFound) {
					apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "DSAR not found.")
					return
				}
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not approve DSAR.")
				return
			}
		case "rejected":
			reason := ""
			if body.RejectionReason != nil {
				reason = strings.TrimSpace(*body.RejectionReason)
			}
			if err := gdprservice.RejectDSAR(r.Context(), d.Pool, id, adminID, reason); err != nil {
				if errors.Is(err, gdprservice.ErrNotFound) {
					apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "DSAR not found.")
					return
				}
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not reject DSAR.")
				return
			}
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "status must be approved or rejected.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}
}

type postGDPRConsentBody struct {
	Purpose        string `json:"purpose"`
	LawfulBasis    string `json:"lawfulBasis"`
	ConsentVersion string `json:"consentVersion"`
}

// POST /api/v1/compliance/gdpr/consents
func (d Deps) handlePostGDPRConsent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.gdprEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body postGDPRConsentBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if strings.TrimSpace(body.Purpose) == "" || strings.TrimSpace(body.LawfulBasis) == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "purpose and lawfulBasis are required.")
			return
		}
		validBases := map[string]bool{
			"consent": true, "contract": true, "legal_obligation": true,
			"vital_interests": true, "legitimate_interests": true,
		}
		if !validBases[body.LawfulBasis] {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid lawfulBasis.")
			return
		}
		version := body.ConsentVersion
		if version == "" {
			version = "1.0"
		}
		ipHash := gdprIPHash(r.RemoteAddr)
		id, err := gdprservice.GrantConsent(r.Context(), d.Pool, userID, body.Purpose, body.LawfulBasis, version, &ipHash)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not record consent.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": id.String()})
	}
}

// GET /api/v1/compliance/gdpr/consents
func (d Deps) handleGetGDPRConsents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.gdprEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		consents, err := gdprservice.ListConsents(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load consents.")
			return
		}
		type item struct {
			ID             string  `json:"id"`
			Purpose        string  `json:"purpose"`
			LawfulBasis    string  `json:"lawfulBasis"`
			ConsentVersion string  `json:"consentVersion"`
			GrantedAt      string  `json:"grantedAt"`
			WithdrawnAt    *string `json:"withdrawnAt,omitempty"`
		}
		out := make([]item, 0, len(consents))
		for _, c := range consents {
			it := item{
				ID:             c.ID.String(),
				Purpose:        c.Purpose,
				LawfulBasis:    c.LawfulBasis,
				ConsentVersion: c.ConsentVersion,
				GrantedAt:      c.GrantedAt.UTC().Format(time.RFC3339),
			}
			if c.WithdrawnAt != nil {
				s := c.WithdrawnAt.UTC().Format(time.RFC3339)
				it.WithdrawnAt = &s
			}
			out = append(out, it)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"consents": out})
	}
}

// DELETE /api/v1/compliance/gdpr/consents/{id}
func (d Deps) handleDeleteGDPRConsent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.gdprEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid consent id.")
			return
		}
		if err := gdprservice.WithdrawConsent(r.Context(), d.Pool, id, userID); err != nil {
			if errors.Is(err, gdprservice.ErrNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Consent record not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not withdraw consent.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}
}

// GET /api/v1/compliance/gdpr/ropa
func (d Deps) handleGetRoPA() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.gdprEnabled(w) {
			return
		}
		_, orgID, ok := d.requireGDPRAdmin(w, r)
		if !ok {
			return
		}
		entries, err := gdprservice.ListRoPAEntries(r.Context(), d.Pool, orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load RoPA entries.")
			return
		}
		type item struct {
			ID              string   `json:"id"`
			ActivityName    string   `json:"activityName"`
			Purpose         string   `json:"purpose"`
			LawfulBasis     string   `json:"lawfulBasis"`
			DataCategories  []string `json:"dataCategories"`
			DataSubjects    []string `json:"dataSubjects"`
			RetentionPeriod string   `json:"retentionPeriod"`
			SubProcessors   []string `json:"subProcessors"`
			UpdatedAt       string   `json:"updatedAt"`
		}
		out := make([]item, 0, len(entries))
		for _, e := range entries {
			out = append(out, item{
				ID:              e.ID.String(),
				ActivityName:    e.ActivityName,
				Purpose:         e.Purpose,
				LawfulBasis:     e.LawfulBasis,
				DataCategories:  gdprNonNilSlice(e.DataCategories),
				DataSubjects:    gdprNonNilSlice(e.DataSubjects),
				RetentionPeriod: e.RetentionPeriod,
				SubProcessors:   gdprNonNilSlice(e.SubProcessors),
				UpdatedAt:       e.UpdatedAt.UTC().Format(time.RFC3339),
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"entries": out})
	}
}

type postRoPABody struct {
	ActivityName    string   `json:"activityName"`
	Purpose         string   `json:"purpose"`
	LawfulBasis     string   `json:"lawfulBasis"`
	DataCategories  []string `json:"dataCategories"`
	DataSubjects    []string `json:"dataSubjects"`
	RetentionPeriod string   `json:"retentionPeriod"`
	SubProcessors   []string `json:"subProcessors"`
}

// POST /api/v1/compliance/gdpr/ropa
func (d Deps) handlePostRoPAEntry() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.gdprEnabled(w) {
			return
		}
		_, orgID, ok := d.requireGDPRAdmin(w, r)
		if !ok {
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body postRoPABody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if strings.TrimSpace(body.ActivityName) == "" || strings.TrimSpace(body.Purpose) == "" || strings.TrimSpace(body.LawfulBasis) == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "activityName, purpose, and lawfulBasis are required.")
			return
		}
		id, err := gdprservice.AddRoPAEntry(r.Context(), d.Pool, orgID,
			body.ActivityName, body.Purpose, body.LawfulBasis, body.RetentionPeriod,
			gdprNonNilSlice(body.DataCategories), gdprNonNilSlice(body.DataSubjects), gdprNonNilSlice(body.SubProcessors))
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not add RoPA entry.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": id.String()})
	}
}

// DELETE /api/v1/compliance/gdpr/ropa/{id}
func (d Deps) handleDeleteRoPAEntry() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.gdprEnabled(w) {
			return
		}
		_, orgID, ok := d.requireGDPRAdmin(w, r)
		if !ok {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid RoPA entry id.")
			return
		}
		if err := gdprservice.DeleteRoPAEntry(r.Context(), d.Pool, id, orgID); err != nil {
			if errors.Is(err, gdprservice.ErrNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "RoPA entry not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not delete RoPA entry.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}
}

// GET /api/v1/compliance/gdpr/dpa-template
func (d Deps) handleGetDPATemplate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.gdprEnabled(w) {
			return
		}
		_, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		privacyURL := d.effectiveConfig().PublicWebOrigin + "/privacy"
		tpl := gdprservice.GenerateDPATemplate("Your Organization", privacyURL)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(tpl)
	}
}

// dsarsToJSON converts a slice of DSARRequest to JSON-friendly map slices.
func dsarsToJSON(requests []gdpr.DSARRequest) []map[string]any {
	out := make([]map[string]any, 0, len(requests))
	for _, req := range requests {
		m := map[string]any{
			"id":          req.ID.String(),
			"userId":      req.UserID.String(),
			"requestType": req.RequestType,
			"status":      req.Status,
			"requestedAt": req.RequestedAt.UTC().Format(time.RFC3339),
			"dueAt":       req.DueAt.UTC().Format(time.RFC3339),
		}
		if req.OrgID != nil {
			m["orgId"] = req.OrgID.String()
		}
		if req.RejectionReason != nil {
			m["rejectionReason"] = *req.RejectionReason
		}
		if req.CompletedAt != nil {
			m["completedAt"] = req.CompletedAt.UTC().Format(time.RFC3339)
		}
		if req.ActionedBy != nil {
			m["actionedBy"] = req.ActionedBy.String()
		}
		out = append(out, m)
	}
	return out
}

func gdprNonNilSlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// gdprIPHash returns a SHA-256 hex digest of the client IP address.
func gdprIPHash(remoteAddr string) string {
	ip := remoteAddr
	if i := strings.LastIndex(remoteAddr, ":"); i > 0 {
		ip = remoteAddr[:i]
	}
	sum := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(sum[:])
}
