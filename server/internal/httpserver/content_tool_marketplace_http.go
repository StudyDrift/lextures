package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/toolmarket"
	tmsvc "github.com/lextures/lextures/server/internal/service/toolmarket"
)

func (d Deps) contentToolMarketplaceEnabled(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFContentToolMarketplace {
		apierr.WriteJSON(w, http.StatusNotImplemented, apierr.CodeNotImplemented, "Content tool marketplace is not enabled on this server.")
		return false
	}
	return true
}

func (d Deps) toolMarketService() tmsvc.Service {
	return tmsvc.Service{Pool: d.Pool, Cfg: d.effectiveConfig()}
}

func (d Deps) registerContentToolMarketplaceRoutes(r chi.Router) {
	// Developer portal
	r.Get("/api/v1/developer/tools", d.handleDeveloperListTools())
	r.Post("/api/v1/developer/tools", d.handleDeveloperCreateTool())
	r.Post("/api/v1/developer/tools/{tool_id}/releases", d.handleDeveloperCreateRelease())
	r.Post("/api/v1/developer/tools/{tool_id}/releases/{version}/submit", d.handleDeveloperSubmitRelease())
	r.Get("/api/v1/developer/tools/{tool_id}/analytics", d.handleDeveloperToolAnalytics())
	r.Post("/api/v1/developer/tools/{tool_id}/access-grants", d.handleDeveloperGrantAccess())
	r.Post("/api/v1/developer/tools/{tool_id}/sunset", d.handleDeveloperSunset())

	// Marketplace browse
	r.Get("/api/v1/tool-marketplace/tools", d.handleToolMarketplaceBrowse())
	r.Get("/api/v1/tool-marketplace/tools/{tool_id}", d.handleToolMarketplaceDetail())

	// Org installations
	r.Get("/api/v1/orgs/{orgId}/tool-installations", d.handleOrgToolInstallationsList())
	r.Get("/api/v1/orgs/{orgId}/tool-installations/preview", d.handleOrgToolInstallPreview())
	r.Post("/api/v1/orgs/{orgId}/tool-installations", d.handleOrgToolInstall())
	r.Patch("/api/v1/orgs/{orgId}/tool-installations/{id}", d.handleOrgToolInstallPatch())
	r.Delete("/api/v1/orgs/{orgId}/tool-installations/{id}", d.handleOrgToolInstallRevoke())

	// Platform review queue
	r.Get("/api/v1/admin/tool-reviews", d.handleAdminToolReviewsList())
	r.Post("/api/v1/admin/tool-reviews/{release_id}/decision", d.handleAdminToolReviewDecision())
	r.Post("/api/v1/admin/tool-reviews/auto-updates", d.handleAdminToolAutoUpdates())
}

func toolToJSON(t toolmarket.Tool) map[string]any {
	out := map[string]any{
		"id":            t.ID.String(),
		"toolId":        t.ToolID,
		"displayName":   t.DisplayName,
		"summary":       t.Summary,
		"descriptionMd": t.DescriptionMD,
		"subjectTags":   t.SubjectTags,
		"gradeTags":     t.GradeTags,
		"visibility":    t.Visibility,
		"pricingModel":  t.PricingModel,
		"status":        t.Status,
		"createdAt":     t.CreatedAt.UTC().Format(time.RFC3339),
		"updatedAt":     t.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if t.SupportURL != nil {
		out["supportUrl"] = *t.SupportURL
	}
	if t.PrivacyURL != nil {
		out["privacyUrl"] = *t.PrivacyURL
	}
	return out
}

func releaseToJSON(r *toolmarket.Release) map[string]any {
	out := map[string]any{
		"id":           r.ID.String(),
		"toolPk":       r.ToolPK.String(),
		"version":      r.Version,
		"manifest":     json.RawMessage(r.ManifestJSON),
		"dataSheet":    json.RawMessage(r.DataSheetJSON),
		"bundleSri":    r.BundleSRI,
		"bundleBytes":  r.BundleBytes,
		"checks":       json.RawMessage(r.ChecksJSON),
		"reviewStatus": r.ReviewStatus,
		"createdAt":    r.CreatedAt.UTC().Format(time.RFC3339),
	}
	if r.ReviewNotes != nil {
		out["reviewNotes"] = *r.ReviewNotes
	}
	if r.PublishedAt != nil {
		out["publishedAt"] = r.PublishedAt.UTC().Format(time.RFC3339)
	}
	if r.SunsetAt != nil {
		out["sunsetAt"] = r.SunsetAt.UTC().Format(time.RFC3339)
	}
	if r.SoakUntil != nil {
		out["soakUntil"] = r.SoakUntil.UTC().Format(time.RFC3339)
	}
	return out
}

func installationToToolJSON(i toolmarket.Installation) map[string]any {
	out := map[string]any{
		"id":                    i.ID.String(),
		"orgId":                 i.OrgID.String(),
		"toolId":                i.ToolID,
		"displayName":           i.DisplayName,
		"pinnedMajor":           i.PinnedMajor,
		"currentVersion":        i.CurrentVersion,
		"consentedCapabilities": i.ConsentedCapabilities,
		"consentedHosts":        i.ConsentedHosts,
		"autoUpdateMinor":       i.AutoUpdateMinor,
		"status":                i.Status,
		"installedAt":           i.InstalledAt.UTC().Format(time.RFC3339),
	}
	if i.InstalledBy != nil {
		out["installedBy"] = i.InstalledBy.String()
	}
	if i.RevokedAt != nil {
		out["revokedAt"] = i.RevokedAt.UTC().Format(time.RFC3339)
	}
	return out
}
