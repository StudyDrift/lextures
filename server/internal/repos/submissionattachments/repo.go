package submissionattachments

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AttachmentRow struct {
	FileID           uuid.UUID
	SortOrder        int
	OriginalFilename string
	MimeType         string
}

func ListForSubmission(ctx context.Context, pool *pgxpool.Pool, submissionID uuid.UUID) ([]AttachmentRow, error) {
	rows, err := pool.Query(ctx, `
SELECT sa.file_id, sa.sort_order, cf.original_filename, cf.mime_type
FROM course.submission_attachments sa
INNER JOIN course.course_files cf ON cf.id = sa.file_id
WHERE sa.submission_id = $1
ORDER BY sa.sort_order ASC, sa.created_at ASC, sa.id ASC
`, submissionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]AttachmentRow, 0)
	for rows.Next() {
		var row AttachmentRow
		if err := rows.Scan(&row.FileID, &row.SortOrder, &row.OriginalFilename, &row.MimeType); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func ListForSubmissionInTransaction(ctx context.Context, tx pgx.Tx, submissionID uuid.UUID) ([]AttachmentRow, error) {
	rows, err := tx.Query(ctx, `
SELECT sa.file_id, sa.sort_order, cf.original_filename, cf.mime_type
FROM course.submission_attachments sa
INNER JOIN course.course_files cf ON cf.id = sa.file_id
WHERE sa.submission_id = $1
ORDER BY sa.sort_order ASC, sa.created_at ASC, sa.id ASC
`, submissionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]AttachmentRow, 0)
	for rows.Next() {
		var row AttachmentRow
		if err := rows.Scan(&row.FileID, &row.SortOrder, &row.OriginalFilename, &row.MimeType); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func ReplaceForSubmission(ctx context.Context, pool *pgxpool.Pool, submissionID uuid.UUID, fileIDs []uuid.UUID) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := ReplaceForSubmissionInTransaction(ctx, tx, submissionID, fileIDs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func ReplaceForSubmissionInTransaction(ctx context.Context, tx pgx.Tx, submissionID uuid.UUID, fileIDs []uuid.UUID) error {
	if _, err := tx.Exec(ctx, `DELETE FROM course.submission_attachments WHERE submission_id = $1`, submissionID); err != nil {
		return err
	}
	return insertOrdered(ctx, tx, submissionID, fileIDs)
}

func insertOrdered(ctx context.Context, tx pgx.Tx, submissionID uuid.UUID, fileIDs []uuid.UUID) error {
	for i, fileID := range fileIDs {
		if fileID == uuid.Nil {
			continue
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO course.submission_attachments (submission_id, file_id, sort_order)
VALUES ($1, $2, $3)
ON CONFLICT (submission_id, file_id) DO UPDATE SET sort_order = EXCLUDED.sort_order
`, submissionID, fileID, i); err != nil {
			return err
		}
	}
	return nil
}