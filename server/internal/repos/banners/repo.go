// Package banners persists and queries platform.banners (plan 18.6).
package banners

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Severity is the banner severity level.
type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

// Scope is global or org-scoped.
type Scope string

const (
	ScopeGlobal Scope = "global"
	ScopeOrg    Scope = "org"
)

// Banner is one maintenance/outage notice row.
type Banner struct {
	ID         uuid.UUID
	Scope      Scope
	OrgID      *uuid.UUID
	Message    string
	Severity   Severity
	CTAText    *string
	CTAURL     *string
	StartsAt   *time.Time
	ExpiresAt  *time.Time
	IsActive   bool
	ExternalID *string
	CreatedBy  uuid.UUID
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// CreateParams holds fields for inserting a banner.
type CreateParams struct {
	Scope      Scope
	OrgID      *uuid.UUID
	Message    string
	Severity   Severity
	CTAText    *string
	CTAURL     *string
	StartsAt   *time.Time
	ExpiresAt  *time.Time
	ExternalID *string
	CreatedBy  uuid.UUID
}

// UpdateParams holds mutable banner fields.
type UpdateParams struct {
	Message   string
	Severity  Severity
	CTAText   *string
	CTAURL    *string
	StartsAt  *time.Time
	ExpiresAt *time.Time
	IsActive  bool
}

const bannerColumns = `
	id, scope, org_id, message, severity, cta_text, cta_url,
	starts_at, expires_at, is_active, external_id, created_by, created_at, updated_at
`

func scanBanner(row pgx.Row) (Banner, error) {
	var b Banner
	var scope, severity string
	err := row.Scan(
		&b.ID, &scope, &b.OrgID, &b.Message, &severity, &b.CTAText, &b.CTAURL,
		&b.StartsAt, &b.ExpiresAt, &b.IsActive, &b.ExternalID, &b.CreatedBy, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		return Banner{}, err
	}
	b.Scope = Scope(scope)
	b.Severity = Severity(severity)
	return b, nil
}

func activeTimeClause(now time.Time) string {
	_ = now
	return `
		AND is_active = TRUE
		AND (starts_at IS NULL OR starts_at <= $1)
		AND (expires_at IS NULL OR expires_at > $1)
	`
}

// GetActiveForOrg returns the highest-priority active banner for a viewer.
// Org-scoped banners take precedence over global banners.
func GetActiveForOrg(ctx context.Context, pool *pgxpool.Pool, orgID *uuid.UUID, now time.Time) (*Banner, error) {
	if orgID != nil {
		row := pool.QueryRow(ctx, `
SELECT `+bannerColumns+`
FROM platform.banners
WHERE scope = 'org' AND org_id = $2`+activeTimeClause(now)+`
ORDER BY updated_at DESC
LIMIT 1
`, now, *orgID)
		b, err := scanBanner(row)
		if err == nil {
			return &b, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}
	row := pool.QueryRow(ctx, `
SELECT `+bannerColumns+`
FROM platform.banners
WHERE scope = 'global'`+activeTimeClause(now)+`
ORDER BY updated_at DESC
LIMIT 1
`, now)
	b, err := scanBanner(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// List returns banners for admin management, optionally filtered by org.
func List(ctx context.Context, pool *pgxpool.Pool, orgID *uuid.UUID, globalOnly bool) ([]Banner, error) {
	var rows pgx.Rows
	var err error
	if globalOnly {
		rows, err = pool.Query(ctx, `
SELECT `+bannerColumns+`
FROM platform.banners
WHERE scope = 'global'
ORDER BY updated_at DESC
LIMIT 100`)
	} else if orgID != nil {
		rows, err = pool.Query(ctx, `
SELECT `+bannerColumns+`
FROM platform.banners
WHERE scope = 'org' AND org_id = $1
ORDER BY updated_at DESC
LIMIT 100`, *orgID)
	} else {
		rows, err = pool.Query(ctx, `
SELECT `+bannerColumns+`
FROM platform.banners
ORDER BY updated_at DESC
LIMIT 100`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Banner
	for rows.Next() {
		b, err := scanBanner(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// GetByID loads one banner by primary key.
func GetByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Banner, error) {
	row := pool.QueryRow(ctx, `SELECT `+bannerColumns+` FROM platform.banners WHERE id = $1`, id)
	b, err := scanBanner(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// Create inserts a new banner.
func Create(ctx context.Context, pool *pgxpool.Pool, p CreateParams) (Banner, error) {
	row := pool.QueryRow(ctx, `
INSERT INTO platform.banners
  (scope, org_id, message, severity, cta_text, cta_url, starts_at, expires_at, external_id, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING `+bannerColumns,
		string(p.Scope), p.OrgID, p.Message, string(p.Severity),
		p.CTAText, p.CTAURL, p.StartsAt, p.ExpiresAt, p.ExternalID, p.CreatedBy,
	)
	return scanBanner(row)
}

// UpsertByExternalID creates or updates a banner keyed by external_id (Statuspage incidents).
func UpsertByExternalID(ctx context.Context, pool *pgxpool.Pool, p CreateParams) (Banner, error) {
	if p.ExternalID == nil || *p.ExternalID == "" {
		return Create(ctx, pool, p)
	}
	row := pool.QueryRow(ctx, `
INSERT INTO platform.banners
  (scope, org_id, message, severity, cta_text, cta_url, starts_at, expires_at, external_id, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (external_id) WHERE external_id IS NOT NULL AND external_id <> ''
DO UPDATE SET
  message = EXCLUDED.message,
  severity = EXCLUDED.severity,
  cta_text = EXCLUDED.cta_text,
  cta_url = EXCLUDED.cta_url,
  starts_at = EXCLUDED.starts_at,
  expires_at = EXCLUDED.expires_at,
  is_active = TRUE,
  updated_at = now()
RETURNING `+bannerColumns,
		string(p.Scope), p.OrgID, p.Message, string(p.Severity),
		p.CTAText, p.CTAURL, p.StartsAt, p.ExpiresAt, p.ExternalID, p.CreatedBy,
	)
	return scanBanner(row)
}

// Update mutates an existing banner.
func Update(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, p UpdateParams) (*Banner, error) {
	row := pool.QueryRow(ctx, `
UPDATE platform.banners
SET message = $2, severity = $3, cta_text = $4, cta_url = $5,
    starts_at = $6, expires_at = $7, is_active = $8, updated_at = now()
WHERE id = $1
RETURNING `+bannerColumns,
		id, p.Message, string(p.Severity), p.CTAText, p.CTAURL, p.StartsAt, p.ExpiresAt, p.IsActive,
	)
	b, err := scanBanner(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// GetByExternalID loads a banner keyed by external_id (Statuspage incidents).
func GetByExternalID(ctx context.Context, pool *pgxpool.Pool, externalID string) (*Banner, error) {
	row := pool.QueryRow(ctx, `
SELECT `+bannerColumns+`
FROM platform.banners
WHERE external_id = $1
LIMIT 1`, externalID)
	b, err := scanBanner(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// DeactivateByExternalID marks a webhook-managed banner inactive.
func DeactivateByExternalID(ctx context.Context, pool *pgxpool.Pool, externalID string) error {
	_, err := pool.Exec(ctx, `
UPDATE platform.banners
SET is_active = FALSE, updated_at = now()
WHERE external_id = $1`, externalID)
	return err
}

// Delete removes a banner row.
func Delete(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (bool, error) {
	tag, err := pool.Exec(ctx, `DELETE FROM platform.banners WHERE id = $1`, id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}
