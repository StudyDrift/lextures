// Package platformpeople provides instance-wide user search and reports for global admins.
package platformpeople

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)



// PersonRow is a search result row.
type PersonRow struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	FirstName   *string   `json:"firstName"`
	LastName    *string   `json:"lastName"`
	DisplayName *string   `json:"displayName"`
	OrgID       string    `json:"orgId"`
	OrgName     string    `json:"orgName"`
	Role        string    `json:"role"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"createdAt"`
}

// ListParams holds search pagination options.
type ListParams struct {
	Query   string
	Page    int
	PerPage int
}

// ListResult is a paginated search response.
type ListResult struct {
	Items      []PersonRow `json:"items"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PerPage    int         `json:"perPage"`
	TotalPages int         `json:"totalPages"`
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

// Search returns users matching free text across the instance. Requires a non-empty query.
func Search(ctx context.Context, pool *pgxpool.Pool, p ListParams) (ListResult, error) {
	q := strings.TrimSpace(p.Query)
	if q == "" {
		return ListResult{Items: []PersonRow{}}, nil
	}
	page, perPage := normalizePagination(p.Page, p.PerPage)
	offset := (page - 1) * perPage
	like := "%" + strings.ToLower(q) + "%"

	rows, err := pool.Query(ctx, `
WITH matched AS (
    SELECT
        u.id,
        u.email,
        u.first_name,
        u.last_name,
        u.display_name,
        u.org_id,
        o.name AS org_name,
        COALESCE((
            SELECT ar.name
            FROM "user".user_app_roles uar
            JOIN "user".app_roles ar ON ar.id = uar.role_id
            WHERE uar.user_id = u.id
            ORDER BY ar.name
            LIMIT 1
        ), '') AS role_name,
        (u.deactivated_at IS NULL AND NOT u.login_blocked) AS active,
        u.created_at,
        GREATEST(
            ts_rank(u.search_vector, websearch_to_tsquery('english', $1)),
            CASE WHEN similarity(u.email, $1) >= 0.3 THEN similarity(u.email, $1) ELSE 0 END,
            CASE WHEN similarity(COALESCE(u.first_name, ''), $1) >= 0.3 THEN similarity(COALESCE(u.first_name, ''), $1) ELSE 0 END,
            CASE WHEN similarity(COALESCE(u.last_name, ''), $1) >= 0.3 THEN similarity(COALESCE(u.last_name, ''), $1) ELSE 0 END,
            CASE WHEN u.email ILIKE $2 OR COALESCE(u.first_name, '') ILIKE $2
                OR COALESCE(u.last_name, '') ILIKE $2
                OR COALESCE(u.display_name, '') ILIKE $2 THEN 0.35 ELSE 0 END
        ) AS rank
    FROM "user".users u
    INNER JOIN tenant.organizations o ON o.id = u.org_id
    WHERE u.account_type <> 'system'
      AND (
          u.search_vector @@ websearch_to_tsquery('english', $1)
          OR u.email ILIKE $2
          OR COALESCE(u.first_name, '') ILIKE $2
          OR COALESCE(u.last_name, '') ILIKE $2
          OR similarity(u.email, $1) >= 0.3
          OR similarity(COALESCE(u.first_name, ''), $1) >= 0.3
          OR similarity(COALESCE(u.last_name, ''), $1) >= 0.3
          OR similarity(COALESCE(u.display_name, ''), $1) >= 0.3
      )
)
SELECT
    id::text,
    email,
    first_name,
    last_name,
    display_name,
    org_id::text,
    org_name,
    role_name,
    active,
    created_at,
    COUNT(*) OVER () AS total
FROM matched
ORDER BY rank DESC, email ASC
LIMIT $3 OFFSET $4
`, q, like, perPage, offset)
	if err != nil {
		return ListResult{}, err
	}
	defer rows.Close()

	var items []PersonRow
	var total int64
	for rows.Next() {
		var row PersonRow
		var roleName string
		if err := rows.Scan(
			&row.ID, &row.Email, &row.FirstName, &row.LastName, &row.DisplayName,
			&row.OrgID, &row.OrgName, &roleName, &row.Active, &row.CreatedAt, &total,
		); err != nil {
			return ListResult{}, err
		}
		row.Role = appRoleToCli(roleName)
		items = append(items, row)
	}
	if items == nil {
		items = []PersonRow{}
	}
	totalPages := int(total) / perPage
	if int(total)%perPage != 0 {
		totalPages++
	}
	return ListResult{
		Items: items, Total: total, Page: page, PerPage: perPage, TotalPages: totalPages,
	}, rows.Err()
}

// EnrollmentRow is one course enrollment on a user report.
type EnrollmentRow struct {
	CourseID     string  `json:"courseId"`
	CourseCode   string  `json:"courseCode"`
	CourseTitle  string  `json:"courseTitle"`
	Role         string  `json:"role"`
	Active       bool    `json:"active"`
	State        string  `json:"state"`
	EnrolledAt   string  `json:"enrolledAt"`
	OrgName      *string `json:"orgName,omitempty"`
}

// ActivityRow is one recent learning-activity event.
type ActivityRow struct {
	EventKind  string `json:"eventKind"`
	CourseCode string `json:"courseCode"`
	CourseTitle string `json:"courseTitle"`
	OccurredAt string `json:"occurredAt"`
}

// Report is the full person report payload.
type Report struct {
	ID            string          `json:"id"`
	Email         string          `json:"email"`
	FirstName     *string         `json:"firstName"`
	LastName      *string         `json:"lastName"`
	DisplayName   *string         `json:"displayName"`
	OrgID         string          `json:"orgId"`
	OrgName       string          `json:"orgName"`
	Role          string          `json:"role"`
	Active        bool            `json:"active"`
	CreatedAt     time.Time       `json:"createdAt"`
	LastActivityAt *time.Time     `json:"lastActivityAt"`
	EnrollmentCount int64         `json:"enrollmentCount"`
	Enrollments   []EnrollmentRow `json:"enrollments"`
	RecentActivity []ActivityRow  `json:"recentActivity"`
}

// UserReport returns profile, enrollments, and recent activity for one user.
func UserReport(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (*Report, error) {
	var rep Report
	var roleName string
	err := pool.QueryRow(ctx, `
SELECT
    u.id::text,
    u.email,
    u.first_name,
    u.last_name,
    u.display_name,
    u.org_id::text,
    o.name,
    COALESCE((
        SELECT ar.name
        FROM "user".user_app_roles uar
        JOIN "user".app_roles ar ON ar.id = uar.role_id
        WHERE uar.user_id = u.id
        ORDER BY ar.name
        LIMIT 1
    ), '') AS role_name,
    (u.deactivated_at IS NULL AND NOT u.login_blocked) AS active,
    u.created_at
FROM "user".users u
INNER JOIN tenant.organizations o ON o.id = u.org_id
WHERE u.id = $1 AND u.account_type <> 'system'
`, userID).Scan(
		&rep.ID, &rep.Email, &rep.FirstName, &rep.LastName, &rep.DisplayName,
		&rep.OrgID, &rep.OrgName, &roleName, &rep.Active, &rep.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	rep.Role = appRoleToCli(roleName)

	_ = pool.QueryRow(ctx, `
SELECT MAX(occurred_at) FROM "user".user_audit WHERE user_id = $1
`, userID).Scan(&rep.LastActivityAt)

	_ = pool.QueryRow(ctx, `
SELECT COUNT(*)::bigint
FROM course.course_enrollments ce
WHERE ce.user_id = $1 AND (ce.active OR ce.invitation_pending)
`, userID).Scan(&rep.EnrollmentCount)

	enrollRows, err := pool.Query(ctx, `
SELECT
    c.id::text,
    c.course_code,
    c.title,
    ce.role,
    ce.active,
    COALESCE(ce.state::text, 'active'),
    ce.created_at,
    ho.name
FROM course.course_enrollments ce
INNER JOIN course.courses c ON c.id = ce.course_id
LEFT JOIN tenant.organizations ho ON ho.id = ce.home_org_id
WHERE ce.user_id = $1
ORDER BY ce.created_at DESC
LIMIT 100
`, userID)
	if err != nil {
		return nil, err
	}
	defer enrollRows.Close()
	for enrollRows.Next() {
		var row EnrollmentRow
		var enrolledAt time.Time
		if err := enrollRows.Scan(
			&row.CourseID, &row.CourseCode, &row.CourseTitle, &row.Role,
			&row.Active, &row.State, &enrolledAt, &row.OrgName,
		); err != nil {
			return nil, err
		}
		row.EnrolledAt = enrolledAt.UTC().Format(time.RFC3339)
		rep.Enrollments = append(rep.Enrollments, row)
	}
	if rep.Enrollments == nil {
		rep.Enrollments = []EnrollmentRow{}
	}

	actRows, err := pool.Query(ctx, `
SELECT
    ua.event_kind,
    c.course_code,
    c.title,
    ua.occurred_at
FROM "user".user_audit ua
INNER JOIN course.courses c ON c.id = ua.course_id
WHERE ua.user_id = $1
ORDER BY ua.occurred_at DESC
LIMIT 50
`, userID)
	if err != nil {
		return nil, err
	}
	defer actRows.Close()
	for actRows.Next() {
		var row ActivityRow
		var at time.Time
		if err := actRows.Scan(&row.EventKind, &row.CourseCode, &row.CourseTitle, &at); err != nil {
			return nil, err
		}
		row.OccurredAt = at.UTC().Format(time.RFC3339)
		rep.RecentActivity = append(rep.RecentActivity, row)
	}
	if rep.RecentActivity == nil {
		rep.RecentActivity = []ActivityRow{}
	}
	return &rep, nil
}

// IsErased reports whether the user email has been anonymized.
func IsErased(email string) bool {
	return strings.HasSuffix(strings.ToLower(email), "@erased.invalid")
}

func appRoleToCli(name string) string {
	switch name {
	case "Teacher":
		return "instructor"
	case "Student":
		return "student"
	case "TA":
		return "ta"
	case "Global Admin":
		return "admin"
	default:
		if name == "" {
			return ""
		}
		return strings.ToLower(name)
	}
}

func CliRoleToApp(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "instructor":
		return "Teacher"
	case "student":
		return "Student"
	case "ta":
		return "TA"
	case "admin":
		return "Global Admin"
	default:
		return role
	}
}

// InsertUser creates a user in the given organization.
func InsertUser(
	ctx context.Context,
	pool *pgxpool.Pool,
	orgID uuid.UUID,
	email, passwordHash, displayName string,
	firstName, lastName *string,
) (uuid.UUID, error) {
	var uid uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO "user".users (email, password_hash, display_name, first_name, last_name, org_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id
`, email, passwordHash, displayName, firstName, lastName, orgID).Scan(&uid)
	return uid, err
}

// SetActive toggles login access for a user.
func SetActive(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, active bool) error {
	if active {
		_, err := pool.Exec(ctx, `
UPDATE "user".users SET deactivated_at = NULL, login_blocked = FALSE WHERE id = $1
`, userID)
		return err
	}
	tag, err := pool.Exec(ctx, `
UPDATE "user".users
SET deactivated_at = COALESCE(deactivated_at, NOW()), login_blocked = TRUE
WHERE id = $1
`, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("platformpeople: user not found")
	}
	return nil
}