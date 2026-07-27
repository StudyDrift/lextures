package toolmarket

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/repos/toolmarket"
	ctsvc "github.com/lextures/lextures/server/internal/service/contenttools"
	webhooksvc "github.com/lextures/lextures/server/internal/service/webhooks"
	"github.com/lextures/lextures/server/internal/webhooks"
)

// ConsentCapability is plain-language consent item.
type ConsentCapability struct {
	Capability string `json:"capability"`
	Plain      string `json:"plainLanguage"`
}

// InstallPreview builds the admin consent screen payload (FR-5).
func (s Service) InstallPreview(ctx context.Context, toolID string, orgID uuid.UUID) (map[string]any, error) {
	listing, tool, rel, err := toolmarket.GetPublicListing(ctx, s.Pool, toolID, &orgID)
	if err != nil {
		return nil, err
	}
	if listing == nil || tool == nil || rel == nil {
		return nil, fmt.Errorf("not found")
	}
	caps := ExtractCapabilities(rel.ManifestJSON)
	hosts := ExtractHosts(rel.ManifestJSON)
	items := make([]ConsentCapability, 0, len(caps))
	for _, c := range caps {
		items = append(items, ConsentCapability{Capability: c, Plain: CapabilityPlainLanguage(c)})
	}
	return map[string]any{
		"toolId":       tool.ToolID,
		"displayName":  tool.DisplayName,
		"version":      rel.Version,
		"capabilities": items,
		"hosts":        hosts,
		"dataSheet":    json.RawMessage(rel.DataSheetJSON),
		"pricingModel": tool.PricingModel,
		"bundleSri":    rel.BundleSRI,
	}, nil
}

// InstallInput installs with admin consent.
type InstallInput struct {
	OrgID           uuid.UUID
	ToolID          string
	AdminUserID     uuid.UUID
	AutoUpdateMinor *bool
	Consented       bool
}

// Install records org installation (FR-5 / FR-6 / AC-2).
func (s Service) Install(ctx context.Context, in InstallInput) (*toolmarket.Installation, error) {
	if !in.Consented {
		return nil, fmt.Errorf("admin consent required")
	}
	listing, tool, rel, err := toolmarket.GetPublicListing(ctx, s.Pool, in.ToolID, &in.OrgID)
	if err != nil {
		return nil, err
	}
	if listing == nil || tool == nil || rel == nil {
		// AC-9: unlisted without grant → 404
		return nil, fmt.Errorf("not found")
	}
	existing, err := toolmarket.GetInstallationByOrgTool(ctx, s.Pool, in.OrgID, tool.ID)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.Status == toolmarket.InstallActive {
		return nil, fmt.Errorf("already installed")
	}
	sv, err := ctsvc.ParseSemVer(rel.Version)
	if err != nil {
		return nil, err
	}
	auto := true
	if in.AutoUpdateMinor != nil {
		auto = *in.AutoUpdateMinor
	}
	caps := ExtractCapabilities(rel.ManifestJSON)
	hosts := ExtractHosts(rel.ManifestJSON)
	ins, err := toolmarket.CreateInstallation(ctx, s.Pool, toolmarket.CreateInstallationParams{
		OrgID:                 in.OrgID,
		ToolPK:                tool.ID,
		PinnedMajor:           sv.Major,
		CurrentVersion:        rel.Version,
		ConsentedCapabilities: caps,
		ConsentedHosts:        hosts,
		AutoUpdateMinor:       auto,
		InstalledBy:           in.AdminUserID,
	})
	if err != nil {
		// Re-activate previously revoked install.
		if existing != nil {
			status := toolmarket.InstallActive
			patched, perr := toolmarket.PatchInstallation(ctx, s.Pool, existing.ID, toolmarket.PatchInstallationParams{
				CurrentVersion:        &rel.Version,
				PinnedMajor:           &sv.Major,
				ConsentedCapabilities: caps,
				ConsentedHosts:        hosts,
				AutoUpdateMinor:       &auto,
				Status:                &status,
			})
			if perr != nil {
				return nil, perr
			}
			ins = patched
		} else {
			return nil, err
		}
	}
	ins.ToolID = tool.ToolID
	ins.DisplayName = tool.DisplayName
	IncInstall(tool.ToolID, "install")
	details, _ := json.Marshal(map[string]any{
		"version":      rel.Version,
		"capabilities": caps,
		"hosts":        hosts,
		"installed_by": in.AdminUserID.String(),
	})
	_ = toolmarket.RecordLifecycle(ctx, s.Pool, &tool.ID, &rel.ID, &in.OrgID, "tool.installed", &in.AdminUserID, details)
	webhooksvc.EmitAsync(s.Pool, s.Cfg, in.OrgID, webhooks.EventToolInstalled, map[string]any{
		"tool_id":         tool.ToolID,
		"version":         rel.Version,
		"installation_id": ins.ID.String(),
		"installed_by":    in.AdminUserID.String(),
	})
	return ins, nil
}

// ConsentMajorUpdate re-consents an org onto a new major (AC-6).
func (s Service) ConsentMajorUpdate(ctx context.Context, installationID, orgID, adminID uuid.UUID, version string, consented bool) (*toolmarket.Installation, error) {
	if !consented {
		return nil, fmt.Errorf("admin consent required")
	}
	ins, err := toolmarket.GetInstallation(ctx, s.Pool, installationID)
	if err != nil || ins == nil {
		if err == nil {
			err = fmt.Errorf("not found")
		}
		return nil, err
	}
	if ins.OrgID != orgID {
		return nil, fmt.Errorf("forbidden")
	}
	rel, err := toolmarket.GetReleaseByVersion(ctx, s.Pool, ins.ToolPK, version)
	if err != nil || rel == nil || rel.ReviewStatus != toolmarket.ReviewApproved {
		return nil, fmt.Errorf("release not found")
	}
	sv, err := ctsvc.ParseSemVer(version)
	if err != nil {
		return nil, err
	}
	// Majors never auto-migrate; require explicit pin change.
	caps := ExtractCapabilities(rel.ManifestJSON)
	hosts := ExtractHosts(rel.ManifestJSON)
	// FR-6: consented hosts frozen — major re-consent may replace the set.
	updated, err := toolmarket.PatchInstallation(ctx, s.Pool, installationID, toolmarket.PatchInstallationParams{
		CurrentVersion:        &version,
		PinnedMajor:           &sv.Major,
		ConsentedCapabilities: caps,
		ConsentedHosts:        hosts,
	})
	if err != nil {
		return nil, err
	}
	IncInstall(ins.ToolID, "major_update")
	details, _ := json.Marshal(map[string]any{"version": version})
	_ = toolmarket.RecordLifecycle(ctx, s.Pool, &ins.ToolPK, &rel.ID, &orgID, "tool.updated", &adminID, details)
	webhooksvc.EmitAsync(s.Pool, s.Cfg, orgID, webhooks.EventToolUpdated, map[string]any{
		"tool_id":         ins.ToolID,
		"version":         version,
		"installation_id": installationID.String(),
	})
	return updated, nil
}

// Revoke immediately revokes org-wide (FR-9 / AC-7).
func (s Service) Revoke(ctx context.Context, installationID, orgID, adminID uuid.UUID) (*toolmarket.Installation, error) {
	ins, err := toolmarket.GetInstallation(ctx, s.Pool, installationID)
	if err != nil || ins == nil {
		if err == nil {
			err = fmt.Errorf("not found")
		}
		return nil, err
	}
	if ins.OrgID != orgID {
		return nil, fmt.Errorf("forbidden")
	}
	updated, err := toolmarket.RevokeInstallation(ctx, s.Pool, installationID)
	if err != nil {
		return nil, err
	}
	IncInstall(ins.ToolID, "revoke")
	_ = toolmarket.RecordLifecycle(ctx, s.Pool, &ins.ToolPK, nil, &orgID, "tool.revoked", &adminID, nil)
	webhooksvc.EmitAsync(s.Pool, s.Cfg, orgID, webhooks.EventToolRevoked, map[string]any{
		"tool_id":         ins.ToolID,
		"installation_id": installationID.String(),
	})
	return updated, nil
}

// AnnounceSunset sets sunset with >= 90 day notice (FR-13 / AC-11).
func (s Service) AnnounceSunset(ctx context.Context, toolID string, ownerUserID uuid.UUID, sunsetAt time.Time) error {
	tool, err := toolmarket.GetToolByToolID(ctx, s.Pool, toolID)
	if err != nil || tool == nil {
		if err == nil {
			err = fmt.Errorf("tool not found")
		}
		return err
	}
	if tool.OwnerUserID == nil || *tool.OwnerUserID != ownerUserID {
		return fmt.Errorf("forbidden")
	}
	min := time.Now().UTC().Add(time.Duration(toolmarket.DefaultSunsetDays) * 24 * time.Hour)
	if sunsetAt.Before(min) {
		return fmt.Errorf("sunset notice must be at least %d days", toolmarket.DefaultSunsetDays)
	}
	rels, err := toolmarket.ListReleases(ctx, s.Pool, tool.ID)
	if err != nil {
		return err
	}
	for _, r := range rels {
		if r.ReviewStatus == toolmarket.ReviewApproved {
			_ = toolmarket.SetReleaseSunset(ctx, s.Pool, r.ID, sunsetAt)
		}
	}
	_ = toolmarket.UpdateToolStatus(ctx, s.Pool, tool.ID, toolmarket.StatusSunset)
	details, _ := json.Marshal(map[string]any{"sunset_at": sunsetAt.UTC().Format(time.RFC3339)})
	_ = toolmarket.RecordLifecycle(ctx, s.Pool, &tool.ID, nil, nil, "tool.sunset", &ownerUserID, details)
	installs, _ := toolmarket.ListActiveInstallationsForTool(ctx, s.Pool, tool.ID)
	for _, ins := range installs {
		webhooksvc.EmitAsync(s.Pool, s.Cfg, ins.OrgID, webhooks.EventToolSunset, map[string]any{
			"tool_id":   tool.ToolID,
			"sunset_at": sunsetAt.UTC().Format(time.RFC3339),
		})
	}
	return nil
}

// ApplyAutoUpdates moves installs to soaked minor/patch versions (AC-5).
func (s Service) ApplyAutoUpdates(ctx context.Context, now time.Time) (int, error) {
	installs, err := toolmarket.ListInstallationsDueForAutoUpdate(ctx, s.Pool, now, 200)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, ins := range installs {
		rel, err := toolmarket.LatestSoakedReleaseWithinMajor(ctx, s.Pool, ins.ToolPK, ins.PinnedMajor, now)
		if err != nil || rel == nil {
			continue
		}
		if rel.Version == ins.CurrentVersion {
			continue
		}
		// Never cross major via auto-update.
		if !ctsvc.SameMajor(ins.CurrentVersion, rel.Version) {
			continue
		}
		ver := rel.Version
		_, err = toolmarket.PatchInstallation(ctx, s.Pool, ins.ID, toolmarket.PatchInstallationParams{
			CurrentVersion: &ver,
		})
		if err != nil {
			continue
		}
		n++
		IncInstall(ins.ToolID, "auto_update")
		webhooksvc.EmitAsync(s.Pool, s.Cfg, ins.OrgID, webhooks.EventToolUpdated, map[string]any{
			"tool_id":         ins.ToolID,
			"version":         ver,
			"installation_id": ins.ID.String(),
			"auto":            true,
		})
	}
	return n, nil
}

// DeveloperAnalytics is aggregate-only (FR-15 / AC-8).
type DeveloperAnalytics struct {
	ToolID          string `json:"toolId"`
	ActiveInstalls  int    `json:"activeInstalls"`
	RevokedInstalls int    `json:"revokedInstalls"`
	// UsageEvents is an aggregate counter placeholder (no student ids).
	UsageEvents int `json:"usageEvents"`
}

// AnalyticsForTool returns aggregate installs/usage with no student PII.
func (s Service) AnalyticsForTool(ctx context.Context, toolID string, ownerUserID uuid.UUID) (*DeveloperAnalytics, error) {
	tool, err := toolmarket.GetToolByToolID(ctx, s.Pool, toolID)
	if err != nil || tool == nil {
		if err == nil {
			err = fmt.Errorf("tool not found")
		}
		return nil, err
	}
	if tool.OwnerUserID == nil || *tool.OwnerUserID != ownerUserID {
		return nil, fmt.Errorf("forbidden")
	}
	active, revoked, err := toolmarket.CountInstalls(ctx, s.Pool, tool.ID)
	if err != nil {
		return nil, err
	}
	return &DeveloperAnalytics{
		ToolID:          tool.ToolID,
		ActiveInstalls:  active,
		RevokedInstalls: revoked,
		UsageEvents:     0,
	}, nil
}

// ResolveInstalledManifest returns the consented release for an org tool, or tombstone info.
type ResolvedInstall struct {
	ToolID        string
	Version       string
	ManifestJSON  json.RawMessage
	BundleSRI     string
	Hosts         []string
	Status        string // active|revoked|suspended
	Tombstone     bool
	DisplayName   string
	SandboxForced string
}

// ResolveForOrg resolves a third-party tool for runtime (catalog / render).
func (s Service) ResolveForOrg(ctx context.Context, orgID uuid.UUID, toolID string) (*ResolvedInstall, error) {
	start := time.Now()
	tool, err := toolmarket.GetToolByToolID(ctx, s.Pool, toolID)
	if err != nil || tool == nil {
		if err != nil {
			IncThirdPartyError(toolID)
		}
		return nil, err
	}
	ins, err := toolmarket.GetInstallationByOrgTool(ctx, s.Pool, orgID, tool.ID)
	if err != nil || ins == nil {
		if err != nil {
			IncThirdPartyError(toolID)
		}
		return nil, err
	}
	out := &ResolvedInstall{
		ToolID:        tool.ToolID,
		Version:       ins.CurrentVersion,
		Hosts:         append([]string{}, ins.ConsentedHosts...),
		Status:        ins.Status,
		DisplayName:   tool.DisplayName,
		SandboxForced: ctsvc.SandboxIframe,
	}
	defer func() {
		ObserveBundleLoad(toolID, time.Since(start).Seconds())
	}()
	if ins.Status != toolmarket.InstallActive {
		out.Tombstone = true
		return out, nil
	}
	rel, err := toolmarket.GetReleaseByVersion(ctx, s.Pool, tool.ID, ins.CurrentVersion)
	if err != nil || rel == nil {
		out.Tombstone = true
		return out, nil
	}
	out.ManifestJSON = rel.ManifestJSON
	out.BundleSRI = rel.BundleSRI
	return out, nil
}

// ListActiveToolIDsForOrg returns tool ids installed and active for catalog merge.
func (s Service) ListActiveToolIDsForOrg(ctx context.Context, orgID uuid.UUID) ([]toolmarket.Installation, error) {
	all, err := toolmarket.ListInstallationsByOrg(ctx, s.Pool, orgID)
	if err != nil {
		return nil, err
	}
	out := make([]toolmarket.Installation, 0, len(all))
	for _, i := range all {
		if i.Status == toolmarket.InstallActive {
			out = append(out, i)
		}
	}
	return out, nil
}

// GrantUnlistedAccess grants an org access to a private/unlisted tool.
func (s Service) GrantUnlistedAccess(ctx context.Context, toolID string, ownerUserID, orgID uuid.UUID) error {
	tool, err := toolmarket.GetToolByToolID(ctx, s.Pool, toolID)
	if err != nil || tool == nil {
		if err == nil {
			err = fmt.Errorf("tool not found")
		}
		return err
	}
	if tool.OwnerUserID == nil || *tool.OwnerUserID != ownerUserID {
		return fmt.Errorf("forbidden")
	}
	return toolmarket.GrantAccess(ctx, s.Pool, tool.ID, orgID, ownerUserID)
}
