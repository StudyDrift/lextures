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

// CreateToolParams creates a draft tool.
type CreateToolParams struct {
	ToolID        string
	OwnerUserID   uuid.UUID
	OwnerOrgID    *uuid.UUID
	DisplayName   string
	Summary       string
	DescriptionMD string
	SubjectTags   []string
	GradeTags     []string
	SupportURL    *string
	PrivacyURL    *string
	Visibility    string
	PricingModel  string
}

// CreateTool inserts a draft marketplace tool.
func CreateTool(ctx context.Context, pool *pgxpool.Pool, p CreateToolParams) (*Tool, error) {
	if p.SubjectTags == nil {
		p.SubjectTags = []string{}
	}
	if p.GradeTags == nil {
		p.GradeTags = []string{}
	}
	if p.Visibility == "" {
		p.Visibility = VisibilityPrivate
	}
	if p.PricingModel == "" {
		p.PricingModel = PricingFree
	}
	row := pool.QueryRow(ctx, `
		INSERT INTO toolmarket.tools (
			tool_id, owner_user_id, owner_org_id, display_name, summary, description_md,
			subject_tags, grade_tags, support_url, privacy_url, visibility, pricing_model
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING id, tool_id, owner_user_id, owner_org_id, display_name, summary, description_md,
		          subject_tags, grade_tags, support_url, privacy_url, visibility, pricing_model,
		          status, created_at, updated_at`,
		p.ToolID, p.OwnerUserID, p.OwnerOrgID, p.DisplayName, p.Summary, p.DescriptionMD,
		p.SubjectTags, p.GradeTags, p.SupportURL, p.PrivacyURL, p.Visibility, p.PricingModel,
	)
	return scanTool(row)
}

// GetToolByToolID loads a tool by its immutable namespace id.
func GetToolByToolID(ctx context.Context, pool *pgxpool.Pool, toolID string) (*Tool, error) {
	row := pool.QueryRow(ctx, `
		SELECT id, tool_id, owner_user_id, owner_org_id, display_name, summary, description_md,
		       subject_tags, grade_tags, support_url, privacy_url, visibility, pricing_model,
		       status, created_at, updated_at
		FROM toolmarket.tools WHERE tool_id = $1`, toolID)
	t, err := scanTool(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return t, err
}

// GetToolByPK loads a tool by primary key.
func GetToolByPK(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Tool, error) {
	row := pool.QueryRow(ctx, `
		SELECT id, tool_id, owner_user_id, owner_org_id, display_name, summary, description_md,
		       subject_tags, grade_tags, support_url, privacy_url, visibility, pricing_model,
		       status, created_at, updated_at
		FROM toolmarket.tools WHERE id = $1`, id)
	t, err := scanTool(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return t, err
}

// ListToolsByOwner lists tools owned by a developer.
func ListToolsByOwner(ctx context.Context, pool *pgxpool.Pool, ownerUserID uuid.UUID) ([]Tool, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, tool_id, owner_user_id, owner_org_id, display_name, summary, description_md,
		       subject_tags, grade_tags, support_url, privacy_url, visibility, pricing_model,
		       status, created_at, updated_at
		FROM toolmarket.tools
		WHERE owner_user_id = $1
		ORDER BY updated_at DESC`, ownerUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Tool{}
	for rows.Next() {
		t, err := scanTool(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

// UpdateToolStatus sets tool status.
func UpdateToolStatus(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, status string) error {
	_, err := pool.Exec(ctx, `
		UPDATE toolmarket.tools SET status = $2, updated_at = NOW() WHERE id = $1`, id, status)
	return err
}

// CreateReleaseParams creates a release row.
type CreateReleaseParams struct {
	ToolPK         uuid.UUID
	Version        string
	ManifestJSON   json.RawMessage
	DataSheetJSON  json.RawMessage
	BundleObjectID *uuid.UUID
	BundleSRI      string
	BundleBytes    int
	ChecksJSON     json.RawMessage
	SoakUntil      *time.Time
}

// CreateRelease inserts a release (pending review).
func CreateRelease(ctx context.Context, pool *pgxpool.Pool, p CreateReleaseParams) (*Release, error) {
	if len(p.ChecksJSON) == 0 {
		p.ChecksJSON = json.RawMessage(`{}`)
	}
	row := pool.QueryRow(ctx, `
		INSERT INTO toolmarket.tool_releases (
			tool_pk, version, manifest_json, data_sheet_json, bundle_object_id, bundle_sri,
			bundle_bytes, checks_json, soak_until
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, tool_pk, version, manifest_json, data_sheet_json, bundle_object_id, bundle_sri,
		          bundle_bytes, checks_json, review_status, reviewed_by, review_notes,
		          published_at, sunset_at, soak_until, created_at`,
		p.ToolPK, p.Version, p.ManifestJSON, p.DataSheetJSON, p.BundleObjectID, p.BundleSRI,
		p.BundleBytes, p.ChecksJSON, p.SoakUntil,
	)
	return scanRelease(row)
}

// GetRelease loads a release by id.
func GetRelease(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Release, error) {
	row := pool.QueryRow(ctx, `
		SELECT id, tool_pk, version, manifest_json, data_sheet_json, bundle_object_id, bundle_sri,
		       bundle_bytes, checks_json, review_status, reviewed_by, review_notes,
		       published_at, sunset_at, soak_until, created_at
		FROM toolmarket.tool_releases WHERE id = $1`, id)
	r, err := scanRelease(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return r, err
}

// GetReleaseByVersion loads a release by tool pk + version.
func GetReleaseByVersion(ctx context.Context, pool *pgxpool.Pool, toolPK uuid.UUID, version string) (*Release, error) {
	row := pool.QueryRow(ctx, `
		SELECT id, tool_pk, version, manifest_json, data_sheet_json, bundle_object_id, bundle_sri,
		       bundle_bytes, checks_json, review_status, reviewed_by, review_notes,
		       published_at, sunset_at, soak_until, created_at
		FROM toolmarket.tool_releases WHERE tool_pk = $1 AND version = $2`, toolPK, version)
	r, err := scanRelease(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return r, err
}

// ListReleases lists releases for a tool newest-first.
func ListReleases(ctx context.Context, pool *pgxpool.Pool, toolPK uuid.UUID) ([]Release, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, tool_pk, version, manifest_json, data_sheet_json, bundle_object_id, bundle_sri,
		       bundle_bytes, checks_json, review_status, reviewed_by, review_notes,
		       published_at, sunset_at, soak_until, created_at
		FROM toolmarket.tool_releases WHERE tool_pk = $1
		ORDER BY created_at DESC`, toolPK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Release{}
	for rows.Next() {
		r, err := scanRelease(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

// DecideRelease sets review decision; on approve also sets published_at.
func DecideRelease(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, status string, reviewer uuid.UUID, notes string) (*Release, error) {
	var published any
	if status == ReviewApproved {
		published = time.Now().UTC()
	}
	row := pool.QueryRow(ctx, `
		UPDATE toolmarket.tool_releases
		SET review_status = $2, reviewed_by = $3, review_notes = $4,
		    published_at = COALESCE($5::timestamptz, published_at)
		WHERE id = $1
		RETURNING id, tool_pk, version, manifest_json, data_sheet_json, bundle_object_id, bundle_sri,
		          bundle_bytes, checks_json, review_status, reviewed_by, review_notes,
		          published_at, sunset_at, soak_until, created_at`,
		id, status, reviewer, notes, published,
	)
	return scanRelease(row)
}

// SetReleaseSunset sets sunset_at on a release.
func SetReleaseSunset(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, sunsetAt time.Time) error {
	_, err := pool.Exec(ctx, `
		UPDATE toolmarket.tool_releases SET sunset_at = $2 WHERE id = $1`, id, sunsetAt)
	return err
}

// ListPendingReviews returns releases awaiting human review.
func ListPendingReviews(ctx context.Context, pool *pgxpool.Pool, status string, limit int) ([]Release, error) {
	if status == "" {
		status = ReviewPending
	}
	if limit <= 0 {
		limit = 50
	}
	rows, err := pool.Query(ctx, `
		SELECT id, tool_pk, version, manifest_json, data_sheet_json, bundle_object_id, bundle_sri,
		       bundle_bytes, checks_json, review_status, reviewed_by, review_notes,
		       published_at, sunset_at, soak_until, created_at
		FROM toolmarket.tool_releases
		WHERE review_status = $1
		ORDER BY created_at ASC
		LIMIT $2`, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Release{}
	for rows.Next() {
		r, err := scanRelease(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}
