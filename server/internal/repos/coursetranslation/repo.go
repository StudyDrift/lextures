// Package coursetranslation persists course content translations, TM, and glossaries (plan 11.5).
package coursetranslation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	tmsvc "github.com/lextures/lextures/server/internal/service/translationmemory"
)

// ItemType identifies translatable course structure items.
type ItemType string

const (
	TypeContentPage ItemType = "content_page"
	TypeAssignment  ItemType = "assignment"
)

// Translation row for a locale variant.
type Translation struct {
	ID                      uuid.UUID
	SourceItemID            uuid.UUID
	SourceItemType          ItemType
	SourceLocale            string
	TargetLocale            string
	TranslatedTitle         *string
	TranslatedBody          *string
	IsDraft                 bool
	MachineTranslationDraft bool
	ReviewedBy              *uuid.UUID
	PublishedAt             *time.Time
	Version                 int64
	UpdatedAt               time.Time
}

// TMMatch is a translation memory suggestion.
type TMMatch struct {
	TranslatedText string  `json:"translatedText"`
	Similarity     float64 `json:"similarity"`
	Exact          bool    `json:"exact"`
}

// GlossaryRow is a course glossary entry.
type GlossaryRow struct {
	ID          uuid.UUID `json:"id"`
	SourceTerm  string    `json:"sourceTerm"`
	TargetTerm  string    `json:"targetTerm"`
	SourceLocale string   `json:"sourceLocale"`
	TargetLocale string   `json:"targetLocale"`
}

// TranslatableItem is a course item that can be translated.
type TranslatableItem struct {
	ItemID   uuid.UUID `json:"itemId"`
	ItemType ItemType  `json:"itemType"`
	Title    string    `json:"title"`
	Body     string    `json:"body"`
	HasPublished bool  `json:"hasPublished"`
	HasDraft     bool  `json:"hasDraft"`
}

// Coverage summarizes translation progress for a target locale.
type Coverage struct {
	TargetLocale   string             `json:"targetLocale"`
	TotalItems     int                `json:"totalItems"`
	TranslatedItems int               `json:"translatedItems"`
	Percent        float64            `json:"percent"`
	Untranslated   []TranslatableItem `json:"untranslated,omitempty"`
}

// UpsertTranslationInput is the payload for saving a translation variant.
type UpsertTranslationInput struct {
	SourceItemID            uuid.UUID
	SourceItemType          ItemType
	SourceLocale            string
	TargetLocale            string
	TranslatedTitle         *string
	TranslatedBody          *string
	IsDraft                 bool
	MachineTranslationDraft bool
	ExpectedVersion         *int64
}

// ListTranslatableItems returns content pages and assignments in a course.
func ListTranslatableItems(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]TranslatableItem, error) {
	rows, err := pool.Query(ctx, `
SELECT csi.id,
       CASE
           WHEN csi.kind = 'content_page' THEN 'content_page'
           WHEN csi.kind = 'assignment' THEN 'assignment'
       END AS item_type,
       COALESCE(csi.title, '') AS title,
       COALESCE(mcp.markdown, ma.markdown, '') AS body
FROM course.course_structure_items csi
LEFT JOIN course.module_content_pages mcp ON mcp.structure_item_id = csi.id
LEFT JOIN course.module_assignments ma ON ma.structure_item_id = csi.id
WHERE csi.course_id = $1
  AND csi.kind IN ('content_page', 'assignment')
  AND NOT csi.archived
ORDER BY csi.sort_order
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TranslatableItem
	for rows.Next() {
		var it TranslatableItem
		var kind string
		if err := rows.Scan(&it.ItemID, &kind, &it.Title, &it.Body); err != nil {
			return nil, err
		}
		it.ItemType = ItemType(kind)
		out = append(out, it)
	}
	return out, rows.Err()
}

// GetCoverage returns translation coverage for a course locale.
func GetCoverage(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, targetLocale string) (Coverage, error) {
	items, err := ListTranslatableItems(ctx, pool, courseID)
	if err != nil {
		return Coverage{}, err
	}
	cov := Coverage{TargetLocale: targetLocale, TotalItems: len(items)}
	if len(items) == 0 {
		return cov, nil
	}
	ids := make([]uuid.UUID, len(items))
	types := make([]string, len(items))
	for i, it := range items {
		ids[i] = it.ItemID
		types[i] = string(it.ItemType)
	}
	rows, err := pool.Query(ctx, `
SELECT ct.source_item_id, ct.is_draft, ct.published_at IS NOT NULL AS published
FROM i18n.content_translations ct
WHERE ct.target_locale = $1
  AND ct.source_item_id = ANY($2::uuid[])
`, targetLocale, ids)
	if err != nil {
		return Coverage{}, err
	}
	defer rows.Close()
	pub := make(map[uuid.UUID]bool)
	draft := make(map[uuid.UUID]bool)
	for rows.Next() {
		var id uuid.UUID
		var isDraft, published bool
		if err := rows.Scan(&id, &isDraft, &published); err != nil {
			return Coverage{}, err
		}
		if published && !isDraft {
			pub[id] = true
		}
		if isDraft {
			draft[id] = true
		}
	}
	if err := rows.Err(); err != nil {
		return Coverage{}, err
	}
	for i := range items {
		if pub[items[i].ItemID] {
			cov.TranslatedItems++
			items[i].HasPublished = true
		}
		items[i].HasDraft = draft[items[i].ItemID]
	}
	if cov.TotalItems > 0 {
		cov.Percent = float64(cov.TranslatedItems) / float64(cov.TotalItems) * 100
	}
	for _, it := range items {
		if !pub[it.ItemID] {
			cov.Untranslated = append(cov.Untranslated, it)
		}
	}
	return cov, nil
}

// GetTranslation loads a translation row for editors (includes drafts).
func GetTranslation(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID, itemType ItemType, targetLocale string) (*Translation, error) {
	var t Translation
	err := pool.QueryRow(ctx, `
SELECT id, source_item_id, source_item_type, source_locale, target_locale,
       translated_title, translated_body, is_draft, machine_translation_draft,
       reviewed_by, published_at, version, updated_at
FROM i18n.content_translations
WHERE source_item_id = $1 AND source_item_type = $2 AND target_locale = $3
`, itemID, string(itemType), targetLocale).Scan(
		&t.ID, &t.SourceItemID, &t.SourceItemType, &t.SourceLocale, &t.TargetLocale,
		&t.TranslatedTitle, &t.TranslatedBody, &t.IsDraft, &t.MachineTranslationDraft,
		&t.ReviewedBy, &t.PublishedAt, &t.Version, &t.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// GetPublishedForStudent returns a published translation for student viewing.
func GetPublishedForStudent(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID, itemType ItemType, targetLocale string) (*Translation, error) {
	var t Translation
	err := pool.QueryRow(ctx, `
SELECT id, source_item_id, source_item_type, source_locale, target_locale,
       translated_title, translated_body, is_draft, machine_translation_draft,
       reviewed_by, published_at, version, updated_at
FROM i18n.content_translations
WHERE source_item_id = $1 AND source_item_type = $2 AND target_locale = $3
  AND published_at IS NOT NULL AND NOT is_draft
`, itemID, string(itemType), targetLocale).Scan(
		&t.ID, &t.SourceItemID, &t.SourceItemType, &t.SourceLocale, &t.TargetLocale,
		&t.TranslatedTitle, &t.TranslatedBody, &t.IsDraft, &t.MachineTranslationDraft,
		&t.ReviewedBy, &t.PublishedAt, &t.Version, &t.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// UpsertTranslation saves or updates a translation with optimistic locking.
func UpsertTranslation(ctx context.Context, pool *pgxpool.Pool, in UpsertTranslationInput) (*Translation, error) {
	if in.ExpectedVersion != nil {
		var cur int64
		err := pool.QueryRow(ctx, `
SELECT version FROM i18n.content_translations
WHERE source_item_id = $1 AND source_item_type = $2 AND target_locale = $3
`, in.SourceItemID, string(in.SourceItemType), in.TargetLocale).Scan(&cur)
		if errors.Is(err, pgx.ErrNoRows) {
			if *in.ExpectedVersion != 0 {
				return nil, fmt.Errorf("version conflict")
			}
		} else if err != nil {
			return nil, err
		} else if cur != *in.ExpectedVersion {
			return nil, fmt.Errorf("version conflict")
		}
	}
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO i18n.content_translations (
  source_item_id, source_item_type, source_locale, target_locale,
  translated_title, translated_body, is_draft, machine_translation_draft, version
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 1)
ON CONFLICT (source_item_id, source_item_type, target_locale) DO UPDATE SET
  translated_title = EXCLUDED.translated_title,
  translated_body = EXCLUDED.translated_body,
  is_draft = EXCLUDED.is_draft,
  machine_translation_draft = EXCLUDED.machine_translation_draft,
  version = i18n.content_translations.version + 1,
  updated_at = NOW()
RETURNING id
`, in.SourceItemID, string(in.SourceItemType), in.SourceLocale, in.TargetLocale,
		in.TranslatedTitle, in.TranslatedBody, in.IsDraft, in.MachineTranslationDraft).Scan(&id)
	if err != nil {
		if strings.Contains(err.Error(), "version conflict") {
			return nil, err
		}
		return nil, err
	}
	if in.TranslatedBody != nil && strings.TrimSpace(*in.TranslatedBody) != "" {
		srcTitle, srcBody, srcErr := GetSourceContent(ctx, pool, in.SourceItemID, in.SourceItemType)
		if srcErr == nil {
			segment := strings.TrimSpace(srcBody)
			if segment == "" {
				segment = strings.TrimSpace(srcTitle)
			}
			if segment != "" {
				_ = UpsertTM(ctx, pool, in.SourceLocale, in.TargetLocale, segment, strings.TrimSpace(*in.TranslatedBody))
			}
		}
	}
	_ = id
	return GetTranslation(ctx, pool, in.SourceItemID, in.SourceItemType, in.TargetLocale)
}

// PublishTranslation marks a translation as reviewed and published.
func PublishTranslation(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID, itemType ItemType, targetLocale string, reviewer uuid.UUID) (*Translation, error) {
	tag, err := pool.Exec(ctx, `
UPDATE i18n.content_translations
SET is_draft = FALSE,
    machine_translation_draft = FALSE,
    reviewed_by = $4,
    published_at = NOW(),
    updated_at = NOW(),
    version = version + 1
WHERE source_item_id = $1 AND source_item_type = $2 AND target_locale = $3
`, itemID, string(itemType), targetLocale, reviewer)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, pgx.ErrNoRows
	}
	return GetTranslation(ctx, pool, itemID, itemType, targetLocale)
}

// UpsertTM stores a segment in translation memory.
func UpsertTM(ctx context.Context, pool *pgxpool.Pool, sourceLocale, targetLocale, sourceText, translatedText string) error {
	if !tmsvc.IsMostlyText(sourceText) {
		return nil
	}
	hash := tmsvc.SourceHash(sourceText)
	_, err := pool.Exec(ctx, `
INSERT INTO i18n.translation_memory (source_locale, target_locale, source_text, source_hash, translated_text)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (source_locale, target_locale, source_hash)
DO UPDATE SET translated_text = EXCLUDED.translated_text, source_text = EXCLUDED.source_text
`, sourceLocale, targetLocale, sourceText, hash, translatedText)
	return err
}

// QueryTM returns TM suggestions ordered by similarity (pg_trgm + exact hash).
func QueryTM(ctx context.Context, pool *pgxpool.Pool, sourceLocale, targetLocale, text string, limit int) ([]TMMatch, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 5
	}
	hash := tmsvc.SourceHash(text)
	var exactTranslated string
	if err := pool.QueryRow(ctx, `
SELECT translated_text FROM i18n.translation_memory
WHERE source_locale = $1 AND target_locale = $2 AND source_hash = $3
`, sourceLocale, targetLocale, hash).Scan(&exactTranslated); err == nil {
		return []TMMatch{{TranslatedText: exactTranslated, Similarity: 1, Exact: true}}, nil
	}

	rows, err := pool.Query(ctx, `
SELECT translated_text,
       similarity(source_text, $3) AS sim
FROM i18n.translation_memory
WHERE source_locale = $1 AND target_locale = $2
  AND source_text % $3
ORDER BY sim DESC
LIMIT $4
`, sourceLocale, targetLocale, text, limit)
	if err != nil {
		// pg_trgm unavailable in short tests — fall back to hash-only
		var translated string
		err2 := pool.QueryRow(ctx, `
SELECT translated_text FROM i18n.translation_memory
WHERE source_locale = $1 AND target_locale = $2 AND source_hash = $3
`, sourceLocale, targetLocale, hash).Scan(&translated)
		if errors.Is(err2, pgx.ErrNoRows) {
			return nil, nil
		}
		if err2 != nil {
			return nil, err2
		}
		return []TMMatch{{TranslatedText: translated, Similarity: 1, Exact: true}}, nil
	}
	defer rows.Close()
	var out []TMMatch
	for rows.Next() {
		var m TMMatch
		if err := rows.Scan(&m.TranslatedText, &m.Similarity); err != nil {
			return nil, err
		}
		m.Exact = m.Similarity >= 0.99
		out = append(out, m)
	}
	return out, rows.Err()
}

// ListGlossary returns glossary entries for a course locale pair.
func ListGlossary(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, sourceLocale, targetLocale string) ([]GlossaryRow, error) {
	rows, err := pool.Query(ctx, `
SELECT id, source_term, target_term, source_locale, target_locale
FROM i18n.course_glossaries
WHERE course_id = $1 AND source_locale = $2 AND target_locale = $3
ORDER BY lower(source_term)
`, courseID, sourceLocale, targetLocale)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GlossaryRow
	for rows.Next() {
		var g GlossaryRow
		if err := rows.Scan(&g.ID, &g.SourceTerm, &g.TargetTerm, &g.SourceLocale, &g.TargetLocale); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// AddGlossaryEntry inserts a glossary term.
func AddGlossaryEntry(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, sourceLocale, targetLocale, sourceTerm, targetTerm string) (GlossaryRow, error) {
	var g GlossaryRow
	err := pool.QueryRow(ctx, `
INSERT INTO i18n.course_glossaries (course_id, source_locale, target_locale, source_term, target_term)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (course_id, source_locale, target_locale, source_term)
DO UPDATE SET target_term = EXCLUDED.target_term
RETURNING id, source_term, target_term, source_locale, target_locale
`, courseID, sourceLocale, targetLocale, sourceTerm, targetTerm).Scan(
		&g.ID, &g.SourceTerm, &g.TargetTerm, &g.SourceLocale, &g.TargetLocale,
	)
	return g, err
}

// ResolveItemType returns content_page or assignment for a structure item in a course.
func ResolveItemType(ctx context.Context, pool *pgxpool.Pool, courseID, itemID uuid.UUID) (ItemType, error) {
	var kind string
	err := pool.QueryRow(ctx, `
SELECT kind FROM course.course_structure_items
WHERE id = $1 AND course_id = $2
`, itemID, courseID).Scan(&kind)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	if err != nil {
		return "", err
	}
	switch kind {
	case "content_page":
		return TypeContentPage, nil
	case "assignment":
		return TypeAssignment, nil
	default:
		return "", fmt.Errorf("unsupported item kind %q", kind)
	}
}

// GetSourceContent loads title and body for a translatable item.
func GetSourceContent(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID, itemType ItemType) (title, body string, err error) {
	switch itemType {
	case TypeContentPage:
		err = pool.QueryRow(ctx, `
SELECT COALESCE(c.title, ''), COALESCE(m.markdown, '')
FROM course.course_structure_items c
INNER JOIN course.module_content_pages m ON m.structure_item_id = c.id
WHERE c.id = $1
`, itemID).Scan(&title, &body)
	case TypeAssignment:
		err = pool.QueryRow(ctx, `
SELECT COALESCE(c.title, ''), COALESCE(m.markdown, '')
FROM course.course_structure_items c
INNER JOIN course.module_assignments m ON m.structure_item_id = c.id
WHERE c.id = $1
`, itemID).Scan(&title, &body)
	default:
		err = fmt.Errorf("unknown item type")
	}
	return title, body, err
}

// GetEnrollmentContentLocale returns the student's preferred content locale for a course.
func GetEnrollmentContentLocale(ctx context.Context, pool *pgxpool.Pool, courseID, userID uuid.UUID) (*string, error) {
	var loc *string
	err := pool.QueryRow(ctx, `
SELECT content_locale FROM course.course_enrollments
WHERE course_id = $1 AND user_id = $2 AND active
`, courseID, userID).Scan(&loc)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return loc, err
}

// SetEnrollmentContentLocale updates the student's preferred content locale.
func SetEnrollmentContentLocale(ctx context.Context, pool *pgxpool.Pool, courseID, userID uuid.UUID, locale *string) error {
	tag, err := pool.Exec(ctx, `
UPDATE course.course_enrollments
SET content_locale = $3
WHERE course_id = $1 AND user_id = $2 AND active
`, courseID, userID, locale)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// LocalesWithCoverage returns target locales with any published translations in the course.
func LocalesWithCoverage(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]Coverage, error) {
	rows, err := pool.Query(ctx, `
SELECT ct.target_locale,
       COUNT(*) FILTER (WHERE ct.published_at IS NOT NULL AND NOT ct.is_draft) AS published_count
FROM i18n.content_translations ct
INNER JOIN course.course_structure_items csi ON csi.id = ct.source_item_id
WHERE csi.course_id = $1
GROUP BY ct.target_locale
HAVING COUNT(*) FILTER (WHERE ct.published_at IS NOT NULL AND NOT ct.is_draft) > 0
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items, err := ListTranslatableItems(ctx, pool, courseID)
	if err != nil {
		return nil, err
	}
	total := len(items)
	var out []Coverage
	for rows.Next() {
		var loc string
		var n int
		if err := rows.Scan(&loc, &n); err != nil {
			return nil, err
		}
		pct := 0.0
		if total > 0 {
			pct = float64(n) / float64(total) * 100
		}
		out = append(out, Coverage{
			TargetLocale:    loc,
			TotalItems:      total,
			TranslatedItems: n,
			Percent:         pct,
		})
	}
	return out, rows.Err()
}
