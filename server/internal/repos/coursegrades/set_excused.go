package coursegrades

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/repos/gradeauditevents"
)

// ErrGradebookItemNotInCourse is returned when the cell's module item is not in the course.
var ErrGradebookItemNotInCourse = errors.New("grade item is not in this course")

const excusedReasonMax = 2000

// SetExcusedForCell toggles course.course_grades.excused for one student and item
// and appends a grade_audit_events row (action excused or unexcused) in the same transaction.
// A cell with no grade row is inserted with points_earned 0 so the flag can persist;
// an existing score is left unchanged. Un-excusing a cell that has no row is a no-op.
func SetExcusedForCell(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID, actor, studentID, itemID uuid.UUID,
	excused bool,
	reason *string,
) error {
	if pool == nil {
		return errors.New("nil pool")
	}

	var policy string
	err := pool.QueryRow(ctx, `
SELECT COALESCE(NULLIF(TRIM(ma.posting_policy), ''), 'automatic')
FROM course.course_structure_items csi
LEFT JOIN course.module_assignments ma ON ma.structure_item_id = csi.id
WHERE csi.course_id = $1 AND csi.id = $2
`, courseID, itemID).Scan(&policy)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrGradebookItemNotInCourse
	}
	if err != nil {
		return err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var prevPts float64
	var prevExcused bool
	err = tx.QueryRow(ctx, `
SELECT points_earned, excused
FROM course.course_grades
WHERE course_id = $1 AND student_user_id = $2 AND module_item_id = $3
FOR UPDATE
`, courseID, studentID, itemID).Scan(&prevPts, &prevExcused)
	hasPrev := err == nil
	if errors.Is(err, pgx.ErrNoRows) {
		hasPrev = false
	} else if err != nil {
		return err
	}

	if hasPrev && prevExcused == excused {
		return tx.Commit(ctx)
	}
	if !hasPrev && !excused {
		return tx.Commit(ctx)
	}

	score := 0.0
	if hasPrev {
		score = prevPts
		_, err = tx.Exec(ctx, `
UPDATE course.course_grades
SET excused = $4, updated_at = NOW()
WHERE course_id = $1 AND student_user_id = $2 AND module_item_id = $3
`, courseID, studentID, itemID, excused)
	} else if policy == "automatic" {
		_, err = tx.Exec(ctx, `
INSERT INTO course.course_grades (
	course_id, student_user_id, module_item_id, points_earned, updated_at, posted_at, excused
) VALUES ($1, $2, $3, 0, NOW(), NOW(), TRUE)
`, courseID, studentID, itemID)
	} else {
		_, err = tx.Exec(ctx, `
INSERT INTO course.course_grades (
	course_id, student_user_id, module_item_id, points_earned, updated_at, posted_at, excused
) VALUES ($1, $2, $3, 0, NOW(), NULL, TRUE)
`, courseID, studentID, itemID)
	}
	if err != nil {
		return err
	}

	action := "excused"
	newStatus := "excused"
	if !excused {
		action = "unexcused"
		newStatus = "graded"
	}
	var prevPtr *float64
	var prevStatus *string
	if hasPrev {
		p := prevPts
		prevPtr = &p
		ps := "graded"
		if prevExcused {
			ps = "excused"
		}
		prevStatus = &ps
	}
	newPtr := score
	ns := newStatus
	if err := gradeauditevents.Insert(ctx, tx, courseID, itemID, studentID, &actor, action, prevPtr, &newPtr, prevStatus, &ns, trimExcusedReason(reason)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func trimExcusedReason(reason *string) *string {
	if reason == nil {
		return nil
	}
	s := strings.TrimSpace(*reason)
	if s == "" {
		return nil
	}
	if len(s) > excusedReasonMax {
		s = s[:excusedReasonMax]
	}
	return &s
}
