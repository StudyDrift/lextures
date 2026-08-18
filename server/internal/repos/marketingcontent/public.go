package marketingcontent

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type PublicArticle struct {
	Article
	CategorySlug     *string           `json:"categorySlug"`
	CategoryTitle    *string           `json:"categoryTitle"`
	Author           *Author           `json:"author"`
	Reviewer         *Author           `json:"reviewer"`
	Tags             []string          `json:"tags"`
	AvailableLocales []AvailableLocale `json:"availableLocales,omitempty"`
	IsFallback       bool              `json:"isFallback,omitempty"`
}

type PublicCategory struct {
	Slug, Locale, Title, Description, PlatformPath string
	SortOrder, ArticleCount                        int
}

type SearchResult struct {
	Path        string  `json:"path"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Snippet     string  `json:"snippet"`
	Kind        string  `json:"kind"`
	Locale      string  `json:"locale"`
	IsFallback  bool    `json:"isFallback,omitempty"`
	Rank        float64 `json:"-"`
}

const publicJoins = ` FROM marketing.content_articles a
 LEFT JOIN marketing.content_categories c ON c.id=a.category_id
 JOIN marketing.content_authors au ON au.slug=a.author_slug
 LEFT JOIN marketing.content_authors rv ON rv.slug=a.reviewer_slug`

const publicColumns = `a.id,a.kind,a.slug,a.locale,a.translation_group_id,a.category_id,a.path,
 a.title,a.description,a.body_md,a.status,a.author_slug,a.reviewer_slug,a.published_at,
 a.first_published_at,a.scheduled_for,a.content_updated_at,a.reviewed_at,a.review_due_on,
 a.primary_question,a.cluster,a.pillar,a.brief_ref,a.verified_against,a.keywords,a.related_to,
 a.roles,a.segments,a.citations,a.hero_media_id,a.quality_score,a.quality_report,a.noindex,
 a.canonical_override,a.extra,a.revision_no,a.source_article_id,a.source_synced_revision,a.source_synced_at,a.created_by,a.updated_by,a.created_at,a.updated_at,a.deleted_at,c.slug,c.title,
 au.slug,au.name,au.job_title,au.bio,au.status,au.knows_about,au.image_media_id,au.user_id,au.links,au.created_by,au.updated_by,au.created_at,au.updated_at,
 rv.slug,rv.name,rv.job_title,rv.bio,rv.status,rv.knows_about,rv.image_media_id,rv.user_id,rv.links,rv.created_by,rv.updated_by,rv.created_at,rv.updated_at,
 COALESCE((SELECT array_agg(t.slug ORDER BY t.slug) FROM marketing.content_article_tags at JOIN marketing.content_tags t ON t.id=at.tag_id WHERE at.article_id=a.id),'{}')`

// scanPublic is kept explicit because pgx cannot scan nullable joined structs.
func scanPublic(row pgx.Row) (*PublicArticle, error) {
	var p PublicArticle
	var au Author
	var rs, rn, rj, rb, rst *string
	var rk []string
	var ri, ru, rcb, rub *uuid.UUID
	var rl []byte
	var rca, rua *time.Time
	err := row.Scan(&p.ID, &p.Kind, &p.Slug, &p.Locale, &p.TranslationGroupID, &p.CategoryID, &p.Path, &p.Title, &p.Description, &p.BodyMD, &p.Status, &p.AuthorSlug, &p.ReviewerSlug, &p.PublishedAt, &p.FirstPublishedAt, &p.ScheduledFor, &p.ContentUpdatedAt, &p.ReviewedAt, &p.ReviewDueOn, &p.PrimaryQuestion, &p.Cluster, &p.Pillar, &p.BriefRef, &p.VerifiedAgainst, &p.Keywords, &p.RelatedTo, &p.Roles, &p.Segments, &p.Citations, &p.HeroMediaID, &p.QualityScore, &p.QualityReport, &p.Noindex, &p.CanonicalOverride, &p.Extra, &p.RevisionNo, &p.SourceArticleID, &p.SourceSyncedRevision, &p.SourceSyncedAt, &p.CreatedBy, &p.UpdatedBy, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt,
		&p.CategorySlug, &p.CategoryTitle, &au.Slug, &au.Name, &au.JobTitle, &au.Bio, &au.Status, &au.KnowsAbout, &au.ImageMediaID, &au.UserID, &au.Links, &au.CreatedBy, &au.UpdatedBy, &au.CreatedAt, &au.UpdatedAt,
		&rs, &rn, &rj, &rb, &rst, &rk, &ri, &ru, &rl, &rcb, &rub, &rca, &rua, &p.Tags)
	if err != nil {
		return nil, err
	}
	applySocialFromExtra(&p.Article)
	p.Author = &au
	if rs != nil {
		p.Reviewer = &Author{Slug: *rs, Name: value(rn), JobTitle: value(rj), Bio: value(rb), Status: value(rst), KnowsAbout: rk, ImageMediaID: ri, UserID: ru, Links: rl, CreatedBy: rcb, UpdatedBy: rub}
		if rca != nil {
			p.Reviewer.CreatedAt = *rca
		}
		if rua != nil {
			p.Reviewer.UpdatedAt = *rua
		}
	}
	return &p, nil
}

func value(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func GetPublishedArticleByPath(ctx context.Context, q querier, path string) (*PublicArticle, error) {
	return scanPublic(q.QueryRow(ctx, `SELECT `+publicColumns+publicJoins+` WHERE a.path=$1 AND a.status='published' AND a.deleted_at IS NULL AND a.published_at<=now()`, path))
}

// GetPreviewArticleByPath may see non-public rows only so the service can verify
// an article-bound token. Handlers must never return it before verification.
func GetPreviewArticleByPath(ctx context.Context, q querier, path string) (*PublicArticle, error) {
	return scanPublic(q.QueryRow(ctx, `SELECT `+publicColumns+publicJoins+` WHERE a.path=$1 AND a.deleted_at IS NULL`, path))
}

func ListPublishedArticles(ctx context.Context, q querier, f PublicArticleFilter) ([]PublicArticle, string, error) {
	limit := f.Limit
	if limit < 1 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	var ct time.Time
	var cid uuid.UUID
	if f.Cursor != "" {
		b, e := base64.RawURLEncoding.DecodeString(f.Cursor)
		if e != nil {
			return nil, "", fmt.Errorf("invalid cursor")
		}
		parts := strings.SplitN(string(b), "/", 2)
		if len(parts) != 2 {
			return nil, "", fmt.Errorf("invalid cursor")
		}
		ct, e = time.Parse(time.RFC3339Nano, parts[0])
		if e != nil {
			return nil, "", fmt.Errorf("invalid cursor")
		}
		cid, e = uuid.Parse(parts[1])
		if e != nil {
			return nil, "", fmt.Errorf("invalid cursor")
		}
	}
	rows, e := q.Query(ctx, `SELECT `+publicColumns+publicJoins+` WHERE a.status='published' AND a.deleted_at IS NULL AND a.published_at<=now()
 AND ($1='' OR a.kind=$1) AND ($2='' OR a.locale=$2) AND ($3='' OR c.slug=$3) AND ($4='' OR a.author_slug=$4)
 AND ($5='' OR EXISTS (SELECT 1 FROM marketing.content_article_tags x JOIN marketing.content_tags t ON t.id=x.tag_id WHERE x.article_id=a.id AND t.slug=$5))
 AND ($6='' OR a.search_tsv @@ websearch_to_tsquery(COALESCE((SELECT l.ts_config FROM marketing.content_locales l WHERE l.code=a.locale),'simple')::regconfig,$6)) AND ($7::timestamptz IS NULL OR (a.published_at,a.id)<($7,$8))
 ORDER BY a.published_at DESC,a.id DESC LIMIT $9`, f.Kind, f.Locale, f.CategorySlug, f.AuthorSlug, f.Tag, f.Q, nullTime(ct), nullUUID(cid), limit+1)
	if e != nil {
		return nil, "", e
	}
	defer rows.Close()
	out := make([]PublicArticle, 0, limit+1)
	for rows.Next() {
		p, e := scanPublic(rows)
		if e != nil {
			return nil, "", e
		}
		out = append(out, *p)
	}
	if e = rows.Err(); e != nil {
		return nil, "", e
	}
	next := ""
	if len(out) > limit {
		last := out[limit-1]
		next = base64.RawURLEncoding.EncodeToString([]byte(last.PublishedAt.Format(time.RFC3339Nano) + "/" + last.ID.String()))
		out = out[:limit]
	}
	return out, next, nil
}

func ListPublicCategories(ctx context.Context, q querier, locale string) ([]PublicCategory, error) {
	rows, e := q.Query(ctx, `SELECT c.slug,c.locale,c.title,c.description,c.sort_order,c.platform_path,count(a.id) FROM marketing.content_categories c LEFT JOIN marketing.content_articles a ON a.category_id=c.id AND a.status='published' AND a.deleted_at IS NULL AND a.published_at<=now() WHERE ($1='' OR c.locale=$1) GROUP BY c.id ORDER BY c.sort_order,c.slug`, locale)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []PublicCategory
	for rows.Next() {
		var c PublicCategory
		if e = rows.Scan(&c.Slug, &c.Locale, &c.Title, &c.Description, &c.SortOrder, &c.PlatformPath, &c.ArticleCount); e != nil {
			return nil, e
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
func ListPublicAuthors(ctx context.Context, q querier) ([]Author, error) {
	rows, e := q.Query(ctx, `SELECT slug,name,job_title,bio,knows_about,image_media_id,links,user_id,status,created_by,updated_by,created_at,updated_at FROM marketing.content_authors WHERE status='active' ORDER BY name,slug`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []Author
	for rows.Next() {
		var a Author
		if e = rows.Scan(&a.Slug, &a.Name, &a.JobTitle, &a.Bio, &a.KnowsAbout, &a.ImageMediaID, &a.Links, &a.UserID, &a.Status, &a.CreatedBy, &a.UpdatedBy, &a.CreatedAt, &a.UpdatedAt); e != nil {
			return nil, e
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
func GetPublicAuthor(ctx context.Context, q querier, slug string) (*Author, error) {
	var a Author
	e := q.QueryRow(ctx, `SELECT slug,name,job_title,bio,knows_about,image_media_id,links,user_id,status,created_by,updated_by,created_at,updated_at FROM marketing.content_authors WHERE slug=$1 AND status='active'`, slug).Scan(&a.Slug, &a.Name, &a.JobTitle, &a.Bio, &a.KnowsAbout, &a.ImageMediaID, &a.Links, &a.UserID, &a.Status, &a.CreatedBy, &a.UpdatedBy, &a.CreatedAt, &a.UpdatedAt)
	return &a, e
}
func SearchPublished(ctx context.Context, q querier, query, kind, locale string, limit int) ([]SearchResult, error) {
	if limit < 1 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	if locale == "" {
		locale = DefaultLocale
	}
	cfg, err := LocaleTSConfig(ctx, q, locale)
	if err != nil {
		cfg = "simple"
	}
	rows, e := q.Query(ctx, `WITH x AS (SELECT websearch_to_tsquery($4::regconfig,$1) q) SELECT a.path,a.title,a.description,ts_headline($4::regconfig,a.body_md,x.q,'MaxWords=30,MinWords=10,StartSel=<mark>,StopSel=</mark>'),a.kind,a.locale,ts_rank_cd(a.search_tsv,x.q) FROM marketing.content_articles a,x WHERE $1<>'' AND a.status='published' AND a.deleted_at IS NULL AND a.published_at<=now() AND a.search_tsv@@x.q AND ($2='' OR a.kind=$2) AND a.locale=$3 ORDER BY 7 DESC,a.published_at DESC LIMIT $5`, query, kind, locale, cfg, limit)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []SearchResult
	for rows.Next() {
		var s SearchResult
		if e = rows.Scan(&s.Path, &s.Title, &s.Description, &s.Snippet, &s.Kind, &s.Locale, &s.Rank); e != nil {
			return nil, e
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func PublicContentVersion(ctx context.Context, q querier) (string, error) {
	var v string
	e := q.QueryRow(ctx, `SELECT md5(concat_ws('|',COALESCE((SELECT max(updated_at)::text FROM marketing.content_articles),''),COALESCE((SELECT max(updated_at)::text FROM marketing.content_categories),''),COALESCE((SELECT max(updated_at)::text FROM marketing.content_authors),''),COALESCE((SELECT max(updated_at)::text FROM marketing.content_redirects),''),COALESCE((SELECT max(updated_at)::text FROM marketing.content_tags),'')))`).Scan(&v)
	return v, e
}
