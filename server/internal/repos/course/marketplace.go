// Marketplace listing helpers for the in-app course marketplace (plan MKT1).
package course

import (
	"context"
	"errors"
	"strings"
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

// IsStorefrontHeroReadable reports whether a course's hero image may be served without
// enrollment — published courses that are marketplace-listed and/or publicly catalogued.
func IsStorefrontHeroReadable(ctx context.Context, pool *pgxpool.Pool, courseCode string) (bool, error) {
	var ok bool
	err := pool.QueryRow(ctx, `
SELECT TRUE
FROM course.courses
WHERE course_code = $1
  AND published = TRUE
  AND (marketplace_listed = TRUE OR is_public = TRUE)
`, courseCode).Scan(&ok)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return ok, err
}

// IsCourseHeroFile reports whether fileID is the course's configured hero_image_url target.
func IsCourseHeroFile(ctx context.Context, pool *pgxpool.Pool, courseCode string, fileID uuid.UUID) (bool, error) {
	var heroURL *string
	err := pool.QueryRow(ctx, `
SELECT hero_image_url FROM course.courses WHERE course_code = $1
`, courseCode).Scan(&heroURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if heroURL == nil {
		return false, nil
	}
	want := strings.TrimSpace(*heroURL)
	got := "/api/v1/courses/" + courseCode + "/course-files/" + fileID.String() + "/content"
	return want == got, nil
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
