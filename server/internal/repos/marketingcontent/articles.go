package marketingcontent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type querier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

const articleColumns = `id, kind, slug, locale, translation_group_id, category_id, path,
 title, description, body_md, status, author_slug, reviewer_slug, published_at,
 first_published_at, scheduled_for, content_updated_at, reviewed_at, review_due_on,
 primary_question, cluster, pillar, brief_ref, verified_against, keywords, related_to,
 roles, segments, citations, hero_media_id, quality_score, quality_report, noindex,
 canonical_override, extra, revision_no, source_article_id, source_synced_revision,
 source_synced_at, created_by, updated_by, created_at, updated_at, deleted_at`

func scanArticle(row pgx.Row) (*Article, error) {
	var a Article
	err := row.Scan(&a.ID, &a.Kind, &a.Slug, &a.Locale, &a.TranslationGroupID, &a.CategoryID,
		&a.Path, &a.Title, &a.Description, &a.BodyMD, &a.Status, &a.AuthorSlug, &a.ReviewerSlug,
		&a.PublishedAt, &a.FirstPublishedAt, &a.ScheduledFor, &a.ContentUpdatedAt, &a.ReviewedAt,
		&a.ReviewDueOn, &a.PrimaryQuestion, &a.Cluster, &a.Pillar, &a.BriefRef, &a.VerifiedAgainst,
		&a.Keywords, &a.RelatedTo, &a.Roles, &a.Segments, &a.Citations, &a.HeroMediaID,
		&a.QualityScore, &a.QualityReport, &a.Noindex, &a.CanonicalOverride, &a.Extra,
		&a.RevisionNo, &a.SourceArticleID, &a.SourceSyncedRevision, &a.SourceSyncedAt,
		&a.CreatedBy, &a.UpdatedBy, &a.CreatedAt, &a.UpdatedAt, &a.DeletedAt)
	if err == nil {
		normalizeArticleCollections(&a)
	}
	return &a, err
}

func normalizeArticleCollections(a *Article) {
	if a.Keywords == nil {
		a.Keywords = []string{}
	}
	if a.RelatedTo == nil {
		a.RelatedTo = []string{}
	}
	if a.Roles == nil {
		a.Roles = []string{}
	}
	if a.Segments == nil {
		a.Segments = []string{}
	}
	if a.Citations == nil {
		a.Citations = []string{}
	}
}

func GetArticleByID(ctx context.Context, q querier, id uuid.UUID) (*Article, error) {
	a, err := scanArticle(q.QueryRow(ctx, `SELECT `+articleColumns+` FROM marketing.content_articles WHERE id=$1 AND deleted_at IS NULL`, id))
	if err != nil {
		return nil, err
	}
	if a.SourceArticleID != nil {
		var srcUpdated *time.Time
		if e := q.QueryRow(ctx, `SELECT content_updated_at FROM marketing.content_articles WHERE id=$1`, *a.SourceArticleID).Scan(&srcUpdated); e == nil {
			a.Stale = TranslationIsStale(srcUpdated, a.SourceSyncedAt)
		}
	}
	return a, nil
}

func GetArticleByPath(ctx context.Context, q querier, path string) (*Article, error) {
	return scanArticle(q.QueryRow(ctx, `SELECT `+articleColumns+` FROM marketing.content_articles WHERE path=$1 AND deleted_at IS NULL`, path))
}

func GetArticleByKey(ctx context.Context, q querier, kind, locale, slug string) (*Article, error) {
	return scanArticle(q.QueryRow(ctx, `SELECT `+articleColumns+` FROM marketing.content_articles WHERE kind=$1 AND locale=$2 AND slug=$3 AND deleted_at IS NULL`, kind, locale, slug))
}

// ArticleCounts returns live article totals grouped by kind and workflow status.
func ArticleCounts(ctx context.Context, q querier) (map[[2]string]int64, error) {
	rows, err := q.Query(ctx, `SELECT kind,status,count(*) FROM marketing.content_articles WHERE deleted_at IS NULL GROUP BY kind,status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[[2]string]int64)
	for rows.Next() {
		var kind, status string
		var count int64
		if err := rows.Scan(&kind, &status, &count); err != nil {
			return nil, err
		}
		out[[2]string{kind, status}] = count
	}
	return out, rows.Err()
}

func ListArticles(ctx context.Context, q querier, f ArticleFilter) ([]ArticleSummary, string, error) {
	limit := f.Limit
	if limit < 1 || limit > 100 {
		limit = 25
	}
	var cursorTime time.Time
	var cursorID uuid.UUID
	if f.Cursor != "" {
		b, err := base64.RawURLEncoding.DecodeString(f.Cursor)
		if err != nil {
			return nil, "", fmt.Errorf("decode cursor: %w", err)
		}
		parts := strings.SplitN(string(b), "/", 2)
		if len(parts) != 2 {
			return nil, "", fmt.Errorf("invalid cursor")
		}
		cursorTime, err = time.Parse(time.RFC3339Nano, parts[0])
		if err != nil {
			return nil, "", fmt.Errorf("invalid cursor: %w", err)
		}
		cursorID, err = uuid.Parse(parts[1])
		if err != nil {
			return nil, "", fmt.Errorf("invalid cursor: %w", err)
		}
	}
	sortColumn := "a.updated_at"
	switch f.Sort {
	case "published":
		sortColumn = "COALESCE(a.published_at, '-infinity'::timestamptz)"
	case "title":
		sortColumn = "a.updated_at" // title sorting cannot share the timestamp cursor; keep cursor stable.
	}
	rows, err := q.Query(ctx, `SELECT a.id,a.kind,a.slug,a.locale,a.path,a.title,a.description,a.status,
 a.author_slug,COALESCE(au.name,a.author_slug),a.reviewer_slug,rv.name,a.category_id,c.slug,c.title,
 a.review_due_on,a.quality_score,a.published_at,a.revision_no,a.updated_at,a.translation_group_id,
 COALESCE((SELECT array_agg(g.locale ORDER BY g.locale) FROM marketing.content_articles g WHERE g.translation_group_id=a.translation_group_id AND g.deleted_at IS NULL), ARRAY[a.locale]::text[]),
 COALESCE(src.content_updated_at IS NOT NULL AND a.source_synced_at IS NOT NULL AND src.content_updated_at > a.source_synced_at, false)
 FROM marketing.content_articles a LEFT JOIN marketing.content_categories c ON c.id=a.category_id
 LEFT JOIN marketing.content_authors au ON au.slug=a.author_slug
 LEFT JOIN marketing.content_authors rv ON rv.slug=a.reviewer_slug
 LEFT JOIN marketing.content_articles src ON src.id=a.source_article_id
 WHERE a.deleted_at IS NULL AND ($1='' OR a.kind=$1) AND ($2='' OR a.status=$2)
 AND ($3='' OR a.locale=$3) AND ($4='' OR c.slug=$4)
 AND ($5='' OR a.search_tsv @@ websearch_to_tsquery(COALESCE((SELECT l.ts_config FROM marketing.content_locales l WHERE l.code=a.locale),'simple')::regconfig,$5))
 AND ($6='' OR a.author_slug=$6) AND (NOT $7 OR a.review_due_on < CURRENT_DATE)
 AND ($8::timestamptz IS NULL OR (`+sortColumn+`,a.id) < ($8,$9))
 ORDER BY `+sortColumn+` DESC,a.id DESC LIMIT $10`, f.Kind, f.Status, f.Locale, f.CategorySlug, f.Q,
		f.AuthorSlug, f.Overdue, nullTime(cursorTime), nullUUID(cursorID), limit+1)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]ArticleSummary, 0, limit+1)
	for rows.Next() {
		var a ArticleSummary
		if err := rows.Scan(&a.ID, &a.Kind, &a.Slug, &a.Locale, &a.Path, &a.Title, &a.Description, &a.Status, &a.AuthorSlug, &a.AuthorName, &a.ReviewerSlug, &a.ReviewerName, &a.CategoryID, &a.CategorySlug, &a.CategoryTitle, &a.ReviewDueOn, &a.QualityScore, &a.PublishedAt, &a.RevisionNo, &a.UpdatedAt, &a.TranslationGroupID, &a.GroupLocales, &a.Stale); err != nil {
			return nil, "", err
		}
		items = append(items, a)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	next := ""
	if len(items) > limit {
		last := items[limit-1]
		cursorAt := last.UpdatedAt
		if f.Sort == "published" && last.PublishedAt != nil {
			cursorAt = *last.PublishedAt
		}
		next = base64.RawURLEncoding.EncodeToString([]byte(cursorAt.Format(time.RFC3339Nano) + "/" + last.ID.String()))
		items = items[:limit]
	}
	return items, next, nil
}

func nullTime(v time.Time) any {
	if v.IsZero() {
		return nil
	}
	return v
}
func nullUUID(v uuid.UUID) any {
	if v == uuid.Nil {
		return nil
	}
	return v
}

func articlePath(ctx context.Context, q querier, kind, locale, slug string, categoryID *uuid.UUID) (string, error) {
	if kind == "blog" {
		return PublicPath("blog", locale, "", slug), nil
	}
	if kind != "doc" || categoryID == nil {
		return "", fmt.Errorf("doc article requires category")
	}
	var categorySlug string
	if err := q.QueryRow(ctx, `SELECT slug FROM marketing.content_categories WHERE id=$1`, *categoryID).Scan(&categorySlug); err != nil {
		return "", err
	}
	return PublicPath("doc", locale, categorySlug, slug), nil
}

func InsertArticle(ctx context.Context, tx pgx.Tx, in NewArticle) (*Article, error) {
	in = normalizeArticleInput(in)
	path, err := articlePath(ctx, tx, in.Kind, in.Locale, in.Slug, in.CategoryID)
	if err != nil {
		return nil, err
	}
	if in.Locale == "" {
		in.Locale = "en"
	}
	if in.Status == "" {
		in.Status = "draft"
	}
	if in.TranslationGroupID == uuid.Nil {
		in.TranslationGroupID = uuid.New()
	}
	a, err := scanArticle(tx.QueryRow(ctx, `INSERT INTO marketing.content_articles (`+
		`kind,slug,locale,translation_group_id,category_id,path,title,description,body_md,status,author_slug,reviewer_slug,published_at,first_published_at,scheduled_for,content_updated_at,reviewed_at,review_due_on,primary_question,cluster,pillar,brief_ref,verified_against,keywords,related_to,roles,segments,citations,hero_media_id,quality_score,quality_report,noindex,canonical_override,extra,source_article_id,source_synced_revision,source_synced_at,created_by,updated_by) `+
		`VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37,$38,$38) RETURNING `+articleColumns,
		in.Kind, in.Slug, in.Locale, in.TranslationGroupID, in.CategoryID, path, in.Title, in.Description, in.BodyMD, in.Status, in.AuthorSlug, in.ReviewerSlug, in.PublishedAt, in.FirstPublishedAt, in.ScheduledFor, in.ContentUpdatedAt, in.ReviewedAt, in.ReviewDueOn, in.PrimaryQuestion, in.Cluster, in.Pillar, in.BriefRef, in.VerifiedAgainst, in.Keywords, in.RelatedTo, in.Roles, in.Segments, in.Citations, in.HeroMediaID, in.QualityScore, in.QualityReport, in.Noindex, in.CanonicalOverride, in.Extra, in.SourceArticleID, in.SourceSyncedRevision, in.SourceSyncedAt, nullUUID(in.ActorID)))
	if err != nil {
		return nil, mapConstraint(err)
	}
	if err := insertRevision(ctx, tx, a, in.ChangeNote, in.ActorID); err != nil {
		return nil, err
	}
	return a, nil
}

func normalizeArticleInput(in NewArticle) NewArticle {
	if in.Keywords == nil {
		in.Keywords = []string{}
	}
	if in.RelatedTo == nil {
		in.RelatedTo = []string{}
	}
	if in.Roles == nil {
		in.Roles = []string{}
	}
	if in.Segments == nil {
		in.Segments = []string{}
	}
	if in.Citations == nil {
		in.Citations = []string{}
	}
	if len(in.Extra) == 0 {
		in.Extra = json.RawMessage(`{}`)
	}
	return in
}

func mapConstraint(err error) error {
	var pgErr *pgconn.PgError
	if errorsAs(err, &pgErr) && pgErr.Code == "23505" && (pgErr.ConstraintName == "idx_mc_articles_slug_live" || pgErr.ConstraintName == "idx_mc_articles_path_live") {
		return ErrDuplicateSlug
	}
	return err
}

// Kept behind a helper to keep pgconn handling testable without exporting implementation detail.
var errorsAs = func(err error, target any) bool { return errors.As(err, target) }
