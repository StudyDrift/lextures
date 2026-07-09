// Marketplace listing helpers for the in-app course marketplace (plan MKT1).
package course

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MarketplaceListing is the course-level marketplace state used by MKT2–MKT5.
// Published is exposed so MKT2 can refuse listing draft/unpublished courses (FR-9).
type MarketplaceListing struct {
	CourseID            uuid.UUID
	MarketplaceListed   bool
	MarketplaceListedAt *time.Time
	Published           bool
	PriceCents          int
	PriceCurrency       string
}

// IsMarketplaceListed reports whether the course is opted into the in-app storefront.
func IsMarketplaceListed(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (bool, error) {
	var listed bool
	err := pool.QueryRow(ctx, `
SELECT marketplace_listed FROM course.courses WHERE id = $1
`, courseID).Scan(&listed)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return listed, err
}

// GetMarketplaceListing loads marketplace columns plus publish state and pricing.
// Returns nil when the course does not exist.
func GetMarketplaceListing(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (*MarketplaceListing, error) {
	var m MarketplaceListing
	err := pool.QueryRow(ctx, `
SELECT id, marketplace_listed, marketplace_listed_at, published, price_cents, price_currency
FROM course.courses WHERE id = $1
`, courseID).Scan(
		&m.CourseID,
		&m.MarketplaceListed,
		&m.MarketplaceListedAt,
		&m.Published,
		&m.PriceCents,
		&m.PriceCurrency,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// IsMarketplaceListable reports whether the course may be listed (must be published).
// Write-path enforcement lives in MKT2; this helper is the shared check (FR-9).
func IsMarketplaceListable(m *MarketplaceListing) bool {
	return m != nil && m.Published
}

// IsFree reports whether the course fee is free (price_cents = 0).
func IsFree(priceCents int) bool {
	return priceCents <= 0
}
