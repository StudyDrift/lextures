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
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/repos/organization"
	reposp "github.com/lextures/lextures/server/internal/repos/stateprivacy"
	stateprivacyservice "github.com/lextures/lextures/server/internal/service/stateprivacy"
)

func (d Deps) statePrivacyEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().StatePrivacyEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "State privacy module is not enabled.")
		return false
	}
	return true
}

func (d Deps) requireStatePrivacyAdmin(w http.ResponseWriter, r *http.Request) (userID uuid.UUID, ok bool) {
	uid, ok := d.meUserID(w, r)
	if !ok {
		return uuid.UUID{}, false
	}
	isAdmin, err := stateprivacyservice.CheckAdmin(r.Context(), d.Pool, uid)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Permission check failed.")
		return uuid.UUID{}, false
	}
	if !isAdmin {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
		return uuid.UUID{}, false
	}
	return uid, true
}

func (d Deps) registerStatePrivacyRoutes(r chi.Router) {
	r.Get("/api/v1/compliance/state/disclosure/{studentId}", d.handleGetStateDisclosure())
	r.Post("/api/v1/compliance/state/deletion-request", d.handlePostStateDeletionRequest())
	r.Get("/api/v1/compliance/state/deletion-request/{id}", d.handleGetStateDeletionRequest())
	r.Patch("/api/v1/compliance/state/deletion-request/{id}", d.handlePatchStateDeletionRequest())
	r.Get("/api/v1/compliance/state/checklist", d.handleGetStateChecklist())
	r.Get("/api/v1/compliance/state/dpa-addendum/{state}", d.handleGetStateDPAAddendum())
	r.Get("/api/v1/compliance/state/prohibitions", d.handleGetStateProhibitions())
}

// GET /api/v1/compliance/state/disclosure/{studentId}
// Returns sub-processor and school-official access events for a student (CA SOPIPA § 49073.1(b)(7)).
// Gated to parent/guardian role via requireParentViewer + requireParentLink.
func (d Deps) handleGetStateDisclosure() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.statePrivacyEnabled(w) {
			return
		}
		parentID, orgID, ok := d.requireParentViewer(w, r)
		if !ok {
			return
		}
		studentID, err := uuid.Parse(chi.URLParam(r, "studentId"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid studentId.")
			return
		}
		if _, ok := d.requireParentLink(w, r, parentID, orgID, studentID); !ok {
			return
		}
		events, err := stateprivacyservice.GetParentDisclosure(r.Context(), d.Pool, studentID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load disclosure events.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"studentId": studentID.String(),
			"events":    disclosureEventsToJSON(events),
		})
	}
}

type postDeletionRequestBody struct {
	StudentID string `json:"studentId"`
}

// POST /api/v1/compliance/state/deletion-request
// Submits an IL SOPPA parent data-deletion request (105 ILCS 85/25).
func (d Deps) handlePostStateDeletionRequest() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.statePrivacyEnabled(w) {
			return
		}
		parentID, orgID, ok := d.requireParentViewer(w, r)
		if !ok {
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body postDeletionRequestBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		studentID, err := uuid.Parse(body.StudentID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid studentId.")
			return
		}
		if _, ok := d.requireParentLink(w, r, parentID, orgID, studentID); !ok {
			return
		}
		requesterEmail := statePrivacyRequesterEmail(r, d.JWTSigner)
		id, err := stateprivacyservice.SubmitDeletionRequest(r.Context(), d.Pool, orgID, studentID, &parentID, requesterEmail)
		if err != nil {
			if errors.Is(err, stateprivacyservice.ErrAlreadyExists) {
				apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "A pending deletion request already exists for this student.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not submit deletion request.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": id.String()})
	}
}

// GET /api/v1/compliance/state/deletion-request/{id}
// Returns a deletion request by ID (admin or parent who submitted it).
func (d Deps) handleGetStateDeletionRequest() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.statePrivacyEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid request id.")
			return
		}
		req, err := stateprivacyservice.GetDeletionRequest(r.Context(), d.Pool, id)
		if err != nil {
			if errors.Is(err, stateprivacyservice.ErrNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Request not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load request.")
			return
		}
		// Allow: admin, or the parent who submitted the request.
		isAdmin, _ := stateprivacyservice.CheckAdmin(r.Context(), d.Pool, userID)
		isOwner := req.RequesterID != nil && *req.RequesterID == userID
		if !isAdmin && !isOwner {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to view this request.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(deletionRequestToJSON(*req))
	}
}

type patchDeletionRequestBody struct {
	Status string  `json:"status"`
	Notes  *string `json:"notes"`
}

// PATCH /api/v1/compliance/state/deletion-request/{id}
// Admin approves or denies a deletion request.
func (d Deps) handlePatchStateDeletionRequest() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.statePrivacyEnabled(w) {
			return
		}
		adminID, ok := d.requireStatePrivacyAdmin(w, r)
		if !ok {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid request id.")
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body patchDeletionRequestBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		notes := ""
		if body.Notes != nil {
			notes = *body.Notes
		}
		switch body.Status {
		case "completed":
			if err := stateprivacyservice.ApproveDeletionRequest(r.Context(), d.Pool, id, adminID, notes); err != nil {
				if errors.Is(err, stateprivacyservice.ErrNotFound) {
					apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Request not found.")
					return
				}
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not complete request.")
				return
			}
		case "denied":
			if err := stateprivacyservice.DenyDeletionRequest(r.Context(), d.Pool, id, adminID, notes); err != nil {
				if errors.Is(err, stateprivacyservice.ErrNotFound) {
					apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Request not found.")
					return
				}
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not deny request.")
				return
			}
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "status must be completed or denied.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}
}

// GET /api/v1/compliance/state/checklist
// Returns the state compliance checklist for the admin's org.
func (d Deps) handleGetStateChecklist() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.statePrivacyEnabled(w) {
			return
		}
		adminID, ok := d.requireStatePrivacyAdmin(w, r)
		if !ok {
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, adminID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not resolve org.")
			return
		}
		items, err := stateprivacyservice.ComplianceChecklist(r.Context(), d.Pool, orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load checklist.")
			return
		}
		jurisdiction, err := stateprivacyservice.GetOrgJurisdiction(r.Context(), d.Pool, orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load jurisdiction.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jurisdiction": jurisdiction,
			"items":        items,
		})
	}
}

// GET /api/v1/compliance/state/dpa-addendum/{state}
// Returns the state-specific DPA addendum content for CA, NY, or IL.
func (d Deps) handleGetStateDPAAddendum() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.statePrivacyEnabled(w) {
			return
		}
		if _, ok := d.requireStatePrivacyAdmin(w, r); !ok {
			return
		}
		state := chi.URLParam(r, "state")
		content, err := stateprivacyservice.DPAAddendum(state)
		if err != nil {
			if errors.Is(err, stateprivacyservice.ErrInvalidJurisdiction) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid state. Must be CA, NY, or IL.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load DPA addendum.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(content)
	}
}

// GET /api/v1/compliance/state/prohibitions
// Public endpoint — returns the platform-wide prohibition attestation required by all three laws.
func (d Deps) handleGetStateProhibitions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.statePrivacyEnabled(w) {
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"prohibitions": stateprivacyservice.ProhibitionAttestation(),
		})
	}
}

func disclosureEventToJSON(e reposp.DisclosureEvent) map[string]any {
	return map[string]any{
		"id":           e.ID.String(),
		"accessor":     e.Accessor,
		"purpose":      e.Purpose,
		"dataElements": e.DataElements,
		"occurredAt":   e.OccurredAt.UTC().Format(time.RFC3339),
	}
}

func disclosureEventsToJSON(events []reposp.DisclosureEvent) []map[string]any {
	out := make([]map[string]any, 0, len(events))
	for _, e := range events {
		out = append(out, disclosureEventToJSON(e))
	}
	return out
}

func deletionRequestToJSON(r reposp.DeletionRequest) map[string]any {
	m := map[string]any{
		"id":             r.ID.String(),
		"orgId":          r.OrgID.String(),
		"studentId":      r.StudentID.String(),
		"requesterEmail": r.RequesterEmail,
		"status":         r.Status,
		"submittedAt":    r.SubmittedAt.UTC().Format(time.RFC3339),
		"dueAt":          r.DueAt.UTC().Format(time.RFC3339),
	}
	if r.RequesterID != nil {
		m["requesterId"] = r.RequesterID.String()
	}
	if r.CompletedAt != nil {
		m["completedAt"] = r.CompletedAt.UTC().Format(time.RFC3339)
	}
	if r.ActionedBy != nil {
		m["actionedBy"] = r.ActionedBy.String()
	}
	if r.ResponseNotes != nil {
		m["responseNotes"] = *r.ResponseNotes
	}
	return m
}

func statePrivacyRequesterEmail(r *http.Request, signer *auth.JWTSigner) string {
	if signer == nil {
		return ""
	}
	u, err := auth.UserFromRequest(r, signer)
	if err != nil {
		return ""
	}
	return u.Email
}
