// Package dpa persists SDPC/NDPA DPA versions, acceptance records, and data inventory (plan 10.5).
package dpa

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DPAVersion is one row from compliance.dpa_versions.
type DPAVersion struct {
	ID          uuid.UUID
	VersionStr  string
	TemplateURL string
	EffectiveAt time.Time
	Notes       *string
	CreatedAt   time.Time
}

// DPAAcceptance is one row from compliance.dpa_acceptances.
type DPAAcceptance struct {
	ID           uuid.UUID
	OrgID        uuid.UUID
	DPAVersionID uuid.UUID
	AcceptedBy   uuid.UUID
	AcceptedAt   time.Time
	IPAddress    *string
}

// DataInventoryItem is one row from compliance.data_inventory.
type DataInventoryItem struct {
	ID                       uuid.UUID
	ElementName              string
	Category                 string
	Purpose                  string
	LegalBasis               string
	RetentionDays            *int
	SharedWithSubProcessors  bool
	SubProcessorNames        []string
	UpdatedAt                time.Time
}

// GetCurrentVersion returns the most recently effective DPA version.
func GetCurrentVersion(ctx context.Context, pool *pgxpool.Pool) (*DPAVersion, error) {
	var v DPAVersion
	err := pool.QueryRow(ctx, `
SELECT id, version_str, template_url, effective_at, notes, created_at
  FROM compliance.dpa_versions
 WHERE effective_at <= NOW()
 ORDER BY effective_at DESC
 LIMIT 1
`).Scan(&v.ID, &v.VersionStr, &v.TemplateURL, &v.EffectiveAt, &v.Notes, &v.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// ListVersions returns all DPA versions ordered by effective_at descending.
func ListVersions(ctx context.Context, pool *pgxpool.Pool) ([]DPAVersion, error) {
	rows, err := pool.Query(ctx, `
SELECT id, version_str, template_url, effective_at, notes, created_at
  FROM compliance.dpa_versions
 ORDER BY effective_at DESC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DPAVersion
	for rows.Next() {
		var v DPAVersion
		if err := rows.Scan(&v.ID, &v.VersionStr, &v.TemplateURL, &v.EffectiveAt, &v.Notes, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// InsertAcceptance records a DPA acceptance. Returns the row ID.
// Idempotent: if the org+version pair was already accepted, returns the existing ID.
// Does not use ON CONFLICT because PostgreSQL disallows that with rule-protected tables.
func InsertAcceptance(ctx context.Context, pool *pgxpool.Pool, orgID, dpaVersionID, acceptedBy uuid.UUID, ip net.IP) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO compliance.dpa_acceptances (org_id, dpa_version_id, accepted_by, ip_address)
VALUES ($1, $2, $3, $4)
RETURNING id
`, orgID, dpaVersionID, acceptedBy, ip).Scan(&id)
	if err != nil {
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == "23505" {
			// Unique violation — already accepted; return existing row ID.
			err2 := pool.QueryRow(ctx, `
SELECT id FROM compliance.dpa_acceptances WHERE org_id = $1 AND dpa_version_id = $2
`, orgID, dpaVersionID).Scan(&id)
			return id, err2
		}
		return uuid.UUID{}, err
	}
	return id, nil
}

// GetAcceptanceByOrgVersion returns the acceptance record for a given org+version, or nil.
func GetAcceptanceByOrgVersion(ctx context.Context, pool *pgxpool.Pool, orgID, dpaVersionID uuid.UUID) (*DPAAcceptance, error) {
	var a DPAAcceptance
	var ipStr *string
	err := pool.QueryRow(ctx, `
SELECT id, org_id, dpa_version_id, accepted_by, accepted_at, host(ip_address)
  FROM compliance.dpa_acceptances
 WHERE org_id = $1 AND dpa_version_id = $2
`, orgID, dpaVersionID).Scan(&a.ID, &a.OrgID, &a.DPAVersionID, &a.AcceptedBy, &a.AcceptedAt, &ipStr)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	a.IPAddress = ipStr
	return &a, nil
}

// ListAcceptances returns all DPA acceptance records ordered by accepted_at descending.
func ListAcceptances(ctx context.Context, pool *pgxpool.Pool) ([]DPAAcceptance, error) {
	rows, err := pool.Query(ctx, `
SELECT id, org_id, dpa_version_id, accepted_by, accepted_at, host(ip_address)
  FROM compliance.dpa_acceptances
 ORDER BY accepted_at DESC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DPAAcceptance
	for rows.Next() {
		var a DPAAcceptance
		var ipStr *string
		if err := rows.Scan(&a.ID, &a.OrgID, &a.DPAVersionID, &a.AcceptedBy, &a.AcceptedAt, &ipStr); err != nil {
			return nil, err
		}
		a.IPAddress = ipStr
		out = append(out, a)
	}
	return out, rows.Err()
}

// ListDataInventory returns all data inventory rows ordered by category, element_name.
func ListDataInventory(ctx context.Context, pool *pgxpool.Pool) ([]DataInventoryItem, error) {
	rows, err := pool.Query(ctx, `
SELECT id, element_name, category, purpose, legal_basis,
       retention_days, shared_with_sub_processors, sub_processor_names, updated_at
  FROM compliance.data_inventory
 ORDER BY category, element_name
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DataInventoryItem
	for rows.Next() {
		var item DataInventoryItem
		if err := rows.Scan(
			&item.ID, &item.ElementName, &item.Category, &item.Purpose, &item.LegalBasis,
			&item.RetentionDays, &item.SharedWithSubProcessors, &item.SubProcessorNames, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if item.SubProcessorNames == nil {
			item.SubProcessorNames = []string{}
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
