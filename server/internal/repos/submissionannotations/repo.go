package submissionannotations

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AnnotationRow struct {
	ID           uuid.UUID
	SubmissionID uuid.UUID
	AnnotatorID  uuid.UUID
	ClientID     string
	Page         int32
	ToolType     string
	Colour       string
	CoordsJSON   json.RawMessage
	Body         *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

type AnnotationUpsertWrite struct {
	SubmissionID uuid.UUID
	AnnotatorID  uuid.UUID
	ClientID     string
	Page         int32
	ToolType     string
	Colour       string
	CoordsJSON   json.RawMessage
	Body         *string
}

const selectColumns = `id, submission_id, annotator_id, client_id, page, tool_type, colour, coords_json, body, created_at, updated_at, deleted_at`

func scanRow(row pgx.Row) (*AnnotationRow, error) {
	var a AnnotationRow
	if err := row.Scan(
		&a.ID, &a.SubmissionID, &a.AnnotatorID, &a.ClientID, &a.Page,
		&a.ToolType, &a.Colour, &a.CoordsJSON, &a.Body,
		&a.CreatedAt, &a.UpdatedAt, &a.DeletedAt,
	); err != nil {
		return nil, err
	}
	return &a, nil
}

// ListBySubmission returns all live (non-deleted) annotations for a submission, oldest first.
func ListBySubmission(ctx context.Context, pool *pgxpool.Pool, submissionID uuid.UUID) ([]AnnotationRow, error) {
	rows, err := pool.Query(ctx, `
SELECT `+selectColumns+`
FROM course.submission_annotations
WHERE submission_id = $1 AND deleted_at IS NULL
ORDER BY created_at ASC
`, submissionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]AnnotationRow, 0)
	for rows.Next() {
		a, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

// Upsert inserts a new annotation, or — when (submission_id, annotator_id, client_id) already
// exists — updates its geometry/body in place. This makes POST idempotent on retry and lets the
// client edit its own pending annotation by re-posting the same client id.
func Upsert(ctx context.Context, pool *pgxpool.Pool, w AnnotationUpsertWrite) (*AnnotationRow, error) {
	coords := w.CoordsJSON
	if len(coords) == 0 {
		coords = json.RawMessage(`{}`)
	}
	return scanRow(pool.QueryRow(ctx, `
INSERT INTO course.submission_annotations
    (submission_id, annotator_id, client_id, page, tool_type, colour, coords_json, body)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (submission_id, annotator_id, client_id) DO UPDATE SET
    page = EXCLUDED.page,
    tool_type = EXCLUDED.tool_type,
    colour = EXCLUDED.colour,
    coords_json = EXCLUDED.coords_json,
    body = EXCLUDED.body,
    deleted_at = NULL,
    updated_at = NOW()
RETURNING `+selectColumns,
		w.SubmissionID, w.AnnotatorID, w.ClientID, w.Page,
		w.ToolType, w.Colour, coords, w.Body,
	))
}

// SoftDelete marks an annotation deleted. Returns false when no live annotation matched.
func SoftDelete(ctx context.Context, pool *pgxpool.Pool, submissionID, annotationID uuid.UUID) (bool, error) {
	tag, err := pool.Exec(ctx, `
UPDATE course.submission_annotations
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1 AND submission_id = $2 AND deleted_at IS NULL
`, annotationID, submissionID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// GetByID loads a single live annotation, or nil when missing.
func GetByID(ctx context.Context, pool *pgxpool.Pool, annotationID uuid.UUID) (*AnnotationRow, error) {
	a, err := scanRow(pool.QueryRow(ctx, `
SELECT `+selectColumns+`
FROM course.submission_annotations
WHERE id = $1 AND deleted_at IS NULL
`, annotationID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}
