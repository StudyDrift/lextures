package httpserver

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/mail"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/parentlinks"
	"github.com/lextures/lextures/server/internal/repos/user"
	coppaservice "github.com/lextures/lextures/server/internal/service/coppa"
)

func (d Deps) coppaEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().CoppaWorkflowEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "COPPA workflow is not enabled.")
		return false
	}
	return true
}

// handleCoppaStatus is GET /api/v1/compliance/coppa/status
// Returns the authenticated user's own COPPA consent status.
func (d Deps) handleCoppaStatus() http.HandlerFunc {
	type resp struct {
		CoppaMinor        bool   `json:"coppaMinor"`
		ConsentStatus     string `json:"consentStatus"`
		ParentEmail       string `json:"parentEmail,omitempty"`
		AIFeaturesEnabled bool   `json:"aiFeaturesEnabled"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.coppaEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		status, err := coppaservice.GetUserConsentStatus(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load COPPA status.")
			return
		}
		out := resp{
			CoppaMinor:        status.CoppaMinor,
			ConsentStatus:     string(status.ConsentStatus),
			AIFeaturesEnabled: status.AIFeaturesEnabled,
		}
		if status.ParentEmail != nil {
			out.ParentEmail = *status.ParentEmail
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

// handleCoppaConsentToken is POST /api/v1/compliance/coppa/consent-token
// Public endpoint (signed link): parent submits their token to confirm consent.
func (d Deps) handleCoppaConsentToken() http.HandlerFunc {
	type req struct {
		Token string `json:"token"`
	}
	type resp struct {
		ConsentID string `json:"consentId"`
		StudentID string `json:"studentId"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.coppaEnabled(w) {
			return
		}
		var body req
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		tok := strings.TrimSpace(body.Token)
		if tok == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Token is required.")
			return
		}
		rec, err := coppaservice.ConsumeConsentToken(r.Context(), d.Pool, tok, time.Now().UTC())
		if err != nil {
			switch err {
			case coppaservice.ErrTokenExpired:
				apierr.WriteJSON(w, http.StatusGone, apierr.CodeInvalidResetToken, "This consent link has expired. Please request a new one.")
			case coppaservice.ErrTokenInvalid:
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid consent token.")
			case coppaservice.ErrAlreadyApproved:
				// still return success with the record
			default:
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to process consent.")
				return
			}
			if err != coppaservice.ErrAlreadyApproved {
				return
			}
		}
		// Send confirmation email (best-effort).
		go func() {
			if rec != nil {
				_ = mail.SendCoppaConsentConfirmation(d.effectiveConfig(), rec.ParentEmail, "", nil)
			}
		}()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp{
			ConsentID: rec.ID.String(),
			StudentID: rec.StudentID.String(),
		})
	}
}

// handleCoppaParentDashboard is GET /api/v1/compliance/coppa/parent-dashboard
// Parent role: returns all COPPA consent records for their linked children.
func (d Deps) handleCoppaParentDashboard() http.HandlerFunc {
	type consentOut struct {
		ID                 string  `json:"id"`
		StudentID          string  `json:"studentId"`
		ParentEmail        string  `json:"parentEmail"`
		ConsentMethod      string  `json:"consentMethod"`
		ConsentedAt        *string `json:"consentedAt"`
		RevokedAt          *string `json:"revokedAt"`
		AIFeaturesEnabled  bool    `json:"aiFeaturesEnabled"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.coppaEnabled(w) {
			return
		}
		parentID, orgID, ok := d.requireParentViewer(w, r)
		if !ok {
			return
		}

		children, err := parentlinks.ListChildrenForParent(r.Context(), d.Pool, parentID, orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list children.")
			return
		}

		var out []consentOut
		for _, child := range children {
			recs, err := coppaservice.ListConsentRecords(r.Context(), d.Pool, child.StudentUserID)
			if err != nil {
				continue
			}
			for _, rec := range recs {
				co := consentOut{
					ID:                rec.ID.String(),
					StudentID:         rec.StudentID.String(),
					ParentEmail:       rec.ParentEmail,
					ConsentMethod:     string(rec.ConsentMethod),
					AIFeaturesEnabled: rec.AIFeaturesEnabled,
				}
				if rec.ConsentedAt != nil {
					s := rec.ConsentedAt.UTC().Format(time.RFC3339Nano)
					co.ConsentedAt = &s
				}
				if rec.RevokedAt != nil {
					s := rec.RevokedAt.UTC().Format(time.RFC3339Nano)
					co.RevokedAt = &s
				}
				out = append(out, co)
			}
		}
		if out == nil {
			out = []consentOut{}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"consents": out})
	}
}

// handleCoppaConsentRevoke is DELETE /api/v1/compliance/coppa/consent/{id}
// Parent role: revoke a consent record.
func (d Deps) handleCoppaConsentRevoke() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.Header().Set("Allow", http.MethodDelete)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.coppaEnabled(w) {
			return
		}
		parentID, _, ok := d.requireParentViewer(w, r)
		if !ok {
			return
		}
		consentID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid consent id.")
			return
		}
		if err := coppaservice.RevokeConsent(r.Context(), d.Pool, consentID, parentID, time.Now().UTC()); err != nil {
			switch err {
			case coppaservice.ErrNotFound:
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Consent record not found.")
			case coppaservice.ErrAlreadyRevoked:
				apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Consent already revoked.")
			default:
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to revoke consent.")
			}
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleCoppaAIOptIn is PATCH /api/v1/compliance/coppa/ai-opt-in
// Parent role: toggle per-feature AI opt-in for a minor student.
func (d Deps) handleCoppaAIOptIn() http.HandlerFunc {
	type req struct {
		StudentID string `json:"studentId"`
		Enabled   bool   `json:"enabled"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			w.Header().Set("Allow", http.MethodPatch)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.coppaEnabled(w) {
			return
		}
		parentID, orgID, ok := d.requireParentViewer(w, r)
		if !ok {
			return
		}
		var body req
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		studentID, err := uuid.Parse(strings.TrimSpace(body.StudentID))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid studentId.")
			return
		}
		if _, ok := d.requireParentLink(w, r, parentID, orgID, studentID); !ok {
			return
		}
		if err := coppaservice.SetAIOptIn(r.Context(), d.Pool, studentID, body.Enabled); err != nil {
			switch err {
			case coppaservice.ErrNotFound:
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "No active consent record found.")
			default:
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update AI opt-in.")
			}
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"enabled": body.Enabled})
	}
}

// handleCoppaBulkImport is POST /api/v1/compliance/coppa/bulk-import
// Org admin: CSV bulk district consent import.
func (d Deps) handleCoppaBulkImport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.coppaEnabled(w) {
			return
		}
		orgID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgId")))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid organization id.")
			return
		}
		if _, ok := d.orgRoleAccess(w, r, orgID, true); !ok {
			return
		}

		cr := csv.NewReader(r.Body)
		cr.TrimLeadingSpace = true
		cr.FieldsPerRecord = -1

		var rows []coppaservice.BulkImportRow
		lineNum := 0
		for {
			rec, err := cr.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid CSV.")
				return
			}
			lineNum++
			if lineNum == 1 {
				// skip header
				if len(rec) > 0 && strings.EqualFold(strings.TrimSpace(rec[0]), "student_id") {
					continue
				}
			}
			if len(rec) < 2 || strings.TrimSpace(rec[0]) == "" || strings.TrimSpace(rec[1]) == "" {
				continue
			}
			sid, err := uuid.Parse(strings.TrimSpace(rec[0]))
			if err != nil {
				continue
			}
			parentEmail := strings.ToLower(strings.TrimSpace(rec[1]))
			var consentDate time.Time
			if len(rec) >= 3 && strings.TrimSpace(rec[2]) != "" {
				consentDate, _ = time.Parse("2006-01-02", strings.TrimSpace(rec[2]))
			}
			method := coppaservice.ConsentMethodSchoolAuthorization
			if len(rec) >= 4 {
				switch strings.ToLower(strings.TrimSpace(rec[3])) {
				case "email_signed":
					method = coppaservice.ConsentMethodEmailSigned
				case "upload":
					method = coppaservice.ConsentMethodUpload
				case "direct":
					method = coppaservice.ConsentMethodDirect
				}
			}
			rows = append(rows, coppaservice.BulkImportRow{
				StudentID:     sid,
				ParentEmail:   parentEmail,
				ConsentDate:   consentDate,
				ConsentMethod: method,
			})
		}

		result := coppaservice.BulkSchoolAuthorization(r.Context(), d.Pool, orgID, rows)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"imported": result.Imported,
			"skipped":  result.Skipped,
			"errors":   result.Errors,
		})
	}
}

// handleCoppaInitiateConsent is POST /api/v1/compliance/coppa/initiate
// Used internally or by registration to trigger the consent email for a minor.
func (d Deps) handleCoppaInitiateConsent() http.HandlerFunc {
	type req struct {
		StudentID   string `json:"studentId"`
		ParentEmail string `json:"parentEmail"`
	}
	type resp struct {
		ConsentID string `json:"consentId"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if !d.coppaEnabled(w) {
			return
		}
		actorID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, actorID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load organization.")
			return
		}
		var body req
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		studentID, err := uuid.Parse(strings.TrimSpace(body.StudentID))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid studentId.")
			return
		}
		parentEmail := strings.ToLower(strings.TrimSpace(body.ParentEmail))
		if parentEmail == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "parentEmail is required.")
			return
		}

		// Verify student is a minor.
		status, err := coppaservice.GetUserConsentStatus(r.Context(), d.Pool, studentID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Student not found.")
			return
		}
		if !status.CoppaMinor {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Student is not flagged as a COPPA minor.")
			return
		}

		ct, err := coppaservice.InitiateEmailConsent(r.Context(), d.Pool, orgID, studentID, parentEmail)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to initiate consent.")
			return
		}

		// Look up student name for email.
		studentRow, _ := user.FindByID(r.Context(), d.Pool, studentID)
		studentName := ""
		if studentRow != nil && studentRow.DisplayName != nil {
			studentName = *studentRow.DisplayName
		}
		if studentName == "" && studentRow != nil {
			studentName = studentRow.Email
		}

		consentURL := fmt.Sprintf("%s/coppa/consent?token=%s", strings.TrimRight(d.effectiveConfig().PublicWebOrigin, "/"), ct.RawToken)
		go func() {
			_ = mail.SendCoppaConsentNotice(d.effectiveConfig(), parentEmail, studentName, consentURL, nil)
		}()

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp{ConsentID: ct.ConsentID.String()})
	}
}

func (d Deps) registerCoppaRoutes(r chi.Router) {
	r.Get("/api/v1/compliance/coppa/status", d.handleCoppaStatus())
	r.Post("/api/v1/compliance/coppa/consent-token", d.handleCoppaConsentToken())
	r.Get("/api/v1/compliance/coppa/parent-dashboard", d.handleCoppaParentDashboard())
	r.Delete("/api/v1/compliance/coppa/consent/{id}", d.handleCoppaConsentRevoke())
	r.Patch("/api/v1/compliance/coppa/ai-opt-in", d.handleCoppaAIOptIn())
	r.Post("/api/v1/compliance/coppa/initiate", d.handleCoppaInitiateConsent())
	r.Post("/api/v1/compliance/coppa/bulk-import/{orgId}", d.handleCoppaBulkImport())
}
