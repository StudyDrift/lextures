package httpserver

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/orgroles"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/toolmarket"
	"github.com/lextures/lextures/server/internal/service/adminaudit"
	tmsvc "github.com/lextures/lextures/server/internal/service/toolmarket"
)

func (d Deps) handleToolMarketplaceBrowse() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.contentToolMarketplaceEnabled(w) {
			return
		}
		var orgPtr *uuid.UUID
		if userID, ok := d.meSessionUserIDOptional(r); ok {
			if orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID); err == nil && orgID != uuid.Nil {
				orgPtr = &orgID
			}
		}
		listings, err := toolmarket.BrowseListings(r.Context(), d.Pool, toolmarket.BrowseFilters{
			Subject: r.URL.Query().Get("subject"),
			Grade:   r.URL.Query().Get("grade"),
			Query:   r.URL.Query().Get("q"),
			OrgID:   orgPtr,
			Limit:   50,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to browse marketplace.")
			return
		}
		out := make([]map[string]any, 0, len(listings))
		for _, l := range listings {
			item := map[string]any{
				"toolId":       l.ToolID,
				"displayName":  l.DisplayName,
				"summary":      l.Summary,
				"subjectTags":  l.SubjectTags,
				"gradeTags":    l.GradeTags,
				"visibility":   l.Visibility,
				"pricingModel": l.PricingModel,
				"status":       l.Status,
				"version":      l.Version,
				"wcagLevel":    l.WCAGLevel,
				"capabilities": l.Capabilities,
			}
			if l.SunsetAt != nil {
				item["sunsetAt"] = l.SunsetAt.UTC().Format(time.RFC3339)
			}
			out = append(out, item)
		}
		writeJSON(w, http.StatusOK, map[string]any{"tools": out})
	}
}

func (d Deps) meSessionUserIDOptional(r *http.Request) (uuid.UUID, bool) {
	// Soft auth for public browse — ignore failures.
	h := r.Header.Get("Authorization")
	if h == "" || d.JWTSigner == nil {
		return uuid.Nil, false
	}
	w := &discardWriter{}
	id, ok := d.meSessionUserID(w, r)
	return id, ok
}

type discardWriter struct {
	h http.Header
}

func (d *discardWriter) Header() http.Header {
	if d.h == nil {
		d.h = make(http.Header)
	}
	return d.h
}
func (d *discardWriter) Write(b []byte) (int, error) { return len(b), nil }
func (d *discardWriter) WriteHeader(int)             {}

func (d Deps) handleToolMarketplaceDetail() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.contentToolMarketplaceEnabled(w) {
			return
		}
		var orgPtr *uuid.UUID
		if userID, ok := d.meSessionUserIDOptional(r); ok {
			if orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, userID); err == nil && orgID != uuid.Nil {
				orgPtr = &orgID
			}
		}
		toolID := chi.URLParam(r, "tool_id")
		listing, tool, rel, err := toolmarket.GetPublicListing(r.Context(), d.Pool, toolID, orgPtr)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load listing.")
			return
		}
		if listing == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Tool not found.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"tool":      toolToJSON(*tool),
			"listing":   listing,
			"release":   releaseToJSON(rel),
			"dataSheet": json.RawMessage(rel.DataSheetJSON),
		})
	}
}

func (d Deps) requireOrgToolInstallAccess(w http.ResponseWriter, r *http.Request) (actor, orgID uuid.UUID, ok bool) {
	actor, ok = d.meUserID(w, r)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	parsed, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "orgId")))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid orgId.")
		return uuid.Nil, uuid.Nil, false
	}
	ga, err := rbac.UserHasPermission(r.Context(), d.Pool, actor, permGlobalRBACManage)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return uuid.Nil, uuid.Nil, false
	}
	if !ga {
		uOrg, err := organization.OrgIDForUser(r.Context(), d.Pool, actor)
		if err != nil || uOrg != parsed {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return uuid.Nil, uuid.Nil, false
		}
		has, err := orgroles.UserHasRole(r.Context(), d.Pool, actor, parsed, orgroles.RoleOrgAdmin)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
			return uuid.Nil, uuid.Nil, false
		}
		if !has {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
			return uuid.Nil, uuid.Nil, false
		}
	}
	return actor, parsed, true
}

func (d Deps) handleOrgToolInstallationsList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.contentToolMarketplaceEnabled(w) {
			return
		}
		_, orgID, ok := d.requireOrgToolInstallAccess(w, r)
		if !ok {
			return
		}
		list, err := toolmarket.ListInstallationsByOrg(r.Context(), d.Pool, orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list installations.")
			return
		}
		out := make([]map[string]any, 0, len(list))
		for _, i := range list {
			out = append(out, installationToToolJSON(i))
		}
		writeJSON(w, http.StatusOK, map[string]any{"installations": out})
	}
}

func (d Deps) handleOrgToolInstallPreview() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.contentToolMarketplaceEnabled(w) {
			return
		}
		_, orgID, ok := d.requireOrgToolInstallAccess(w, r)
		if !ok {
			return
		}
		toolID := strings.TrimSpace(r.URL.Query().Get("toolId"))
		if toolID == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "toolId is required.")
			return
		}
		preview, err := d.toolMarketService().InstallPreview(r.Context(), toolID, orgID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Tool not found.")
			return
		}
		writeJSON(w, http.StatusOK, preview)
	}
}

type installBody struct {
	ToolID          string `json:"toolId"`
	Consented       bool   `json:"consented"`
	AutoUpdateMinor *bool  `json:"autoUpdateMinor"`
}

func (d Deps) handleOrgToolInstall() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.contentToolMarketplaceEnabled(w) {
			return
		}
		actor, orgID, ok := d.requireOrgToolInstallAccess(w, r)
		if !ok {
			return
		}
		var body installBody
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		ins, err := d.toolMarketService().Install(r.Context(), tmsvc.InstallInput{
			OrgID:           orgID,
			ToolID:          body.ToolID,
			AdminUserID:     actor,
			AutoUpdateMinor: body.AutoUpdateMinor,
			Consented:       body.Consented,
		})
		if err != nil {
			msg := err.Error()
			if msg == "not found" {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Tool not found.")
				return
			}
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, msg)
			return
		}
		after, _ := json.Marshal(map[string]any{
			"toolId": body.ToolID, "action": "install", "installationId": ins.ID.String(),
			"consentedCapabilities": ins.ConsentedCapabilities, "consentedHosts": ins.ConsentedHosts,
		})
		_, _ = adminaudit.Record(r.Context(), d.Pool, adminaudit.RecordParams{
			OrgID:      &orgID,
			EventType:  adminaudit.EventContentToolMarketplace,
			ActorID:    actor,
			AfterValue: after,
		})
		writeJSON(w, http.StatusCreated, installationToToolJSON(*ins))
	}
}

type patchInstallBody struct {
	AutoUpdateMinor *bool  `json:"autoUpdateMinor"`
	ConsentMajorTo  string `json:"consentMajorTo"`
	Consented       bool   `json:"consented"`
}

func (d Deps) handleOrgToolInstallPatch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.contentToolMarketplaceEnabled(w) {
			return
		}
		actor, orgID, ok := d.requireOrgToolInstallAccess(w, r)
		if !ok {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid installation id.")
			return
		}
		var body patchInstallBody
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if body.ConsentMajorTo != "" {
			ins, err := d.toolMarketService().ConsentMajorUpdate(r.Context(), id, orgID, actor, body.ConsentMajorTo, body.Consented)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, installationToToolJSON(*ins))
			return
		}
		if body.AutoUpdateMinor != nil {
			ins, err := toolmarket.PatchInstallation(r.Context(), d.Pool, id, toolmarket.PatchInstallationParams{
				AutoUpdateMinor: body.AutoUpdateMinor,
			})
			if err != nil || ins == nil {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Installation not found.")
				return
			}
			if ins.OrgID != orgID {
				apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Forbidden.")
				return
			}
			writeJSON(w, http.StatusOK, installationToToolJSON(*ins))
			return
		}
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "No patch fields provided.")
	}
}

func (d Deps) handleOrgToolInstallRevoke() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.contentToolMarketplaceEnabled(w) {
			return
		}
		actor, orgID, ok := d.requireOrgToolInstallAccess(w, r)
		if !ok {
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid installation id.")
			return
		}
		ins, err := d.toolMarketService().Revoke(r.Context(), id, orgID, actor)
		if err != nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Installation not found.")
			return
		}
		after, _ := json.Marshal(map[string]any{"toolId": ins.ToolID, "action": "revoke", "installationId": id.String()})
		_, _ = adminaudit.Record(r.Context(), d.Pool, adminaudit.RecordParams{
			OrgID:      &orgID,
			EventType:  adminaudit.EventContentToolMarketplace,
			ActorID:    actor,
			AfterValue: after,
		})
		writeJSON(w, http.StatusOK, installationToToolJSON(*ins))
	}
}

func (d Deps) handleAdminToolReviewsList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.contentToolMarketplaceEnabled(w) {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		status := r.URL.Query().Get("status")
		list, err := toolmarket.ListPendingReviews(r.Context(), d.Pool, status, 100)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list reviews.")
			return
		}
		out := make([]map[string]any, 0, len(list))
		for i := range list {
			item := releaseToJSON(&list[i])
			if tool, _ := toolmarket.GetToolByPK(r.Context(), d.Pool, list[i].ToolPK); tool != nil {
				item["toolId"] = tool.ToolID
				item["displayName"] = tool.DisplayName
				item["visibility"] = tool.Visibility
			}
			out = append(out, item)
		}
		writeJSON(w, http.StatusOK, map[string]any{"reviews": out})
	}
}

type reviewDecisionBody struct {
	Approve bool   `json:"approve"`
	Notes   string `json:"notes"`
}

func (d Deps) handleAdminToolReviewDecision() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.contentToolMarketplaceEnabled(w) {
			return
		}
		reviewer, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		releaseID, err := uuid.Parse(chi.URLParam(r, "release_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid release id.")
			return
		}
		var body reviewDecisionBody
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		rel, err := d.toolMarketService().DecideReview(r.Context(), tmsvc.ReviewDecisionInput{
			ReleaseID:  releaseID,
			ReviewerID: reviewer,
			Approve:    body.Approve,
			Notes:      body.Notes,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		action := "reject"
		if body.Approve {
			action = "approve"
		}
		after, _ := json.Marshal(map[string]any{"releaseId": releaseID.String(), "action": action, "notes": body.Notes})
		_, _ = adminaudit.Record(r.Context(), d.Pool, adminaudit.RecordParams{
			EventType:  adminaudit.EventContentToolMarketplace,
			ActorID:    reviewer,
			AfterValue: after,
		})
		writeJSON(w, http.StatusOK, releaseToJSON(rel))
	}
}

func (d Deps) handleAdminToolAutoUpdates() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.contentToolMarketplaceEnabled(w) {
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		n, err := d.toolMarketService().ApplyAutoUpdates(r.Context(), time.Now().UTC())
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to apply auto-updates.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"updated": n})
	}
}
