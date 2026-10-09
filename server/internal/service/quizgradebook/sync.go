// Package quizgradebook writes auto-graded quiz scores into the gradebook.
package quizgradebook

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/repos/coursegrades"
	"github.com/lextures/lextures/server/internal/repos/coursemodulequizzes"
	"github.com/lextures/lextures/server/internal/repos/quizattempts"
)

// SyncStudentCell recomputes the student's gradebook score for a quiz from their submitted
// attempts (honouring the quiz's attempt policy) and upserts course.course_grades.
//
// Attempts that still need manual grading are ignored, so a quiz with essay questions never
// receives a partial score; staff finish those from the quiz grader. It returns true when a
// gradebook cell was written.
func SyncStudentCell(ctx context.Context, pool *pgxpool.Pool, courseID, studentID, itemID uuid.UUID) (bool, error) {
	if pool == nil {
		return false, errors.New("nil pool")
	}
	quizRow, err := coursemodulequizzes.GetForCourseItem(ctx, pool, courseID, itemID)
	if err != nil {
		return false, err
	}
	if quizRow == nil {
		return false, nil
	}
	points, ready, err := quizattempts.PolicyPointsForStudent(
		ctx, pool, courseID, itemID, studentID, quizRow.GradeAttemptPolicy,
	)
	if err != nil || !ready {
		return false, err
	}
	// Quizzes have no posting policy: the score is visible as soon as the attempt is graded.
	if err := coursegrades.UpsertCellWithFlags(
		ctx, pool, courseID, studentID, itemID, points, nil, nil, nil, "automatic", false,
	); err != nil {
		return false, err
	}
	return true, nil
}
