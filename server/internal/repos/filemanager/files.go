package filemanager

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func GetFileItem(ctx context.Context, db *pgxpool.Pool, courseID, itemID uuid.UUID) (*FileItem, error) {
	row := db.QueryRow(ctx, `
		SELECT id, course_id, folder_id, storage_key, original_filename, display_name,
		       mime_type, byte_size, uploaded_by, canvas_file_id, created_at, updated_at
		FROM course.file_items
		WHERE id = $1 AND course_id = $2
	`, itemID, courseID)
	fi, err := scanFileItem(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &fi, nil
}

// CreateFileItem inserts a file metadata record. Called after the blob is stored.
func CreateFileItem(ctx context.Context, db *pgxpool.Pool, courseID uuid.UUID, folderID *uuid.UUID, storageKey, originalFilename, displayName, mimeType string, byteSize int64, uploadedBy *uuid.UUID) (*FileItem, error) {
	id := uuid.New()
	_, err := db.Exec(ctx, `
		INSERT INTO course.file_items
		    (id, course_id, folder_id, storage_key, original_filename, display_name, mime_type, byte_size, uploaded_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, id, courseID, folderID, storageKey, originalFilename, displayName, mimeType, byteSize, uploadedBy)
	if err != nil {
		return nil, err
	}
	return GetFileItem(ctx, db, courseID, id)
}

// CreateFileItemWithCanvas is like CreateFileItem but also stores the canvas_file_id.
func CreateFileItemWithCanvas(ctx context.Context, db *pgxpool.Pool, courseID uuid.UUID, folderID *uuid.UUID, storageKey, originalFilename, displayName, mimeType string, byteSize int64, uploadedBy *uuid.UUID, canvasFileID int64) (*FileItem, error) {
	id := uuid.New()
	_, err := db.Exec(ctx, `
		INSERT INTO course.file_items
		    (id, course_id, folder_id, storage_key, original_filename, display_name, mime_type, byte_size, uploaded_by, canvas_file_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (storage_key) DO NOTHING
	`, id, courseID, folderID, storageKey, originalFilename, displayName, mimeType, byteSize, uploadedBy, canvasFileID)
	if err != nil {
		return nil, err
	}
	// Return the row (may already exist from a previous import)
	row := db.QueryRow(ctx, `
		SELECT id, course_id, folder_id, storage_key, original_filename, display_name,
		       mime_type, byte_size, uploaded_by, canvas_file_id, created_at, updated_at
		FROM course.file_items WHERE storage_key = $1 AND course_id = $2
	`, storageKey, courseID)
	fi, scanErr := scanFileItem(row)
	if scanErr != nil {
		return nil, scanErr
	}
	return &fi, nil
}

func MoveFileItem(ctx context.Context, db *pgxpool.Pool, courseID, itemID uuid.UUID, newFolderID *uuid.UUID) (*FileItem, error) {
	tag, err := db.Exec(ctx, `
		UPDATE course.file_items SET folder_id = $1, updated_at = $2
		WHERE id = $3 AND course_id = $4
	`, newFolderID, time.Now(), itemID, courseID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, nil
	}
	return GetFileItem(ctx, db, courseID, itemID)
}

func RenameFileItem(ctx context.Context, db *pgxpool.Pool, courseID, itemID uuid.UUID, displayName string) (*FileItem, error) {
	tag, err := db.Exec(ctx, `
		UPDATE course.file_items SET display_name = $1, updated_at = $2
		WHERE id = $3 AND course_id = $4
	`, displayName, time.Now(), itemID, courseID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, nil
	}
	return GetFileItem(ctx, db, courseID, itemID)
}

func DeleteFileItem(ctx context.Context, db *pgxpool.Pool, courseID, itemID uuid.UUID) (string, bool, error) {
	var storageKey string
	err := db.QueryRow(ctx, `
		DELETE FROM course.file_items WHERE id = $1 AND course_id = $2 RETURNING storage_key
	`, itemID, courseID).Scan(&storageKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return storageKey, true, nil
}

func scanFileItem(row pgx.Row) (FileItem, error) {
	var fi FileItem
	var folderID *uuid.UUID
	var uploadedBy *uuid.UUID
	var canvasFileID *int64
	err := row.Scan(
		&fi.ID, &fi.CourseID, &folderID, &fi.StorageKey, &fi.OriginalFilename, &fi.DisplayName,
		&fi.MimeType, &fi.ByteSize, &uploadedBy, &canvasFileID, &fi.CreatedAt, &fi.UpdatedAt,
	)
	if err != nil {
		return FileItem{}, err
	}
	fi.FolderID = folderID
	fi.UploadedBy = uploadedBy
	fi.CanvasFileID = canvasFileID
	return fi, nil
}
