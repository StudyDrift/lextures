package board

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DefaultExportRetentionDays is how long completed export files are kept (VC.10 FR-7).
const DefaultExportRetentionDays = 30

// DefaultArchivedBoardRetentionDays is how long archived boards are retained before purge.
const DefaultArchivedBoardRetentionDays = 365

// ExpiredExport is an export job past the retention window with a storage key.
type ExpiredExport struct {
	ID         uuid.UUID
	BoardID    uuid.UUID
	StorageKey string
}

// ListExpiredExports returns completed export jobs older than cutoff that still have files.
func ListExpiredExports(ctx context.Context, pool *pgxpool.Pool, cutoff time.Time) ([]ExpiredExport, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, board_id, storage_key
		FROM board.export_jobs
		WHERE status = 'done'
		  AND storage_key IS NOT NULL
		  AND storage_key <> ''
		  AND COALESCE(completed_at, created_at) < $1
		ORDER BY created_at ASC
		LIMIT 500
	`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ExpiredExport, 0)
	for rows.Next() {
		var e ExpiredExport
		if err := rows.Scan(&e.ID, &e.BoardID, &e.StorageKey); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ClearExportStorageKey clears the storage key after the blob was deleted.
func ClearExportStorageKey(ctx context.Context, pool *pgxpool.Pool, jobID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
		UPDATE board.export_jobs
		SET storage_key = NULL
		WHERE id = $1
	`, jobID)
	return err
}

// ArchivedBoardForPurge is an archived board past the retention window.
type ArchivedBoardForPurge struct {
	ID          uuid.UUID
	CourseCode  string
	StorageKeys []string
}

// ListArchivedBoardsForPurge returns archived boards whose updated_at is older than cutoff.
func ListArchivedBoardsForPurge(ctx context.Context, pool *pgxpool.Pool, cutoff time.Time, limit int) ([]ArchivedBoardForPurge, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := pool.Query(ctx, `
		SELECT b.id, c.course_code
		FROM board.boards b
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE b.archived = TRUE AND b.updated_at < $1
		ORDER BY b.updated_at ASC
		LIMIT $2
	`, cutoff, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ArchivedBoardForPurge, 0)
	for rows.Next() {
		var b ArchivedBoardForPurge
		if err := rows.Scan(&b.ID, &b.CourseCode); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		keys, err := listAttachmentKeysForBoard(ctx, pool, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].StorageKeys = keys
		exportKeys, err := listExportKeysForBoard(ctx, pool, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].StorageKeys = append(out[i].StorageKeys, exportKeys...)
	}
	return out, nil
}

func listAttachmentKeysForBoard(ctx context.Context, pool *pgxpool.Pool, boardID uuid.UUID) ([]string, error) {
	rows, err := pool.Query(ctx, `
		SELECT storage_key FROM board.post_attachments WHERE board_id = $1
	`, boardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	keys := make([]string, 0)
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func listExportKeysForBoard(ctx context.Context, pool *pgxpool.Pool, boardID uuid.UUID) ([]string, error) {
	rows, err := pool.Query(ctx, `
		SELECT storage_key FROM board.export_jobs
		WHERE board_id = $1 AND storage_key IS NOT NULL AND storage_key <> ''
	`, boardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	keys := make([]string, 0)
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// PurgeBoard deletes a board row (cascades posts/attachments/exports metadata).
func PurgeBoard(ctx context.Context, pool *pgxpool.Pool, boardID uuid.UUID) error {
	_, err := pool.Exec(ctx, `DELETE FROM board.boards WHERE id = $1`, boardID)
	return err
}
