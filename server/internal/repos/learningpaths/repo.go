// Package learningpaths stores learning path bundles and path enrollments (plan 15.4).
package learningpaths

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

// Path is one learningpath.learning_paths row.
type Path struct {
	ID               uuid.UUID
	CreatorID        uuid.UUID
	Title            string
	Description      string
	Slug             *string
	BundlePriceCents *int
	StripeProductID  *string
	IsPublic         bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// PathCourse is a constituent course in a path.
type PathCourse struct {
	CourseID         uuid.UUID
	Position         int
	CourseCode       string
	Title            string
	Description      string
	ListPriceCents   *int
	DurationMinutes  int
	SkillTags        []string
}

// PathEnrollment is a learner path enrollment row.
type PathEnrollment struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	PathID      uuid.UUID
	EnrolledAt  time.Time
	CompletedAt *time.Time
}

// CatalogPathSummary is a public catalog list item.
type CatalogPathSummary struct {
	ID               uuid.UUID
	Title            string
	Description      string
	Slug             string
	BundlePriceCents *int
	CourseCount      int
	TotalDurationMin int
	SkillTags        []string
	IndividualTotal  int
}

// CatalogPathDetail is the public landing page payload.
type CatalogPathDetail struct {
	Path
	Courses         []PathCourse
	IndividualTotal int
	TotalDurationMin int
	SkillTags       []string
}

// SlugifyTitle produces a URL-safe slug from a title.
func SlugifyTitle(title string) string {
	s := strings.ToLower(strings.TrimSpace(title))
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == ' ' || r == '-' || r == '_':
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "path"
	}
	return out
}

// MintUniqueSlug returns a unique slug for a new path.
func MintUniqueSlug(ctx context.Context, pool *pgxpool.Pool, title string) (string, error) {
	base := SlugifyTitle(title)
	for i := 0; i < 8; i++ {
		candidate := base
		if i > 0 {
			candidate = fmt.Sprintf("%s-%d", base, i)
		}
		var exists bool
		err := pool.QueryRow(ctx, `
SELECT EXISTS (SELECT 1 FROM learningpath.learning_paths WHERE slug = $1)
`, candidate).Scan(&exists)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}
	return fmt.Sprintf("%s-%s", base, uuid.New().String()[:8]), nil
}

// CreatePath inserts a path and ordered courses in one transaction.
func CreatePath(ctx context.Context, pool *pgxpool.Pool, creatorID uuid.UUID, title, description string, courseIDs []uuid.UUID, bundlePriceCents *int, isPublic bool) (*Path, error) {
	if len(courseIDs) == 0 {
		return nil, errors.New("path must include at least one course")
	}
	if len(courseIDs) > 20 {
		return nil, errors.New("path cannot contain more than 20 courses")
	}
	slug, err := MintUniqueSlug(ctx, pool, title)
	if err != nil {
		return nil, err
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var p Path
	err = tx.QueryRow(ctx, `
INSERT INTO learningpath.learning_paths (creator_id, title, description, slug, bundle_price_cents, is_public)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, creator_id, title, description, slug, bundle_price_cents, stripe_product_id, is_public, created_at, updated_at
`, creatorID, title, description, slug, bundlePriceCents, isPublic).Scan(
		&p.ID, &p.CreatorID, &p.Title, &p.Description, &p.Slug, &p.BundlePriceCents,
		&p.StripeProductID, &p.IsPublic, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	for i, cid := range courseIDs {
		if _, err := tx.Exec(ctx, `
INSERT INTO learningpath.learning_path_courses (path_id, course_id, position)
VALUES ($1, $2, $3)
`, p.ID, cid, i); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &p, nil
}

// GetPathByID returns a path or nil.
func GetPathByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Path, error) {
	var p Path
	err := pool.QueryRow(ctx, `
SELECT id, creator_id, title, description, slug, bundle_price_cents, stripe_product_id, is_public, created_at, updated_at
FROM learningpath.learning_paths WHERE id = $1
`, id).Scan(
		&p.ID, &p.CreatorID, &p.Title, &p.Description, &p.Slug, &p.BundlePriceCents,
		&p.StripeProductID, &p.IsPublic, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// GetPathBySlug returns a public path by slug or nil.
func GetPathBySlug(ctx context.Context, pool *pgxpool.Pool, slug string) (*Path, error) {
	var p Path
	err := pool.QueryRow(ctx, `
SELECT id, creator_id, title, description, slug, bundle_price_cents, stripe_product_id, is_public, created_at, updated_at
FROM learningpath.learning_paths WHERE slug = $1 AND is_public = TRUE
`, slug).Scan(
		&p.ID, &p.CreatorID, &p.Title, &p.Description, &p.Slug, &p.BundlePriceCents,
		&p.StripeProductID, &p.IsPublic, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ListPathsByCreator returns paths owned by a creator.
func ListPathsByCreator(ctx context.Context, pool *pgxpool.Pool, creatorID uuid.UUID) ([]Path, error) {
	rows, err := pool.Query(ctx, `
SELECT id, creator_id, title, description, slug, bundle_price_cents, stripe_product_id, is_public, created_at, updated_at
FROM learningpath.learning_paths
WHERE creator_id = $1
ORDER BY updated_at DESC
`, creatorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPaths(rows)
}

// ListPublicPaths returns public catalog paths with optional search.
func ListPublicPaths(ctx context.Context, pool *pgxpool.Pool, q string, sort string) ([]CatalogPathSummary, error) {
	q = strings.TrimSpace(q)
	order := "lp.created_at DESC"
	switch strings.TrimSpace(sort) {
	case "title":
		order = "lp.title ASC"
	case "price":
		order = "lp.bundle_price_cents ASC NULLS LAST"
	}
	var rows pgx.Rows
	var err error
	if q != "" {
		pattern := "%" + q + "%"
		rows, err = pool.Query(ctx, fmt.Sprintf(`
SELECT lp.id, lp.title, lp.description, lp.slug, lp.bundle_price_cents,
       COUNT(lpc.course_id)::int AS course_count
FROM learningpath.learning_paths lp
LEFT JOIN learningpath.learning_path_courses lpc ON lpc.path_id = lp.id
WHERE lp.is_public = TRUE
  AND (lp.title ILIKE $1 OR lp.description ILIKE $1)
GROUP BY lp.id
ORDER BY %s
`, order), pattern)
	} else {
		rows, err = pool.Query(ctx, fmt.Sprintf(`
SELECT lp.id, lp.title, lp.description, lp.slug, lp.bundle_price_cents,
       COUNT(lpc.course_id)::int AS course_count
FROM learningpath.learning_paths lp
LEFT JOIN learningpath.learning_path_courses lpc ON lpc.path_id = lp.id
WHERE lp.is_public = TRUE
GROUP BY lp.id
ORDER BY %s
`, order))
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CatalogPathSummary
	for rows.Next() {
		var s CatalogPathSummary
		if err := rows.Scan(&s.ID, &s.Title, &s.Description, &s.Slug, &s.BundlePriceCents, &s.CourseCount); err != nil {
			return nil, err
		}
		detail, err := enrichPathSummary(ctx, pool, s.ID)
		if err != nil {
			return nil, err
		}
		s.TotalDurationMin = detail.TotalDurationMin
		s.SkillTags = detail.SkillTags
		s.IndividualTotal = detail.IndividualTotal
		out = append(out, s)
	}
	return out, rows.Err()
}

func enrichPathSummary(ctx context.Context, pool *pgxpool.Pool, pathID uuid.UUID) (CatalogPathSummary, error) {
	courses, err := ListPathCourses(ctx, pool, pathID)
	if err != nil {
		return CatalogPathSummary{}, err
	}
	var s CatalogPathSummary
	tagSet := map[string]struct{}{}
	for _, c := range courses {
		s.TotalDurationMin += c.DurationMinutes
		if c.ListPriceCents != nil {
			s.IndividualTotal += *c.ListPriceCents
		}
		for _, t := range c.SkillTags {
			tagSet[t] = struct{}{}
		}
	}
	for t := range tagSet {
		s.SkillTags = append(s.SkillTags, t)
	}
	return s, nil
}

// ListPathCourses returns ordered courses for a path.
func ListPathCourses(ctx context.Context, pool *pgxpool.Pool, pathID uuid.UUID) ([]PathCourse, error) {
	rows, err := pool.Query(ctx, `
SELECT lpc.course_id, lpc.position, c.course_code, c.title, COALESCE(c.description, ''),
       c.list_price_cents,
       COALESCE((
           SELECT COUNT(*)::int
           FROM course.course_structure_items csi
           WHERE csi.course_id = c.id AND csi.published = TRUE
       ), 0) * 15 AS duration_minutes,
       COALESCE((
           SELECT array_agg(DISTINCT lo.title ORDER BY lo.title)
           FROM course.course_learning_outcomes lo
           WHERE lo.course_id = c.id
           LIMIT 8
       ), '{}') AS skill_tags
FROM learningpath.learning_path_courses lpc
INNER JOIN course.courses c ON c.id = lpc.course_id
WHERE lpc.path_id = $1
ORDER BY lpc.position ASC
`, pathID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PathCourse
	for rows.Next() {
		var pc PathCourse
		if err := rows.Scan(
			&pc.CourseID, &pc.Position, &pc.CourseCode, &pc.Title, &pc.Description,
			&pc.ListPriceCents, &pc.DurationMinutes, &pc.SkillTags,
		); err != nil {
			return nil, err
		}
		out = append(out, pc)
	}
	return out, rows.Err()
}

// GetCatalogDetail loads a public path landing page.
func GetCatalogDetail(ctx context.Context, pool *pgxpool.Pool, slug string) (*CatalogPathDetail, error) {
	p, err := GetPathBySlug(ctx, pool, slug)
	if err != nil || p == nil {
		return nil, err
	}
	courses, err := ListPathCourses(ctx, pool, p.ID)
	if err != nil {
		return nil, err
	}
	d := &CatalogPathDetail{Path: *p, Courses: courses}
	tagSet := map[string]struct{}{}
	for _, c := range courses {
		d.TotalDurationMin += c.DurationMinutes
		if c.ListPriceCents != nil {
			d.IndividualTotal += *c.ListPriceCents
		}
		for _, t := range c.SkillTags {
			tagSet[t] = struct{}{}
		}
	}
	for t := range tagSet {
		d.SkillTags = append(d.SkillTags, t)
	}
	return d, nil
}

// UpdatePath patches path metadata and optionally replaces course order.
func UpdatePath(ctx context.Context, pool *pgxpool.Pool, pathID, creatorID uuid.UUID, title, description *string, bundlePriceCents *int, bundlePriceSet bool, isPublic *bool, courseIDs []uuid.UUID) (*Path, error) {
	p, err := GetPathByID(ctx, pool, pathID)
	if err != nil {
		return nil, err
	}
	if p == nil || p.CreatorID != creatorID {
		return nil, nil
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if title != nil || description != nil || bundlePriceSet || isPublic != nil {
		_, err = tx.Exec(ctx, `
UPDATE learningpath.learning_paths
SET title = COALESCE($2, title),
    description = COALESCE($3, description),
    bundle_price_cents = CASE WHEN $4 THEN $5 ELSE bundle_price_cents END,
    is_public = COALESCE($6, is_public),
    updated_at = NOW()
WHERE id = $1
`, pathID, title, description, bundlePriceSet, bundlePriceCents, isPublic)
		if err != nil {
			return nil, err
		}
	}
	if courseIDs != nil {
		if len(courseIDs) == 0 {
			return nil, errors.New("path must include at least one course")
		}
		if len(courseIDs) > 20 {
			return nil, errors.New("path cannot contain more than 20 courses")
		}
		if _, err := tx.Exec(ctx, `DELETE FROM learningpath.learning_path_courses WHERE path_id = $1`, pathID); err != nil {
			return nil, err
		}
		for i, cid := range courseIDs {
			if _, err := tx.Exec(ctx, `
INSERT INTO learningpath.learning_path_courses (path_id, course_id, position) VALUES ($1, $2, $3)
`, pathID, cid, i); err != nil {
				return nil, err
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return GetPathByID(ctx, pool, pathID)
}

// DeletePath removes a path owned by creator.
func DeletePath(ctx context.Context, pool *pgxpool.Pool, pathID, creatorID uuid.UUID) (bool, error) {
	tag, err := pool.Exec(ctx, `
DELETE FROM learningpath.learning_paths WHERE id = $1 AND creator_id = $2
`, pathID, creatorID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// UserTeachesCourse returns true when user has an active teacher enrollment.
func UserTeachesCourse(ctx context.Context, pool *pgxpool.Pool, userID, courseID uuid.UUID) (bool, error) {
	var ok bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM course.course_enrollments ce
  WHERE ce.course_id = $1 AND ce.user_id = $2 AND ce.active AND ce.role = 'teacher'
)
`, courseID, userID).Scan(&ok)
	return ok, err
}

// GetPathEnrollment returns enrollment for user+path or nil.
func GetPathEnrollment(ctx context.Context, pool *pgxpool.Pool, userID, pathID uuid.UUID) (*PathEnrollment, error) {
	var e PathEnrollment
	err := pool.QueryRow(ctx, `
SELECT id, user_id, path_id, enrolled_at, completed_at
FROM learningpath.path_enrollments
WHERE user_id = $1 AND path_id = $2
`, userID, pathID).Scan(&e.ID, &e.UserID, &e.PathID, &e.EnrolledAt, &e.CompletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// ListPathEnrollmentsByUser returns all path enrollments for a learner.
func ListPathEnrollmentsByUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]PathEnrollment, error) {
	rows, err := pool.Query(ctx, `
SELECT id, user_id, path_id, enrolled_at, completed_at
FROM learningpath.path_enrollments
WHERE user_id = $1
ORDER BY enrolled_at DESC
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PathEnrollment
	for rows.Next() {
		var e PathEnrollment
		if err := rows.Scan(&e.ID, &e.UserID, &e.PathID, &e.EnrolledAt, &e.CompletedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// MarkPathCompleted sets completed_at when nil.
func MarkPathCompleted(ctx context.Context, pool *pgxpool.Pool, enrollmentID uuid.UUID) (bool, error) {
	tag, err := pool.Exec(ctx, `
UPDATE learningpath.path_enrollments
SET completed_at = NOW()
WHERE id = $1 AND completed_at IS NULL
`, enrollmentID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func scanPaths(rows pgx.Rows) ([]Path, error) {
	var out []Path
	for rows.Next() {
		var p Path
		if err := rows.Scan(
			&p.ID, &p.CreatorID, &p.Title, &p.Description, &p.Slug, &p.BundlePriceCents,
			&p.StripeProductID, &p.IsPublic, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
