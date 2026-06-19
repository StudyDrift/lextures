package publicapi

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	repoCourse "github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
)

// ListCourses returns courses the user can access, optionally filtered by token course allowlist.
func ListCourses(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, allowedCourseIDs []uuid.UUID) ([]CourseResource, error) {
	courses, err := repoCourse.ListForEnrolledUser(ctx, pool, userID, nil)
	if err != nil {
		return nil, err
	}
	if len(allowedCourseIDs) > 0 {
		allowed := make(map[string]struct{}, len(allowedCourseIDs))
		for _, id := range allowedCourseIDs {
			allowed[id.String()] = struct{}{}
		}
		filtered := make([]repoCourse.CoursePublic, 0, len(courses))
		for _, c := range courses {
			if _, ok := allowed[c.ID]; ok {
				filtered = append(filtered, c)
			}
		}
		courses = filtered
	}
	out := make([]CourseResource, 0, len(courses))
	for _, c := range courses {
		out = append(out, CourseResource{
			ID:          c.ID,
			CourseCode:  c.CourseCode,
			Title:       c.Title,
			Description: c.Description,
			Published:   c.Published,
			CreatedAt:   c.CreatedAt.Format(time.RFC3339),
		})
	}
	return out, nil
}

// GetCourseByID returns a course when the user has access.
func GetCourseByID(ctx context.Context, pool *pgxpool.Pool, userID, courseID uuid.UUID) (*CourseResource, error) {
	code, err := repoCourse.GetCourseCodeByID(ctx, pool, courseID)
	if err != nil || code == nil {
		return nil, err
	}
	ok, err := enrollment.UserHasAccess(ctx, pool, *code, userID)
	if err != nil || !ok {
		return nil, err
	}
	c, err := repoCourse.GetPublicByCourseCode(ctx, pool, *code)
	if err != nil || c == nil {
		return nil, err
	}
	return &CourseResource{
		ID:          c.ID,
		CourseCode:  c.CourseCode,
		Title:       c.Title,
		Description: c.Description,
		Published:   c.Published,
		CreatedAt:   c.CreatedAt.Format(time.RFC3339),
	}, nil
}

// ListEnrollments returns roster rows for a course the user can access.
func ListEnrollments(ctx context.Context, pool *pgxpool.Pool, userID, courseID uuid.UUID) ([]EnrollmentResource, error) {
	code, err := repoCourse.GetCourseCodeByID(ctx, pool, courseID)
	if err != nil || code == nil {
		return nil, err
	}
	ok, err := enrollment.UserHasAccess(ctx, pool, *code, userID)
	if err != nil || !ok {
		return []EnrollmentResource{}, nil
	}
	rows, err := enrollment.ListRosterForCourse(ctx, pool, *code)
	if err != nil {
		return nil, err
	}
	out := make([]EnrollmentResource, 0, len(rows))
	for _, r := range rows {
		out = append(out, EnrollmentResource{
			ID:     r.ID.String(),
			UserID: r.UserID.String(),
			Role:   r.Role,
			State:  r.State,
		})
	}
	return out, nil
}

// GetUser returns a user visible to the requester.
func GetUser(ctx context.Context, pool *pgxpool.Pool, _ uuid.UUID, targetID uuid.UUID, includePII bool) (*UserResource, error) {
	q := `SELECT id::text, email, display_name, first_name, last_name FROM "user".users WHERE id = $1`
	var id, email string
	var displayName, firstName, lastName *string
	err := pool.QueryRow(ctx, q, targetID).Scan(&id, &email, &displayName, &firstName, &lastName)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	res := &UserResource{ID: id, DisplayName: displayName}
	if includePII {
		res.Email = &email
		res.FirstName = firstName
		res.LastName = lastName
	}
	return res, nil
}

// ListAssignments returns assignments across courses the user can access.
func ListAssignments(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, allowedCourseIDs []uuid.UUID) ([]AssignmentResource, error) {
	courses, err := ListCourses(ctx, pool, userID, allowedCourseIDs)
	if err != nil {
		return nil, err
	}
	var out []AssignmentResource
	for _, c := range courses {
		cid, _ := uuid.Parse(c.ID)
		rows, err := pool.Query(ctx, `
SELECT csi.id, csi.title, ma.due_at
FROM course.course_structure_items csi
INNER JOIN course.module_assignments ma ON ma.structure_item_id = csi.id
WHERE csi.course_id = $1 AND csi.item_type = 'assignment' AND csi.published = true
ORDER BY csi.sort_order
`, cid)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var id uuid.UUID
			var title string
			var dueAt *time.Time
			if err := rows.Scan(&id, &title, &dueAt); err != nil {
				rows.Close()
				return nil, err
			}
			out = append(out, AssignmentResource{
				ID:       id.String(),
				CourseID: c.ID,
				Title:    title,
				DueAt:    dueAt,
			})
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	if out == nil {
		out = []AssignmentResource{}
	}
	return out, nil
}

// ListGrades returns posted grade cells across accessible courses.
func ListGrades(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, allowedCourseIDs []uuid.UUID) ([]GradeResource, error) {
	courses, err := ListCourses(ctx, pool, userID, allowedCourseIDs)
	if err != nil {
		return nil, err
	}
	var out []GradeResource
	for _, c := range courses {
		cid, _ := uuid.Parse(c.ID)
		rows, err := pool.Query(ctx, `
SELECT student_user_id, module_item_id, points_earned::text
FROM course.course_grades
WHERE course_id = $1 AND posted_at IS NOT NULL
`, cid)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var studentID, itemID uuid.UUID
			var pts string
			if err := rows.Scan(&studentID, &itemID, &pts); err != nil {
				rows.Close()
				return nil, err
			}
			out = append(out, GradeResource{
				CourseID:      c.ID,
				StudentUserID: studentID.String(),
				ModuleItemID:  itemID.String(),
				PointsEarned:  pts,
			})
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	if out == nil {
		out = []GradeResource{}
	}
	return out, nil
}
