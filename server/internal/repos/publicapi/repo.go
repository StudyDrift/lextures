// Package publicapi exposes read models for the versioned REST API (plan 16.1).
package publicapi

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AssignmentItem is a cross-course assignment visible to the token owner.
type AssignmentItem struct {
	ID         uuid.UUID  `json:"id"`
	CourseID   uuid.UUID  `json:"courseId"`
	CourseCode string     `json:"courseCode"`
	Title      string     `json:"title"`
	DueAt      *time.Time `json:"dueAt,omitempty"`
	PointsWorth *int      `json:"pointsWorth,omitempty"`
}

// GradeItem is a posted grade visible to the token owner.
type GradeItem struct {
	ID           uuid.UUID  `json:"id"`
	CourseID     uuid.UUID  `json:"courseId"`
	CourseCode   string     `json:"courseCode"`
	AssignmentID uuid.UUID  `json:"assignmentId"`
	Title        string     `json:"title"`
	PointsEarned *float64   `json:"pointsEarned,omitempty"`
	PostedAt     *time.Time `json:"postedAt,omitempty"`
}

// UserPublic is a user directory record with optional PII.
type UserPublic struct {
	ID          uuid.UUID `json:"id"`
	DisplayName string    `json:"displayName"`
	Email       string    `json:"email,omitempty"`
	Role        string    `json:"role,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

// ListAssignmentsForUser returns published assignments in courses the user can access.
func ListAssignmentsForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, courseIDs []uuid.UUID) ([]AssignmentItem, error) {
	args := []any{userID}
	courseFilter := ""
	if len(courseIDs) > 0 {
		courseFilter = ` AND c.id = ANY($2::uuid[])`
		args = append(args, courseIDs)
	}
	rows, err := pool.Query(ctx, `
SELECT csi.id, c.id, c.course_code, csi.title, csi.due_at, ma.points_worth
FROM course.course_structure_items csi
INNER JOIN course.module_assignments ma ON ma.structure_item_id = csi.id
INNER JOIN course.courses c ON c.id = csi.course_id
WHERE csi.kind = 'assignment' AND csi.published AND NOT csi.archived
  AND c.id IN (
    SELECT e.course_id FROM course.course_enrollments e
    WHERE e.user_id = $1 AND e.active
  )`+courseFilter+`
ORDER BY c.title ASC, csi.due_at NULLS LAST, csi.sort_order ASC
`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AssignmentItem
	for rows.Next() {
		var item AssignmentItem
		if err := rows.Scan(&item.ID, &item.CourseID, &item.CourseCode, &item.Title, &item.DueAt, &item.PointsWorth); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// ListGradesForUser returns posted grades for courses the user can access.
func ListGradesForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, courseIDs []uuid.UUID) ([]GradeItem, error) {
	args := []any{userID}
	courseFilter := ""
	if len(courseIDs) > 0 {
		courseFilter = ` AND c.id = ANY($2::uuid[])`
		args = append(args, courseIDs)
	}
	rows, err := pool.Query(ctx, `
SELECT cg.id, c.id, c.course_code, csi.id, csi.title, cg.points_earned, cg.updated_at
FROM course.course_grades cg
INNER JOIN course.course_structure_items csi ON csi.id = cg.module_item_id
INNER JOIN course.courses c ON c.id = csi.course_id
WHERE cg.student_user_id = $1
  AND c.id IN (
    SELECT e.course_id FROM course.course_enrollments e
    WHERE e.user_id = $1 AND e.active
  )`+courseFilter+`
ORDER BY cg.updated_at DESC NULLS LAST
`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GradeItem
	for rows.Next() {
		var item GradeItem
		if err := rows.Scan(&item.ID, &item.CourseID, &item.CourseCode, &item.AssignmentID, &item.Title, &item.PointsEarned, &item.PostedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// GetUserByID loads a user for the public API.
func GetUserByID(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (*UserPublic, error) {
	var u UserPublic
	var dn, roleName *string
	err := pool.QueryRow(ctx, `
SELECT u.id,
       COALESCE(u.display_name, ''),
       u.email,
       (SELECT ar.name FROM "user".user_app_roles uar
        JOIN "user".app_roles ar ON ar.id = uar.role_id
        WHERE uar.user_id = u.id ORDER BY ar.name LIMIT 1),
       u.created_at
FROM "user".users u
WHERE u.id = $1
`, userID).Scan(&u.ID, &dn, &u.Email, &roleName, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	if dn != nil {
		u.DisplayName = *dn
	}
	if roleName != nil {
		u.Role = *roleName
	}
	return &u, nil
}

// ViewerMayReadUser is true when viewer shares an enrollment with target or is the same user.
func ViewerMayReadUser(ctx context.Context, pool *pgxpool.Pool, viewerID, targetID uuid.UUID) (bool, error) {
	if viewerID == targetID {
		return true, nil
	}
	var ok bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1
  FROM course.course_enrollments e1
  INNER JOIN course.course_enrollments e2
    ON e1.course_id = e2.course_id AND e1.active AND e2.active
  WHERE e1.user_id = $1 AND e2.user_id = $2
)
`, viewerID, targetID).Scan(&ok)
	return ok, err
}
