package webhooksvc

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/repos/coursegrades"
)

// EmitGradePostedCells emits grade.posted for each posted cell.
func EmitGradePostedCells(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, courseID, moduleItemID uuid.UUID, cells []coursegrades.PostedCell) {
	if !cfg.FFWebhooks || pool == nil || len(cells) == 0 {
		return
	}
	var orgID uuid.UUID
	var courseCode string
	err := pool.QueryRow(ctx, `SELECT org_id, course_code FROM course.courses WHERE id = $1`, courseID).Scan(&orgID, &courseCode)
	if err != nil {
		return
	}
	for _, cell := range cells {
		var points float64
		_ = pool.QueryRow(ctx, `
SELECT points_earned FROM course.course_grades
WHERE course_id = $1 AND student_user_id = $2 AND module_item_id = $3
`, courseID, cell.StudentUserID, moduleItemID).Scan(&points)
		EmitGradePosted(pool, cfg, orgID, GradePostedData{
			CourseID:      courseID.String(),
			CourseCode:    courseCode,
			ModuleItemID:  moduleItemID.String(),
			StudentUserID: cell.StudentUserID.String(),
			PointsEarned:  points,
		})
	}
}

// EmitSingleGradePosted emits one grade.posted event.
func EmitSingleGradePosted(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, courseID, moduleItemID, studentUserID uuid.UUID, points float64) {
	if !cfg.FFWebhooks || pool == nil {
		return
	}
	var orgID uuid.UUID
	var courseCode string
	if err := pool.QueryRow(ctx, `SELECT org_id, course_code FROM course.courses WHERE id = $1`, courseID).Scan(&orgID, &courseCode); err != nil {
		return
	}
	EmitGradePosted(pool, cfg, orgID, GradePostedData{
		CourseID:      courseID.String(),
		CourseCode:    courseCode,
		ModuleItemID:  moduleItemID.String(),
		StudentUserID: studentUserID.String(),
		PointsEarned:  points,
	})
}

// EmitEnrollmentCreatedEvent emits enrollment.created.
func EmitEnrollmentCreatedEvent(pool *pgxpool.Pool, cfg config.Config, orgID, courseID uuid.UUID, courseCode string, enrollmentID, studentUserID uuid.UUID, role string) {
	if !cfg.FFWebhooks || pool == nil {
		return
	}
	EmitEnrollmentCreated(pool, cfg, orgID, EnrollmentCreatedData{
		CourseID:      courseID.String(),
		CourseCode:    courseCode,
		EnrollmentID:  enrollmentID.String(),
		StudentUserID: studentUserID.String(),
		Role:          role,
	})
}

// EmitAssignmentSubmittedEvent emits assignment.submitted.
func EmitAssignmentSubmittedEvent(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, courseID uuid.UUID, courseCode string, moduleItemID, submissionID, submittedBy uuid.UUID) {
	if !cfg.FFWebhooks || pool == nil {
		return
	}
	var orgID uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT org_id FROM course.courses WHERE id = $1`, courseID).Scan(&orgID); err != nil {
		return
	}
	EmitAssignmentSubmitted(pool, cfg, orgID, AssignmentSubmittedData{
		CourseID:     courseID.String(),
		CourseCode:   courseCode,
		ModuleItemID: moduleItemID.String(),
		SubmissionID: submissionID.String(),
		SubmittedBy:  submittedBy.String(),
	})
}

// EmitGradeReleasedEvent emits grade.released (DM-only by default for bots).
func EmitGradeReleasedEvent(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, courseID, moduleItemID, studentUserID uuid.UUID, points float64) {
	if !cfg.FFWebhooks || pool == nil {
		return
	}
	var orgID uuid.UUID
	var courseCode string
	if err := pool.QueryRow(ctx, `SELECT org_id, course_code FROM course.courses WHERE id = $1`, courseID).Scan(&orgID, &courseCode); err != nil {
		return
	}
	data := GradeReleasedData{
		CourseID:      courseID.String(),
		CourseCode:    courseCode,
		ModuleItemID:  moduleItemID.String(),
		StudentUserID: studentUserID.String(),
		PointsEarned:  points,
	}
	if cfg.PublicWebOrigin != "" {
		data.URL = cfg.PublicWebOrigin + "/courses/" + courseID.String()
	}
	EmitGradeReleased(pool, cfg, orgID, data)
}

// EmitQuizCompletedEvent emits quiz.completed after a quiz submission.
func EmitQuizCompletedEvent(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, courseID uuid.UUID, courseCode string, moduleItemID, attemptID, studentUserID uuid.UUID, pointsEarned, scorePercent float64) {
	if !cfg.FFWebhooks || pool == nil {
		return
	}
	var orgID uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT org_id FROM course.courses WHERE id = $1`, courseID).Scan(&orgID); err != nil {
		return
	}
	EmitQuizCompleted(pool, cfg, orgID, QuizCompletedData{
		CourseID:      courseID.String(),
		CourseCode:    courseCode,
		ModuleItemID:  moduleItemID.String(),
		AttemptID:     attemptID.String(),
		StudentUserID: studentUserID.String(),
		PointsEarned:  pointsEarned,
		ScorePercent:  scorePercent,
	})
}
