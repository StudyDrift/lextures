// Package platformcourses provides instance-wide course search and admin access for global admins.
package platformcourses

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/courseroles"
)

// CourseRow is a search result row.
type CourseRow struct {
	ID              string    `json:"id"`
	CourseCode      string    `json:"courseCode"`
	Title           string    `json:"title"`
	Status          string    `json:"status"`
	OrgID           string    `json:"orgId"`
	OrgName         string    `json:"orgName"`
	InstructorName  *string   `json:"instructorName"`
	TermID          *string   `json:"termId"`
	TermName        *string   `json:"termName"`
	EnrollmentCount int64     `json:"enrollmentCount"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// ListParams holds search pagination options.
type ListParams struct {
	Query   string
	Status  string
	Page    int
	PerPage int
}

// ListResult is a paginated search response.
type ListResult struct {
	Items      []CourseRow `json:"items"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PerPage    int         `json:"perPage"`
	TotalPages int         `json:"totalPages"`
}

// Report is the full course report payload.
type Report struct {
	ID              string    `json:"id"`
	CourseCode      string    `json:"courseCode"`
	Title           string    `json:"title"`
	Description     *string   `json:"description"`
	Status          string    `json:"status"`
	OrgID           string    `json:"orgId"`
	OrgName         string    `json:"orgName"`
	InstructorName  *string   `json:"instructorName"`
	TermID          *string   `json:"termId"`
	TermName        *string   `json:"termName"`
	EnrollmentCount int64     `json:"enrollmentCount"`
	Published       bool      `json:"published"`
	Archived        bool      `json:"archived"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

func normalizePagination(page, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	switch perPage {
	case 25, 50, 100:
	default:
		perPage = 25
	}
	return page, perPage
}

func courseStatus(archived, published bool) string {
	if archived {
		return "archived"
	}
	if !published {
		return "draft"
	}
	return "active"
}

func statusFilterSQL(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "all":
		return ""
	case "active":
		return " AND c.archived = false AND c.published = true"
	case "archived":
		return " AND c.archived = true"
	case "draft":
		return " AND c.archived = false AND c.published = false"
	default:
		// "open" and empty: exclude archived (active + draft).
		return " AND c.archived = false"
	}
}

// Search returns courses matching free text across the instance. Requires a non-empty query.
func Search(ctx context.Context, pool *pgxpool.Pool, p ListParams) (ListResult, error) {
	q := strings.TrimSpace(p.Query)
	if q == "" {
		return ListResult{Items: []CourseRow{}}, nil
	}
	page, perPage := normalizePagination(p.Page, p.PerPage)
	offset := (page - 1) * perPage
	like := "%" + strings.ToLower(q) + "%"
	statusClause := statusFilterSQL(p.Status)

	rows, err := pool.Query(ctx, fmt.Sprintf(`
WITH matched AS (
    SELECT
        c.id,
        c.course_code,
        c.title,
        c.archived,
        c.published,
        c.org_id,
        o.name AS org_name,
        COALESCE(NULLIF(TRIM(COALESCE(inst.display_name, '')), ''), inst.email) AS instructor_name,
        c.term_id,
        t.name AS term_name,
        (SELECT COUNT(*)::bigint FROM course.course_enrollments ce
         WHERE ce.course_id = c.id AND ce.active) AS enrollment_count,
        c.created_at,
        c.updated_at,
        GREATEST(
            ts_rank(c.search_vector, websearch_to_tsquery('english', $1)),
            CASE WHEN similarity(c.title, $1) >= 0.3 THEN similarity(c.title, $1) ELSE 0 END,
            CASE WHEN similarity(c.course_code, $1) >= 0.3 THEN similarity(c.course_code, $1) ELSE 0 END,
            CASE WHEN c.course_code ILIKE $2 OR c.title ILIKE $2 THEN 0.35 ELSE 0 END
        ) AS rank
    FROM course.courses c
    INNER JOIN tenant.organizations o ON o.id = c.org_id
    LEFT JOIN "user".users inst ON inst.id = c.created_by_user_id
    LEFT JOIN tenant.terms t ON t.id = c.term_id
    WHERE (
          c.search_vector @@ websearch_to_tsquery('english', $1)
          OR c.course_code ILIKE $2
          OR c.title ILIKE $2
          OR similarity(c.title, $1) >= 0.3
          OR similarity(c.course_code, $1) >= 0.3
      )%s
)
SELECT
    id::text,
    course_code,
    title,
    archived,
    published,
    org_id::text,
    org_name,
    instructor_name,
    term_id::text,
    term_name,
    enrollment_count,
    created_at,
    updated_at,
    COUNT(*) OVER () AS total
FROM matched
ORDER BY rank DESC, title ASC
LIMIT $3 OFFSET $4
`, statusClause), q, like, perPage, offset)
	if err != nil {
		return ListResult{}, err
	}
	defer rows.Close()

	var items []CourseRow
	var total int64
	for rows.Next() {
		var row CourseRow
		var archived, published bool
		if err := rows.Scan(
			&row.ID, &row.CourseCode, &row.Title, &archived, &published,
			&row.OrgID, &row.OrgName, &row.InstructorName, &row.TermID, &row.TermName,
			&row.EnrollmentCount, &row.CreatedAt, &row.UpdatedAt, &total,
		); err != nil {
			return ListResult{}, err
		}
		row.Status = courseStatus(archived, published)
		items = append(items, row)
	}
	if items == nil {
		items = []CourseRow{}
	}
	totalPages := int(total) / perPage
	if int(total)%perPage != 0 {
		totalPages++
	}
	return ListResult{
		Items: items, Total: total, Page: page, PerPage: perPage, TotalPages: totalPages,
	}, rows.Err()
}

// CourseReport returns profile details for one course.
func CourseReport(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (*Report, error) {
	var rep Report
	var archived, published bool
	err := pool.QueryRow(ctx, `
SELECT
    c.id::text,
    c.course_code,
    c.title,
    c.description,
    c.archived,
    c.published,
    c.org_id::text,
    o.name,
    COALESCE(NULLIF(TRIM(COALESCE(inst.display_name, '')), ''), inst.email),
    c.term_id::text,
    t.name,
    (SELECT COUNT(*)::bigint FROM course.course_enrollments ce
     WHERE ce.course_id = c.id AND ce.active),
    c.created_at,
    c.updated_at
FROM course.courses c
INNER JOIN tenant.organizations o ON o.id = c.org_id
LEFT JOIN "user".users inst ON inst.id = c.created_by_user_id
LEFT JOIN tenant.terms t ON t.id = c.term_id
WHERE c.id = $1
`, courseID).Scan(
		&rep.ID, &rep.CourseCode, &rep.Title, &rep.Description, &archived, &published,
		&rep.OrgID, &rep.OrgName, &rep.InstructorName, &rep.TermID, &rep.TermName,
		&rep.EnrollmentCount, &rep.CreatedAt, &rep.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	rep.Archived = archived
	rep.Published = published
	rep.Status = courseStatus(archived, published)
	return &rep, nil
}

// EnsureAdminAccess enrolls the user as a teacher when needed and refreshes course grants.
func EnsureAdminAccess(ctx context.Context, pool *pgxpool.Pool, courseID, userID uuid.UUID) error {
	var courseCode string
	if err := pool.QueryRow(ctx, `SELECT course_code FROM course.courses WHERE id = $1`, courseID).Scan(&courseCode); err != nil {
		return err
	}

	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
UPDATE course.course_enrollments
SET active = true, invitation_pending = false
WHERE course_id = $1 AND user_id = $2 AND role = 'teacher'
`, courseID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		var enrollmentID uuid.UUID
		err = tx.QueryRow(ctx, `
INSERT INTO course.course_enrollments (course_id, user_id, role, active, invitation_pending)
VALUES ($1, $2, 'teacher', true, false)
ON CONFLICT (course_id, user_id, role) DO NOTHING
RETURNING id
`, courseID, userID).Scan(&enrollmentID)
		if err != nil && err != pgx.ErrNoRows {
			return err
		}
		if err == pgx.ErrNoRows {
			// Race or unexpected conflict — try re-activate again.
			if _, err := tx.Exec(ctx, `
UPDATE course.course_enrollments
SET active = true, invitation_pending = false
WHERE course_id = $1 AND user_id = $2 AND role = 'teacher'
`, courseID, userID); err != nil {
				return err
			}
		}
	}
	if err := courseroles.RefreshManagedGrantsForCourseUser(ctx, tx, userID, courseID, courseCode); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

// LookupCourseID resolves a course UUID from its primary key string.
func LookupCourseID(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) error {
	var exists bool
	err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM course.courses WHERE id = $1)`, courseID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("platformcourses: course not found")
	}
	return nil
}