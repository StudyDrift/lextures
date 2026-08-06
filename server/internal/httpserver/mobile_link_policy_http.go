package httpserver

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/mobilelinkpolicy"
)

type mobileLinkHandlingJSON struct {
	MobileLinkHandling string `json:"mobileLinkHandling"`
	// When true, the org has an explicit override; when false, platform default applies.
	OrgOverride bool `json:"orgOverride"`
}

// handleGetOrgMobileLinkHandling is GET /api/v1/orgs/{orgId}/settings/mobile-link-handling
func (d Deps) handleGetOrgMobileLinkHandling() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID, ok := d.parseOrgID(w, r)
		if !ok {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		handling, browserOn, err := mobilelinkpolicy.Effective(r.Context(), d.Pool, &orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load mobile link policy.")
			return
		}
		_, hasOverride, _ := mobilelinkpolicy.GetOrgOverride(r.Context(), d.Pool, orgID)
		_ = browserOn
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(mobileLinkHandlingJSON{
			MobileLinkHandling: string(handling),
			OrgOverride:        hasOverride,
		})
	}
}

type putOrgMobileLinkBody struct {
	MobileLinkHandling *string `json:"mobileLinkHandling"`
	// When true, clears the org override so the platform default applies.
	ClearOverride bool `json:"clearOverride"`
}

// handlePutOrgMobileLinkHandling is PUT /api/v1/orgs/{orgId}/settings/mobile-link-handling
func (d Deps) handlePutOrgMobileLinkHandling() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TD.5 FR-6: registered for both PUT and PATCH — dispatch is load-bearing.
		if r.Method != http.MethodPut && r.Method != http.MethodPatch {
			w.Header().Set("Allow", "PUT, PATCH")
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		orgID, ok := d.parseOrgID(w, r)
		if !ok {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Database is not configured.")
			return
		}
		b, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		var body putOrgMobileLinkBody
		if err := json.Unmarshal(b, &body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if body.ClearOverride {
			if err := mobilelinkpolicy.ClearOrgOverride(r.Context(), d.Pool, orgID); err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to clear org mobile link policy.")
				return
			}
		} else if body.MobileLinkHandling != nil {
			s := strings.TrimSpace(*body.MobileLinkHandling)
			if !mobilelinkpolicy.IsValid(s) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "mobileLinkHandling must be in_app, system, or blocked.")
				return
			}
			if err := mobilelinkpolicy.SetOrgOverride(r.Context(), d.Pool, orgID, s); err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save org mobile link policy.")
				return
			}
		}
		handling, _, err := mobilelinkpolicy.Effective(r.Context(), d.Pool, &orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load mobile link policy.")
			return
		}
		_, hasOverride, _ := mobilelinkpolicy.GetOrgOverride(r.Context(), d.Pool, orgID)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(mobileLinkHandlingJSON{
			MobileLinkHandling: string(handling),
			OrgOverride:        hasOverride,
		})
	}
}

func (d Deps) parseOrgID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	raw := strings.TrimSpace(chi.URLParam(r, "orgId"))
	if raw == "" {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "orgId is required.")
		return uuid.Nil, false
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid orgId.")
		return uuid.Nil, false
	}
	return id, true
}
