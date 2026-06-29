// Package adminconsole provides org-scoped queries for the admin console (plan 18.1).
package adminconsole

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Overview holds dashboard KPIs for an organization.
type Overview struct {
	TotalUsers           int64 `json:"totalUsers"`
	ActiveCourses        int64 `json:"activeCourses"`
	PendingEnrollments   int64 `json:"pendingEnrollments"`
	StorageBytes         int64 `json:"storageBytes"`
}

// UserRow is one row in the admin user-management table.
type UserRow struct {
	ID          string     `json:"id"`
	Email       string     `json:"email"`
	DisplayName *string    `json:"displayName"`
	Role        string     `json:"role"`
	OrgRole     *string    `json:"orgRole"`
	Active      bool       `json:"active"`
	CreatedAt   time.Time  `json:"createdAt"`
}

// CourseRow is one row in the admin course-management table.
type CourseRow struct {
	ID           string    `json:"id"`
	CourseCode   string    `json:"courseCode"`
	Title        string    `json:"title"`
	Status       string    `json:"status"`
	Instructor   *string   `json:"instructorName"`
	TermID       *string   `json:"termId"`
	TermName     *string   `json:"termName"`
	EnrollmentCount int64  `json:"enrollmentCount"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// ListParams holds pagination and filter options.
type ListParams struct {
	Query    string
	Role     string
	Status   string
	TermID   *uuid.UUID
	Page     int
	PerPage  int
}

// ListResult is a paginated list response.
type ListResult[T any] struct {
	Items      []T   `json:"items"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PerPage    int   `json:"perPage"`
	TotalPages int   `json:"totalPages"`
}

func normalizePagination(page, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	switch perPage {
	case 50, 100:
	default:
		perPage = 25
	}
	return page, perPage
}

// OverviewForOrg returns KPI counts scoped to orgID.
func OverviewForOrg(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) (Overview, error) {
	var o Overview
	err := pool.QueryRow(ctx, `
SELECT
  (SELECT COUNT(*)::bigint FROM "user".users u
   WHERE u.org_id = $1 AND u.id <> 'a0000000-0000-4000-8000-000000000001'::uuid
     AND u.deactivated_at IS NULL AND NOT u.login_blocked),
  (SELECT COUNT(*)::bigint FROM course.courses c
   WHERE c.org_id = $1 AND c.archived = false AND c.published = true),
  (SELECT COUNT(*)::bigint FROM course.course_enrollments ce
   INNER JOIN course.courses c ON c.id = ce.course_id
   WHERE c.org_id = $1 AND ce.invitation_pending = true)
`, orgID).Scan(&o.TotalUsers, &o.ActiveCourses, &o.PendingEnrollments)
	if err != nil {
		return Overview{}, err
	}
	_ = pool.QueryRow(ctx, `
SELECT COALESCE(SUM(cf.byte_size), 0)::bigint
FROM course.course_files cf
INNER JOIN course.courses c ON c.id = cf.course_id
WHERE c.org_id = $1
`, orgID).Scan(&o.StorageBytes)
	return o, nil
}

// ListUsers returns paginated users in orgID matching filters.
func ListUsers(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, p ListParams) (ListResult[UserRow], error) {
	page, perPage := normalizePagination(p.Page, p.PerPage)
	offset := (page - 1) * perPage

	args := []any{orgID}
	where := []string{
		`u.org_id = $1`,
		`u.id <> 'a0000000-0000-4000-8000-000000000001'::uuid`,
	}
	argIdx := 2

	if q := strings.TrimSpace(p.Query); q != "" {
		like := "%" + strings.ToLower(q) + "%"
		where = append(where, fmt.Sprintf(`(LOWER(u.email) LIKE $%d OR LOWER(COALESCE(u.display_name,'')) LIKE $%d)`, argIdx, argIdx))
		args = append(args, like)
		argIdx++
	}

	roleJoin := ""
	if role := strings.TrimSpace(p.Role); role != "" {
		dbRole := cliRoleToAppRole(role)
		roleJoin = fmt.Sprintf(`
INNER JOIN "user".user_app_roles uar_f ON uar_f.user_id = u.id
INNER JOIN "user".app_roles ar_f ON ar_f.id = uar_f.role_id AND ar_f.name = $%d`, argIdx)
		args = append(args, dbRole)
		argIdx++
	}

	whereSQL := strings.Join(where, " AND ")
	countQ := fmt.Sprintf(`SELECT COUNT(*)::bigint FROM "user".users u %s WHERE %s`, roleJoin, whereSQL)
	var total int64
	if err := pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return ListResult[UserRow]{}, err
	}

	args = append(args, perPage, offset)
	limitIdx := argIdx
	offsetIdx := argIdx + 1
	listQ := fmt.Sprintf(`
SELECT u.id::text, u.email, u.display_name,
       COALESCE((SELECT ar.name FROM "user".user_app_roles uar
        JOIN "user".app_roles ar ON ar.id = uar.role_id
        WHERE uar.user_id = u.id ORDER BY ar.name LIMIT 1), '') AS role,
       (SELECT g.role FROM "user".org_role_grants g
        WHERE g.org_id = u.org_id AND g.user_id = u.id
          AND (g.expires_at IS NULL OR g.expires_at > NOW())
        ORDER BY g.granted_at DESC LIMIT 1) AS org_role,
       (u.deactivated_at IS NULL AND NOT u.login_blocked) AS active,
       u.created_at
FROM "user".users u
%s
WHERE %s
ORDER BY u.email ASC
LIMIT $%d OFFSET $%d
`, roleJoin, whereSQL, limitIdx, offsetIdx)

	rows, err := pool.Query(ctx, listQ, args...)
	if err != nil {
		return ListResult[UserRow]{}, err
	}
	defer rows.Close()

	var items []UserRow
	for rows.Next() {
		var row UserRow
		var roleName string
		if err := rows.Scan(&row.ID, &row.Email, &row.DisplayName, &roleName, &row.OrgRole, &row.Active, &row.CreatedAt); err != nil {
			return ListResult[UserRow]{}, err
		}
		row.Role = appRoleToCliRole(roleName)
		items = append(items, row)
	}
	if items == nil {
		items = []UserRow{}
	}
	totalPages := int(total) / perPage
	if int(total)%perPage != 0 {
		totalPages++
	}
	return ListResult[UserRow]{
		Items: items, Total: total, Page: page, PerPage: perPage, TotalPages: totalPages,
	}, rows.Err()
}

// ListCourses returns paginated courses in orgID matching filters.
func ListCourses(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, p ListParams) (ListResult[CourseRow], error) {
	page, perPage := normalizePagination(p.Page, p.PerPage)
	offset := (page - 1) * perPage

	args := []any{orgID}
	where := []string{`c.org_id = $1`}
	argIdx := 2

	if q := strings.TrimSpace(p.Query); q != "" {
		like := "%" + strings.ToLower(q) + "%"
		where = append(where, fmt.Sprintf(`(LOWER(c.title) LIKE $%d OR LOWER(c.course_code) LIKE $%d
  OR LOWER(COALESCE(inst.display_name, inst.email, '')) LIKE $%d)`, argIdx, argIdx, argIdx))
		args = append(args, like)
		argIdx++
	}

	switch strings.ToLower(strings.TrimSpace(p.Status)) {
	case "active":
		where = append(where, `c.archived = false AND c.published = true`)
	case "archived":
		where = append(where, `c.archived = true`)
	case "draft":
		where = append(where, `c.archived = false AND c.published = false`)
	}

	if p.TermID != nil {
		where = append(where, fmt.Sprintf(`c.term_id = $%d`, argIdx))
		args = append(args, *p.TermID)
		argIdx++
	}

	whereSQL := strings.Join(where, " AND ")
	fromJoin := `
FROM course.courses c
LEFT JOIN "user".users inst ON inst.id = c.created_by_user_id
LEFT JOIN tenant.terms t ON t.id = c.term_id`

	countQ := fmt.Sprintf(`SELECT COUNT(*)::bigint %s WHERE %s`, fromJoin, whereSQL)
	var total int64
	if err := pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return ListResult[CourseRow]{}, err
	}

	args = append(args, perPage, offset)
	limitIdx := argIdx
	offsetIdx := argIdx + 1
	listQ := fmt.Sprintf(`
SELECT c.id::text, c.course_code, c.title,
       CASE WHEN c.archived THEN 'archived'
            WHEN NOT c.published THEN 'draft'
            ELSE 'active' END AS status,
       COALESCE(NULLIF(TRIM(COALESCE(inst.display_name, '')), ''), inst.email) AS instructor_name,
       c.term_id::text, t.name,
       (SELECT COUNT(*)::bigint FROM course.course_enrollments ce
        WHERE ce.course_id = c.id AND ce.active) AS enrollment_count,
       c.updated_at
%s
WHERE %s
ORDER BY c.title ASC
LIMIT $%d OFFSET $%d
`, fromJoin, whereSQL, limitIdx, offsetIdx)

	rows, err := pool.Query(ctx, listQ, args...)
	if err != nil {
		return ListResult[CourseRow]{}, err
	}
	defer rows.Close()

	var items []CourseRow
	for rows.Next() {
		var row CourseRow
		var termID, termName *string
		if err := rows.Scan(
			&row.ID, &row.CourseCode, &row.Title, &row.Status,
			&row.Instructor, &termID, &termName, &row.EnrollmentCount, &row.UpdatedAt,
		); err != nil {
			return ListResult[CourseRow]{}, err
		}
		row.TermID = termID
		row.TermName = termName
		items = append(items, row)
	}
	if items == nil {
		items = []CourseRow{}
	}
	totalPages := int(total) / perPage
	if int(total)%perPage != 0 {
		totalPages++
	}
	return ListResult[CourseRow]{
		Items: items, Total: total, Page: page, PerPage: perPage, TotalPages: totalPages,
	}, rows.Err()
}

func appRoleToCliRole(name string) string {
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

func cliRoleToAppRole(role string) string {
	switch strings.ToLower(role) {
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
