package originalityreports

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Report is one row from course.originality_reports.
type Report struct {
	ID               uuid.UUID
	SubmissionID     uuid.UUID
	Provider         string
	Status           string
	SimilarityPct    *float64
	AIProbability    *float64
	ReportURL        *string
	ReportToken      *string
	ProviderReportID *string
	ErrorMessage     *string
	ReportStorageKey *string
	SnapshotStorageKey *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func scanReport(row pgx.Row) (*Report, error) {
	var r Report
	var sim, ai sql.NullFloat64
	var reportURL, reportToken, providerReportID, errMsg, reportKey, snapKey sql.NullString
	err := row.Scan(
		&r.ID, &r.SubmissionID, &r.Provider, &r.Status,
		&sim, &ai, &reportURL, &reportToken, &providerReportID, &errMsg,
		&reportKey, &snapKey, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if sim.Valid {
		v := sim.Float64
		r.SimilarityPct = &v
	}
	if ai.Valid {
		v := ai.Float64
		r.AIProbability = &v
	}
	if reportURL.Valid {
		v := reportURL.String
		r.ReportURL = &v
	}
	if reportToken.Valid {
		v := reportToken.String
		r.ReportToken = &v
	}
	if providerReportID.Valid {
		v := providerReportID.String
		r.ProviderReportID = &v
	}
	if errMsg.Valid {
		v := errMsg.String
		r.ErrorMessage = &v
	}
	if reportKey.Valid {
		v := reportKey.String
		r.ReportStorageKey = &v
	}
	if snapKey.Valid {
		v := snapKey.String
		r.SnapshotStorageKey = &v
	}
	return &r, nil
}

const reportSelectCols = `
SELECT id, submission_id, provider, status,
	similarity_pct::float8, ai_probability::float8,
	report_url, report_token, provider_report_id, error_message,
	report_storage_key, snapshot_storage_key, created_at, updated_at
FROM course.originality_reports
`

// ListBySubmission returns all reports for a submission ordered by provider.
func ListBySubmission(ctx context.Context, pool *pgxpool.Pool, submissionID uuid.UUID) ([]Report, error) {
	rows, err := pool.Query(ctx, reportSelectCols+`
WHERE submission_id = $1
ORDER BY provider ASC`, submissionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Report
	for rows.Next() {
		r, err := scanReport(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

// InsertPending creates a pending report row if one does not exist for (submission, provider).
func InsertPending(ctx context.Context, pool *pgxpool.Pool, submissionID uuid.UUID, provider string) (*Report, error) {
	r, err := scanReport(pool.QueryRow(ctx, `
INSERT INTO course.originality_reports (submission_id, provider, status)
VALUES ($1, $2, 'pending')
ON CONFLICT (submission_id, provider) DO NOTHING
RETURNING id, submission_id, provider, status,
	similarity_pct::float8, ai_probability::float8,
	report_url, report_token, provider_report_id, error_message,
	report_storage_key, snapshot_storage_key, created_at, updated_at
`, submissionID, provider))
	if errors.Is(err, pgx.ErrNoRows) {
		return GetBySubmissionProvider(ctx, pool, submissionID, provider)
	}
	return r, err
}

// GetBySubmissionProvider returns the report for a submission and provider.
func GetBySubmissionProvider(ctx context.Context, pool *pgxpool.Pool, submissionID uuid.UUID, provider string) (*Report, error) {
	r, err := scanReport(pool.QueryRow(ctx, reportSelectCols+`
WHERE submission_id = $1 AND provider = $2`, submissionID, provider))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return r, err
}

// ClaimNextPending picks one pending report for processing (FOR UPDATE SKIP LOCKED).
func ClaimNextPending(ctx context.Context, pool *pgxpool.Pool) (*Report, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	r, err := scanReport(tx.QueryRow(ctx, reportSelectCols+`
WHERE status = 'pending'
ORDER BY created_at
LIMIT 1
FOR UPDATE SKIP LOCKED`))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `
UPDATE course.originality_reports
SET status = 'processing', updated_at = NOW()
WHERE id = $1`, r.ID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	r.Status = "processing"
	return r, nil
}

// MarkDone updates a report with scan results.
func MarkDone(ctx context.Context, pool *pgxpool.Pool, reportID uuid.UUID, similarityPct, aiProbability *float64, reportURL, reportToken, providerReportID *string) error {
	_, err := pool.Exec(ctx, `
UPDATE course.originality_reports
SET status = 'done',
	similarity_pct = COALESCE($2::numeric, similarity_pct),
	ai_probability = COALESCE($3::numeric, ai_probability),
	report_url = COALESCE($4, report_url),
	report_token = COALESCE($5, report_token),
	provider_report_id = COALESCE($6, provider_report_id),
	error_message = NULL,
	updated_at = NOW()
WHERE id = $1`, reportID, similarityPct, aiProbability, reportURL, reportToken, providerReportID)
	return err
}

// MarkFailed records a scan failure.
func MarkFailed(ctx context.Context, pool *pgxpool.Pool, reportID uuid.UUID, message string) error {
	_, err := pool.Exec(ctx, `
UPDATE course.originality_reports
SET status = 'failed', error_message = $2, updated_at = NOW()
WHERE id = $1`, reportID, message)
	return err
}

// ResetForRetry sets a failed report back to pending.
func ResetForRetry(ctx context.Context, pool *pgxpool.Pool, submissionID uuid.UUID, provider string) (bool, error) {
	tag, err := pool.Exec(ctx, `
UPDATE course.originality_reports
SET status = 'pending', error_message = NULL, updated_at = NOW()
WHERE submission_id = $1 AND provider = $2 AND status = 'failed'`, submissionID, provider)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// SubmissionNeedingScan is a submission that should have originality reports enqueued.
type SubmissionNeedingScan struct {
	SubmissionID       uuid.UUID
	CourseID           uuid.UUID
	CourseCode         string
	ModuleItemID       uuid.UUID
	OriginalityMode    string
	AttachmentFileID   *uuid.UUID
}

// ListSubmissionsNeedingEnqueue finds submissions with originality enabled but missing report rows.
func ListSubmissionsNeedingEnqueue(ctx context.Context, pool *pgxpool.Pool, limit int) ([]SubmissionNeedingScan, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := pool.Query(ctx, `
SELECT s.id, s.course_id, c.course_code, s.module_item_id, m.originality_detection, s.attachment_file_id
FROM course.module_assignment_submissions s
INNER JOIN course.courses c ON c.id = s.course_id
INNER JOIN course.module_assignments m ON m.structure_item_id = s.module_item_id
WHERE COALESCE(c.plagiarism_checks_enabled, true) = true
  AND m.originality_detection <> 'disabled'
  AND NOT EXISTS (
    SELECT 1 FROM course.originality_reports r WHERE r.submission_id = s.id
  )
ORDER BY s.submitted_at DESC
LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SubmissionNeedingScan
	for rows.Next() {
		var row SubmissionNeedingScan
		var attach *uuid.UUID
		if err := rows.Scan(&row.SubmissionID, &row.CourseID, &row.CourseCode, &row.ModuleItemID, &row.OriginalityMode, &attach); err != nil {
			return nil, err
		}
		row.AttachmentFileID = attach
		out = append(out, row)
	}
	return out, rows.Err()
}

// SubmissionContext loads assignment + course settings for a submission.
type SubmissionContext struct {
	SubmissionID     uuid.UUID
	CourseID         uuid.UUID
	CourseCode       string
	ModuleItemID     uuid.UUID
	SubmittedBy      uuid.UUID
	OriginalityMode  string
	StudentVisibility string
	AttachmentFileID *uuid.UUID
	PlagiarismEnabled bool
}

// GetSubmissionContext loads metadata needed for originality access checks and scanning.
func GetSubmissionContext(ctx context.Context, pool *pgxpool.Pool, courseCode string, submissionID uuid.UUID) (*SubmissionContext, error) {
	var sc SubmissionContext
	var attach *uuid.UUID
	err := pool.QueryRow(ctx, `
SELECT s.id, s.course_id, c.course_code, s.module_item_id, s.submitted_by,
       m.originality_detection, m.originality_student_visibility, s.attachment_file_id,
       COALESCE(c.plagiarism_checks_enabled, true)
FROM course.module_assignment_submissions s
INNER JOIN course.courses c ON c.id = s.course_id AND c.course_code = $1
INNER JOIN course.module_assignments m ON m.structure_item_id = s.module_item_id
WHERE s.id = $2
`, courseCode, submissionID).Scan(
		&sc.SubmissionID, &sc.CourseID, &sc.CourseCode, &sc.ModuleItemID, &sc.SubmittedBy,
		&sc.OriginalityMode, &sc.StudentVisibility, &attach, &sc.PlagiarismEnabled,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	sc.AttachmentFileID = attach
	return &sc, nil
}

// GetSubmissionContextByID loads submission metadata by submission id only.
func GetSubmissionContextByID(ctx context.Context, pool *pgxpool.Pool, submissionID uuid.UUID) (*SubmissionContext, error) {
	var sc SubmissionContext
	var attach *uuid.UUID
	err := pool.QueryRow(ctx, `
SELECT s.id, s.course_id, c.course_code, s.module_item_id, s.submitted_by,
       m.originality_detection, m.originality_student_visibility, s.attachment_file_id,
       COALESCE(c.plagiarism_checks_enabled, true)
FROM course.module_assignment_submissions s
INNER JOIN course.courses c ON c.id = s.course_id
INNER JOIN course.module_assignments m ON m.structure_item_id = s.module_item_id
WHERE s.id = $1
`, submissionID).Scan(
		&sc.SubmissionID, &sc.CourseID, &sc.CourseCode, &sc.ModuleItemID, &sc.SubmittedBy,
		&sc.OriginalityMode, &sc.StudentVisibility, &attach, &sc.PlagiarismEnabled,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	sc.AttachmentFileID = attach
	return &sc, nil
}
