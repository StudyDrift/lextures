package marketplacecourses

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

	"github.com/lextures/lextures/server/internal/repos/coursefiles"
)

const (
	heroBannerMIME           = "image/jpeg"
	heroBannerObjectPosition = "50% 50%"
)

//go:embed assets/ai-essentials-banner.jpg
var aiEssentialsBannerJPEG []byte

//go:embed assets/introduction-to-python-banner.jpg
var introductionToPythonBannerJPEG []byte

//go:embed assets/personal-finance-banner.jpg
var personalFinanceBannerJPEG []byte

type heroBannerAsset struct {
	filename string
	jpeg     []byte
}

// heroBannerForSlug returns the embedded course banner for a catalog slug, if any.
func heroBannerForSlug(catalogSlug string) (heroBannerAsset, bool) {
	switch strings.TrimSpace(catalogSlug) {
	case "ai-essentials":
		return heroBannerAsset{
			filename: "ai-essentials-banner.jpg",
			jpeg:     aiEssentialsBannerJPEG,
		}, true
	case "introduction-to-python":
		return heroBannerAsset{
			filename: "introduction-to-python-banner.jpg",
			jpeg:     introductionToPythonBannerJPEG,
		}, true
	case "personal-finance":
		return heroBannerAsset{
			filename: "personal-finance-banner.jpg",
			jpeg:     personalFinanceBannerJPEG,
		}, true
	default:
		return heroBannerAsset{}, false
	}
}

// EnsureHeroBanner idempotently stores the course banner (when one exists) and wires hero_image_url.
func EnsureHeroBanner(ctx context.Context, tx pgx.Tx, courseID uuid.UUID, courseCode, catalogSlug, courseFilesRoot string) error {
	asset, ok := heroBannerForSlug(catalogSlug)
	if !ok {
		return nil
	}
	if len(asset.jpeg) == 0 {
		return fmt.Errorf("marketplace course %s banner asset missing", catalogSlug)
	}

	storageKey := fmt.Sprintf("files/%s/%s", courseCode, asset.filename)
	root := strings.TrimSpace(courseFilesRoot)
	if root == "" {
		root = "data/course-files"
	}
	// Write both layouts: BlobDiskPath (basename under course dir) and the
	// LocalDriver key path (preserves "files/<code>/…" segments) so reads work
	// whether Storage.GetObject or the disk fallback is used.
	diskPath := coursefiles.BlobDiskPath(root, courseCode, storageKey)
	if err := writeHeroBannerFile(diskPath, asset.jpeg); err != nil {
		return fmt.Errorf("write marketplace course banner: %w", err)
	}
	localDriverPath := filepath.Join(root, filepath.FromSlash(storageKey))
	if localDriverPath != diskPath {
		if err := writeHeroBannerFile(localDriverPath, asset.jpeg); err != nil {
			return fmt.Errorf("write marketplace course banner (storage key path): %w", err)
		}
	}

	byteSize := int64(len(asset.jpeg))
	fileID, err := upsertHeroBannerFile(ctx, tx, courseID, storageKey, asset.filename, byteSize)
	if err != nil {
		return err
	}

	heroURL := fmt.Sprintf("/api/v1/courses/%s/course-files/%s/content", courseCode, fileID.String())
	_, err = tx.Exec(ctx, `
UPDATE course.courses
SET
    hero_image_url = $2,
    hero_image_object_position = $3,
    updated_at = NOW()
WHERE id = $1
`, courseID, heroURL, heroBannerObjectPosition)
	return err
}

func writeHeroBannerFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func upsertHeroBannerFile(
	ctx context.Context,
	tx pgx.Tx,
	courseID uuid.UUID,
	storageKey, filename string,
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
`, existing, filename, heroBannerMIME, byteSize, SystemPublisherID); err != nil {
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
		SystemPublisherID,
		storageKey,
		filename,
		heroBannerMIME,
		byteSize,
	)
}
