package marketingcontent

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type EditorialSettings struct {
	ReviewIntervalDocDays   int             `json:"reviewIntervalDocDays"`
	ReviewIntervalBlogDays  int             `json:"reviewIntervalBlogDays"`
	StaleThresholdPct       float64         `json:"staleThresholdPct"`
	RevisionRetentionMonths int             `json:"revisionRetentionMonths"`
	LocalesEnabled          bool            `json:"localesEnabled"`
	Pillars                 json.RawMessage `json:"pillars"`
}

type ReviewQueueItem struct {
	ArticleSummary
	ReviewerID       *uuid.UUID `json:"reviewerId"`
	SubmittedAt      time.Time  `json:"submittedAt"`
	BlockingFindings int        `json:"blockingFindings"`
}

type Review struct {
	ID                  uuid.UUID `json:"id"`
	ArticleID           uuid.UUID `json:"articleId"`
	RevisionNo          int       `json:"revisionNo"`
	Action              string    `json:"action"`
	ReviewerID, ActorID *uuid.UUID
	Note                string    `json:"note"`
	CreatedAt           time.Time `json:"createdAt"`
}

type Brief struct {
	ID              uuid.UUID  `json:"id"`
	Title           string     `json:"title"`
	Kind            string     `json:"kind"`
	Pillar          string     `json:"pillar"`
	Cluster         string     `json:"cluster"`
	PrimaryQuestion string     `json:"primaryQuestion"`
	OwnerID         *uuid.UUID `json:"ownerId"`
	OwnerName       string     `json:"ownerName"`
	TargetDate      *time.Time `json:"targetDate"`
	BriefRef        string     `json:"briefRef"`
	ArticleID       *uuid.UUID `json:"articleId"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

type executor interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func GetEditorialSettings(ctx context.Context, q querier) (EditorialSettings, error) {
	var s EditorialSettings
	err := q.QueryRow(ctx, `SELECT review_interval_doc_days,review_interval_blog_days,stale_threshold_pct,revision_retention_months,COALESCE(locales_enabled,false),pillars FROM marketing.content_editorial_settings WHERE singleton`).Scan(&s.ReviewIntervalDocDays, &s.ReviewIntervalBlogDays, &s.StaleThresholdPct, &s.RevisionRetentionMonths, &s.LocalesEnabled, &s.Pillars)
	return s, err
}

func ListReviewQueue(ctx context.Context, q querier, actor uuid.UUID, assigned bool) ([]ReviewQueueItem, error) {
	rows, err := q.Query(ctx, `SELECT a.id,a.kind,a.slug,a.locale,a.path,a.title,a.description,a.status,a.author_slug,
	 COALESCE(au.name,a.author_slug),a.reviewer_slug,rv.name,a.category_id,c.slug,c.title,a.review_due_on,
	 a.quality_score,a.published_at,a.revision_no,a.updated_at,a.reviewer_id,COALESCE(a.review_submitted_at,a.updated_at),
	 COALESCE((SELECT count(*) FROM jsonb_array_elements(COALESCE(a.quality_report->'findings','[]')) f WHERE f->>'severity'='error'),0)
	 FROM marketing.content_articles a LEFT JOIN marketing.content_categories c ON c.id=a.category_id
	 LEFT JOIN marketing.content_authors au ON au.slug=a.author_slug LEFT JOIN marketing.content_authors rv ON rv.slug=a.reviewer_slug
	 WHERE a.status='in_review' AND a.deleted_at IS NULL AND (NOT $2 OR a.reviewer_id=$1)
	 ORDER BY a.review_submitted_at ASC NULLS LAST,a.updated_at ASC LIMIT 100`, actor, assigned)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ReviewQueueItem{}
	for rows.Next() {
		var v ReviewQueueItem
		if err = rows.Scan(&v.ID, &v.Kind, &v.Slug, &v.Locale, &v.Path, &v.Title, &v.Description, &v.Status, &v.AuthorSlug, &v.AuthorName, &v.ReviewerSlug, &v.ReviewerName, &v.CategoryID, &v.CategorySlug, &v.CategoryTitle, &v.ReviewDueOn, &v.QualityScore, &v.PublishedAt, &v.RevisionNo, &v.UpdatedAt, &v.ReviewerID, &v.SubmittedAt, &v.BlockingFindings); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func InsertReview(ctx context.Context, tx pgx.Tx, articleID uuid.UUID, revision int, action string, reviewerID, actorID *uuid.UUID, note string) error {
	_, err := tx.Exec(ctx, `INSERT INTO marketing.content_reviews(article_id,revision_no,action,reviewer_id,actor_id,note) VALUES($1,$2,$3,$4,$5,$6)`, articleID, revision, action, reviewerID, actorID, note)
	return err
}

func ListReviews(ctx context.Context, q querier, articleID uuid.UUID) ([]Review, error) {
	rows, err := q.Query(ctx, `SELECT id,article_id,revision_no,action,reviewer_id,actor_id,note,created_at FROM marketing.content_reviews WHERE article_id=$1 ORDER BY created_at DESC`, articleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Review{}
	for rows.Next() {
		var v Review
		if err = rows.Scan(&v.ID, &v.ArticleID, &v.RevisionNo, &v.Action, &v.ReviewerID, &v.ActorID, &v.Note, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func NotifyOnce(ctx context.Context, tx pgx.Tx, key string, articleID, recipient uuid.UUID, event, title, body, url string) error {
	var enabled bool
	if err := tx.QueryRow(ctx, `SELECT COALESCE((SELECT push_enabled FROM settings.notification_preferences WHERE user_id=$1 AND event_type=$2),true)`, recipient, event).Scan(&enabled); err != nil {
		return err
	}
	if !enabled {
		return nil
	}
	var inserted bool
	err := tx.QueryRow(ctx, `INSERT INTO marketing.content_notification_log(dedupe_key,article_id,event_type,recipient_id) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING RETURNING true`, key, articleID, event, recipient).Scan(&inserted)
	if err == pgx.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO settings.notifications(user_id,event_type,title,body,action_url) VALUES($1,$2,$3,$4,$5)`, recipient, event, title, body, url)
	return err
}

func ListBriefs(ctx context.Context, q querier, from, to time.Time) ([]Brief, error) {
	rows, err := q.Query(ctx, `SELECT b.id,b.title,b.kind,b.pillar,b.cluster,b.primary_question,b.owner_id,COALESCE(NULLIF(trim(u.display_name),''),u.email,''),b.target_date,b.brief_ref,b.article_id,b.status,b.created_at,b.updated_at FROM marketing.content_briefs b LEFT JOIN "user".users u ON u.id=b.owner_id WHERE b.target_date BETWEEN $1::date AND $2::date ORDER BY b.target_date,b.title`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Brief{}
	for rows.Next() {
		var v Brief
		if err = rows.Scan(&v.ID, &v.Title, &v.Kind, &v.Pillar, &v.Cluster, &v.PrimaryQuestion, &v.OwnerID, &v.OwnerName, &v.TargetDate, &v.BriefRef, &v.ArticleID, &v.Status, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func InsertBrief(ctx context.Context, tx pgx.Tx, b Brief) (Brief, error) {
	err := tx.QueryRow(ctx, `INSERT INTO marketing.content_briefs(title,kind,pillar,cluster,primary_question,owner_id,target_date,brief_ref,article_id,status) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id,created_at,updated_at`, b.Title, b.Kind, b.Pillar, b.Cluster, b.PrimaryQuestion, b.OwnerID, b.TargetDate, b.BriefRef, b.ArticleID, b.Status).Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt)
	return b, err
}

func UpdateBrief(ctx context.Context, q executor, b Brief) (int64, error) {
	tag, err := q.Exec(ctx, `UPDATE marketing.content_briefs SET title=COALESCE(NULLIF($2,''),title),kind=COALESCE(NULLIF($3,''),kind),pillar=$4,cluster=$5,primary_question=$6,owner_id=$7,target_date=$8,brief_ref=$9,article_id=$10,status=COALESCE(NULLIF($11,''),status) WHERE id=$1`, b.ID, b.Title, b.Kind, b.Pillar, b.Cluster, b.PrimaryQuestion, b.OwnerID, b.TargetDate, b.BriefRef, b.ArticleID, b.Status)
	return tag.RowsAffected(), err
}
func DeleteBrief(ctx context.Context, q executor, id uuid.UUID) error {
	_, err := q.Exec(ctx, `DELETE FROM marketing.content_briefs WHERE id=$1`, id)
	return err
}
func UpdateEditorialSettings(ctx context.Context, q executor, s EditorialSettings) error {
	_, err := q.Exec(ctx, `UPDATE marketing.content_editorial_settings SET review_interval_doc_days=$1,review_interval_blog_days=$2,stale_threshold_pct=$3,revision_retention_months=$4,locales_enabled=$5,pillars=$6,updated_at=now() WHERE singleton`, s.ReviewIntervalDocDays, s.ReviewIntervalBlogDays, s.StaleThresholdPct, s.RevisionRetentionMonths, s.LocalesEnabled, s.Pillars)
	return err
}
func RetireAuthor(ctx context.Context, q executor, slug string, actor uuid.UUID) (bool, error) {
	tag, err := q.Exec(ctx, `UPDATE marketing.content_authors SET status='retired',updated_by=$2 WHERE slug=$1 AND status<>'retired'`, slug, actor)
	return tag.RowsAffected() > 0, err
}
