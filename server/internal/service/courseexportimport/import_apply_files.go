package courseexportimport

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/models/courseexport"
)

// applyCourseFiles restores embedded course files + file-manager hierarchy.
// IDs are preserved so markdown / hero URLs keep resolving.
//
// Legacy export bundles without any file rows leave the target file set unchanged
// (so older backups do not wipe images). When the export includes file rows:
//   - erase: replace the whole file set
//   - overwrite: upsert export rows and prune extras
//   - mergeAdd: insert missing ids only
func applyCourseFiles(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID uuid.UUID,
	targetCourseCode string,
	mode courseexport.CourseImportMode,
	ex *Bundle,
	blobOpts BlobOptions,
) error {
	hasAny := len(ex.CourseFiles) > 0 || len(ex.FileFolders) > 0 || len(ex.FileItems) > 0
	if !hasAny {
		return nil
	}

	switch mode {
	case courseexport.CourseImportModeErase:
		if err := deleteAllCourseFiles(ctx, pool, courseID); err != nil {
			return err
		}
		if err := importFileFolders(ctx, pool, courseID, ex.FileFolders, false); err != nil {
			return err
		}
		if err := importCourseFileRows(ctx, pool, courseID, targetCourseCode, ex.CourseFiles, blobOpts, false); err != nil {
			return err
		}
		return importFileItems(ctx, pool, courseID, targetCourseCode, ex.FileItems, blobOpts, false)

	case courseexport.CourseImportModeOverwrite:
		if err := importFileFolders(ctx, pool, courseID, ex.FileFolders, false); err != nil {
			return err
		}
		if err := importCourseFileRows(ctx, pool, courseID, targetCourseCode, ex.CourseFiles, blobOpts, false); err != nil {
			return err
		}
		if err := importFileItems(ctx, pool, courseID, targetCourseCode, ex.FileItems, blobOpts, false); err != nil {
			return err
		}
		return pruneCourseFilesNotInExport(ctx, pool, courseID, ex)

	case courseexport.CourseImportModeMergeAdd:
		if err := importFileFolders(ctx, pool, courseID, ex.FileFolders, true); err != nil {
			return err
		}
		if err := importCourseFileRows(ctx, pool, courseID, targetCourseCode, ex.CourseFiles, blobOpts, true); err != nil {
			return err
		}
		return importFileItems(ctx, pool, courseID, targetCourseCode, ex.FileItems, blobOpts, true)
	}
	return nil
}

func deleteAllCourseFiles(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) error {
	// file_items first (no FK from folders), then folders (self-referential parent), then course_files.
	if _, err := pool.Exec(ctx, `DELETE FROM course.file_items WHERE course_id = $1`, courseID); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `DELETE FROM course.file_folders WHERE course_id = $1`, courseID); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `DELETE FROM course.course_files WHERE course_id = $1`, courseID); err != nil {
		return err
	}
	return nil
}

func pruneCourseFilesNotInExport(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, ex *Bundle) error {
	keepCF := make([]uuid.UUID, 0, len(ex.CourseFiles))
	for _, f := range ex.CourseFiles {
		keepCF = append(keepCF, f.ID)
	}
	if len(keepCF) == 0 {
		if _, err := pool.Exec(ctx, `DELETE FROM course.course_files WHERE course_id = $1`, courseID); err != nil {
			return err
		}
	} else {
		if _, err := pool.Exec(ctx, `
			DELETE FROM course.course_files
			WHERE course_id = $1 AND NOT (id = ANY($2::uuid[]))
		`, courseID, keepCF); err != nil {
			return err
		}
	}

	keepItems := make([]uuid.UUID, 0, len(ex.FileItems))
	for _, f := range ex.FileItems {
		keepItems = append(keepItems, f.ID)
	}
	if len(keepItems) == 0 {
		if _, err := pool.Exec(ctx, `DELETE FROM course.file_items WHERE course_id = $1`, courseID); err != nil {
			return err
		}
	} else {
		if _, err := pool.Exec(ctx, `
			DELETE FROM course.file_items
			WHERE course_id = $1 AND NOT (id = ANY($2::uuid[]))
		`, courseID, keepItems); err != nil {
			return err
		}
	}

	keepFolders := make([]uuid.UUID, 0, len(ex.FileFolders))
	for _, f := range ex.FileFolders {
		keepFolders = append(keepFolders, f.ID)
	}
	if len(keepFolders) == 0 {
		if _, err := pool.Exec(ctx, `DELETE FROM course.file_folders WHERE course_id = $1`, courseID); err != nil {
			return err
		}
	} else {
		// Delete deepest folders first by repeating until none left (children cascade via parent FK).
		// Safer: delete folders not in keep list; ON DELETE CASCADE on parent_id removes children,
		// but we want to keep descendants that are in the export — so only delete folders whose
		// id is not in keep (children not in keep also get deleted when parent is kept? No —
		// orphan children would remain. Delete all not-in-keep; CASCADE only fires when parent
		// is deleted. A kept parent with deleted child is fine.
		if _, err := pool.Exec(ctx, `
			DELETE FROM course.file_folders
			WHERE course_id = $1 AND NOT (id = ANY($2::uuid[]))
		`, courseID, keepFolders); err != nil {
			return err
		}
	}
	return nil
}

func importFileFolders(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID uuid.UUID,
	folders []courseexport.ExportedFileFolder,
	mergeSkipExisting bool,
) error {
	// Export orders by created_at so parents typically appear before children.
	// Re-order defensively: roots first, then anything whose parent is already inserted.
	remaining := make([]courseexport.ExportedFileFolder, 0, len(folders))
	remaining = append(remaining, folders...)
	inserted := map[uuid.UUID]struct{}{}
	for len(remaining) > 0 {
		progress := 0
		next := remaining[:0]
		for _, f := range remaining {
			if f.ID == uuid.Nil {
				return InvalidInput("Each file folder needs a valid id.")
			}
			if f.ParentID != nil {
				if _, ok := inserted[*f.ParentID]; !ok {
					// Parent may already exist in the DB (merge) or appear later in remaining.
					var exists bool
					_ = pool.QueryRow(ctx, `
						SELECT EXISTS(SELECT 1 FROM course.file_folders WHERE id = $1 AND course_id = $2)
					`, *f.ParentID, courseID).Scan(&exists)
					if !exists {
						// Keep for a later pass if parent is still in the export batch.
						next = append(next, f)
						continue
					}
				}
			}
			if mergeSkipExisting {
				var exists bool
				if err := pool.QueryRow(ctx, `
					SELECT EXISTS(SELECT 1 FROM course.file_folders WHERE id = $1 AND course_id = $2)
				`, f.ID, courseID).Scan(&exists); err != nil {
					return err
				}
				if exists {
					inserted[f.ID] = struct{}{}
					progress++
					continue
				}
			}
			name := strings.TrimSpace(f.Name)
			if name == "" {
				return InvalidInput(fmt.Sprintf("File folder %s needs a name.", f.ID))
			}
			if _, err := pool.Exec(ctx, `
				INSERT INTO course.file_folders (id, course_id, parent_id, name, created_at, updated_at)
				VALUES ($1, $2, $3, $4, NOW(), NOW())
				ON CONFLICT (id) DO UPDATE SET
					parent_id = EXCLUDED.parent_id,
					name = EXCLUDED.name,
					updated_at = NOW()
				WHERE course.file_folders.course_id = $2
			`, f.ID, courseID, f.ParentID, name); err != nil {
				return err
			}
			inserted[f.ID] = struct{}{}
			progress++
		}
		if progress == 0 {
			return InvalidInput("File folder parent references form a cycle or reference missing parents.")
		}
		remaining = next
	}
	return nil
}

func importCourseFileRows(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID uuid.UUID,
	courseCode string,
	files []courseexport.ExportedCourseFile,
	blobOpts BlobOptions,
	mergeSkipExisting bool,
) error {
	for _, f := range files {
		if f.ID == uuid.Nil {
			return InvalidInput("Each course file needs a valid id.")
		}
		if mergeSkipExisting {
			var exists bool
			if err := pool.QueryRow(ctx, `
				SELECT EXISTS(SELECT 1 FROM course.course_files WHERE id = $1 AND course_id = $2)
			`, f.ID, courseID).Scan(&exists); err != nil {
				return err
			}
			if exists {
				continue
			}
		}
		data, err := decodeFileContent(fmt.Sprintf("Course file %s", f.ID), f.ContentBase64, f.ByteSize)
		if err != nil {
			return err
		}
		filename := strings.TrimSpace(f.OriginalFilename)
		if filename == "" {
			filename = "file"
		}
		mime := strings.TrimSpace(f.MimeType)
		if mime == "" {
			mime = "application/octet-stream"
		}
		storageKey := newImportStorageKey("files", courseCode, filename)
		byteSize := f.ByteSize
		if data != nil {
			byteSize = int64(len(data))
			if err := blobOpts.writeCourseFileBlob(ctx, courseCode, storageKey, mime, data); err != nil {
				return fmt.Errorf("store course file %s: %w", f.ID, err)
			}
		} else {
			// No body: still register metadata with a fresh key so the id exists for links.
			if byteSize < 0 {
				byteSize = 0
			}
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO course.course_files (id, course_id, storage_key, original_filename, mime_type, byte_size, uploaded_by)
			VALUES ($1, $2, $3, $4, $5, $6, NULL)
			ON CONFLICT (id) DO UPDATE SET
				storage_key = EXCLUDED.storage_key,
				original_filename = EXCLUDED.original_filename,
				mime_type = EXCLUDED.mime_type,
				byte_size = EXCLUDED.byte_size
			WHERE course.course_files.course_id = $2
		`, f.ID, courseID, storageKey, filename, mime, byteSize); err != nil {
			return err
		}
	}
	return nil
}

func importFileItems(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID uuid.UUID,
	courseCode string,
	items []courseexport.ExportedFileItem,
	blobOpts BlobOptions,
	mergeSkipExisting bool,
) error {
	for _, it := range items {
		if it.ID == uuid.Nil {
			return InvalidInput("Each file manager item needs a valid id.")
		}
		if mergeSkipExisting {
			var exists bool
			if err := pool.QueryRow(ctx, `
				SELECT EXISTS(SELECT 1 FROM course.file_items WHERE id = $1 AND course_id = $2)
			`, it.ID, courseID).Scan(&exists); err != nil {
				return err
			}
			if exists {
				continue
			}
		}
		data, err := decodeFileContent(fmt.Sprintf("File item %s", it.ID), it.ContentBase64, it.ByteSize)
		if err != nil {
			return err
		}
		filename := strings.TrimSpace(it.OriginalFilename)
		if filename == "" {
			filename = "file"
		}
		display := strings.TrimSpace(it.DisplayName)
		if display == "" {
			display = filename
		}
		mime := strings.TrimSpace(it.MimeType)
		if mime == "" {
			mime = "application/octet-stream"
		}
		ext := filepath.Ext(filename)
		storageKey := fmt.Sprintf("managed-files/%s/%s%s", courseCode, uuid.New().String(), ext)
		byteSize := it.ByteSize
		if data != nil {
			byteSize = int64(len(data))
			if err := blobOpts.writeFileItemBlob(ctx, courseCode, storageKey, mime, data); err != nil {
				return fmt.Errorf("store file item %s: %w", it.ID, err)
			}
		} else if byteSize < 0 {
			byteSize = 0
		}
		// folder_id may be null (root). If set, ensure it exists in this course (or leave null).
		folderID := it.FolderID
		if folderID != nil {
			var exists bool
			_ = pool.QueryRow(ctx, `
				SELECT EXISTS(SELECT 1 FROM course.file_folders WHERE id = $1 AND course_id = $2)
			`, *folderID, courseID).Scan(&exists)
			if !exists {
				folderID = nil
			}
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO course.file_items (
				id, course_id, folder_id, storage_key, original_filename, display_name,
				mime_type, byte_size, uploaded_by, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULL, NOW(), NOW())
			ON CONFLICT (id) DO UPDATE SET
				folder_id = EXCLUDED.folder_id,
				storage_key = EXCLUDED.storage_key,
				original_filename = EXCLUDED.original_filename,
				display_name = EXCLUDED.display_name,
				mime_type = EXCLUDED.mime_type,
				byte_size = EXCLUDED.byte_size,
				updated_at = NOW()
			WHERE course.file_items.course_id = $2
		`, it.ID, courseID, folderID, storageKey, filename, display, mime, byteSize); err != nil {
			return err
		}
	}
	return nil
}

// rewriteCourseFileURLsInBundle rewrites absolute course-file API paths from the
// source course code to the target so embedded images work after cross-course import.
func rewriteCourseFileURLsInBundle(ex *Bundle, sourceCode, targetCode string) {
	sourceCode = strings.TrimSpace(sourceCode)
	targetCode = strings.TrimSpace(targetCode)
	if sourceCode == "" || targetCode == "" || sourceCode == targetCode {
		return
	}
	fromPrefix := "/api/v1/courses/" + sourceCode + "/"
	toPrefix := "/api/v1/courses/" + targetCode + "/"
	rewrite := func(s string) string {
		return strings.ReplaceAll(s, fromPrefix, toPrefix)
	}

	if ex.Course.HeroImageURL != nil {
		u := rewrite(*ex.Course.HeroImageURL)
		ex.Course.HeroImageURL = &u
	}
	for i := range ex.Syllabus {
		ex.Syllabus[i].Markdown = rewrite(ex.Syllabus[i].Markdown)
	}
	for id, body := range ex.ContentPages {
		body.Markdown = rewrite(body.Markdown)
		ex.ContentPages[id] = body
	}
	for id, body := range ex.Assignments {
		body.Markdown = rewrite(body.Markdown)
		ex.Assignments[id] = body
	}
	for id, body := range ex.Quizzes {
		body.Markdown = rewrite(body.Markdown)
		ex.Quizzes[id] = body
	}
}
