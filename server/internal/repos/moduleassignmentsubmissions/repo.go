package moduleassignmentsubmissions

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/repos/submissionversions"
)

type SubmissionRow struct {
	ID                    uuid.UUID
	CourseID              uuid.UUID
	ModuleItemID          uuid.UUID
	SubmittedBy           uuid.UUID
	AttachmentFileID      *uuid.UUID
	BodyText              string
	SubmittedAt           time.Time
	UpdatedAt             time.Time
	ResubmissionRequested bool
	RevisionDueAt         *time.Time
	RevisionFeedback      *string
	VersionNumber         int32
}

// HasBodyText reports whether the submission has non-empty typed text-entry content.
func HasBodyText(row SubmissionRow) bool {
	return strings.TrimSpace(row.BodyText) != ""
}

// HasFileAttachment reports whether the submission references a stored file.
func HasFileAttachment(row SubmissionRow) bool {
	return row.AttachmentFileID != nil
}

type GradedFilter string

const (
	GradedFilterAll      GradedFilter = "all"
	GradedFilterGraded   GradedFilter = "graded"
	GradedFilterUngraded GradedFilter = "ungraded"
)

func scanSubmission(scanner interface{ Scan(...any) error }) (*SubmissionRow, error) {
	var s SubmissionRow
	err := scanner.Scan(
		&s.ID, &s.CourseID, &s.ModuleItemID, &s.SubmittedBy, &s.AttachmentFileID, &s.BodyText, &s.SubmittedAt, &s.UpdatedAt,
		&s.ResubmissionRequested, &s.RevisionDueAt, &s.RevisionFeedback, &s.VersionNumber,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func GetForCourseItemUser(ctx context.Context, pool *pgxpool.Pool, courseID, moduleItemID, submittedBy uuid.UUID) (*SubmissionRow, error) {
	s, err := scanSubmission(pool.QueryRow(ctx, `
SELECT id, course_id, module_item_id, submitted_by, attachment_file_id, body_text, submitted_at, updated_at,
       resubmission_requested, revision_due_at, revision_feedback, version_number
FROM course.module_assignment_submissions
WHERE course_id = $1 AND module_item_id = $2 AND submitted_by = $3
`, courseID, moduleItemID, submittedBy))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

func GetByID(ctx context.Context, pool *pgxpool.Pool, submissionID uuid.UUID) (*SubmissionRow, error) {
	s, err := scanSubmission(pool.QueryRow(ctx, `
SELECT id, course_id, module_item_id, submitted_by, attachment_file_id, body_text, submitted_at, updated_at,
       resubmission_requested, revision_due_at, revision_feedback, version_number
FROM course.module_assignment_submissions
WHERE id = $1
`, submissionID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

func GetByIDForCourse(ctx context.Context, pool *pgxpool.Pool, courseID, submissionID uuid.UUID) (*SubmissionRow, error) {
	s, err := scanSubmission(pool.QueryRow(ctx, `
SELECT id, course_id, module_item_id, submitted_by, attachment_file_id, body_text, submitted_at, updated_at,
       resubmission_requested, revision_due_at, revision_feedback, version_number
FROM course.module_assignment_submissions
WHERE course_id = $1 AND id = $2
`, courseID, submissionID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

// UngradedAssignmentCount is one assignment with at least one ungraded submission.
type UngradedAssignmentCount struct {
	ModuleItemID  uuid.UUID
	Title         string
	UngradedCount int64
}

// ListUngradedCountsForCourse returns assignments in a course that have ungraded submissions.
func ListUngradedCountsForCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]UngradedAssignmentCount, error) {
	rows, err := pool.Query(ctx, `
SELECT si.id, si.title, COUNT(s.id)::bigint
FROM course.course_structure_items si
INNER JOIN course.module_assignment_submissions s
	ON s.module_item_id = si.id AND s.course_id = $1
LEFT JOIN course.course_grades g
	ON g.module_item_id = s.module_item_id AND g.student_user_id = s.submitted_by
WHERE si.course_id = $1 AND si.kind = 'assignment' AND g.student_user_id IS NULL
GROUP BY si.id, si.title
HAVING COUNT(s.id) > 0
ORDER BY MIN(s.submitted_at) ASC, si.title ASC
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]UngradedAssignmentCount, 0)
	for rows.Next() {
		var row UngradedAssignmentCount
		if err := rows.Scan(&row.ModuleItemID, &row.Title, &row.UngradedCount); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func CountUngradedForAssignment(ctx context.Context, pool *pgxpool.Pool, courseID, moduleItemID uuid.UUID) (int64, error) {
	var n *int64
	err := pool.QueryRow(ctx, `
SELECT COUNT(*)::bigint
FROM course.module_assignment_submissions s
LEFT JOIN course.course_grades g ON g.module_item_id = s.module_item_id AND g.student_user_id = s.submitted_by
WHERE s.course_id = $1 AND s.module_item_id = $2 AND g.student_user_id IS NULL
`, courseID, moduleItemID).Scan(&n)
	if err != nil {
		return 0, err
	}
	if n == nil {
		return 0, nil
	}
	return *n, nil
}

func ListForAssignment(ctx context.Context, pool *pgxpool.Pool, courseID, moduleItemID uuid.UUID, filter GradedFilter) ([]SubmissionRow, error) {
	var gradedClause string
	switch filter {
	case GradedFilterGraded:
		gradedClause = "AND g.student_user_id IS NOT NULL"
	case GradedFilterUngraded:
		gradedClause = "AND g.student_user_id IS NULL"
	}
	rows, err := pool.Query(ctx, `
SELECT s.id, s.course_id, s.module_item_id, s.submitted_by, s.attachment_file_id, s.body_text, s.submitted_at, s.updated_at,
       s.resubmission_requested, s.revision_due_at, s.revision_feedback, s.version_number
FROM course.module_assignment_submissions s
LEFT JOIN course.course_grades g ON g.module_item_id = s.module_item_id AND g.student_user_id = s.submitted_by
WHERE s.course_id = $1 AND s.module_item_id = $2 `+gradedClause+`
ORDER BY s.submitted_at ASC, s.id ASC
`, courseID, moduleItemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]SubmissionRow, 0)
	for rows.Next() {
		s, err := scanSubmission(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func UpsertAttachment(ctx context.Context, pool *pgxpool.Pool, courseID, moduleItemID, submittedBy, attachmentFileID uuid.UUID) (*SubmissionRow, error) {
	return scanSubmission(pool.QueryRow(ctx, `
INSERT INTO course.module_assignment_submissions (course_id, module_item_id, submitted_by, attachment_file_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT (module_item_id, submitted_by) DO UPDATE
SET attachment_file_id = EXCLUDED.attachment_file_id, updated_at = NOW()
RETURNING id, course_id, module_item_id, submitted_by, attachment_file_id, body_text, submitted_at, updated_at,
          resubmission_requested, revision_due_at, revision_feedback, version_number
`, courseID, moduleItemID, submittedBy, attachmentFileID))
}

// UpsertBodyText records or updates typed text-entry content for a submission.
func UpsertBodyText(ctx context.Context, pool *pgxpool.Pool, courseID, moduleItemID, submittedBy uuid.UUID, bodyText string) (*SubmissionRow, error) {
	return scanSubmission(pool.QueryRow(ctx, `
INSERT INTO course.module_assignment_submissions (course_id, module_item_id, submitted_by, body_text)
VALUES ($1, $2, $3, $4)
ON CONFLICT (module_item_id, submitted_by) DO UPDATE
SET body_text = EXCLUDED.body_text, updated_at = NOW()
RETURNING id, course_id, module_item_id, submitted_by, attachment_file_id, body_text, submitted_at, updated_at,
          resubmission_requested, revision_due_at, revision_feedback, version_number
`, courseID, moduleItemID, submittedBy, bodyText))
}

// UpsertImportedInTransaction records or updates a Canvas-imported submission row inside an import tx.
func UpsertImportedInTransaction(
	ctx context.Context,
	tx pgx.Tx,
	courseID, moduleItemID, submittedBy uuid.UUID,
	attachmentFileID *uuid.UUID,
	submittedAt time.Time,
) (uuid.UUID, error) {
	var id uuid.UUID
	err := tx.QueryRow(ctx, `
INSERT INTO course.module_assignment_submissions (
	course_id, module_item_id, submitted_by, attachment_file_id, submitted_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $5)
ON CONFLICT (module_item_id, submitted_by) DO UPDATE SET
	attachment_file_id = COALESCE(EXCLUDED.attachment_file_id, course.module_assignment_submissions.attachment_file_id),
	submitted_at = CASE
		WHEN EXCLUDED.submitted_at > course.module_assignment_submissions.submitted_at THEN EXCLUDED.submitted_at
		ELSE course.module_assignment_submissions.submitted_at
	END,
	updated_at = GREATEST(course.module_assignment_submissions.updated_at, EXCLUDED.updated_at)
RETURNING id
`, courseID, moduleItemID, submittedBy, attachmentFileID, submittedAt).Scan(&id)
	return id, err
}

func ResubmitVersionedInTransaction(ctx context.Context, tx pgx.Tx, now time.Time, courseID, submissionID, newAttachmentFileID uuid.UUID) (*SubmissionRow, error) {
	if tx == nil {
		return nil, errors.New("db tx is nil")
	}
	cur, err := scanSubmission(tx.QueryRow(ctx, `
SELECT id, course_id, module_item_id, submitted_by, attachment_file_id, body_text, submitted_at, updated_at,
       resubmission_requested, revision_due_at, revision_feedback, version_number
FROM course.module_assignment_submissions
WHERE course_id = $1 AND id = $2
FOR UPDATE
`, courseID, submissionID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !cur.ResubmissionRequested {
		return nil, nil
	}
	if cur.RevisionDueAt != nil && cur.RevisionDueAt.Before(now) {
		return nil, nil
	}
	if cur.VersionNumber >= 10 {
		return nil, nil
	}
	if _, err := submissionversions.InsertArchived(ctx, tx, cur.CourseID, cur.ModuleItemID, cur.SubmittedBy, cur.VersionNumber, cur.AttachmentFileID, cur.SubmittedAt); err != nil {
		return nil, err
	}
	nextV := cur.VersionNumber + 1
	return scanSubmission(tx.QueryRow(ctx, `
UPDATE course.module_assignment_submissions
SET attachment_file_id = $1, submitted_at = $2, updated_at = $2, version_number = $3,
    resubmission_requested = false, revision_due_at = NULL, revision_feedback = NULL
WHERE id = $4
RETURNING id, course_id, module_item_id, submitted_by, attachment_file_id, body_text, submitted_at, updated_at,
          resubmission_requested, revision_due_at, revision_feedback, version_number
`, newAttachmentFileID, now, nextV, cur.ID))
}

func SetRevisionRequest(ctx context.Context, pool *pgxpool.Pool, courseID, submissionID uuid.UUID, revisionDueAt *time.Time, revisionFeedback *string) (*SubmissionRow, error) {
	return setRevisionRequestExec(ctx, pool, courseID, submissionID, revisionDueAt, revisionFeedback)
}

func SetRevisionRequestInTransaction(ctx context.Context, tx pgx.Tx, courseID, submissionID uuid.UUID, revisionDueAt *time.Time, revisionFeedback *string) (*SubmissionRow, error) {
	return setRevisionRequestExec(ctx, tx, courseID, submissionID, revisionDueAt, revisionFeedback)
}

type queryRower interface{ QueryRow(context.Context, string, ...any) pgx.Row }

func setRevisionRequestExec(ctx context.Context, q queryRower, courseID, submissionID uuid.UUID, revisionDueAt *time.Time, revisionFeedback *string) (*SubmissionRow, error) {
	s, err := scanSubmission(q.QueryRow(ctx, `
UPDATE course.module_assignment_submissions
SET resubmission_requested = true, revision_due_at = $1, revision_feedback = $2, updated_at = NOW()
WHERE course_id = $3 AND id = $4
RETURNING id, course_id, module_item_id, submitted_by, attachment_file_id, body_text, submitted_at, updated_at,
          resubmission_requested, revision_due_at, revision_feedback, version_number
`, revisionDueAt, revisionFeedback, courseID, submissionID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return s, err
}
