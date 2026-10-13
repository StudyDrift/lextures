package kidmessaging

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Contact kinds.
const (
	KindParent  = "parent"
	KindTeacher = "teacher"
)

// Contact is a person a managed learner may exchange inbox messages with.
type Contact struct {
	UserID       uuid.UUID
	DisplayName  string
	Email        string
	Kind         string
	CourseTitles []string
	// Approved is true for parents and for teachers the parent approved.
	Approved bool
}

// ListParentContacts returns the active parents or guardians linked to the learner.
func ListParentContacts(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID) ([]Contact, error) {
	rows, err := pool.Query(ctx, `
SELECT DISTINCT pu.id, COALESCE(NULLIF(TRIM(pu.display_name), ''), pu.email), pu.email
FROM "user".parent_student_links l
INNER JOIN "user".users pu ON pu.id = l.parent_user_id
WHERE l.student_user_id = $1
  AND l.status = 'active'
  AND pu.deactivated_at IS NULL
ORDER BY 2
`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Contact, 0)
	for rows.Next() {
		c := Contact{Kind: KindParent, Approved: true}
		if err := rows.Scan(&c.UserID, &c.DisplayName, &c.Email); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ListTeacherCandidates returns grading staff (owner, teacher, instructor, TA) on the learner's
// active courses, with whether the parent approved each one as a message contact.
func ListTeacherCandidates(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID) ([]Contact, error) {
	rows, err := pool.Query(ctx, `
SELECT tu.id,
       COALESCE(NULLIF(TRIM(tu.display_name), ''), tu.email),
       tu.email,
       ARRAY_AGG(DISTINCT c.title ORDER BY c.title),
       BOOL_OR(mc.contact_user_id IS NOT NULL)
FROM course.course_enrollments se
INNER JOIN course.enrollment_roles ser ON ser.role_key = se.role AND ser.is_student_equivalent = true
INNER JOIN course.courses c ON c.id = se.course_id AND c.archived = false
INNER JOIN course.course_enrollments te ON te.course_id = c.id AND te.active
INNER JOIN course.enrollment_roles ter ON ter.role_key = te.role AND ter.is_staff AND ter.can_grade
INNER JOIN "user".users tu ON tu.id = te.user_id AND tu.deactivated_at IS NULL
LEFT JOIN communication.managed_learner_contacts mc
  ON mc.student_user_id = se.user_id AND mc.contact_user_id = tu.id
WHERE se.user_id = $1 AND se.active AND tu.id <> $1
GROUP BY tu.id, tu.display_name, tu.email
ORDER BY 2
`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Contact, 0)
	for rows.Next() {
		c := Contact{Kind: KindTeacher}
		if err := rows.Scan(&c.UserID, &c.DisplayName, &c.Email, &c.CourseTitles, &c.Approved); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ApproveContact records that approvedBy allowed contactID to message the learner (idempotent).
func ApproveContact(ctx context.Context, pool *pgxpool.Pool, studentID, contactID, approvedBy uuid.UUID) error {
	_, err := pool.Exec(ctx, `
INSERT INTO communication.managed_learner_contacts (student_user_id, contact_user_id, approved_by)
VALUES ($1, $2, $3)
ON CONFLICT (student_user_id, contact_user_id) DO NOTHING
`, studentID, contactID, approvedBy)
	return err
}

// RevokeContact removes an approved contact. Returns false when no approval existed.
func RevokeContact(ctx context.Context, pool *pgxpool.Pool, studentID, contactID uuid.UUID) (bool, error) {
	tag, err := pool.Exec(ctx, `
DELETE FROM communication.managed_learner_contacts
WHERE student_user_id = $1 AND contact_user_id = $2
`, studentID, contactID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}
