package courseexportimport

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/models/courseexport"
	"github.com/lextures/lextures/server/internal/repos/coursefiles"
)

const (
	maxExportCourseFiles     = 5000
	maxExportFileFolders     = 5000
	maxExportFileItems       = 5000
	maxExportSingleFileBytes = 50 << 20  // 50 MiB
	maxExportTotalFileBytes  = 250 << 20 // 250 MiB
)

func (o BlobOptions) filesRoot() string {
	root := strings.TrimSpace(o.FilesRoot)
	if root == "" {
		return "data/course-files"
	}
	return filepath.Clean(root)
}

// readCourseFileBlob loads bytes for a course.course_files storage key.
// Mirrors httpserver.readCourseFileRowBytes (storage driver, then disk layouts).
func (o BlobOptions) readCourseFileBlob(ctx context.Context, courseCode, storageKey string) ([]byte, error) {
	if o.Storage != nil {
		rc, err := o.Storage.GetObject(ctx, storageKey)
		if err == nil {
			defer func() { _ = rc.Close() }()
			return io.ReadAll(rc)
		}
	}
	root := o.filesRoot()
	if b, err := os.ReadFile(coursefiles.BlobDiskPath(root, courseCode, storageKey)); err == nil {
		return b, nil
	}
	legacyPath := filepath.Join(root, courseCode, storageKey)
	b, err := os.ReadFile(legacyPath)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// readFileItemBlob loads bytes for a course.file_items storage key.
func (o BlobOptions) readFileItemBlob(ctx context.Context, courseCode, storageKey string) ([]byte, error) {
	if o.Storage != nil {
		rc, err := o.Storage.GetObject(ctx, storageKey)
		if err != nil {
			return nil, err
		}
		defer func() { _ = rc.Close() }()
		return io.ReadAll(rc)
	}
	root := o.filesRoot()
	p := filepath.Join(root, courseCode, storageKey)
	return os.ReadFile(p)
}

// writeCourseFileBlob stores an embedded course.course_files blob.
func (o BlobOptions) writeCourseFileBlob(ctx context.Context, courseCode, storageKey, mimeType string, data []byte) error {
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	if o.Storage != nil {
		return o.Storage.PutObject(ctx, storageKey, bytes.NewReader(data), int64(len(data)), mimeType)
	}
	// Match handlePostCourseFileMultipart local layout (BlobDiskPath).
	p := coursefiles.BlobDiskPath(o.filesRoot(), courseCode, storageKey)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

// writeFileItemBlob stores a course.file_items blob.
func (o BlobOptions) writeFileItemBlob(ctx context.Context, courseCode, storageKey, mimeType string, data []byte) error {
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	if o.Storage != nil {
		return o.Storage.PutObject(ctx, storageKey, bytes.NewReader(data), int64(len(data)), mimeType)
	}
	// Match handlePostCourseFileItem / canvas import local layout.
	p := filepath.Join(o.filesRoot(), courseCode, filepath.FromSlash(storageKey))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

func attachCourseFilesToExport(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID uuid.UUID,
	courseCode string,
	blobOpts BlobOptions,
	bundle *Bundle,
) error {
	var totalBytes int64

	// --- course.course_files (embedded content) ---
	cfRows, err := pool.Query(ctx, `
		SELECT id, storage_key, original_filename, mime_type, byte_size
		FROM course.course_files
		WHERE course_id = $1
		ORDER BY created_at ASC, id ASC
	`, courseID)
	if err != nil {
		return err
	}
	defer cfRows.Close()
	courseFiles := make([]courseexport.ExportedCourseFile, 0)
	for cfRows.Next() {
		var (
			id               uuid.UUID
			storageKey       string
			originalFilename string
			mimeType         string
			byteSize         int64
		)
		if err := cfRows.Scan(&id, &storageKey, &originalFilename, &mimeType, &byteSize); err != nil {
			return err
		}
		if len(courseFiles) >= maxExportCourseFiles {
			return fmt.Errorf("too many course files to export (max %d)", maxExportCourseFiles)
		}
		ex := courseexport.ExportedCourseFile{
			ID:               id,
			OriginalFilename: originalFilename,
			MimeType:         mimeType,
			ByteSize:         byteSize,
		}
		if data, rerr := blobOpts.readCourseFileBlob(ctx, courseCode, storageKey); rerr == nil {
			if int64(len(data)) > maxExportSingleFileBytes {
				return fmt.Errorf("course file %q exceeds export size limit (%d bytes)", originalFilename, maxExportSingleFileBytes)
			}
			totalBytes += int64(len(data))
			if totalBytes > maxExportTotalFileBytes {
				return fmt.Errorf("total course file payload exceeds export size limit (%d bytes)", maxExportTotalFileBytes)
			}
			ex.ContentBase64 = base64.StdEncoding.EncodeToString(data)
			ex.ByteSize = int64(len(data))
		}
		// Missing blobs: still export metadata so import can recreate the row id
		// (content URL will 404 until re-uploaded).
		courseFiles = append(courseFiles, ex)
	}
	if err := cfRows.Err(); err != nil {
		return err
	}
	if len(courseFiles) > 0 {
		bundle.CourseFiles = courseFiles
	}

	// --- file manager folders (parents before children via created_at) ---
	folderRows, err := pool.Query(ctx, `
		SELECT id, parent_id, name
		FROM course.file_folders
		WHERE course_id = $1
		ORDER BY created_at ASC, id ASC
	`, courseID)
	if err != nil {
		return err
	}
	defer folderRows.Close()
	folders := make([]courseexport.ExportedFileFolder, 0)
	for folderRows.Next() {
		var f courseexport.ExportedFileFolder
		if err := folderRows.Scan(&f.ID, &f.ParentID, &f.Name); err != nil {
			return err
		}
		if len(folders) >= maxExportFileFolders {
			return fmt.Errorf("too many file folders to export (max %d)", maxExportFileFolders)
		}
		folders = append(folders, f)
	}
	if err := folderRows.Err(); err != nil {
		return err
	}
	if len(folders) > 0 {
		bundle.FileFolders = folders
	}

	// --- file manager items ---
	itemRows, err := pool.Query(ctx, `
		SELECT id, folder_id, storage_key, original_filename, display_name, mime_type, byte_size
		FROM course.file_items
		WHERE course_id = $1
		ORDER BY created_at ASC, id ASC
	`, courseID)
	if err != nil {
		return err
	}
	defer itemRows.Close()
	items := make([]courseexport.ExportedFileItem, 0)
	for itemRows.Next() {
		var (
			id               uuid.UUID
			folderID         *uuid.UUID
			storageKey       string
			originalFilename string
			displayName      string
			mimeType         string
			byteSize         int64
		)
		if err := itemRows.Scan(&id, &folderID, &storageKey, &originalFilename, &displayName, &mimeType, &byteSize); err != nil {
			return err
		}
		if len(items) >= maxExportFileItems {
			return fmt.Errorf("too many file manager items to export (max %d)", maxExportFileItems)
		}
		ex := courseexport.ExportedFileItem{
			ID:               id,
			FolderID:         folderID,
			OriginalFilename: originalFilename,
			DisplayName:      displayName,
			MimeType:         mimeType,
			ByteSize:         byteSize,
		}
		if data, rerr := blobOpts.readFileItemBlob(ctx, courseCode, storageKey); rerr == nil {
			if int64(len(data)) > maxExportSingleFileBytes {
				return fmt.Errorf("file manager item %q exceeds export size limit (%d bytes)", displayName, maxExportSingleFileBytes)
			}
			totalBytes += int64(len(data))
			if totalBytes > maxExportTotalFileBytes {
				return fmt.Errorf("total course file payload exceeds export size limit (%d bytes)", maxExportTotalFileBytes)
			}
			ex.ContentBase64 = base64.StdEncoding.EncodeToString(data)
			ex.ByteSize = int64(len(data))
		}
		items = append(items, ex)
	}
	if err := itemRows.Err(); err != nil {
		return err
	}
	if len(items) > 0 {
		bundle.FileItems = items
	}
	return nil
}

// decodeFileContent decodes base64 content and enforces size caps.
func decodeFileContent(label, b64 string, declaredSize int64) ([]byte, error) {
	b64 = strings.TrimSpace(b64)
	if b64 == "" {
		return nil, nil
	}
	// Guard against pathological base64 length before allocating.
	// base64 expands ~4/3; allow a little overhead for padding.
	approx := (int64(len(b64)) * 3) / 4
	if approx > maxExportSingleFileBytes+1024 {
		return nil, InvalidInput(fmt.Sprintf("%s content is too large.", label))
	}
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		// Also accept raw URL-safe encodings that some clients produce.
		data, err = base64.URLEncoding.DecodeString(b64)
		if err != nil {
			return nil, InvalidInput(fmt.Sprintf("%s contentBase64 is not valid base64.", label))
		}
	}
	if int64(len(data)) > maxExportSingleFileBytes {
		return nil, InvalidInput(fmt.Sprintf("%s content exceeds size limit (%d bytes).", label, maxExportSingleFileBytes))
	}
	if declaredSize > 0 && declaredSize != int64(len(data)) {
		// Trust decoded length; mismatch is allowed (declared may be stale).
		_ = declaredSize
	}
	return data, nil
}

// newImportStorageKey builds a storage key unique to the target course.
func newImportStorageKey(prefix, courseCode, filename string) string {
	ext := filepath.Ext(filename)
	return fmt.Sprintf("%s/%s/%s%s", prefix, courseCode, uuid.New().String(), ext)
}
