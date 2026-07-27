package contenttools

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LinkSourceRow is course.content_tool_link_sources.
type LinkSourceRow struct {
	ID                uuid.UUID
	OrgID             *uuid.UUID
	URLHash           string
	URL               string
	FinalURL          *string
	ContentType       *string
	Title             *string
	Lang              *string
	ExtractedText     *string
	ExtractionVersion int
	ByteSize          *int
	ETag              *string
	LastModified      *string
	Status            string
	Error             *string
	FetchedAt         *time.Time
	ExpiresAt         *time.Time
	CreatedAt         time.Time
}

// LinkChunkRow is course.content_tool_link_chunks.
type LinkChunkRow struct {
	ID         uuid.UUID
	SourceID   uuid.UUID
	Ordinal    int
	Text       string
	TokenCount int
}

// ActivitySourceRow is course.content_tool_activity_sources joined with source.
type ActivitySourceRow struct {
	ID              uuid.UUID
	CourseID        uuid.UUID
	StructureItemID *uuid.UUID
	SourceID        *uuid.UUID
	Origin          string
	CourseFileID    *uuid.UUID
	Excluded        bool
	ExcludedBy      *uuid.UUID
	CreatedAt       time.Time
	// Joined source fields
	URL           string
	Title         *string
	Status        string
	Error         *string
	FetchedAt     *time.Time
	ByteSize      *int
	ExtractedText *string
}

// GetLinkSourceByHash returns a cached source for org+hash+version.
func GetLinkSourceByHash(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, urlHash string, version int) (*LinkSourceRow, error) {
	var r LinkSourceRow
	err := pool.QueryRow(ctx, `
SELECT id, org_id, url_hash, url, final_url, content_type, title, lang, extracted_text,
       extraction_version, byte_size, etag, last_modified, status, error, fetched_at, expires_at, created_at
FROM course.content_tool_link_sources
WHERE org_id = $1 AND url_hash = $2 AND extraction_version = $3
`, orgID, urlHash, version).Scan(
		&r.ID, &r.OrgID, &r.URLHash, &r.URL, &r.FinalURL, &r.ContentType, &r.Title, &r.Lang, &r.ExtractedText,
		&r.ExtractionVersion, &r.ByteSize, &r.ETag, &r.LastModified, &r.Status, &r.Error, &r.FetchedAt, &r.ExpiresAt, &r.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// UpsertLinkSource inserts or updates a link source row.
func UpsertLinkSource(ctx context.Context, pool *pgxpool.Pool, r LinkSourceRow) (*LinkSourceRow, error) {
	var out LinkSourceRow
	err := pool.QueryRow(ctx, `
INSERT INTO course.content_tool_link_sources (
  org_id, url_hash, url, final_url, content_type, title, lang, extracted_text,
  extraction_version, byte_size, etag, last_modified, status, error, fetched_at, expires_at
) VALUES (
  $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16
)
ON CONFLICT (org_id, url_hash, extraction_version) DO UPDATE SET
  url = EXCLUDED.url,
  final_url = EXCLUDED.final_url,
  content_type = EXCLUDED.content_type,
  title = EXCLUDED.title,
  lang = EXCLUDED.lang,
  extracted_text = EXCLUDED.extracted_text,
  byte_size = EXCLUDED.byte_size,
  etag = EXCLUDED.etag,
  last_modified = EXCLUDED.last_modified,
  status = EXCLUDED.status,
  error = EXCLUDED.error,
  fetched_at = EXCLUDED.fetched_at,
  expires_at = EXCLUDED.expires_at
RETURNING id, org_id, url_hash, url, final_url, content_type, title, lang, extracted_text,
          extraction_version, byte_size, etag, last_modified, status, error, fetched_at, expires_at, created_at
`, r.OrgID, r.URLHash, r.URL, r.FinalURL, r.ContentType, r.Title, r.Lang, r.ExtractedText,
		r.ExtractionVersion, r.ByteSize, r.ETag, r.LastModified, r.Status, r.Error, r.FetchedAt, r.ExpiresAt,
	).Scan(
		&out.ID, &out.OrgID, &out.URLHash, &out.URL, &out.FinalURL, &out.ContentType, &out.Title, &out.Lang, &out.ExtractedText,
		&out.ExtractionVersion, &out.ByteSize, &out.ETag, &out.LastModified, &out.Status, &out.Error, &out.FetchedAt, &out.ExpiresAt, &out.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ReplaceLinkChunks replaces all chunks for a source.
func ReplaceLinkChunks(ctx context.Context, pool *pgxpool.Pool, sourceID uuid.UUID, chunks []LinkChunkRow) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DELETE FROM course.content_tool_link_chunks WHERE source_id = $1`, sourceID); err != nil {
		return err
	}
	for _, c := range chunks {
		if _, err := tx.Exec(ctx, `
INSERT INTO course.content_tool_link_chunks (source_id, ordinal, text, token_count)
VALUES ($1,$2,$3,$4)
`, sourceID, c.Ordinal, c.Text, c.TokenCount); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// ListLinkChunks returns chunks for a source ordered by ordinal.
func ListLinkChunks(ctx context.Context, pool *pgxpool.Pool, sourceID uuid.UUID) ([]LinkChunkRow, error) {
	rows, err := pool.Query(ctx, `
SELECT id, source_id, ordinal, text, token_count
FROM course.content_tool_link_chunks
WHERE source_id = $1
ORDER BY ordinal
`, sourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LinkChunkRow
	for rows.Next() {
		var c LinkChunkRow
		if err := rows.Scan(&c.ID, &c.SourceID, &c.Ordinal, &c.Text, &c.TokenCount); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// UpsertActivitySource links a source to an activity.
func UpsertActivitySource(ctx context.Context, pool *pgxpool.Pool, courseID, itemID, sourceID uuid.UUID, origin string) error {
	_, err := pool.Exec(ctx, `
INSERT INTO course.content_tool_activity_sources (course_id, structure_item_id, source_id, origin)
VALUES ($1,$2,$3,$4)
ON CONFLICT (structure_item_id, source_id) WHERE source_id IS NOT NULL AND course_file_id IS NULL
DO NOTHING
`, courseID, itemID, sourceID, origin)
	return err
}

// ListActivitySources returns instructor-visible corpus for an item.
func ListActivitySources(ctx context.Context, pool *pgxpool.Pool, courseID, itemID uuid.UUID) ([]ActivitySourceRow, error) {
	rows, err := pool.Query(ctx, `
SELECT a.id, a.course_id, a.structure_item_id, a.source_id, a.origin, a.course_file_id,
       a.excluded, a.excluded_by, a.created_at,
       COALESCE(s.url, ''), s.title, COALESCE(s.status, 'pending'), s.error, s.fetched_at, s.byte_size, s.extracted_text
FROM course.content_tool_activity_sources a
LEFT JOIN course.content_tool_link_sources s ON s.id = a.source_id
WHERE a.course_id = $1 AND a.structure_item_id = $2
ORDER BY a.created_at ASC
`, courseID, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ActivitySourceRow
	for rows.Next() {
		var r ActivitySourceRow
		if err := rows.Scan(
			&r.ID, &r.CourseID, &r.StructureItemID, &r.SourceID, &r.Origin, &r.CourseFileID,
			&r.Excluded, &r.ExcludedBy, &r.CreatedAt,
			&r.URL, &r.Title, &r.Status, &r.Error, &r.FetchedAt, &r.ByteSize, &r.ExtractedText,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// SetActivitySourceExcluded toggles instructor exclude (FR-17, AC-11).
func SetActivitySourceExcluded(ctx context.Context, pool *pgxpool.Pool, courseID, activitySourceID uuid.UUID, excluded bool, actor uuid.UUID) error {
	var excludedBy *uuid.UUID
	if excluded {
		excludedBy = &actor
	}
	tag, err := pool.Exec(ctx, `
UPDATE course.content_tool_activity_sources
SET excluded = $3, excluded_by = $4
WHERE id = $1 AND course_id = $2
`, activitySourceID, courseID, excluded, excludedBy)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// GetActivitySource returns one activity source by id.
func GetActivitySource(ctx context.Context, pool *pgxpool.Pool, courseID, activitySourceID uuid.UUID) (*ActivitySourceRow, error) {
	var r ActivitySourceRow
	err := pool.QueryRow(ctx, `
SELECT a.id, a.course_id, a.structure_item_id, a.source_id, a.origin, a.course_file_id,
       a.excluded, a.excluded_by, a.created_at,
       COALESCE(s.url, ''), s.title, COALESCE(s.status, 'pending'), s.error, s.fetched_at, s.byte_size, s.extracted_text
FROM course.content_tool_activity_sources a
LEFT JOIN course.content_tool_link_sources s ON s.id = a.source_id
WHERE a.id = $1 AND a.course_id = $2
`, activitySourceID, courseID).Scan(
		&r.ID, &r.CourseID, &r.StructureItemID, &r.SourceID, &r.Origin, &r.CourseFileID,
		&r.Excluded, &r.ExcludedBy, &r.CreatedAt,
		&r.URL, &r.Title, &r.Status, &r.Error, &r.FetchedAt, &r.ByteSize, &r.ExtractedText,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// GetLinkSource returns a source by id.
func GetLinkSource(ctx context.Context, pool *pgxpool.Pool, sourceID uuid.UUID) (*LinkSourceRow, error) {
	var r LinkSourceRow
	err := pool.QueryRow(ctx, `
SELECT id, org_id, url_hash, url, final_url, content_type, title, lang, extracted_text,
       extraction_version, byte_size, etag, last_modified, status, error, fetched_at, expires_at, created_at
FROM course.content_tool_link_sources WHERE id = $1
`, sourceID).Scan(
		&r.ID, &r.OrgID, &r.URLHash, &r.URL, &r.FinalURL, &r.ContentType, &r.Title, &r.Lang, &r.ExtractedText,
		&r.ExtractionVersion, &r.ByteSize, &r.ETag, &r.LastModified, &r.Status, &r.Error, &r.FetchedAt, &r.ExpiresAt, &r.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// MarkLinkSourcePending sets status=pending for re-ingest.
func MarkLinkSourcePending(ctx context.Context, pool *pgxpool.Pool, sourceID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
UPDATE course.content_tool_link_sources
SET status = 'pending', error = NULL, expires_at = NULL
WHERE id = $1
`, sourceID)
	return err
}

// CountUserAICallsToday counts content-tool AI usage for a user since UTC midnight.
func CountUserAICallsToday(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, featurePrefix string) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
SELECT COUNT(*)::int
FROM analytics.ai_usage_log
WHERE user_id = $1
  AND feature LIKE $2
  AND created_at >= date_trunc('day', NOW() AT TIME ZONE 'UTC')
`, userID, featurePrefix+"%").Scan(&n)
	return n, err
}

// SumCourseAITokensMonth sums tokens for a course in the current UTC month.
func SumCourseAITokensMonth(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, featurePrefix string) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `
SELECT COALESCE(SUM(total_tokens), 0)::bigint
FROM analytics.ai_usage_log
WHERE course_id = $1
  AND feature LIKE $2
  AND created_at >= date_trunc('month', NOW() AT TIME ZONE 'UTC')
`, courseID, featurePrefix+"%").Scan(&n)
	return n, err
}

// CourseOrgID returns the org for a course.
func CourseOrgID(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (*uuid.UUID, error) {
	var orgID *uuid.UUID
	err := pool.QueryRow(ctx, `SELECT org_id FROM course.courses WHERE id = $1`, courseID).Scan(&orgID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return orgID, err
}

// CourseTitles loads course + item + module titles for pack framing.
func CourseTitles(ctx context.Context, pool *pgxpool.Pool, courseID, itemID uuid.UUID) (courseTitle, itemTitle, moduleTitle string, err error) {
	err = pool.QueryRow(ctx, `
SELECT COALESCE(c.title, ''), COALESCE(i.title, ''), COALESCE(m.title, '')
FROM course.courses c
LEFT JOIN course.course_structure_items i ON i.id = $2 AND i.course_id = c.id
LEFT JOIN course.course_structure_items m ON m.id = i.parent_id
WHERE c.id = $1
`, courseID, itemID).Scan(&courseTitle, &itemTitle, &moduleTitle)
	return
}
