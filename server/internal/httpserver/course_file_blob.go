package httpserver

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/lextures/lextures/server/internal/repos/filemanager"
)

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
