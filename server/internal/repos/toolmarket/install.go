package toolmarket

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CreateInstallationParams installs a tool for an org.
type CreateInstallationParams struct {
	OrgID                 uuid.UUID
	ToolPK                uuid.UUID
	PinnedMajor           int
	CurrentVersion        string
	ConsentedCapabilities []string
	ConsentedHosts        []string
	AutoUpdateMinor       bool
	InstalledBy           uuid.UUID
}

// CreateInstallation inserts an active installation.
func CreateInstallation(ctx context.Context, pool *pgxpool.Pool, p CreateInstallationParams) (*Installation, error) {
	if p.ConsentedCapabilities == nil {
		p.ConsentedCapabilities = []string{}
	}
	if p.ConsentedHosts == nil {
		p.ConsentedHosts = []string{}
	}
	row := pool.QueryRow(ctx, `
		INSERT INTO toolmarket.tool_installations (
			org_id, tool_pk, pinned_major, current_version, consented_capabilities,
			consented_hosts, auto_update_minor, installed_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id, org_id, tool_pk, pinned_major, current_version, consented_capabilities,
		          consented_hosts, auto_update_minor, status, installed_by, installed_at, revoked_at`,
		p.OrgID, p.ToolPK, p.PinnedMajor, p.CurrentVersion, p.ConsentedCapabilities,
		p.ConsentedHosts, p.AutoUpdateMinor, p.InstalledBy,
	)
	return scanInstallation(row)
}

// GetInstallation loads an installation by id.
func GetInstallation(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Installation, error) {
	row := pool.QueryRow(ctx, `
		SELECT i.id, i.org_id, i.tool_pk, i.pinned_major, i.current_version, i.consented_capabilities,
		       i.consented_hosts, i.auto_update_minor, i.status, i.installed_by, i.installed_at, i.revoked_at,
		       t.tool_id, t.display_name
		FROM toolmarket.tool_installations i
		JOIN toolmarket.tools t ON t.id = i.tool_pk
		WHERE i.id = $1`, id)
	ins, err := scanInstallationJoined(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return ins, err
}

// GetInstallationByOrgTool loads installation for org+tool.
func GetInstallationByOrgTool(ctx context.Context, pool *pgxpool.Pool, orgID, toolPK uuid.UUID) (*Installation, error) {
	row := pool.QueryRow(ctx, `
		SELECT i.id, i.org_id, i.tool_pk, i.pinned_major, i.current_version, i.consented_capabilities,
		       i.consented_hosts, i.auto_update_minor, i.status, i.installed_by, i.installed_at, i.revoked_at,
		       t.tool_id, t.display_name
		FROM toolmarket.tool_installations i
		JOIN toolmarket.tools t ON t.id = i.tool_pk
		WHERE i.org_id = $1 AND i.tool_pk = $2`, orgID, toolPK)
	ins, err := scanInstallationJoined(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return ins, err
}

// ListInstallationsByOrg lists installs for an org.
func ListInstallationsByOrg(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) ([]Installation, error) {
	rows, err := pool.Query(ctx, `
		SELECT i.id, i.org_id, i.tool_pk, i.pinned_major, i.current_version, i.consented_capabilities,
		       i.consented_hosts, i.auto_update_minor, i.status, i.installed_by, i.installed_at, i.revoked_at,
		       t.tool_id, t.display_name
		FROM toolmarket.tool_installations i
		JOIN toolmarket.tools t ON t.id = i.tool_pk
		WHERE i.org_id = $1
		ORDER BY i.installed_at DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Installation{}
	for rows.Next() {
		ins, err := scanInstallationJoined(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *ins)
	}
	return out, rows.Err()
}

// ListActiveInstallationsForTool lists active installs (for sunset notice / analytics).
func ListActiveInstallationsForTool(ctx context.Context, pool *pgxpool.Pool, toolPK uuid.UUID) ([]Installation, error) {
	rows, err := pool.Query(ctx, `
		SELECT i.id, i.org_id, i.tool_pk, i.pinned_major, i.current_version, i.consented_capabilities,
		       i.consented_hosts, i.auto_update_minor, i.status, i.installed_by, i.installed_at, i.revoked_at,
		       t.tool_id, t.display_name
		FROM toolmarket.tool_installations i
		JOIN toolmarket.tools t ON t.id = i.tool_pk
		WHERE i.tool_pk = $1 AND i.status = 'active'`, toolPK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Installation{}
	for rows.Next() {
		ins, err := scanInstallationJoined(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *ins)
	}
	return out, rows.Err()
}

// PatchInstallation updates consent/version/auto-update fields.
type PatchInstallationParams struct {
	CurrentVersion        *string
	PinnedMajor           *int
	ConsentedCapabilities []string
	ConsentedHosts        []string
	AutoUpdateMinor       *bool
	Status                *string
	RevokedAt             *time.Time
}

// PatchInstallation applies non-nil fields.
func PatchInstallation(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, p PatchInstallationParams) (*Installation, error) {
	ins, err := GetInstallation(ctx, pool, id)
	if err != nil || ins == nil {
		return ins, err
	}
	ver := ins.CurrentVersion
	if p.CurrentVersion != nil {
		ver = *p.CurrentVersion
	}
	major := ins.PinnedMajor
	if p.PinnedMajor != nil {
		major = *p.PinnedMajor
	}
	caps := ins.ConsentedCapabilities
	if p.ConsentedCapabilities != nil {
		caps = p.ConsentedCapabilities
	}
	hosts := ins.ConsentedHosts
	if p.ConsentedHosts != nil {
		hosts = p.ConsentedHosts
	}
	auto := ins.AutoUpdateMinor
	if p.AutoUpdateMinor != nil {
		auto = *p.AutoUpdateMinor
	}
	status := ins.Status
	if p.Status != nil {
		status = *p.Status
	}
	revoked := ins.RevokedAt
	if p.RevokedAt != nil {
		revoked = p.RevokedAt
	}
	row := pool.QueryRow(ctx, `
		UPDATE toolmarket.tool_installations
		SET current_version = $2, pinned_major = $3, consented_capabilities = $4,
		    consented_hosts = $5, auto_update_minor = $6, status = $7, revoked_at = $8
		WHERE id = $1
		RETURNING id, org_id, tool_pk, pinned_major, current_version, consented_capabilities,
		          consented_hosts, auto_update_minor, status, installed_by, installed_at, revoked_at`,
		id, ver, major, caps, hosts, auto, status, revoked,
	)
	updated, err := scanInstallation(row)
	if err != nil {
		return nil, err
	}
	updated.ToolID = ins.ToolID
	updated.DisplayName = ins.DisplayName
	return updated, nil
}

// RevokeInstallation marks an install revoked.
func RevokeInstallation(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Installation, error) {
	now := time.Now().UTC()
	status := InstallRevoked
	return PatchInstallation(ctx, pool, id, PatchInstallationParams{
		Status:    &status,
		RevokedAt: &now,
	})
}

// GrantAccess grants an org access to an unlisted/private tool.
func GrantAccess(ctx context.Context, pool *pgxpool.Pool, toolPK, orgID, grantedBy uuid.UUID) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO toolmarket.tool_access_grants (tool_pk, org_id, granted_by)
		VALUES ($1,$2,$3)
		ON CONFLICT (tool_pk, org_id) DO NOTHING`, toolPK, orgID, grantedBy)
	return err
}

// HasAccessGrant reports whether org may see/install an unlisted tool.
func HasAccessGrant(ctx context.Context, pool *pgxpool.Pool, toolPK, orgID uuid.UUID) (bool, error) {
	var n int
	err := pool.QueryRow(ctx, `
		SELECT 1 FROM toolmarket.tool_access_grants WHERE tool_pk = $1 AND org_id = $2`, toolPK, orgID).Scan(&n)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// CountInstalls returns aggregate install counts for a tool (no student data).
func CountInstalls(ctx context.Context, pool *pgxpool.Pool, toolPK uuid.UUID) (active int, revoked int, err error) {
	err = pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE status = 'active'),
			COUNT(*) FILTER (WHERE status = 'revoked')
		FROM toolmarket.tool_installations WHERE tool_pk = $1`, toolPK).Scan(&active, &revoked)
	return
}

// RecordLifecycle writes a lifecycle audit row.
func RecordLifecycle(ctx context.Context, pool *pgxpool.Pool, toolPK *uuid.UUID, releaseID *uuid.UUID, orgID *uuid.UUID, action string, actor *uuid.UUID, details json.RawMessage) error {
	if len(details) == 0 {
		details = json.RawMessage(`{}`)
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO toolmarket.tool_lifecycle_events (tool_pk, release_id, org_id, action, actor_user_id, details_json)
		VALUES ($1,$2,$3,$4,$5,$6)`, toolPK, releaseID, orgID, action, actor, details)
	return err
}

// ListInstallationsDueForAutoUpdate returns active installs that can move to a soaked minor/patch.
func ListInstallationsDueForAutoUpdate(ctx context.Context, pool *pgxpool.Pool, now time.Time, limit int) ([]Installation, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := pool.Query(ctx, `
		SELECT i.id, i.org_id, i.tool_pk, i.pinned_major, i.current_version, i.consented_capabilities,
		       i.consented_hosts, i.auto_update_minor, i.status, i.installed_by, i.installed_at, i.revoked_at,
		       t.tool_id, t.display_name
		FROM toolmarket.tool_installations i
		JOIN toolmarket.tools t ON t.id = i.tool_pk
		WHERE i.status = 'active' AND i.auto_update_minor = TRUE
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Installation{}
	for rows.Next() {
		ins, err := scanInstallationJoined(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *ins)
	}
	_ = now
	return out, rows.Err()
}

// LatestSoakedReleaseWithinMajor finds newest approved release within major whose soak window elapsed.
func LatestSoakedReleaseWithinMajor(ctx context.Context, pool *pgxpool.Pool, toolPK uuid.UUID, major int, now time.Time) (*Release, error) {
	row := pool.QueryRow(ctx, `
		SELECT id, tool_pk, version, manifest_json, data_sheet_json, bundle_object_id, bundle_sri,
		       bundle_bytes, checks_json, review_status, reviewed_by, review_notes,
		       published_at, sunset_at, soak_until, created_at
		FROM toolmarket.tool_releases
		WHERE tool_pk = $1
		  AND review_status = 'approved'
		  AND published_at IS NOT NULL
		  AND split_part(version, '.', 1)::int = $2
		  AND (soak_until IS NULL OR soak_until <= $3)
		ORDER BY published_at DESC
		LIMIT 1`, toolPK, major, now)
	r, err := scanRelease(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return r, err
}

func scanTool(row pgx.Row) (*Tool, error) {
	var t Tool
	err := row.Scan(
		&t.ID, &t.ToolID, &t.OwnerUserID, &t.OwnerOrgID, &t.DisplayName, &t.Summary, &t.DescriptionMD,
		&t.SubjectTags, &t.GradeTags, &t.SupportURL, &t.PrivacyURL, &t.Visibility, &t.PricingModel,
		&t.Status, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if t.SubjectTags == nil {
		t.SubjectTags = []string{}
	}
	if t.GradeTags == nil {
		t.GradeTags = []string{}
	}
	return &t, nil
}

func scanRelease(row pgx.Row) (*Release, error) {
	var r Release
	err := row.Scan(
		&r.ID, &r.ToolPK, &r.Version, &r.ManifestJSON, &r.DataSheetJSON, &r.BundleObjectID, &r.BundleSRI,
		&r.BundleBytes, &r.ChecksJSON, &r.ReviewStatus, &r.ReviewedBy, &r.ReviewNotes,
		&r.PublishedAt, &r.SunsetAt, &r.SoakUntil, &r.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func scanInstallation(row pgx.Row) (*Installation, error) {
	var i Installation
	err := row.Scan(
		&i.ID, &i.OrgID, &i.ToolPK, &i.PinnedMajor, &i.CurrentVersion, &i.ConsentedCapabilities,
		&i.ConsentedHosts, &i.AutoUpdateMinor, &i.Status, &i.InstalledBy, &i.InstalledAt, &i.RevokedAt,
	)
	if err != nil {
		return nil, err
	}
	if i.ConsentedCapabilities == nil {
		i.ConsentedCapabilities = []string{}
	}
	if i.ConsentedHosts == nil {
		i.ConsentedHosts = []string{}
	}
	return &i, nil
}

func scanInstallationJoined(row pgx.Row) (*Installation, error) {
	var i Installation
	err := row.Scan(
		&i.ID, &i.OrgID, &i.ToolPK, &i.PinnedMajor, &i.CurrentVersion, &i.ConsentedCapabilities,
		&i.ConsentedHosts, &i.AutoUpdateMinor, &i.Status, &i.InstalledBy, &i.InstalledAt, &i.RevokedAt,
		&i.ToolID, &i.DisplayName,
	)
	if err != nil {
		return nil, err
	}
	if i.ConsentedCapabilities == nil {
		i.ConsentedCapabilities = []string{}
	}
	if i.ConsentedHosts == nil {
		i.ConsentedHosts = []string{}
	}
	return &i, nil
}
