package filemanager

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func ListContents(ctx context.Context, db *pgxpool.Pool, courseID uuid.UUID, folderID *uuid.UUID) (*FolderContents, error) {
	folders, err := listFolders(ctx, db, courseID, folderID)
	if err != nil {
		return nil, err
	}
	files, err := listFiles(ctx, db, courseID, folderID)
	if err != nil {
		return nil, err
	}
	return &FolderContents{FolderID: folderID, Folders: folders, Files: files}, nil
}

func listFolders(ctx context.Context, db *pgxpool.Pool, courseID uuid.UUID, parentID *uuid.UUID) ([]Folder, error) {
	var rows pgx.Rows
	var err error
	if parentID == nil {
		rows, err = db.Query(ctx, `
			SELECT id, course_id, parent_id, name, created_by, created_at, updated_at
			FROM course.file_folders
			WHERE course_id = $1 AND parent_id IS NULL
			ORDER BY name
		`, courseID)
	} else {
		rows, err = db.Query(ctx, `
			SELECT id, course_id, parent_id, name, created_by, created_at, updated_at
			FROM course.file_folders
			WHERE course_id = $1 AND parent_id = $2
			ORDER BY name
		`, courseID, *parentID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Folder
	for rows.Next() {
		f, scanErr := scanFolder(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, f)
	}
	if out == nil {
		out = []Folder{}
	}
	return out, rows.Err()
}

func listFiles(ctx context.Context, db *pgxpool.Pool, courseID uuid.UUID, folderID *uuid.UUID) ([]FileItem, error) {
	var rows pgx.Rows
	var err error
	if folderID == nil {
		rows, err = db.Query(ctx, `
			SELECT id, course_id, folder_id, storage_key, original_filename, display_name,
			       mime_type, byte_size, uploaded_by, canvas_file_id, created_at, updated_at
			FROM course.file_items
			WHERE course_id = $1 AND folder_id IS NULL
			ORDER BY display_name
		`, courseID)
	} else {
		rows, err = db.Query(ctx, `
			SELECT id, course_id, folder_id, storage_key, original_filename, display_name,
			       mime_type, byte_size, uploaded_by, canvas_file_id, created_at, updated_at
			FROM course.file_items
			WHERE course_id = $1 AND folder_id = $2
			ORDER BY display_name
		`, courseID, *folderID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []FileItem
	for rows.Next() {
		fi, scanErr := scanFileItem(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, fi)
	}
	if out == nil {
		out = []FileItem{}
	}
	return out, rows.Err()
}

func GetFolder(ctx context.Context, db *pgxpool.Pool, courseID, folderID uuid.UUID) (*Folder, error) {
	row := db.QueryRow(ctx, `
		SELECT id, course_id, parent_id, name, created_by, created_at, updated_at
		FROM course.file_folders
		WHERE id = $1 AND course_id = $2
	`, folderID, courseID)
	f, err := scanFolder(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func CreateFolder(ctx context.Context, db *pgxpool.Pool, courseID uuid.UUID, parentID *uuid.UUID, name string, createdBy *uuid.UUID) (*Folder, error) {
	id := uuid.New()
	_, err := db.Exec(ctx, `
		INSERT INTO course.file_folders (id, course_id, parent_id, name, created_by)
		VALUES ($1, $2, $3, $4, $5)
	`, id, courseID, parentID, name, createdBy)
	if err != nil {
		return nil, err
	}
	return GetFolder(ctx, db, courseID, id)
}

func RenameFolder(ctx context.Context, db *pgxpool.Pool, courseID, folderID uuid.UUID, name string) (*Folder, error) {
	tag, err := db.Exec(ctx, `
		UPDATE course.file_folders SET name = $1, updated_at = $2
		WHERE id = $3 AND course_id = $4
	`, name, time.Now(), folderID, courseID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, nil
	}
	return GetFolder(ctx, db, courseID, folderID)
}

func MoveFolder(ctx context.Context, db *pgxpool.Pool, courseID, folderID uuid.UUID, parentID *uuid.UUID) (*Folder, error) {
	if parentID != nil {
		if *parentID == folderID {
			return nil, errors.New("cannot move a folder into itself")
		}
		var parentExists bool
		err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM course.file_folders WHERE id = $1 AND course_id = $2)`, *parentID, courseID).Scan(&parentExists)
		if err != nil {
			return nil, err
		}
		if !parentExists {
			return nil, errors.New("target folder not found")
		}

		var isDescendant bool
		err = db.QueryRow(ctx, `
			WITH RECURSIVE tree AS (
				SELECT id FROM course.file_folders WHERE id = $1 AND course_id = $2
				UNION ALL
				SELECT f.id FROM course.file_folders f JOIN tree t ON f.parent_id = t.id
			)
			SELECT EXISTS(SELECT 1 FROM tree WHERE id = $3)
		`, folderID, courseID, *parentID).Scan(&isDescendant)
		if err != nil {
			return nil, err
		}
		if isDescendant {
			return nil, errors.New("cannot move a folder into one of its subfolders")
		}
	}

	tag, err := db.Exec(ctx, `
		UPDATE course.file_folders SET parent_id = $1, updated_at = $2
		WHERE id = $3 AND course_id = $4
	`, parentID, time.Now(), folderID, courseID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, nil
	}
	return GetFolder(ctx, db, courseID, folderID)
}


func DeleteFolder(ctx context.Context, db *pgxpool.Pool, courseID, folderID uuid.UUID) (bool, error) {
	tag, err := db.Exec(ctx, `
		DELETE FROM course.file_folders WHERE id = $1 AND course_id = $2
	`, folderID, courseID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// StorageKeysInFolder returns storage_key values for all file_items inside a folder (recursively).
// Used to clean up blob storage on folder deletion.
func StorageKeysInFolder(ctx context.Context, db *pgxpool.Pool, courseID, folderID uuid.UUID) ([]string, error) {
	rows, err := db.Query(ctx, `
		WITH RECURSIVE tree AS (
			SELECT id FROM course.file_folders WHERE id = $1 AND course_id = $2
			UNION ALL
			SELECT f.id FROM course.file_folders f JOIN tree t ON f.parent_id = t.id
		)
		SELECT fi.storage_key FROM course.file_items fi
		JOIN tree t ON fi.folder_id = t.id
	`, folderID, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var k string
		if e := rows.Scan(&k); e != nil {
			return nil, e
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func scanFolder(row pgx.Row) (Folder, error) {
	var f Folder
	var parentID *uuid.UUID
	var createdBy *uuid.UUID
	err := row.Scan(&f.ID, &f.CourseID, &parentID, &f.Name, &createdBy, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return Folder{}, err
	}
	f.ParentID = parentID
	f.CreatedBy = createdBy
	return f, nil
}
