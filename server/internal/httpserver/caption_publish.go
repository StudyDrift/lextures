package httpserver

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	captionsrepo "github.com/lextures/lextures/server/internal/repos/captions"
	captionssvc "github.com/lextures/lextures/server/internal/service/captions"
)

const captionPublishBlockedMsg = "This course requires captions. Please add captions before publishing."

// ErrCaptionRequired is returned when publishing video content without ready captions.
var ErrCaptionRequired = errors.New(captionPublishBlockedMsg)

// validateStructureItemPublishCaptions enforces require_captions when publishing content with embedded video.
func validateStructureItemPublishCaptions(ctx context.Context, pool *pgxpool.Pool, courseID, itemID uuid.UUID) error {
	var require bool
	var kind string
	var markdown *string
	err := pool.QueryRow(ctx, `
		SELECT c.require_captions, si.kind, mcp.markdown
		FROM course.course_structure_items si
		INNER JOIN course.courses c ON c.id = si.course_id
		LEFT JOIN course.module_content_pages mcp ON mcp.structure_item_id = si.id
		WHERE si.id = $1 AND si.course_id = $2`,
		itemID, courseID,
	).Scan(&require, &kind, &markdown)
	if err != nil {
		return fmt.Errorf("caption publish check: %w", err)
	}
	if !require || kind != "content_page" || markdown == nil {
		return nil
	}
	ids := captionssvc.CourseFileIDPattern.FindAllStringSubmatch(*markdown, -1)
	if len(ids) == 0 {
		return nil
	}
	for _, m := range ids {
		if len(m) < 2 {
			continue
		}
		fileID, parseErr := uuid.Parse(m[1])
		if parseErr != nil {
			continue
		}
		var mime, storageKey string
		if qErr := pool.QueryRow(ctx, `
			SELECT mime_type, storage_key FROM course.course_files WHERE id = $1 AND course_id = $2`,
			fileID, courseID,
		).Scan(&mime, &storageKey); qErr != nil {
			continue
		}
		if mime == "" || mime[:5] != "video" {
			continue
		}
		objID, resErr := captionsrepo.ResolveObjectIDByStorageKey(ctx, pool, storageKey)
		if resErr != nil || objID == nil {
			return ErrCaptionRequired
		}
		ok, capErr := captionsrepo.ObjectHasReadyCaption(ctx, pool, *objID)
		if capErr != nil {
			return capErr
		}
		if !ok {
			return ErrCaptionRequired
		}
	}
	return nil
}
