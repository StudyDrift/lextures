// Marketplace storefront listing/detail queries (plan MKT3).
// Visibility is marketplace_listed + published (not is_public).
package course

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CatalogSortPrice sorts marketplace/catalog results by ascending price.
const CatalogSortPrice = "price"

// MarketplaceCourse is a storefront card/detail record (plan MKT3).
// Owned is resolved per-request in the HTTP layer and is not loaded from SQL.
type MarketplaceCourse struct {
	ID              string   `json:"id"`
	Slug            string   `json:"slug"`
	CourseCode      string   `json:"courseCode"`
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	HeroImageURL    *string  `json:"heroImageUrl"`
	Category        *string  `json:"category"`
	Level           *string  `json:"level"`
	Language        string   `json:"language"`
	PriceCents      int      `json:"priceCents"`
	PriceCurrency   string   `json:"priceCurrency"`
	ListPriceCents  *int     `json:"listPriceCents"`
	EnrollmentCount int      `json:"enrollmentCount"`
	AverageRating   *float64 `json:"averageRating"`
	RatingCount     int      `json:"ratingCount"`
	InstructorName  *string  `json:"instructorName"`
	CreatedAt       string   `json:"createdAt"`
	Owned           bool     `json:"owned"`
}

// MarketplaceWhatsIncluded summarizes course structure for the detail page.
type MarketplaceWhatsIncluded struct {
	ModuleCount              int  `json:"moduleCount"`
	ItemCount                int  `json:"itemCount"`
	EstimatedDurationMinutes *int `json:"estimatedDurationMinutes,omitempty"`
}

// MarketplaceFilter mirrors PublicCatalogFilter with an optional free-only flag.
type MarketplaceFilter struct {
	Q        string
	Category string
	Level    string
	Language string
	PriceMax *int
	FreeOnly bool
	Sort     string
	Offset   int
	Limit    int
}

// ToPublicCatalogFilter adapts a marketplace filter for shared sort/cursor helpers.
func (f MarketplaceFilter) ToPublicCatalogFilter() PublicCatalogFilter {
	pf := PublicCatalogFilter{
		Q:        f.Q,
		Category: f.Category,
		Level:    f.Level,
		Language: f.Language,
		PriceMax: f.PriceMax,
		Sort:     f.Sort,
		Offset:   f.Offset,
		Limit:    f.Limit,
	}
	if f.FreeOnly {
		zero := 0
		pf.PriceMax = &zero
	}
	return pf
}

const marketplaceCatalogSelect = `
    c.id,
    COALESCE(NULLIF(TRIM(c.catalog_slug), ''), c.course_code) AS slug,
    c.course_code,
    c.title,
    c.description,
    c.hero_image_url,
    c.catalog_category,
    c.difficulty_level,
    c.catalog_language,
    c.price_cents,
    COALESCE(NULLIF(TRIM(c.price_currency), ''), 'usd') AS price_currency,
    c.list_price_cents,
    c.enrollment_count,
    c.average_rating,
    c.rating_count,
    NULLIF(TRIM(COALESCE(u.display_name,
        TRIM(COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')))), '') AS instructor_name,
    c.created_at::text
`

const marketplaceCatalogFrom = `
    FROM course.courses c
    LEFT JOIN "user".users u ON u.id = c.created_by_user_id
    WHERE c.marketplace_listed = TRUE
      AND c.published = TRUE
      AND c.archived = FALSE
`

func scanMarketplaceCourse(scan func(dest ...any) error) (MarketplaceCourse, error) {
	var c MarketplaceCourse
	err := scan(
		&c.ID, &c.Slug, &c.CourseCode, &c.Title, &c.Description, &c.HeroImageURL,
		&c.Category, &c.Level, &c.Language, &c.PriceCents, &c.PriceCurrency, &c.ListPriceCents,
		&c.EnrollmentCount, &c.AverageRating, &c.RatingCount, &c.InstructorName, &c.CreatedAt,
	)
	return c, err
}

func marketplaceOrderBy(sort, q string, add func(any) string) string {
	switch sort {
	case CatalogSortRating:
		return "c.average_rating DESC NULLS LAST, c.enrollment_count DESC, c.created_at DESC"
	case CatalogSortNewest:
		return "c.created_at DESC, c.id DESC"
	case CatalogSortPrice:
		return "c.price_cents ASC, c.created_at DESC, c.id DESC"
	case CatalogSortRelevance:
		if strings.TrimSpace(q) != "" {
			rankPh := add(q)
			return fmt.Sprintf(
				"ts_rank(c.search_vector, websearch_to_tsquery('english', %s)) DESC, c.enrollment_count DESC, c.created_at DESC",
				rankPh)
		}
		return "c.enrollment_count DESC, c.created_at DESC"
	default: // popular
		return "c.enrollment_count DESC, c.created_at DESC, c.id DESC"
	}
}

// ListMarketplaceCourses returns a page of marketplace-listed published courses.
func ListMarketplaceCourses(
	ctx context.Context,
	pool *pgxpool.Pool,
	f MarketplaceFilter,
) ([]MarketplaceCourse, int, string, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}

	var where strings.Builder
	args := []any{}
	add := func(v any) string {
		args = append(args, v)
		return "$" + strconv.Itoa(len(args))
	}

	q := strings.TrimSpace(f.Q)
	if q != "" {
		like := "%" + q + "%"
		ph := add(q)
		phLike := add(like)
		fmt.Fprintf(&where, `
      AND (
          c.search_vector @@ websearch_to_tsquery('english', %s)
          OR c.title ILIKE %s
          OR c.description ILIKE %s
          OR COALESCE(u.display_name, '') ILIKE %s
      )`, ph, phLike, phLike, phLike)
	}
	if cat := strings.TrimSpace(f.Category); cat != "" {
		where.WriteString("\n      AND c.catalog_category = " + add(cat))
	}
	if lvl := strings.TrimSpace(f.Level); lvl != "" {
		where.WriteString("\n      AND c.difficulty_level = " + add(lvl))
	}
	if lang := strings.TrimSpace(f.Language); lang != "" {
		where.WriteString("\n      AND c.catalog_language = " + add(lang))
	}
	priceMax := f.PriceMax
	if f.FreeOnly {
		zero := 0
		priceMax = &zero
	}
	if priceMax != nil {
		where.WriteString("\n      AND c.price_cents <= " + add(*priceMax))
	}

	orderBy := marketplaceOrderBy(f.Sort, q, add)
	limitPh := add(limit + 1)
	offsetPh := add(offset)

	query := "SELECT " + marketplaceCatalogSelect + marketplaceCatalogFrom + where.String() +
		"\n    ORDER BY " + orderBy + "\n    LIMIT " + limitPh + " OFFSET " + offsetPh

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, "", err
	}
	defer rows.Close()

	out := make([]MarketplaceCourse, 0, limit)
	for rows.Next() {
		c, err := scanMarketplaceCourse(rows.Scan)
		if err != nil {
			return nil, 0, "", err
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, "", err
	}

	nextCursor := ""
	if len(out) > limit {
		out = out[:limit]
		nextCursor = EncodeCatalogCursor(offset + limit)
	}

	var countArgs []any
	if f.Sort == CatalogSortRelevance && q != "" {
		countArgs = args[:len(args)-3]
	} else {
		countArgs = args[:len(args)-2]
	}
	countQuery := "SELECT COUNT(*)" + marketplaceCatalogFrom + where.String()
	var total int
	if err := pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, "", err
	}

	return out, total, nextCursor, nil
}

// GetMarketplaceCourseBySlug returns one listed+published course by catalog_slug or course_code.
func GetMarketplaceCourseBySlug(
	ctx context.Context,
	pool *pgxpool.Pool,
	slug string,
) (*MarketplaceCourse, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, nil
	}
	query := "SELECT " + marketplaceCatalogSelect + marketplaceCatalogFrom +
		"\n      AND (lower(c.catalog_slug) = lower($1) OR lower(c.course_code) = lower($1))\n    LIMIT 1"
	c, err := scanMarketplaceCourse(pool.QueryRow(ctx, query, slug).Scan)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

// ListMarketplaceCategories returns category facets among marketplace-listed courses.
func ListMarketplaceCategories(ctx context.Context, pool *pgxpool.Pool) ([]CatalogCategory, error) {
	rows, err := pool.Query(ctx, `
SELECT c.catalog_category, COUNT(*) AS n
FROM course.courses c
WHERE c.marketplace_listed = TRUE
  AND c.published = TRUE
  AND c.archived = FALSE
  AND c.catalog_category IS NOT NULL
  AND c.catalog_category <> ''
GROUP BY c.catalog_category
ORDER BY n DESC, c.catalog_category ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CatalogCategory{}
	for rows.Next() {
		var cc CatalogCategory
		if err := rows.Scan(&cc.Category, &cc.Count); err != nil {
			return nil, err
		}
		out = append(out, cc)
	}
	return out, rows.Err()
}

// GetMarketplaceWhatsIncluded returns module/item counts and a rough duration estimate.
func GetMarketplaceWhatsIncluded(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID uuid.UUID,
) (MarketplaceWhatsIncluded, error) {
	var w MarketplaceWhatsIncluded
	err := pool.QueryRow(ctx, `
SELECT
  COUNT(*) FILTER (
    WHERE kind = 'module' AND parent_id IS NULL AND published = TRUE AND archived = FALSE
  )::int AS module_count,
  COUNT(*) FILTER (
    WHERE kind <> 'module' AND published = TRUE AND archived = FALSE
  )::int AS item_count
FROM course.course_structure_items
WHERE course_id = $1
`, courseID).Scan(&w.ModuleCount, &w.ItemCount)
	if err != nil {
		return MarketplaceWhatsIncluded{}, err
	}
	if w.ItemCount > 0 {
		mins := w.ItemCount * 15
		w.EstimatedDurationMinutes = &mins
	}
	return w, nil
}
