package marketingcontent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func ListTranslations(ctx context.Context, q querier, articleID uuid.UUID) ([]TranslationLink, error) {
	rows, err := q.Query(ctx, `SELECT a.id,a.locale,a.path,a.status,a.source_synced_revision,a.published_at,a.title,
 COALESCE(src.content_updated_at IS NOT NULL AND a.source_synced_at IS NOT NULL AND src.content_updated_at > a.source_synced_at, false)
 FROM marketing.content_articles a
 JOIN marketing.content_articles seed ON seed.id=$1 AND seed.deleted_at IS NULL
 LEFT JOIN marketing.content_articles src ON src.id=COALESCE(a.source_article_id, seed.id) AND src.locale='en'
 WHERE a.translation_group_id=seed.translation_group_id AND a.deleted_at IS NULL
 ORDER BY a.locale`, articleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]TranslationLink, 0)
	for rows.Next() {
		var t TranslationLink
		if err := rows.Scan(&t.ID, &t.Locale, &t.Path, &t.Status, &t.SourceSyncedRevision, &t.PublishedAt, &t.Title, &t.Stale); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func PublishedLocalesForGroup(ctx context.Context, q querier, groupID uuid.UUID) ([]AvailableLocale, error) {
	rows, err := q.Query(ctx, `SELECT locale,path FROM marketing.content_articles
 WHERE translation_group_id=$1 AND status='published' AND deleted_at IS NULL AND published_at<=now()
 ORDER BY locale`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]AvailableLocale, 0)
	for rows.Next() {
		var a AvailableLocale
		if err := rows.Scan(&a.Locale, &a.Path); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func MarkTranslationSynced(ctx context.Context, tx pgx.Tx, id, actor uuid.UUID, revision int, at time.Time) (*Article, error) {
	a, err := scanArticle(tx.QueryRow(ctx, `UPDATE marketing.content_articles SET
 source_synced_revision=$2, source_synced_at=$3, revision_no=revision_no+1, updated_by=$4
 WHERE id=$1 AND deleted_at IS NULL RETURNING `+articleColumns, id, revision, at, nullUUID(actor)))
	if err != nil {
		return nil, err
	}
	if err := insertRevision(ctx, tx, a, "Marked translation in sync with source", actor); err != nil {
		return nil, err
	}
	return a, nil
}

func DefaultLocaleSource(ctx context.Context, q querier, groupID uuid.UUID) (*Article, error) {
	return scanArticle(q.QueryRow(ctx, `SELECT `+articleColumns+` FROM marketing.content_articles
 WHERE translation_group_id=$1 AND locale=$2 AND deleted_at IS NULL ORDER BY created_at LIMIT 1`, groupID, DefaultLocale))
}

func StaleTranslationCount(ctx context.Context, q querier) (int64, error) {
	var n int64
	err := q.QueryRow(ctx, `SELECT count(*) FROM marketing.content_articles a
 JOIN marketing.content_articles src ON src.id=a.source_article_id
 WHERE a.deleted_at IS NULL AND a.locale<>$1 AND src.content_updated_at IS NOT NULL
 AND a.source_synced_at IS NOT NULL AND src.content_updated_at > a.source_synced_at`, DefaultLocale).Scan(&n)
	return n, err
}

func ListStaleTranslations(ctx context.Context, q querier) ([]ArticleSummary, error) {
	rows, err := q.Query(ctx, `SELECT a.id,a.kind,a.slug,a.locale,a.path,a.title,a.description,a.status,
 a.author_slug,COALESCE(au.name,a.author_slug),a.reviewer_slug,rv.name,a.category_id,c.slug,c.title,
 a.review_due_on,a.quality_score,a.published_at,a.revision_no,a.updated_at,a.translation_group_id,
 ARRAY[a.locale]::text[], true
 FROM marketing.content_articles a
 JOIN marketing.content_articles src ON src.id=a.source_article_id
 LEFT JOIN marketing.content_categories c ON c.id=a.category_id
 LEFT JOIN marketing.content_authors au ON au.slug=a.author_slug
 LEFT JOIN marketing.content_authors rv ON rv.slug=a.reviewer_slug
 WHERE a.deleted_at IS NULL AND a.locale<>$1 AND src.content_updated_at IS NOT NULL
 AND a.source_synced_at IS NOT NULL AND src.content_updated_at > a.source_synced_at
 ORDER BY src.content_updated_at DESC, a.title LIMIT 200`, DefaultLocale)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ArticleSummary, 0)
	for rows.Next() {
		var a ArticleSummary
		if err := rows.Scan(&a.ID, &a.Kind, &a.Slug, &a.Locale, &a.Path, &a.Title, &a.Description, &a.Status, &a.AuthorSlug, &a.AuthorName, &a.ReviewerSlug, &a.ReviewerName, &a.CategoryID, &a.CategorySlug, &a.CategoryTitle, &a.ReviewDueOn, &a.QualityScore, &a.PublishedAt, &a.RevisionNo, &a.UpdatedAt, &a.TranslationGroupID, &a.GroupLocales, &a.Stale); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func ArticleCountsByLocale(ctx context.Context, q querier) (map[string]int64, error) {
	rows, err := q.Query(ctx, `SELECT locale,count(*) FROM marketing.content_articles WHERE deleted_at IS NULL GROUP BY locale`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]int64)
	for rows.Next() {
		var locale string
		var n int64
		if err := rows.Scan(&locale, &n); err != nil {
			return nil, err
		}
		out[locale] = n
	}
	return out, rows.Err()
}

func EnsureLocaleAllowed(ctx context.Context, q querier, code string) error {
	l, err := GetLocale(ctx, q, code)
	if err == pgx.ErrNoRows {
		return fmt.Errorf("unsupported locale")
	}
	if err != nil {
		return err
	}
	if !l.Enabled {
		return fmt.Errorf("locale %s is not enabled", code)
	}
	return nil
}
