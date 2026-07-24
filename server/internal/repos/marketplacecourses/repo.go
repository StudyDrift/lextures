package marketplacecourses

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CourseLedgerRow is a provisioned official marketplace course.
type CourseLedgerRow struct {
	Slug           string
	CourseID       uuid.UUID
	ContentVersion int
	ProvisionedAt  time.Time
	UpdatedAt      time.Time
}

// LookupBySlug returns the ledger row for slug, or nil when absent.
func LookupBySlug(ctx context.Context, pool *pgxpool.Pool, slug string) (*CourseLedgerRow, error) {
	var r CourseLedgerRow
	err := pool.QueryRow(ctx, `
SELECT slug, course_id, content_version, provisioned_at, updated_at
FROM settings.marketplace_courses
WHERE slug = $1
`, slug).Scan(&r.Slug, &r.CourseID, &r.ContentVersion, &r.ProvisionedAt, &r.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// LookupBySlugTx is the transactional variant used under the provision advisory lock.
func LookupBySlugTx(ctx context.Context, tx pgx.Tx, slug string) (*CourseLedgerRow, error) {
	var r CourseLedgerRow
	err := tx.QueryRow(ctx, `
SELECT slug, course_id, content_version, provisioned_at, updated_at
FROM settings.marketplace_courses
WHERE slug = $1
`, slug).Scan(&r.Slug, &r.CourseID, &r.ContentVersion, &r.ProvisionedAt, &r.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// UpsertLedger records or updates the course-level provisioning ledger.
func UpsertLedger(ctx context.Context, tx pgx.Tx, slug string, courseID uuid.UUID, contentVersion int) error {
	_, err := tx.Exec(ctx, `
INSERT INTO settings.marketplace_courses (slug, course_id, content_version, provisioned_at, updated_at)
VALUES ($1, $2, $3, NOW(), NOW())
ON CONFLICT (slug) DO UPDATE SET
    course_id = EXCLUDED.course_id,
    content_version = EXCLUDED.content_version,
    updated_at = NOW()
`, slug, courseID, contentVersion)
	return err
}

// EnsureSystemPublisher verifies the migration-seeded publisher user exists.
func EnsureSystemPublisher(ctx context.Context, tx pgx.Tx) error {
	var ok bool
	err := tx.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1 FROM "user".users
    WHERE id = $1 AND account_type = 'system'
)
`, SystemPublisherID).Scan(&ok)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("marketplace course system publisher missing; run migrations")
	}
	return nil
}

// DefaultOrgID returns the default tenant org id.
func DefaultOrgID(ctx context.Context, tx pgx.Tx) (uuid.UUID, error) {
	var id uuid.UUID
	err := tx.QueryRow(ctx, `
SELECT id FROM tenant.organizations WHERE slug = 'default' LIMIT 1
`).Scan(&id)
	return id, err
}

// CreateCourseParams are fields for inserting an official marketplace course.
type CreateCourseParams struct {
	CourseCode        string
	ShortCode         string
	Title             string
	Description       string
	CatalogSlug       string
	CatalogCategory   string
	DifficultyLevel   string
	CatalogLanguage   string
	PriceCents        int
	IsPublic          bool
	MarketplaceListed bool
	OrgID             uuid.UUID
	CreatedBy         uuid.UUID
	Now               time.Time
}

// CreateCourse inserts an official marketplace course row.
func CreateCourse(ctx context.Context, tx pgx.Tx, p CreateCourseParams) (uuid.UUID, error) {
	var id uuid.UUID
	err := tx.QueryRow(ctx, `
INSERT INTO course.courses (
    course_code,
    short_code,
    title,
    description,
    course_type,
    created_by_user_id,
    org_id,
    published,
    visible_from,
    grading_scale,
    is_official,
    is_public,
    catalog_slug,
    catalog_category,
    difficulty_level,
    catalog_language,
    price_cents,
    price_currency,
    marketplace_listed,
    marketplace_listed_at,
    course_mode,
    open_enrollment,
    module_gating_enabled
) VALUES (
    $1, $2, $3, $4, 'traditional', $5, $6, TRUE, $7::timestamptz, 'letter_plus_minus',
    TRUE, $8, $9, $10, $11, $12, $13, 'usd', $14,
    CASE WHEN $14 THEN $7::timestamptz ELSE NULL END,
    'self_paced', TRUE, FALSE
)
RETURNING id
`, p.CourseCode, p.ShortCode, p.Title, p.Description, p.CreatedBy, p.OrgID, p.Now,
		p.IsPublic, p.CatalogSlug, nullIfEmpty(p.CatalogCategory), nullIfEmpty(p.DifficultyLevel),
		p.CatalogLanguage, p.PriceCents, p.MarketplaceListed).Scan(&id)
	return id, err
}

// ReconcileCourseParams are fields updated on re-provision without clobbering ratings/enrollments.
type ReconcileCourseParams struct {
	CourseID          uuid.UUID
	Title             string
	Description       string
	CatalogSlug       string
	CatalogCategory   string
	DifficultyLevel   string
	CatalogLanguage   string
	PriceCents        int
	IsPublic          bool
	MarketplaceListed bool
	Now               time.Time
}

// ReconcileCourse updates catalog/marketplace fields on an existing official course.
// Does not reset enrollment_count or average_rating.
func ReconcileCourse(ctx context.Context, tx pgx.Tx, p ReconcileCourseParams) error {
	_, err := tx.Exec(ctx, `
UPDATE course.courses
SET
    title = $2,
    description = $3,
    published = TRUE,
    visible_from = COALESCE(visible_from, $4),
    starts_at = NULL,
    ends_at = NULL,
    hidden_at = NULL,
    grading_scale = 'letter_plus_minus',
    is_official = TRUE,
    is_public = $5,
    catalog_slug = $6,
    catalog_category = $7,
    difficulty_level = $8,
    catalog_language = $9,
    price_cents = $10,
    price_currency = 'usd',
    marketplace_listed = $11,
    marketplace_listed_at = CASE
        WHEN $11 AND marketplace_listed_at IS NULL THEN $4
        WHEN $11 THEN marketplace_listed_at
        ELSE NULL
    END,
    course_mode = 'self_paced',
    open_enrollment = TRUE,
    module_gating_enabled = FALSE,
    updated_at = NOW()
WHERE id = $1
`, p.CourseID, p.Title, p.Description, p.Now, p.IsPublic, p.CatalogSlug,
		nullIfEmpty(p.CatalogCategory), nullIfEmpty(p.DifficultyLevel),
		p.CatalogLanguage, p.PriceCents, p.MarketplaceListed)
	return err
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// EnsureTeacherEnrollment enrolls the system publisher as teacher with grants.
func EnsureTeacherEnrollment(ctx context.Context, tx pgx.Tx, courseID, teacherID uuid.UUID, courseCode string) error {
	if _, err := tx.Exec(ctx, `
INSERT INTO course.course_enrollments (course_id, user_id, role)
VALUES ($1, $2, 'teacher')
ON CONFLICT (course_id, user_id, role) DO NOTHING
`, courseID, teacherID); err != nil {
		return err
	}
	return seedTeacherGrants(ctx, tx, teacherID, courseID, courseCode)
}

func seedTeacherGrants(ctx context.Context, tx pgx.Tx, userID, courseID uuid.UUID, courseCode string) error {
	prefix := "course:" + courseCode + ":"
	perms := []string{
		prefix + "item:create",
		prefix + "items:create",
		prefix + "enrollments:read",
		prefix + "enrollments:update",
		prefix + "gradebook:view",
		prefix + "attendance:manage",
	}
	for _, perm := range perms {
		if _, err := tx.Exec(ctx, `
INSERT INTO course.user_course_grants (user_id, course_id, permission_string)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, course_id, permission_string) DO NOTHING
`, userID, courseID, perm); err != nil {
			return err
		}
	}
	return nil
}

type assignmentGroupSpec struct {
	SortOrder     int
	Name          string
	WeightPercent float64
}

var defaultAssignmentGroups = []assignmentGroupSpec{
	{0, "Participation", 10},
	{1, "Quizzes", 50},
	{2, "Assignments", 40},
}

// EnsureAssignmentGroups seeds the default weighted groups.
func EnsureAssignmentGroups(ctx context.Context, tx pgx.Tx, courseID uuid.UUID) error {
	for _, g := range defaultAssignmentGroups {
		if _, err := tx.Exec(ctx, `
INSERT INTO course.assignment_groups (course_id, sort_order, name, weight_percent)
VALUES ($1, $2, $3, $4)
ON CONFLICT (course_id, sort_order) DO UPDATE SET
    name = EXCLUDED.name,
    weight_percent = EXCLUDED.weight_percent,
    updated_at = NOW()
`, courseID, g.SortOrder, g.Name, g.WeightPercent); err != nil {
			return err
		}
	}
	return nil
}

// AssignmentGroupIDByName resolves an assignment group id by display name.
func AssignmentGroupIDByName(ctx context.Context, tx pgx.Tx, courseID uuid.UUID, name string) (*uuid.UUID, error) {
	var id uuid.UUID
	err := tx.QueryRow(ctx, `
SELECT id FROM course.assignment_groups WHERE course_id = $1 AND name = $2
`, courseID, name).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// SyncLearningOutcomes reconciles course learning outcomes with the manifest list.
// No-ops when titles and sort order already match (avoids churn on re-provision).
func SyncLearningOutcomes(ctx context.Context, tx pgx.Tx, courseID uuid.UUID, outcomes []string) error {
	want := make([]string, 0, len(outcomes))
	for _, title := range outcomes {
		if t := strings.TrimSpace(title); t != "" {
			want = append(want, t)
		}
	}
	rows, err := tx.Query(ctx, `
SELECT title FROM course.course_learning_outcomes
WHERE course_id = $1
ORDER BY sort_order, title
`, courseID)
	if err != nil {
		return err
	}
	var have []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			rows.Close()
			return err
		}
		have = append(have, t)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(have) == len(want) {
		same := true
		for i := range want {
			if have[i] != want[i] {
				same = false
				break
			}
		}
		if same {
			return nil
		}
	}
	if _, err := tx.Exec(ctx, `
DELETE FROM course.course_learning_outcomes WHERE course_id = $1
`, courseID); err != nil {
		return err
	}
	for i, t := range want {
		if _, err := tx.Exec(ctx, `
INSERT INTO course.course_learning_outcomes (course_id, title, description, sort_order)
VALUES ($1, $2, '', $3)
`, courseID, t, i); err != nil {
			return err
		}
	}
	return nil
}

// IsOfficialCourseID reports whether courseID is an official marketplace course.
func IsOfficialCourseID(ctx context.Context, q interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}, courseID uuid.UUID) (bool, error) {
	var ok bool
	err := q.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1 FROM course.courses WHERE id = $1 AND is_official = TRUE
)
`, courseID).Scan(&ok)
	return ok, err
}
