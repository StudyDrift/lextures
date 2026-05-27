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
	"github.com/lextures/lextures/server/internal/auth"
	repoccpa "github.com/lextures/lextures/server/internal/repos/ccpa"
	ccpaservice "github.com/lextures/lextures/server/internal/service/ccpa"
)

func (d Deps) ccpaEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().CCPAModuleEnabled {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "CCPA module is not enabled.")
		return false
	}
	return true
}

func (d Deps) requireCCPAAdmin(w http.ResponseWriter, r *http.Request) (userID uuid.UUID, ok bool) {
	uid, ok := d.meUserID(w, r)
	if !ok {
		return uuid.UUID{}, false
	}
	isAdmin, err := ccpaservice.CheckAdmin(r.Context(), d.Pool, uid)
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

func (d Deps) registerCCPARoutes(r chi.Router) {
	r.Get("/api/v1/compliance/ccpa/opt-out", d.handleGetCCPAOptOut())
	r.Post("/api/v1/compliance/ccpa/opt-out", d.handlePostCCPAOptOut())
	r.Post("/api/v1/compliance/ccpa/requests", d.handlePostCCPARequest())
	r.Get("/api/v1/compliance/ccpa/requests", d.handleGetCCPARequestList())
	r.Get("/api/v1/compliance/ccpa/requests/{id}", d.handleGetCCPARequest())
	r.Patch("/api/v1/compliance/ccpa/requests/{id}", d.handlePatchCCPARequest())
	r.Get("/api/v1/compliance/ccpa/pi-categories", d.handleGetCCPAPICategories())
}

// GET /api/v1/compliance/ccpa/opt-out
func (d Deps) handleGetCCPAOptOut() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.ccpaEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		doNotSell, limitSensitivePI, err := ccpaservice.GetOptOut(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load opt-out state.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"doNotSell":        doNotSell,
			"limitSensitivePI": limitSensitivePI,
		})
	}
}

type postCCPAOptOutBody struct {
	DoNotSell        *bool `json:"doNotSell"`
	LimitSensitivePI *bool `json:"limitSensitivePI"`
}

// POST /api/v1/compliance/ccpa/opt-out
// Also processes the Sec-GPC header as an automatic opt-out signal (CPPA § 7025 / AC-1).
func (d Deps) handlePostCCPAOptOut() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.ccpaEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}

		// Honour GPC signal (AC-1): Sec-GPC: 1 is an automatic opt-out.
		gpcOptOut := r.Header.Get("Sec-GPC") == "1"

		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body postCCPAOptOutBody
		if len(b) > 0 {
			if err := json.Unmarshal(b, &body); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
		}

		if gpcOptOut {
			if err := ccpaservice.SetDoNotSell(r.Context(), d.Pool, userID, true); err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not update opt-out.")
				return
			}
		} else if body.DoNotSell != nil {
			if err := ccpaservice.SetDoNotSell(r.Context(), d.Pool, userID, *body.DoNotSell); err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not update opt-out.")
				return
			}
		}

		if body.LimitSensitivePI != nil {
			if err := ccpaservice.SetLimitSensitivePI(r.Context(), d.Pool, userID, *body.LimitSensitivePI); err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not update limit sensitive PI flag.")
				return
			}
		}

		doNotSell, limitSensitivePI, err := ccpaservice.GetOptOut(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load updated state.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"doNotSell":        doNotSell,
			"limitSensitivePI": limitSensitivePI,
			"gpcHonoured":      gpcOptOut,
		})
	}
}

type postCCPARequestBody struct {
	RequestType string `json:"requestType"`
}

// POST /api/v1/compliance/ccpa/requests
func (d Deps) handlePostCCPARequest() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.ccpaEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body postCCPARequestBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		validTypes := map[string]bool{
			"know_categories": true, "know_specific": true, "delete": true,
			"correct": true, "limit_sensitive": true,
		}
		if !validTypes[body.RequestType] {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid requestType.")
			return
		}

		// Get email from JWT token for the requester_email field.
		requesterEmail := ccpaRequesterEmail(r, d.JWTSigner)

		id, err := ccpaservice.SubmitRequest(r.Context(), d.Pool, userID, requesterEmail, body.RequestType)
		if err != nil {
			if errors.Is(err, ccpaservice.ErrAlreadyExists) {
				apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "A request of this type is already in progress. Please wait for the current one to complete.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not submit request.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": id.String()})
	}
}

// GET /api/v1/compliance/ccpa/requests
func (d Deps) handleGetCCPARequestList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.ccpaEnabled(w) {
			return
		}
		userID, ok := d.meUserID(w, r)
		if !ok {
			return
		}

		// Admins can view the full queue via ?queue=true
		if r.URL.Query().Get("queue") == "true" {
			isAdmin, err := ccpaservice.CheckAdmin(r.Context(), d.Pool, userID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Permission check failed.")
				return
			}
			if !isAdmin {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
				return
			}
			requests, err := ccpaservice.ListPendingRequests(r.Context(), d.Pool)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load request queue.")
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(map[string]any{"requests": ccpaRequestsToJSON(requests)})
			return
		}

		requests, err := ccpaservice.ListRequestsForUser(r.Context(), d.Pool, userID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load requests.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"requests": ccpaRequestsToJSON(requests)})
	}
}

// GET /api/v1/compliance/ccpa/requests/{id}
func (d Deps) handleGetCCPARequest() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.ccpaEnabled(w) {
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
		req, err := ccpaservice.GetRequestForUser(r.Context(), d.Pool, id, userID)
		if err != nil {
			if errors.Is(err, ccpaservice.ErrNotFound) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Request not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not load request.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(ccpaRequestToJSON(*req))
	}
}

type patchCCPARequestBody struct {
	Status       string  `json:"status"`
	DenialReason *string `json:"denialReason"`
}

// PATCH /api/v1/compliance/ccpa/requests/{id}
func (d Deps) handlePatchCCPARequest() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.ccpaEnabled(w) {
			return
		}
		adminID, ok := d.requireCCPAAdmin(w, r)
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
		var body patchCCPARequestBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		switch body.Status {
		case "approved":
			if err := ccpaservice.ApproveRequest(r.Context(), d.Pool, id, adminID); err != nil {
				if errors.Is(err, ccpaservice.ErrNotFound) {
					apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Request not found.")
					return
				}
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not approve request.")
				return
			}
		case "denied":
			reason := ""
			if body.DenialReason != nil {
				reason = strings.TrimSpace(*body.DenialReason)
			}
			if err := ccpaservice.DenyRequest(r.Context(), d.Pool, id, adminID, reason); err != nil {
				if errors.Is(err, ccpaservice.ErrNotFound) {
					apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Request not found.")
					return
				}
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not deny request.")
				return
			}
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "status must be approved or denied.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}
}

// GET /api/v1/compliance/ccpa/pi-categories
// Returns the privacy notice disclosure (CPRA § 1798.100(a)) — public endpoint, no auth required.
func (d Deps) handleGetCCPAPICategories() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.ccpaEnabled(w) {
			return
		}
		cats := ccpaservice.PICategories()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"categories": cats})
	}
}

// ccpaRequestToJSON converts a CCPARequest to a JSON-friendly map.
func ccpaRequestToJSON(req repoccpa.CCPARequest) map[string]any {
	m := map[string]any{
		"id":             req.ID.String(),
		"requesterEmail": req.RequesterEmail,
		"requestType":    req.RequestType,
		"status":         req.Status,
		"requestedAt":    req.RequestedAt.UTC().Format(time.RFC3339),
		"dueAt":          req.DueAt.UTC().Format(time.RFC3339),
		"extended":       req.Extended,
	}
	if req.UserID != nil {
		m["userId"] = req.UserID.String()
	}
	if req.CompletedAt != nil {
		m["completedAt"] = req.CompletedAt.UTC().Format(time.RFC3339)
	}
	if req.ActionedBy != nil {
		m["actionedBy"] = req.ActionedBy.String()
	}
	if req.ResponsePayload != nil {
		m["responsePayload"] = *req.ResponsePayload
	}
	return m
}

// ccpaRequestsToJSON converts a slice of CCPARequest to JSON-friendly map slices.
func ccpaRequestsToJSON(requests []repoccpa.CCPARequest) []map[string]any {
	out := make([]map[string]any, 0, len(requests))
	for _, req := range requests {
		out = append(out, ccpaRequestToJSON(req))
	}
	return out
}

// ccpaRequesterEmail extracts the email from the JWT token for use as requester_email.
func ccpaRequesterEmail(r *http.Request, signer *auth.JWTSigner) string {
	if signer == nil {
		return ""
	}
	u, err := auth.UserFromRequest(r, signer)
	if err != nil {
		return ""
	}
	return u.Email
}
