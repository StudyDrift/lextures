package coursemoduleassignments

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UpsertImportBody inserts or updates an assignment body for course import.
func UpsertImportBody(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID, itemID uuid.UUID,
	markdown string,
	pointsWorth *int32,
	availableFrom, availableUntil *time.Time,
	assignmentAccessCode *string,
	submissionAllowText, submissionAllowFileUpload, submissionAllowURL bool,
	lateSubmissionPolicy string,
	latePenaltyPercent *int32,
	rubricJSON []byte,
	originalityDetection, originalityStudentVisibility string,
	blindGrading bool,
	gradingType *string,
) error {
	if pool == nil {
		return errors.New("db pool is nil")
	}
	if lateSubmissionPolicy == "" {
		lateSubmissionPolicy = "allow"
	}
	if originalityDetection == "" {
		originalityDetection = "disabled"
	}
	if originalityStudentVisibility == "" {
		originalityStudentVisibility = "hide"
	}
	var rubric any
	if len(rubricJSON) > 0 {
		rubric = rubricJSON
	}
	_, err := pool.Exec(ctx, `
INSERT INTO course.module_assignments AS m (
	structure_item_id, markdown, points_worth, updated_at,
	available_from, available_until, assignment_access_code,
	submission_allow_text, submission_allow_file_upload, submission_allow_url,
	late_submission_policy, late_penalty_percent, rubric_json,
	originality_detection, originality_student_visibility,
	blind_grading, grading_type, identities_revealed_at, posting_policy, release_at
)
SELECT c.id, $3, $4, NOW(), $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, NULL, 'automatic', NULL
FROM course.course_structure_items c
WHERE c.id = $1 AND c.course_id = $2 AND c.kind = 'assignment'
ON CONFLICT (structure_item_id) DO UPDATE
SET markdown = EXCLUDED.markdown,
	points_worth = EXCLUDED.points_worth,
	available_from = EXCLUDED.available_from,
	available_until = EXCLUDED.available_until,
	assignment_access_code = EXCLUDED.assignment_access_code,
	submission_allow_text = EXCLUDED.submission_allow_text,
	submission_allow_file_upload = EXCLUDED.submission_allow_file_upload,
	submission_allow_url = EXCLUDED.submission_allow_url,
	late_submission_policy = EXCLUDED.late_submission_policy,
	late_penalty_percent = EXCLUDED.late_penalty_percent,
	rubric_json = EXCLUDED.rubric_json,
	originality_detection = EXCLUDED.originality_detection,
	originality_student_visibility = EXCLUDED.originality_student_visibility,
	blind_grading = EXCLUDED.blind_grading,
	grading_type = EXCLUDED.grading_type,
	identities_revealed_at = NULL,
	settings_version = m.settings_version + 1,
	updated_at = NOW()
`, itemID, courseID, markdown, pointsWorth, availableFrom, availableUntil, assignmentAccessCode,
		submissionAllowText, submissionAllowFileUpload, submissionAllowURL,
		lateSubmissionPolicy, latePenaltyPercent, rubric,
		originalityDetection, originalityStudentVisibility, blindGrading, gradingType)
	return err
}
