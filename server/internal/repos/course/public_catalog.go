package course

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PublicCatalogCourse is a single course as exposed by the public (unauthenticated)
// catalog. It carries only published, public-visibility fields — no PII (plan 15.1).
type PublicCatalogCourse struct {
	ID              string   `json:"id"`
	Slug            string   `json:"slug"`
	CourseCode      string   `json:"courseCode"`
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	HeroImageURL    *string  `json:"heroImageUrl"`
	Category        *string  `json:"category"`
	DifficultyLevel *string  `json:"difficultyLevel"`
	Language        string   `json:"language"`
	PriceCents      int      `json:"priceCents"`
	EnrollmentCount int      `json:"enrollmentCount"`
	AverageRating   *float64 `json:"averageRating"`
	RatingCount     int      `json:"ratingCount"`
	InstructorName  *string  `json:"instructorName"`
	CreatedAt       string   `json:"createdAt"`
}

// CatalogCategory is a catalog browse category with its published course count.
type CatalogCategory struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}

// Valid sort modes for the public catalog.
const (
	CatalogSortPopular   = "popular"
	CatalogSortRating    = "rating"
	CatalogSortNewest    = "newest"
	CatalogSortRelevance = "relevance"
)

// ValidCatalogSort reports whether s is a supported catalog sort mode.
func ValidCatalogSort(s string) bool {
	switch s {
	case CatalogSortPopular, CatalogSortRating, CatalogSortNewest, CatalogSortRelevance:
		return true
	default:
		return false
	}
}

// ValidDifficultyLevel reports whether s is a supported difficulty level.
func ValidDifficultyLevel(s string) bool {
	switch s {
	case "beginner", "intermediate", "advanced":
		return true
	default:
		return false
	}
}

// PublicCatalogFilter holds the query parameters for a catalog list request.
type PublicCatalogFilter struct {
	Q        string // free-text search over title/description/instructor
	Category string // exact catalog_category match
	Level    string // beginner | intermediate | advanced
	Language string // exact language match
	PriceMax *int   // inclusive max price in cents; 0 selects free courses only
	Sort     string // one of the CatalogSort* constants
	Offset   int    // resolved from the opaque cursor
	Limit    int    // page size (clamped 1..50)
}

// EncodeCatalogCursor returns the opaque cursor pointing at the given row offset.
func EncodeCatalogCursor(offset int) string {
	return base64.RawURLEncoding.EncodeToString([]byte("o:" + strconv.Itoa(offset)))
}

// DecodeCatalogCursor parses an opaque catalog cursor into a row offset. An empty
// cursor decodes to offset 0; malformed cursors return an error.
func DecodeCatalogCursor(cursor string) (int, error) {
	cursor = strings.TrimSpace(cursor)
	if cursor == "" {
		return 0, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return 0, fmt.Errorf("invalid cursor")
	}
	s := string(raw)
	if !strings.HasPrefix(s, "o:") {
		return 0, fmt.Errorf("invalid cursor")
	}
	offset, err := strconv.Atoi(strings.TrimPrefix(s, "o:"))
	if err != nil || offset < 0 {
		return 0, fmt.Errorf("invalid cursor")
	}
	return offset, nil
}

// publicCatalogSelect is the column projection shared by list and detail queries.
// instructor name resolves from the course creator's display name (no PII beyond the name).
const publicCatalogSelect = `
    c.id,
    c.catalog_slug,
    c.course_code,
    c.title,
    c.description,
    c.hero_image_url,
    c.catalog_category,
    c.difficulty_level,
    c.catalog_language,
    c.price_cents,
    c.enrollment_count,
    c.average_rating,
    c.rating_count,
    NULLIF(TRIM(COALESCE(u.display_name,
        TRIM(COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')))), '') AS instructor_name,
    c.created_at::text
`

// publicCatalogFrom restricts to published, non-archived, public courses and joins
// the creating user for the instructor name.
const publicCatalogFrom = `
    FROM course.courses c
    LEFT JOIN "user".users u ON u.id = c.created_by_user_id
    WHERE c.is_public = TRUE
      AND c.published = TRUE
      AND c.archived = FALSE
`

func scanPublicCatalogCourse(scan func(dest ...any) error) (PublicCatalogCourse, error) {
	var c PublicCatalogCourse
	err := scan(
		&c.ID, &c.Slug, &c.CourseCode, &c.Title, &c.Description, &c.HeroImageURL,
		&c.Category, &c.DifficultyLevel, &c.Language, &c.PriceCents,
		&c.EnrollmentCount, &c.AverageRating, &c.RatingCount, &c.InstructorName, &c.CreatedAt,
	)
	return c, err
}

// ListPublicCatalog returns a page of public catalog courses matching the filter,
// the total match count, and the opaque cursor for the next page (empty when none).
func ListPublicCatalog(
	ctx context.Context,
	pool *pgxpool.Pool,
	f PublicCatalogFilter,
) ([]PublicCatalogCourse, int, string, error) {
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
	if f.PriceMax != nil {
		where.WriteString("\n      AND c.price_cents <= " + add(*f.PriceMax))
	}

	// Ordering. Relevance requires a query; otherwise it degrades to popularity.
	var orderBy string
	switch f.Sort {
	case CatalogSortRating:
		orderBy = "c.average_rating DESC NULLS LAST, c.enrollment_count DESC, c.created_at DESC"
	case CatalogSortNewest:
		orderBy = "c.created_at DESC, c.id DESC"
	case CatalogSortRelevance:
		if q != "" {
			rankPh := add(q)
			orderBy = fmt.Sprintf(
				"ts_rank(c.search_vector, websearch_to_tsquery('english', %s)) DESC, c.enrollment_count DESC, c.created_at DESC",
				rankPh)
		} else {
			orderBy = "c.enrollment_count DESC, c.created_at DESC"
		}
	default: // popular
		orderBy = "c.enrollment_count DESC, c.created_at DESC, c.id DESC"
	}

	limitPh := add(limit + 1) // fetch one extra to detect a next page
	offsetPh := add(offset)

	query := "SELECT " + publicCatalogSelect + publicCatalogFrom + where.String() +
		"\n    ORDER BY " + orderBy + "\n    LIMIT " + limitPh + " OFFSET " + offsetPh

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, "", err
	}
	defer rows.Close()

	out := make([]PublicCatalogCourse, 0, limit)
	for rows.Next() {
		c, err := scanPublicCatalogCourse(rows.Scan)
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

	// Total count for the "X courses found" UI. Reuse the same filter args (sans
	// the rank/limit/offset placeholders, which the COUNT query does not reference).
	var countArgs []any
	if f.Sort == CatalogSortRelevance && q != "" {
		// drop the rank placeholder and limit/offset
		countArgs = args[:len(args)-3]
	} else {
		countArgs = args[:len(args)-2]
	}
	countQuery := "SELECT COUNT(*)" + publicCatalogFrom + where.String()
	var total int
	if err := pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, "", err
	}

	return out, total, nextCursor, nil
}

// GetPublicCourseBySlug returns one public catalog course by slug (or course_code
// fallback), or nil when no published public course matches.
func GetPublicCourseBySlug(
	ctx context.Context,
	pool *pgxpool.Pool,
	slug string,
) (*PublicCatalogCourse, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, nil
	}
	query := "SELECT " + publicCatalogSelect + publicCatalogFrom +
		"\n      AND (lower(c.catalog_slug) = lower($1) OR lower(c.course_code) = lower($1))\n    LIMIT 1"
	c, err := scanPublicCatalogCourse(pool.QueryRow(ctx, query, slug).Scan)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

// ListCatalogCategories returns the distinct catalog categories among published
// public courses, with a per-category course count, ordered by count desc.
func ListCatalogCategories(ctx context.Context, pool *pgxpool.Pool) ([]CatalogCategory, error) {
	rows, err := pool.Query(ctx, `
SELECT c.catalog_category, COUNT(*) AS n
FROM course.courses c
WHERE c.is_public = TRUE
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
