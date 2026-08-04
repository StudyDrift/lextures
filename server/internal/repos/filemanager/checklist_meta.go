package filemanager

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FileMetaRow is lightweight file metadata for the course checklist snapshot.
type FileMetaRow struct {
	ID          uuid.UUID
	DisplayName string
	ContentType string
	ByteSize    int64
	StorageKey  string
	// TextLayer is filled by the checklist loader after PDF probing (CC.6).
	TextLayer string
}

// ListFileMetaForCourse returns file_items metadata for a course (no content bytes).
// Read-only helper for CC.1 LoadSnapshot.
func ListFileMetaForCourse(ctx context.Context, db *pgxpool.Pool, courseID uuid.UUID) ([]FileMetaRow, error) {
	rows, err := db.Query(ctx, `
SELECT id, display_name, mime_type, byte_size, storage_key
FROM course.file_items
WHERE course_id = $1
ORDER BY display_name ASC
LIMIT 2000
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []FileMetaRow
	for rows.Next() {
		var r FileMetaRow
		if err := rows.Scan(&r.ID, &r.DisplayName, &r.ContentType, &r.ByteSize, &r.StorageKey); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
