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
