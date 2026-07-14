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

// rowQuerier is implemented by *pgxpool.Pool and pgx.Tx.
type rowQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// DashboardStats holds instance-wide course metrics for the admin Courses page.
type DashboardStats struct {
	CreatedLast7Days int64 `json:"createdLast7Days"`
	ActiveCourses    int64 `json:"activeCourses"`
	DraftCourses     int64 `json:"draftCourses"`
	TotalCourses     int64 `json:"totalCourses"`
	ArchivedCourses  int64 `json:"archivedCourses"`
}

// FetchDashboardStats returns aggregate course metrics for the admin dashboard.
func FetchDashboardStats(ctx context.Context, pool *pgxpool.Pool) (DashboardStats, error) {
	return fetchDashboardStats(ctx, pool)
}

func fetchDashboardStats(ctx context.Context, q rowQuerier) (DashboardStats, error) {
	var stats DashboardStats
	err := q.QueryRow(ctx, `
SELECT
    COUNT(*)::bigint AS total_courses,
    COUNT(*) FILTER (
        WHERE c.archived = false AND c.published = true
    )::bigint AS active_courses,
    COUNT(*) FILTER (
        WHERE c.archived = false AND c.published = false
    )::bigint AS draft_courses,
    COUNT(*) FILTER (
        WHERE c.archived = true
    )::bigint AS archived_courses,
    COUNT(*) FILTER (
        WHERE c.created_at >= NOW() - INTERVAL '7 days'
    )::bigint AS created_last_7_days
FROM course.courses c
`).Scan(
		&stats.TotalCourses,
		&stats.ActiveCourses,
		&stats.DraftCourses,
		&stats.ArchivedCourses,
		&stats.CreatedLast7Days,
	)
	if err != nil {
		return DashboardStats{}, err
	}
	return stats, nil
}

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

// Dashboard filter query values for listing courses behind a stats card.
const (
	FilterCreated7d = "created_7d"
	FilterActive    = "active"
	FilterDraft     = "draft"
	FilterTotal     = "total"
	FilterArchived  = "archived"
)

// ListParams holds search / filter pagination options.
type ListParams struct {
	Query  string
	Status string
	// Filter restricts results to a dashboard segment (see Filter* constants).
	// Empty means free-text search only (Query required).
	Filter  string
	Page    int
	PerPage int
}

// NormalizeFilter returns a known filter constant or empty string if unknown/blank.
func NormalizeFilter(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case FilterCreated7d, "created_last_7_days", "new_courses":
		return FilterCreated7d
	case FilterActive, "active_courses":
		return FilterActive
	case FilterDraft, "draft_courses":
		return FilterDraft
	case FilterTotal, "total_courses":
		return FilterTotal
	case FilterArchived, "archived_courses":
		return FilterArchived
	default:
		return ""
	}
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
	return scanCourseList(rows, page, perPage)
}

// ListByFilter returns courses for a dashboard stats segment (optionally narrowed by free text).
func ListByFilter(ctx context.Context, pool *pgxpool.Pool, p ListParams) (ListResult, error) {
	filter := NormalizeFilter(p.Filter)
	if filter == "" {
		return ListResult{}, fmt.Errorf("platformcourses: invalid filter %q", p.Filter)
	}
	page, perPage := normalizePagination(p.Page, p.PerPage)
	offset := (page - 1) * perPage
	q := strings.TrimSpace(p.Query)

	// Predicates match FetchDashboardStats segments.
	where := `TRUE`
	switch filter {
	case FilterCreated7d:
		where = `c.created_at >= NOW() - INTERVAL '7 days'`
	case FilterActive:
		where = `c.archived = false AND c.published = true`
	case FilterDraft:
		where = `c.archived = false AND c.published = false`
	case FilterArchived:
		where = `c.archived = true`
	case FilterTotal:
		// no extra predicate
	}

	args := []any{}
	if q != "" {
		like := "%" + strings.ToLower(q) + "%"
		args = append(args, like)
		where += fmt.Sprintf(`
AND (
    c.course_code ILIKE $%d
    OR c.title ILIKE $%d
)`, len(args), len(args))
	}
	args = append(args, perPage, offset)
	limitPh := len(args) - 1
	offsetPh := len(args)

	sql := fmt.Sprintf(`
SELECT
    c.id::text,
    c.course_code,
    c.title,
    c.archived,
    c.published,
    c.org_id::text,
    o.name AS org_name,
    COALESCE(NULLIF(TRIM(COALESCE(inst.display_name, '')), ''), inst.email) AS instructor_name,
    c.term_id::text,
    t.name AS term_name,
    (SELECT COUNT(*)::bigint FROM course.course_enrollments ce
     WHERE ce.course_id = c.id AND ce.active) AS enrollment_count,
    c.created_at,
    c.updated_at,
    COUNT(*) OVER () AS total
FROM course.courses c
INNER JOIN tenant.organizations o ON o.id = c.org_id
LEFT JOIN "user".users inst ON inst.id = c.created_by_user_id
LEFT JOIN tenant.terms t ON t.id = c.term_id
WHERE %s
ORDER BY c.created_at DESC, c.title ASC
LIMIT $%d OFFSET $%d
`, where, limitPh, offsetPh)

	rows, err := pool.Query(ctx, sql, args...)
	if err != nil {
		return ListResult{}, err
	}
	defer rows.Close()
	return scanCourseList(rows, page, perPage)
}

func scanCourseList(rows pgx.Rows, page, perPage int) (ListResult, error) {
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