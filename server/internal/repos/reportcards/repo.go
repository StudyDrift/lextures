// Package reportcards persists plan 13.4 report card data (comment bank + per-student cards).
package reportcards

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CommentBankEntry is one row from report_card.comment_bank.
type CommentBankEntry struct {
	ID       uuid.UUID
	OrgID    uuid.UUID
	Category string
	Text     string
	Active   bool
}

// ReportCard is one row from report_card.report_cards.
type ReportCard struct {
	ID            uuid.UUID
	StudentID     uuid.UUID
	CourseID      uuid.UUID
	GradingPeriod string
	FinalGradePct *float64
	LetterGrade   *string
	Comment       *string
	Status        string
	PDFURL        *string
	GeneratedAt   *time.Time
	ReleasedAt    *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// UpsertCommentBankEntry creates or updates a comment bank entry (keyed on org+category+text).
func UpsertCommentBankEntry(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, category, text string) (*CommentBankEntry, error) {
	row := pool.QueryRow(ctx, `
INSERT INTO report_card.comment_bank (org_id, category, text)
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING
RETURNING id, org_id, category, text, active`, orgID, category, text)
	e := &CommentBankEntry{}
	err := row.Scan(&e.ID, &e.OrgID, &e.Category, &e.Text, &e.Active)
	if err == pgx.ErrNoRows {
		// Already existed; fetch it.
		row2 := pool.QueryRow(ctx, `
SELECT id, org_id, category, text, active
FROM report_card.comment_bank
WHERE org_id = $1 AND category = $2 AND text = $3`, orgID, category, text)
		err = row2.Scan(&e.ID, &e.OrgID, &e.Category, &e.Text, &e.Active)
	}
	if err != nil {
		return nil, err
	}
	return e, nil
}

// ListCommentBank returns all active entries for an org, optionally filtered by category.
func ListCommentBank(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, category string) ([]CommentBankEntry, error) {
	var rows pgx.Rows
	var err error
	if category != "" {
		rows, err = pool.Query(ctx, `
SELECT id, org_id, category, text, active
FROM report_card.comment_bank
WHERE org_id = $1 AND active AND category = $2
ORDER BY category, text`, orgID, category)
	} else {
		rows, err = pool.Query(ctx, `
SELECT id, org_id, category, text, active
FROM report_card.comment_bank
WHERE org_id = $1 AND active
ORDER BY category, text`, orgID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CommentBankEntry
	for rows.Next() {
		var e CommentBankEntry
		if err := rows.Scan(&e.ID, &e.OrgID, &e.Category, &e.Text, &e.Active); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// DeleteCommentBankEntry soft-deletes (deactivates) a comment bank entry. Returns false if not found.
func DeleteCommentBankEntry(ctx context.Context, pool *pgxpool.Pool, orgID, entryID uuid.UUID) (bool, error) {
	tag, err := pool.Exec(ctx, `
UPDATE report_card.comment_bank SET active = false
WHERE id = $1 AND org_id = $2 AND active`, entryID, orgID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// PatchReportCard updates comment and/or status for an existing card.
func PatchReportCard(ctx context.Context, pool *pgxpool.Pool, cardID uuid.UUID, comment *string, status *string) (*ReportCard, error) {
	row := pool.QueryRow(ctx, `
UPDATE report_card.report_cards SET
    comment    = COALESCE($2, comment),
    status     = COALESCE($3, status),
    updated_at = now()
WHERE id = $1
RETURNING id, student_id, course_id, grading_period, final_grade_pct, letter_grade, comment,
          status, pdf_url, generated_at, released_at, created_at, updated_at`,
		cardID, comment, status)
	return scanReportCard(row)
}

// SetPDFURL persists the generated PDF URL and marks generated_at.
func SetPDFURL(ctx context.Context, pool *pgxpool.Pool, cardID uuid.UUID, pdfURL string) error {
	_, err := pool.Exec(ctx, `
UPDATE report_card.report_cards
SET pdf_url = $2, generated_at = now(), updated_at = now()
WHERE id = $1`, cardID, pdfURL)
	return err
}

// ReleaseCards marks all approved cards for a course/period as released.
func ReleaseCards(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, period string) (int, error) {
	tag, err := pool.Exec(ctx, `
UPDATE report_card.report_cards
SET status = 'released', released_at = now(), updated_at = now()
WHERE course_id = $1 AND grading_period = $2 AND status = 'approved'`, courseID, period)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// ListForCoursePeriod returns all report cards for a given course + grading period.
func ListForCoursePeriod(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, period string) ([]ReportCard, error) {
	rows, err := pool.Query(ctx, `
SELECT id, student_id, course_id, grading_period, final_grade_pct, letter_grade, comment,
       status, pdf_url, generated_at, released_at, created_at, updated_at
FROM report_card.report_cards
WHERE course_id = $1 AND grading_period = $2
ORDER BY created_at`, courseID, period)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ReportCard
	for rows.Next() {
		rc, err := scanReportCardRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *rc)
	}
	return out, rows.Err()
}

// GetByID retrieves a single report card by ID.
func GetByID(ctx context.Context, pool *pgxpool.Pool, cardID uuid.UUID) (*ReportCard, error) {
	row := pool.QueryRow(ctx, `
SELECT id, student_id, course_id, grading_period, final_grade_pct, letter_grade, comment,
       status, pdf_url, generated_at, released_at, created_at, updated_at
FROM report_card.report_cards
WHERE id = $1`, cardID)
	rc, err := scanReportCard(row)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return rc, err
}

// ListReleasedForStudent returns all released report cards for a student.
func ListReleasedForStudent(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID) ([]ReportCard, error) {
	rows, err := pool.Query(ctx, `
SELECT id, student_id, course_id, grading_period, final_grade_pct, letter_grade, comment,
       status, pdf_url, generated_at, released_at, created_at, updated_at
FROM report_card.report_cards
WHERE student_id = $1 AND status = 'released'
ORDER BY grading_period DESC`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ReportCard
	for rows.Next() {
		rc, err := scanReportCardRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *rc)
	}
	return out, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanReportCard(row scanner) (*ReportCard, error) {
	rc := &ReportCard{}
	err := row.Scan(
		&rc.ID, &rc.StudentID, &rc.CourseID, &rc.GradingPeriod,
		&rc.FinalGradePct, &rc.LetterGrade, &rc.Comment,
		&rc.Status, &rc.PDFURL, &rc.GeneratedAt, &rc.ReleasedAt,
		&rc.CreatedAt, &rc.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return rc, nil
}

func scanReportCardRow(rows pgx.Rows) (*ReportCard, error) {
	rc := &ReportCard{}
	err := rows.Scan(
		&rc.ID, &rc.StudentID, &rc.CourseID, &rc.GradingPeriod,
		&rc.FinalGradePct, &rc.LetterGrade, &rc.Comment,
		&rc.Status, &rc.PDFURL, &rc.GeneratedAt, &rc.ReleasedAt,
		&rc.CreatedAt, &rc.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return rc, nil
}
