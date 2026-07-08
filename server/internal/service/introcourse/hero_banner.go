package introcourse

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/repos/coursefiles"
)

const (
	introHeroBannerFilename = "intro-course-banner.jpg"
	introHeroBannerMIME     = "image/jpeg"
	introHeroObjectPosition = "50% 50%"
)

//go:embed assets/intro-course-banner.jpg
var introHeroBannerJPEG []byte

// EnsureHeroBanner idempotently stores the canonical intro course banner and wires hero_image_url.
func EnsureHeroBanner(ctx context.Context, tx pgx.Tx, courseID uuid.UUID, cfg config.Config) error {
	if len(introHeroBannerJPEG) == 0 {
		return fmt.Errorf("intro course banner asset missing")
	}

	storageKey := fmt.Sprintf("files/%s/%s", CourseCode, introHeroBannerFilename)
	root := strings.TrimSpace(cfg.CourseFilesRoot)
	if root == "" {
		root = "data/course-files"
	}
	diskPath := coursefiles.BlobDiskPath(root, CourseCode, storageKey)
	if err := writeIntroHeroBannerFile(diskPath, introHeroBannerJPEG); err != nil {
		return fmt.Errorf("write intro course banner: %w", err)
	}

	byteSize := int64(len(introHeroBannerJPEG))
	fileID, err := upsertIntroHeroBannerFile(ctx, tx, courseID, storageKey, byteSize)
	if err != nil {
		return err
	}

	heroURL := fmt.Sprintf("/api/v1/courses/%s/course-files/%s/content", CourseCode, fileID.String())
	_, err = tx.Exec(ctx, `
UPDATE course.courses
SET
    hero_image_url = $2,
    hero_image_object_position = $3,
    updated_at = NOW()
WHERE id = $1
`, courseID, heroURL, introHeroObjectPosition)
	return err
}

func writeIntroHeroBannerFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func upsertIntroHeroBannerFile(
	ctx context.Context,
	tx pgx.Tx,
	courseID uuid.UUID,
	storageKey string,
	byteSize int64,
) (uuid.UUID, error) {
	var existing uuid.UUID
	err := tx.QueryRow(ctx, `
SELECT id FROM course.course_files WHERE storage_key = $1
`, storageKey).Scan(&existing)
	if err == nil {
		if _, err := tx.Exec(ctx, `
UPDATE course.course_files
SET
    original_filename = $2,
    mime_type = $3,
    byte_size = $4,
    uploaded_by = $5
WHERE id = $1
`, existing, introHeroBannerFilename, introHeroBannerMIME, byteSize, SystemUserID); err != nil {
			return uuid.UUID{}, err
		}
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.UUID{}, err
	}

	return coursefiles.CreateInTransaction(
		ctx,
		tx,
		courseID,
		SystemUserID,
		storageKey,
		introHeroBannerFilename,
		introHeroBannerMIME,
		byteSize,
	)
}