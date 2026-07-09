package course

import (
	"context"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// StripeMinimumPriceCents is the minimum non-zero charge Stripe accepts (~$0.50 USD).
const StripeMinimumPriceCents = 50

// MaxCatalogPriceCents caps course fees at $99,999.99 (plan MKT2 FR-4).
const MaxCatalogPriceCents = 9_999_999

// CatalogListing holds the creator-editable public catalog and marketplace fields
// for a course (plans 15.1 and MKT2).
type CatalogListing struct {
	IsPublic            bool    `json:"isPublic"`
	Category            *string `json:"category"`
	DifficultyLevel     *string `json:"difficultyLevel"`
	Language            string  `json:"language"`
	PriceCents          int     `json:"priceCents"`
	PriceCurrency       string  `json:"priceCurrency"`
	Slug                string  `json:"slug"`
	MarketplaceListed   bool    `json:"marketplaceListed"`
	PublishState        string  `json:"publishState"`
	ActivePurchaseCount int     `json:"activePurchaseCount"`
}

var slugNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// stripeSupportedCurrencies is the allow-list for per-course currency (plan MKT2 open Q2).
var stripeSupportedCurrencies = map[string]struct{}{
	"usd": {}, "eur": {}, "gbp": {}, "cad": {}, "aud": {}, "jpy": {}, "chf": {},
	"sek": {}, "nok": {}, "dkk": {}, "nzd": {}, "sgd": {}, "hkd": {}, "mxn": {},
}

// Slugify converts a title/code into a URL-safe slug.
func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugNonAlnum.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// NormalizePriceCurrency lowercases and defaults empty currency to usd.
func NormalizePriceCurrency(currency string) string {
	c := strings.ToLower(strings.TrimSpace(currency))
	if c == "" {
		return "usd"
	}
	return c
}

// ValidPriceCurrency reports whether the currency is in the Stripe-supported set.
func ValidPriceCurrency(currency string) bool {
	_, ok := stripeSupportedCurrencies[strings.ToLower(strings.TrimSpace(currency))]
	return ok
}

// PublishStateFromBool maps the course.published column to API publishState.
func PublishStateFromBool(published bool) string {
	if published {
		return "published"
	}
	return "draft"
}

// GetCatalogListing returns the current catalog and marketplace fields for a course.
func GetCatalogListing(ctx context.Context, pool *pgxpool.Pool, courseCode string) (*CatalogListing, error) {
	var l CatalogListing
	var slug *string
	var published bool
	err := pool.QueryRow(ctx, `
		SELECT is_public, catalog_category, difficulty_level, catalog_language,
		       price_cents, catalog_slug, marketplace_listed, price_currency, published,
		       (SELECT COUNT(*)::int FROM billing.user_entitlements e
		        WHERE e.course_id = course.courses.id
		          AND e.entitlement_type = 'course_purchase'
		          AND e.status = 'active')
		FROM course.courses WHERE course_code = $1`, courseCode).
		Scan(
			&l.IsPublic, &l.Category, &l.DifficultyLevel, &l.Language,
			&l.PriceCents, &slug, &l.MarketplaceListed, &l.PriceCurrency, &published,
			&l.ActivePurchaseCount,
		)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, nil
		}
		return nil, err
	}
	if slug != nil {
		l.Slug = *slug
	}
	l.PriceCurrency = NormalizePriceCurrency(l.PriceCurrency)
	l.PublishState = PublishStateFromBool(published)
	return &l, nil
}

// SetCatalogListing updates catalog and marketplace fields for a course. When the
// course is made public and has no slug yet, one is derived from the title (with
// the course code as a uniqueness suffix). Returns nil when the course is missing.
func SetCatalogListing(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseCode string,
	in CatalogListing,
) (*CatalogListing, error) {
	lang := strings.TrimSpace(in.Language)
	if lang == "" {
		lang = "en"
	}
	price := in.PriceCents
	if price < 0 {
		price = 0
	}
	currency := NormalizePriceCurrency(in.PriceCurrency)

	slug := Slugify(in.Slug)
	if slug == "" {
		var title string
		if err := pool.QueryRow(ctx, `SELECT title FROM course.courses WHERE course_code = $1`, courseCode).Scan(&title); err != nil {
			if strings.Contains(err.Error(), "no rows") {
				return nil, nil
			}
			return nil, err
		}
		base := Slugify(title)
		suffix := strings.ToLower(Slugify(courseCode))
		if base == "" {
			slug = suffix
		} else {
			slug = base + "-" + suffix
		}
	}

	tag, err := pool.Exec(ctx, `
		UPDATE course.courses
		SET is_public = $1,
		    catalog_category = $2,
		    difficulty_level = $3,
		    catalog_language = $4,
		    price_cents = $5,
		    catalog_slug = $6,
		    marketplace_listed = $7,
		    marketplace_listed_at = CASE WHEN $7 THEN NOW() ELSE NULL END,
		    price_currency = $8,
		    updated_at = NOW()
		WHERE course_code = $9`,
		in.IsPublic, in.Category, in.DifficultyLevel, lang, price, slug,
		in.MarketplaceListed, currency, courseCode,
	)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, nil
	}
	return GetCatalogListing(ctx, pool, courseCode)
}
