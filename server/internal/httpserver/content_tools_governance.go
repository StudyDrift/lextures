package httpserver

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	ctmodel "github.com/lextures/lextures/server/internal/models/contenttools"
	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
	"github.com/lextures/lextures/server/internal/service/adminaudit"
	ctsvc "github.com/lextures/lextures/server/internal/service/contenttools"
)

func (d Deps) registerContentToolsGovernanceRoutes(r chi.Router) {
	r.Get("/api/v1/orgs/{orgId}/content-tool-policy", d.handleOrgContentToolPolicy())
	r.Put("/api/v1/orgs/{orgId}/content-tool-policy", d.handleOrgContentToolPolicy())
	r.Get("/api/v1/content-tools/data-sheets", d.handleContentToolsDataSheets())
	r.Get("/api/v1/content-tools/conformance", d.handleContentToolsConformance())
	r.Post("/api/v1/courses/{course_code}/content-tools/ai-consent", d.handleContentToolsAIConsent())
	r.Get("/api/v1/courses/{course_code}/content-tools/ai-consent", d.handleContentToolsAIConsentGet())
	r.Post("/api/v1/courses/{course_code}/content-tools/instances/{instance_id}/report", d.handleContentToolsReport())
	r.Post("/api/v1/courses/{course_code}/content-tools/instances/{instance_id}/moderate", d.handleContentToolsModerate())
	r.Get("/api/v1/courses/{course_code}/content-tools/instances/{instance_id}/moderation", d.handleContentToolsModerationList())
	r.Get("/api/v1/courses/{course_code}/content-tools/instances/{instance_id}/filter-flags", d.handleContentToolsFilterFlags())
	r.Post("/api/v1/admin/content-tools/kill", d.handleAdminContentToolsKill())
	r.Get("/api/v1/admin/content-tools/kills", d.handleAdminContentToolsKillsList())
}

func (d Deps) handleOrgContentToolPolicy() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgStr := strings.TrimSpace(chi.URLParam(r, "orgId"))
		orgID, err := uuid.Parse(orgStr)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid organization id.")
			return
		}
		actorID, _, ok := d.adminOrgOrUnitAccess(w, r, orgID)
		if !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeInternal, "Database not configured.")
			return
		}
		switch r.Method {
		case http.MethodGet:
			row, err := ctsvc.LoadOrgPolicy(r.Context(), d.Pool, orgID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load policy.")
				return
			}
			writeJSON(w, http.StatusOK, contentToolPolicyToAPI(row))
		case http.MethodPut:
			var body ctmodel.OrgPolicy
			if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
			for _, c := range body.DeniedCapabilities {
				if _, ok := ctsvc.NormalizePolicyCapability(c); !ok {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid deniedCapabilities value.")
					return
				}
			}
			mode := strings.TrimSpace(body.AIDisclosureMode)
			if mode == "" {
				mode = "banner"
			}
			if mode != "none" && mode != "banner" && mode != "acknowledge" {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "aiDisclosureMode must be none, banner, or acknowledge.")
				return
			}
			action := strings.TrimSpace(body.FreeTextFilterAction)
			if action == "" {
				action = "flag"
			}
			if action != "allow" && action != "flag" && action != "block" {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "freeTextFilterAction must be allow, flag, or block.")
				return
			}
			days := body.AILogRetentionDays
			if days <= 0 {
				days = 30
			}
			crisis := true
			if body.CrisisEscalationEnabled != nil {
				crisis = *body.CrisisEscalationEnabled
			}
			updatedBy := actorID
			row := ctrepo.PolicyRow{
				OrgID:                   orgID,
				DeniedCapabilities:      body.DeniedCapabilities,
				DeniedToolIDs:           body.DeniedToolIDs,
				AllowedToolIDs:          body.AllowedToolIDs,
				AIDisclosureMode:        mode,
				FreeTextFilterAction:    action,
				CrisisEscalationEnabled: crisis,
				AILogRetentionDays:      days,
				UpdatedBy:               &updatedBy,
			}
			saved, err := ctrepo.UpsertPolicy(r.Context(), d.Pool, row)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save policy.")
				return
			}
			after, _ := json.Marshal(contentToolPolicyToAPI(saved))
			_, _ = adminaudit.Record(r.Context(), d.Pool, adminaudit.RecordParams{
				OrgID:      &orgID,
				EventType:  adminaudit.EventContentToolPolicyChange,
				ActorID:    actorID,
				TargetType: ctStrPtr("organization"),
				TargetID:   &orgID,
				AfterValue: after,
			})
			writeJSON(w, http.StatusOK, contentToolPolicyToAPI(saved))
		default:
			jobsMethodNotAllowed(w, http.MethodGet+", "+http.MethodPut)
		}
	}
}

func contentToolPolicyToAPI(row *ctrepo.PolicyRow) ctmodel.OrgPolicy {
	if row == nil {
		return ctmodel.OrgPolicy{
			DeniedCapabilities:      []string{},
			DeniedToolIDs:           []string{},
			AllowedToolIDs:          []string{},
			AIDisclosureMode:        "banner",
			FreeTextFilterAction:    "flag",
			CrisisEscalationEnabled: ctBoolPtr(true),
			AILogRetentionDays:      30,
		}
	}
	return ctmodel.OrgPolicy{
		DeniedCapabilities:      nonNil(row.DeniedCapabilities),
		DeniedToolIDs:           nonNil(row.DeniedToolIDs),
		AllowedToolIDs:          nonNil(row.AllowedToolIDs),
		AIDisclosureMode:        row.AIDisclosureMode,
		FreeTextFilterAction:    row.FreeTextFilterAction,
		CrisisEscalationEnabled: ctBoolPtr(row.CrisisEscalationEnabled),
		AILogRetentionDays:      row.AILogRetentionDays,
		UpdatedAt:               &row.UpdatedAt,
	}
}

func nonNil(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}

func ctBoolPtr(v bool) *bool { return &v }

func ctStrPtr(v string) *string { return &v }
