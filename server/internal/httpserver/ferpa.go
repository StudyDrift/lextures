package httpserver

import (
	"encoding/csv"
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
	ferpaservice "github.com/lextures/lextures/server/internal/service/ferpa"
)

func (d Deps) ferpaEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FERPAWorkflowEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "FERPA workflow is not enabled.")
		return false
	}
	return true
}

// requireFERPAAdmin authenticates the caller and enforces compliance:ferpa:admin:*.
func (d Deps) requireFERPAAdmin(w http.ResponseWriter, r *http.Request) (userID uuid.UUID, orgID uuid.UUID, ok bool) {
	uid, ok := d.meUserID(w, r)
	if !ok {
		return uuid.UUID{}, uuid.UUID{}, false
	}
	oid, err := organization.OrgIDForUser(r.Context(), d.Pool, uid)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load organization.")
		return uuid.UUID{}, uuid.UUID{}, false
	}
	isAdmin, err := ferpaservice.CheckAdmin(r.Context(), d.Pool, uid)
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

func (d Deps) registerFERPARoutes(r chi.Router) {
	r.Get("/api/v1/compliance/ferpa/directory-opt-out", d.handleGetDirectoryOptOut())
	r.Put("/api/v1/compliance/ferpa/directory-opt-out", d.handlePutDirectoryOptOut())
	r.Post("/api/v1/compliance/ferpa/record-requests", d.handlePostRecordRequest())
	r.Get("/api/v1/compliance/ferpa/record-requests", d.handleListRecordRequests())
	r.Patch("/api/v1/compliance/ferpa/record-requests/{id}", d.handlePatchRecordRequest())
	r.Get("/api/v1/compliance/ferpa/disclosure-log", d.handleGetDisclosureLog())
	r.Post("/api/v1/compliance/ferpa/consent", d.handlePostConsent())
	r.Delete("/api/v1/compliance/ferpa/consent/{id}", d.handleDeleteConsent())
}

// GET /api/v1/compliance/ferpa/directory-opt-out
func (d Deps) handleGetDirectoryOptOut() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.ferpaEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		optOut, err := ferpaservice.IsDirectoryOptOut(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load opt-out flag.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]bool{"directoryOptOut": optOut})
	}
}

type putDirectoryOptOutBody struct {
	DirectoryOptOut bool `json:"directoryOptOut"`
}

// PUT /api/v1/compliance/ferpa/directory-opt-out
func (d Deps) handlePutDirectoryOptOut() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.ferpaEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body putDirectoryOptOutBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if err := ferpaservice.SetDirectoryOptOut(r.Context(), d.Pool, userID, body.DirectoryOptOut); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not update opt-out flag.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]bool{"directoryOptOut": body.DirectoryOptOut})
	}
}

type postRecordRequestBody struct {
	StudentID      string  `json:"studentId"`
	RequestType    string  `json:"requestType"`
	Notes          string  `json:"notes"`
	AmendmentField *string `json:"amendmentField"`
	AmendmentValue *string `json:"amendmentValue"`
}

// POST /api/v1/compliance/ferpa/record-requests
func (d Deps) handlePostRecordRequest() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.ferpaEnabled(w) {
			return
		}
		requesterID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, requesterID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load organization.")
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body postRecordRequestBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		validTypes := map[string]bool{"inspect": true, "amend": true, "hearing": true}
		if !validTypes[body.RequestType] {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "requestType must be inspect, amend, or hearing.")
			return
		}
		studentID, err := parseStudentID(body.StudentID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid studentId.")
			return
		}
		if body.RequestType == "amend" && (body.AmendmentField == nil || strings.TrimSpace(*body.AmendmentField) == "") {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "amendmentField is required for amend requests.")
			return
		}
		id, err := ferpaservice.SubmitRecordRequest(r.Context(), d.Pool, orgID, studentID, requesterID,
			body.RequestType, body.Notes, body.AmendmentField, body.AmendmentValue)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not submit record request.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": id.String()})
	}
}

// GET /api/v1/compliance/ferpa/record-requests
func (d Deps) handleListRecordRequests() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.ferpaEnabled(w) {
			return
		}
		_, orgID, ok := d.requireFERPAAdmin(w, r)
		if !ok {
			return
		}
		requests, err := ferpaservice.ListRecordRequests(r.Context(), d.Pool, orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load record requests.")
			return
		}
		type item struct {
			ID          string  `json:"id"`
			StudentID   string  `json:"studentId"`
			RequesterID string  `json:"requesterId"`
			RequestType string  `json:"requestType"`
			Status      string  `json:"status"`
			Notes       *string `json:"notes,omitempty"`
			ArchivePath *string `json:"archivePath,omitempty"`
			RequestedAt string  `json:"requestedAt"`
			DueAt       *string `json:"dueAt,omitempty"`
			CompletedAt *string `json:"completedAt,omitempty"`
		}
		out := make([]item, 0, len(requests))
		for _, req := range requests {
			it := item{
				ID:          req.ID.String(),
				StudentID:   req.StudentID.String(),
				RequesterID: req.RequesterID.String(),
				RequestType: req.RequestType,
				Status:      req.Status,
				Notes:       req.Notes,
				ArchivePath: req.ArchivePath,
				RequestedAt: req.RequestedAt.UTC().Format(time.RFC3339),
			}
			if req.DueAt != nil {
				s := req.DueAt.UTC().Format(time.RFC3339)
				it.DueAt = &s
			}
			if req.CompletedAt != nil {
				s := req.CompletedAt.UTC().Format(time.RFC3339)
				it.CompletedAt = &s
			}
			out = append(out, it)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"requests": out})
	}
}

type patchRecordRequestBody struct {
	Status      string  `json:"status"`
	Notes       *string `json:"notes"`
	ArchivePath *string `json:"archivePath"`
}

// PATCH /api/v1/compliance/ferpa/record-requests/{id}
func (d Deps) handlePatchRecordRequest() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.ferpaEnabled(w) {
			return
		}
		adminID, orgID, ok := d.requireFERPAAdmin(w, r)
		if !ok {
			return
		}
		requestID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid request id.")
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body patchRecordRequestBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		validStatuses := map[string]bool{"approved": true, "denied": true, "completed": true}
		if !validStatuses[body.Status] {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "status must be approved, denied, or completed.")
			return
		}
		if err := ferpaservice.UpdateRecordRequest(r.Context(), d.Pool, requestID, adminID, orgID, body.Status, body.Notes, body.ArchivePath); err != nil {
			if errors.Is(err, ferpaservice.ErrNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Record request not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not update record request.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}
}

// GET /api/v1/compliance/ferpa/disclosure-log
func (d Deps) handleGetDisclosureLog() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.ferpaEnabled(w) {
			return
		}
		_, orgID, ok := d.requireFERPAAdmin(w, r)
		if !ok {
			return
		}
		q := r.URL.Query()
		from, to := parseTimeWindow(q.Get("from"), q.Get("to"))
		format := strings.ToLower(strings.TrimSpace(q.Get("format")))
		entries, err := ferpaservice.ListDisclosures(r.Context(), d.Pool, orgID, from, to)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load disclosure log.")
			return
		}
		if format == "csv" {
			w.Header().Set("Content-Type", "text/csv; charset=utf-8")
			w.Header().Set("Content-Disposition", `attachment; filename="ferpa_disclosure_log.csv"`)
			cw := csv.NewWriter(w)
			_ = cw.Write([]string{"id", "accessor_id", "student_id", "data_type", "authority_claim", "recipient", "logged_at"})
			for _, e := range entries {
				rec := strings.Join([]string{e.ID.String(), e.AccessorID.String(), e.StudentID.String(), e.DataType, e.AuthorityClaim, strOrEmpty(e.Recipient), e.LoggedAt.UTC().Format(time.RFC3339)}, ",")
				_ = cw.Write(strings.Split(rec, ","))
			}
			cw.Flush()
			return
		}
		type entry struct {
			ID             string  `json:"id"`
			AccessorID     string  `json:"accessorId"`
			StudentID      string  `json:"studentId"`
			DataType       string  `json:"dataType"`
			AuthorityClaim string  `json:"authorityClaim"`
			Recipient      *string `json:"recipient,omitempty"`
			LoggedAt       string  `json:"loggedAt"`
		}
		out := make([]entry, 0, len(entries))
		for _, e := range entries {
			out = append(out, entry{
				ID:             e.ID.String(),
				AccessorID:     e.AccessorID.String(),
				StudentID:      e.StudentID.String(),
				DataType:       e.DataType,
				AuthorityClaim: e.AuthorityClaim,
				Recipient:      e.Recipient,
				LoggedAt:       e.LoggedAt.UTC().Format(time.RFC3339),
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"entries": out})
	}
}

type postConsentBody struct {
	StudentID   string   `json:"studentId"`
	Recipient   string   `json:"recipient"`
	Purpose     string   `json:"purpose"`
	DataFields  []string `json:"dataFields"`
	ExpiresAt   *string  `json:"expiresAt"`
}

// POST /api/v1/compliance/ferpa/consent
func (d Deps) handlePostConsent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.ferpaEnabled(w) {
			return
		}
		grantedBy, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, grantedBy)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load organization.")
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body postConsentBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if strings.TrimSpace(body.Recipient) == "" || strings.TrimSpace(body.Purpose) == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "recipient and purpose are required.")
			return
		}
		studentID, err := parseStudentID(body.StudentID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid studentId.")
			return
		}
		var expiresAt *time.Time
		if body.ExpiresAt != nil && strings.TrimSpace(*body.ExpiresAt) != "" {
			t, err := time.Parse(time.RFC3339, strings.TrimSpace(*body.ExpiresAt))
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid expiresAt; use RFC3339.")
				return
			}
			expiresAt = &t
		}
		fields := body.DataFields
		if fields == nil {
			fields = []string{}
		}
		id, err := ferpaservice.GrantConsent(r.Context(), d.Pool, orgID, studentID, grantedBy, body.Recipient, body.Purpose, fields, expiresAt)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not save consent record.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": id.String()})
	}
}

// DELETE /api/v1/compliance/ferpa/consent/{id}
func (d Deps) handleDeleteConsent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.ferpaEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		consentID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid consent id.")
			return
		}
		if err := ferpaservice.RevokeConsent(r.Context(), d.Pool, consentID, userID); err != nil {
			if errors.Is(err, ferpaservice.ErrNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Consent record not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not revoke consent.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}
}

func parseStudentID(s string) (uuid.UUID, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return uuid.UUID{}, errors.New("empty")
	}
	return uuid.Parse(s)
}

func parseTimeWindow(fromStr, toStr string) (from, to time.Time) {
	to = time.Now().UTC()
	from = to.AddDate(0, -1, 0)
	if t, err := time.Parse(time.RFC3339, strings.TrimSpace(fromStr)); err == nil {
		from = t
	}
	if t, err := time.Parse(time.RFC3339, strings.TrimSpace(toStr)); err == nil {
		to = t
	}
	return from, to
}

func strOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
