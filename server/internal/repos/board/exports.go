package board

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Export formats (VC.9).
const (
	ExportFormatPDF   = "pdf"
	ExportFormatCSV   = "csv"
	ExportFormatImage = "image"
)

// Export job statuses (VC.9).
const (
	ExportStatusPending = "pending"
	ExportStatusRunning = "running"
	ExportStatusDone    = "done"
	ExportStatusFailed  = "failed"
)

// ExportJob tracks an async board export.
type ExportJob struct {
	ID                string     `json:"id"`
	BoardID           string     `json:"boardId"`
	Format            string     `json:"format"`
	Status            string     `json:"status"`
	StorageKey        *string    `json:"storageKey,omitempty"`
	Error             string     `json:"error"`
	IncludeModeration bool       `json:"includeModeration"`
	RequestedBy       *string    `json:"requestedBy,omitempty"`
	CreatedAt         time.Time  `json:"createdAt"`
	CompletedAt       *time.Time `json:"completedAt,omitempty"`
}

// NormalizeExportFormat validates format.
func NormalizeExportFormat(raw string) (string, error) {
	v := strings.ToLower(strings.TrimSpace(raw))
	switch v {
	case ExportFormatPDF, ExportFormatCSV, ExportFormatImage:
		return v, nil
	default:
		return "", fmt.Errorf("board: invalid export format")
	}
}

// CreateExportJob inserts a pending export job.
func CreateExportJob(
	ctx context.Context,
	pool *pgxpool.Pool,
	boardID uuid.UUID,
	requestedBy uuid.UUID,
	format string,
	includeModeration bool,
) (*ExportJob, error) {
	format, err := NormalizeExportFormat(format)
	if err != nil {
		return nil, err
	}
	var id uuid.UUID
	err = pool.QueryRow(ctx, `
		INSERT INTO board.export_jobs (board_id, format, status, include_moderation, requested_by)
		VALUES ($1, $2, 'pending', $3, $4)
		RETURNING id
	`, boardID, format, includeModeration, requestedBy).Scan(&id)
	if err != nil {
		return nil, err
	}
	return GetExportJob(ctx, pool, id.String())
}

// GetExportJob loads an export job by id.
func GetExportJob(ctx context.Context, pool *pgxpool.Pool, jobID string) (*ExportJob, error) {
	id, err := uuid.Parse(jobID)
	if err != nil {
		return nil, nil
	}
	var j ExportJob
	var bid uuid.UUID
	var storageKey *string
	var requestedBy uuid.NullUUID
	var completedAt *time.Time
	err = pool.QueryRow(ctx, `
		SELECT id, board_id, format, status, storage_key, error, include_moderation,
			requested_by, created_at, completed_at
		FROM board.export_jobs
		WHERE id = $1
	`, id).Scan(
		&id, &bid, &j.Format, &j.Status, &storageKey, &j.Error, &j.IncludeModeration,
		&requestedBy, &j.CreatedAt, &completedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	j.ID = id.String()
	j.BoardID = bid.String()
	j.StorageKey = storageKey
	j.CompletedAt = completedAt
	if requestedBy.Valid {
		s := requestedBy.UUID.String()
		j.RequestedBy = &s
	}
	return &j, nil
}

// GetExportJobForBoard ensures the job belongs to the board in the given course.
func GetExportJobForBoard(ctx context.Context, pool *pgxpool.Pool, courseCode, boardID, jobID string) (*ExportJob, error) {
	j, err := GetExportJob(ctx, pool, jobID)
	if err != nil || j == nil {
		return j, err
	}
	if j.BoardID != boardID {
		return nil, nil
	}
	var code string
	err = pool.QueryRow(ctx, `
		SELECT c.course_code
		FROM board.boards b
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE b.id = $1::uuid
	`, j.BoardID).Scan(&code)
	if err != nil || code != courseCode {
		return nil, nil
	}
	return j, nil
}

// UpdateExportJobStatus updates status/storage/error for an export job.
func UpdateExportJobStatus(
	ctx context.Context,
	pool *pgxpool.Pool,
	jobID string,
	status string,
	storageKey *string,
	errMsg string,
) error {
	id, err := uuid.Parse(jobID)
	if err != nil {
		return fmt.Errorf("board: invalid job id")
	}
	status = strings.ToLower(strings.TrimSpace(status))
	switch status {
	case ExportStatusPending, ExportStatusRunning, ExportStatusDone, ExportStatusFailed:
	default:
		return fmt.Errorf("board: invalid export status")
	}
	completed := status == ExportStatusDone || status == ExportStatusFailed
	_, err = pool.Exec(ctx, `
		UPDATE board.export_jobs
		SET status = $2,
			storage_key = COALESCE($3, storage_key),
			error = $4,
			completed_at = CASE WHEN $5 THEN NOW() ELSE completed_at END
		WHERE id = $1
	`, id, status, storageKey, errMsg, completed)
	return err
}
