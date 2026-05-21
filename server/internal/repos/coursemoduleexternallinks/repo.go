package coursemoduleexternallinks

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const MaxExternalURLLen = 2048

func ValidateExternalHTTPURL(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", fmt.Errorf("URL is required")
	}
	if len(s) > MaxExternalURLLen {
		return "", fmt.Errorf("URL must be at most %d characters", MaxExternalURLLen)
	}
	lower := strings.ToLower(s)
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		return "", fmt.Errorf("URL must start with http:// or https://")
	}
	if strings.HasPrefix(lower, "javascript:") || strings.HasPrefix(lower, "data:") {
		return "", fmt.Errorf("invalid URL")
	}
	return s, nil
}

func InsertEmptyForItem(ctx context.Context, tx pgx.Tx, structureItemID uuid.UUID, url string) error {
	if tx == nil {
		return errors.New("db tx is nil")
	}
	_, err := tx.Exec(ctx, `
INSERT INTO course.module_external_links (structure_item_id, url, provider, updated_at)
VALUES ($1, $2, 'url', NOW())
`, structureItemID, url)
	return err
}

func UpsertImportBody(ctx context.Context, pool *pgxpool.Pool, courseID, structureItemID uuid.UUID, url string) error {
	if pool == nil {
		return errors.New("db pool is nil")
	}
	_, err := pool.Exec(ctx, `
INSERT INTO course.module_external_links (structure_item_id, url, updated_at)
SELECT $2, $3, NOW()
FROM course.course_structure_items c
WHERE c.id = $2 AND c.course_id = $1 AND c.kind = 'external_link'
ON CONFLICT (structure_item_id) DO UPDATE SET
	url = EXCLUDED.url,
	updated_at = NOW()
`, courseID, structureItemID, url)
	return err
}

func URLsForStructureItems(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, structureItemIDs []uuid.UUID) (map[uuid.UUID]string, error) {
	if pool == nil {
		return nil, errors.New("db pool is nil")
	}
	if len(structureItemIDs) == 0 {
		return map[uuid.UUID]string{}, nil
	}
	rows, err := pool.Query(ctx, `
SELECT c.id, m.url
FROM course.course_structure_items c
INNER JOIN course.module_external_links m ON m.structure_item_id = c.id
WHERE c.course_id = $1 AND c.kind = 'external_link' AND c.id = ANY($2)
`, courseID, structureItemIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[uuid.UUID]string{}
	for rows.Next() {
		var id uuid.UUID
		var url string
		if err := rows.Scan(&id, &url); err != nil {
			return nil, err
		}
		out[id] = url
	}
	return out, rows.Err()
}

func GetForCourseItem(ctx context.Context, pool *pgxpool.Pool, courseID, itemID uuid.UUID) (
	title, url, provider string,
	externalID, iconURL, licenseSPDX, attributionText, oerProvider *string,
	updatedAt *time.Time,
	err error,
) {
	if pool == nil {
		return "", "", "", nil, nil, nil, nil, nil, nil, errors.New("db pool is nil")
	}
	var t time.Time
	var prov string
	err = pool.QueryRow(ctx, `
SELECT c.title, m.url, m.provider, m.external_id, m.icon_url, m.license_spdx, m.attribution_text, m.oer_provider, m.updated_at
FROM course.course_structure_items c
INNER JOIN course.module_external_links m ON m.structure_item_id = c.id
WHERE c.id = $1 AND c.course_id = $2 AND c.kind = 'external_link'
`, itemID, courseID).Scan(&title, &url, &prov, &externalID, &iconURL, &licenseSPDX, &attributionText, &oerProvider, &t)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", "", nil, nil, nil, nil, nil, nil, nil
	}
	if err != nil {
		return "", "", "", nil, nil, nil, nil, nil, nil, err
	}
	return title, url, prov, externalID, iconURL, licenseSPDX, attributionText, oerProvider, &t, nil
}

// InsertOERMetadata sets OER attribution fields on an existing external link row.
func InsertOERMetadata(
	ctx context.Context, pool *pgxpool.Pool,
	courseID, itemID uuid.UUID,
	provider, url string,
	externalID, licenseSPDX, attributionText *string,
) (*time.Time, error) {
	if pool == nil {
		return nil, errors.New("db pool is nil")
	}
	var updated time.Time
	err := pool.QueryRow(ctx, `
UPDATE course.module_external_links m
SET url = $3,
    provider = $4,
    external_id = $5,
    license_spdx = $6,
    attribution_text = $7,
    oer_provider = $4,
    updated_at = NOW()
FROM course.course_structure_items c
WHERE m.structure_item_id = c.id
  AND c.id = $1
  AND c.course_id = $2
  AND c.kind = 'external_link'
RETURNING m.updated_at
`, itemID, courseID, url, provider, externalID, licenseSPDX, attributionText).Scan(&updated)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

func UpdateLink(ctx context.Context, pool *pgxpool.Pool, courseID, itemID uuid.UUID, url, provider string, externalID, iconURL *string) (*time.Time, error) {
	if pool == nil {
		return nil, errors.New("db pool is nil")
	}
	var updated time.Time
	err := pool.QueryRow(ctx, `
UPDATE course.module_external_links m
SET url = $3, provider = $4, external_id = $5, icon_url = $6, updated_at = NOW()
FROM course.course_structure_items c
WHERE m.structure_item_id = c.id
  AND c.id = $1
  AND c.course_id = $2
  AND c.kind = 'external_link'
RETURNING m.updated_at
`, itemID, courseID, url, provider, externalID, iconURL).Scan(&updated)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &updated, nil
}
