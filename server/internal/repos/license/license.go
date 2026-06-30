// Package license provides org seat license persistence (plan 18.8).
package license

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Tier names stored in tenant.license_tier.
const (
	TierUnlimited  = "unlimited"
	TierStarter    = "starter"
	TierGrowth     = "growth"
	TierEnterprise = "enterprise"
)

// Row is one org license record.
type Row struct {
	ID            uuid.UUID
	OrgID         uuid.UUID
	OrgName       string
	OrgSlug       string
	Tier          string
	MaxSeats      int
	UsedSeats     int
	ContractStart *time.Time
	ContractEnd   *time.Time
	Notes         *string
	UpdatedBy     *uuid.UUID
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Patch updates mutable license fields.
type Patch struct {
	Tier          *string
	MaxSeats      *int
	ContractStart *time.Time
	ContractEnd   *time.Time
	Notes         *string
	UpdatedBy     *uuid.UUID
}

// ListParams paginates super-admin license listing.
type ListParams struct {
	Limit  int
	Offset int
}

func normalizeTier(t string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case TierUnlimited, TierStarter, TierGrowth, TierEnterprise:
		return strings.ToLower(strings.TrimSpace(t)), nil
	default:
		return "", fmt.Errorf("invalid tier %q", t)
	}
}

func scanRow(row pgx.Row) (Row, error) {
	var r Row
	err := row.Scan(
		&r.ID, &r.OrgID, &r.Tier, &r.MaxSeats, &r.UsedSeats,
		&r.ContractStart, &r.ContractEnd, &r.Notes, &r.UpdatedBy,
		&r.CreatedAt, &r.UpdatedAt,
	)
	return r, err
}

func scanRowWithOrg(row pgx.Row) (Row, error) {
	var r Row
	err := row.Scan(
		&r.ID, &r.OrgID, &r.OrgName, &r.OrgSlug, &r.Tier, &r.MaxSeats, &r.UsedSeats,
		&r.ContractStart, &r.ContractEnd, &r.Notes, &r.UpdatedBy,
		&r.CreatedAt, &r.UpdatedAt,
	)
	return r, err
}

const licenseSelect = `
SELECT l.id, l.org_id, l.tier::text, l.max_seats, l.used_seats,
       l.contract_start, l.contract_end, l.notes, l.updated_by,
       l.created_at, l.updated_at
FROM tenant.licenses l`

// GetByOrg returns the license for orgID, or nil when no explicit record exists.
func GetByOrg(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) (*Row, error) {
	r, err := scanRow(pool.QueryRow(ctx, licenseSelect+` WHERE l.org_id = $1`, orgID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// DefaultForOrg returns the effective license when no row exists (unlimited).
func DefaultForOrg(orgID uuid.UUID) Row {
	now := time.Now().UTC()
	return Row{
		OrgID:     orgID,
		Tier:      TierUnlimited,
		MaxSeats:  -1,
		UsedSeats: 0,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Effective returns stored license or unlimited default.
func Effective(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) (Row, error) {
	row, err := GetByOrg(ctx, pool, orgID)
	if err != nil {
		return Row{}, err
	}
	if row == nil {
		def := DefaultForOrg(orgID)
		used, err := CountLearnerSeats(ctx, pool, orgID)
		if err != nil {
			return Row{}, err
		}
		def.UsedSeats = used
		return def, nil
	}
	return *row, nil
}

// CountLearnerSeats counts active non-admin users for an org.
func CountLearnerSeats(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `SELECT tenant.count_learner_seats($1)`, orgID).Scan(&n)
	return n, err
}

// RefreshUsedSeats recomputes used_seats from source users.
func RefreshUsedSeats(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) error {
	_, err := pool.Exec(ctx, `SELECT tenant.refresh_license_used_seats($1)`, orgID)
	return err
}

// Upsert creates or updates a license row.
func Upsert(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, p Patch) (Row, error) {
	cur, err := GetByOrg(ctx, pool, orgID)
	if err != nil {
		return Row{}, err
	}
	tier := TierUnlimited
	maxSeats := -1
	var contractStart, contractEnd *time.Time
	var notes *string
	if cur != nil {
		tier = cur.Tier
		maxSeats = cur.MaxSeats
		contractStart = cur.ContractStart
		contractEnd = cur.ContractEnd
		notes = cur.Notes
	}
	if p.Tier != nil {
		tier, err = normalizeTier(*p.Tier)
		if err != nil {
			return Row{}, err
		}
	}
	if p.MaxSeats != nil {
		maxSeats = *p.MaxSeats
		if maxSeats != -1 && maxSeats < 0 {
			return Row{}, fmt.Errorf("max_seats must be -1 or non-negative")
		}
	}
	if p.ContractStart != nil {
		contractStart = p.ContractStart
	}
	if p.ContractEnd != nil {
		contractEnd = p.ContractEnd
	}
	if p.Notes != nil {
		notes = p.Notes
	}

	var r Row
	err = pool.QueryRow(ctx, `
INSERT INTO tenant.licenses (org_id, tier, max_seats, contract_start, contract_end, notes, updated_by)
VALUES ($1, $2::tenant.license_tier, $3, $4, $5, $6, $7)
ON CONFLICT (org_id) DO UPDATE SET
    tier = EXCLUDED.tier,
    max_seats = EXCLUDED.max_seats,
    contract_start = EXCLUDED.contract_start,
    contract_end = EXCLUDED.contract_end,
    notes = EXCLUDED.notes,
    updated_by = EXCLUDED.updated_by,
    updated_at = NOW()
RETURNING id, org_id, tier::text, max_seats, used_seats,
          contract_start, contract_end, notes, updated_by, created_at, updated_at
`, orgID, tier, maxSeats, contractStart, contractEnd, notes, p.UpdatedBy).Scan(
		&r.ID, &r.OrgID, &r.Tier, &r.MaxSeats, &r.UsedSeats,
		&r.ContractStart, &r.ContractEnd, &r.Notes, &r.UpdatedBy,
		&r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return Row{}, err
	}
	if err := RefreshUsedSeats(ctx, pool, orgID); err != nil {
		return Row{}, err
	}
	updated, err := GetByOrg(ctx, pool, orgID)
	if err != nil {
		return Row{}, err
	}
	if updated != nil {
		return *updated, nil
	}
	return r, nil
}

// List returns licenses joined with org metadata for super-admin views.
func List(ctx context.Context, pool *pgxpool.Pool, p ListParams) ([]Row, error) {
	limit := p.Limit
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	offset := p.Offset
	if offset < 0 {
		offset = 0
	}
	rows, err := pool.Query(ctx, `
SELECT l.id, l.org_id, o.name, o.slug, l.tier::text, l.max_seats, l.used_seats,
       l.contract_start, l.contract_end, l.notes, l.updated_by,
       l.created_at, l.updated_at
FROM tenant.licenses l
JOIN tenant.organizations o ON o.id = l.org_id
ORDER BY o.name ASC
LIMIT $1 OFFSET $2
`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Row
	for rows.Next() {
		r, err := scanRowWithOrg(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ReconcileAll refreshes used_seats for every org with a license row.
func ReconcileAll(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	tag, err := pool.Exec(ctx, `
UPDATE tenant.licenses l
SET used_seats = tenant.count_learner_seats(l.org_id),
    updated_at = NOW()`)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}
