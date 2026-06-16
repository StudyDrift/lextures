package course

import (
	"context"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CatalogListing holds the creator-editable public catalog fields for a course
// (plan 15.1 — "Making your course publicly discoverable").
type CatalogListing struct {
	IsPublic        bool    `json:"isPublic"`
	Category        *string `json:"category"`
	DifficultyLevel *string `json:"difficultyLevel"`
	Language        string  `json:"language"`
	PriceCents      int     `json:"priceCents"`
	Slug            string  `json:"slug"`
}

var slugNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify converts a title/code into a URL-safe slug.
func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugNonAlnum.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// GetCatalogListing returns the current public catalog fields for a course.
func GetCatalogListing(ctx context.Context, pool *pgxpool.Pool, courseCode string) (*CatalogListing, error) {
	var l CatalogListing
	var slug *string
	err := pool.QueryRow(ctx, `
		SELECT is_public, catalog_category, difficulty_level, catalog_language, price_cents, catalog_slug
		FROM course.courses WHERE course_code = $1`, courseCode).
		Scan(&l.IsPublic, &l.Category, &l.DifficultyLevel, &l.Language, &l.PriceCents, &slug)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, nil
		}
		return nil, err
	}
	if slug != nil {
		l.Slug = *slug
	}
	return &l, nil
}

// SetCatalogListing updates the public catalog fields for a course. When the
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

	slug := Slugify(in.Slug)
	if slug == "" {
		// Derive a deterministic, unique-ish slug from the course title + code.
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
		    updated_at = NOW()
		WHERE course_code = $7`,
		in.IsPublic, in.Category, in.DifficultyLevel, lang, price, slug, courseCode,
	)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, nil
	}
	return GetCatalogListing(ctx, pool, courseCode)
}
