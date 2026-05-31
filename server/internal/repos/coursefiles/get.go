package coursefiles

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Row is a course file metadata row (Rust `CourseFileRow`).
type Row struct {
	ID               uuid.UUID
	CourseID         uuid.UUID
	StorageKey       string
	OriginalFilename string
	MimeType         string
	ByteSize         int64
}

// GetForCourse returns the file if it exists and belongs to the course code.
func GetForCourse(ctx context.Context, pool *pgxpool.Pool, courseCode string, fileID uuid.UUID) (*Row, error) {
	var r Row
	err := pool.QueryRow(ctx, `
		SELECT f.id, f.course_id, f.storage_key, f.original_filename, f.mime_type, f.byte_size
		FROM course.course_files f
		INNER JOIN course.courses c ON c.id = f.course_id AND c.course_code = $2
		WHERE f.id = $1
	`, fileID, courseCode).Scan(
		&r.ID, &r.CourseID, &r.StorageKey, &r.OriginalFilename, &r.MimeType, &r.ByteSize,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// Create inserts a row for a newly-uploaded course file (used by feed image uploads, etc.).
func Create(ctx context.Context, pool *pgxpool.Pool, courseID, uploadedBy uuid.UUID, storageKey, originalFilename, mimeType string, byteSize int64) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
		INSERT INTO course.course_files (course_id, storage_key, original_filename, mime_type, byte_size, uploaded_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, courseID, storageKey, originalFilename, mimeType, byteSize, uploadedBy).Scan(&id)
	return id, err
}
