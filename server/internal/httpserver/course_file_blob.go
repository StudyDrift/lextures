package httpserver

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/lextures/lextures/server/internal/repos/coursefiles"
	"github.com/lextures/lextures/server/internal/repos/filemanager"
)

// readCourseFileRowBytes loads the raw blob for a course.course_files row.
// Canvas-imported submission attachments use storage keys with path segments and are
// read via the configured storage driver; legacy uploads may only exist on disk.
func (d Deps) readCourseFileRowBytes(ctx context.Context, courseCode string, row *coursefiles.Row) ([]byte, error) {
	if row == nil {
		return nil, fmt.Errorf("missing file row")
	}
	if d.Storage != nil {
		rc, err := d.Storage.GetObject(ctx, row.StorageKey)
		if err == nil {
			defer func() { _ = rc.Close() }()
			return io.ReadAll(rc)
		}
	}
	cfg := d.effectiveConfig()
	root := strings.TrimSpace(cfg.CourseFilesRoot)
	if root == "" {
		root = "data/course-files"
	}
	if b, err := os.ReadFile(coursefiles.BlobDiskPath(root, courseCode, row.StorageKey)); err == nil {
		return b, nil
	}
	legacyPath := filepath.Join(root, courseCode, row.StorageKey)
	b, err := os.ReadFile(legacyPath)
	if err != nil {
		return nil, fmt.Errorf("read blob: %w", err)
	}
	return b, nil
}

// readCourseFileItemBytes loads the raw blob for a course.file_items row.
func (d Deps) readCourseFileItemBytes(ctx context.Context, courseCode string, item *filemanager.FileItem) ([]byte, error) {
	cfg := d.effectiveConfig()
	if d.Storage != nil {
		rc, err := d.Storage.GetObject(ctx, item.StorageKey)
		if err != nil {
			return nil, err
		}
		defer func() { _ = rc.Close() }()
		return io.ReadAll(rc)
	}
	root := strings.TrimSpace(cfg.CourseFilesRoot)
	if root == "" {
		root = "data/course-files"
	}
	legacyPath := filepath.Join(root, courseCode, item.StorageKey)
	b, err := os.ReadFile(legacyPath)
	if err != nil {
		return nil, fmt.Errorf("read blob: %w", err)
	}
	return b, nil
}
