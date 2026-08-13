// Package marketingeditorial owns marketing-content governance read models and maintenance jobs.
package marketingeditorial

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	repo "github.com/lextures/lextures/server/internal/repos/marketingcontent"
)

type Service struct {
	Pool   *pgxpool.Pool
	Now    func() time.Time
	Client *http.Client
}
type Health struct {
	TakenAt           time.Time       `json:"takenAt"`
	Total             int             `json:"total"`
	Overdue           int             `json:"overdue"`
	Percent           float64         `json:"percent"`
	Threshold         float64         `json:"threshold"`
	AboveThreshold    bool            `json:"aboveThreshold"`
	ByOwner           []HealthGroup   `json:"byOwner"`
	ByCategory        []HealthGroup   `json:"byCategory"`
	LinkFailures      []LinkFailure   `json:"linkFailures"`
	StaleTranslations []HealthArticle `json:"staleTranslations"`
}
type HealthGroup struct {
	Key      string          `json:"key"`
	Label    string          `json:"label"`
	Count    int             `json:"count"`
	Articles []HealthArticle `json:"articles"`
}
type HealthArticle struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Path        string    `json:"path"`
	Owner       string    `json:"owner"`
	Category    string    `json:"category"`
	Locale      string    `json:"locale,omitempty"`
	ReviewDueOn time.Time `json:"reviewDueOn"`
}
type LinkFailure struct {
	ArticleID  uuid.UUID `json:"articleId"`
	Title      string    `json:"title"`
	Path       string    `json:"path"`
	URL        string    `json:"url"`
	StatusCode *int      `json:"statusCode"`
	Error      string    `json:"error"`
	CheckedAt  time.Time `json:"checkedAt"`
}
type CalendarItem struct {
	ID        uuid.UUID  `json:"id"`
	Type      string     `json:"type"`
	Title     string     `json:"title"`
	Path      string     `json:"path"`
	Date      time.Time  `json:"date"`
	ArticleID *uuid.UUID `json:"articleId,omitempty"`
}
type Pillar struct {
	ID        string `json:"id"`
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	Floor     int    `json:"floor"`
	Count     int    `json:"count"`
	Remaining int    `json:"remaining"`
	Gap       bool   `json:"gap"`
}
type Override struct {
	ID            uuid.UUID `json:"id"`
	ArticleID     uuid.UUID `json:"articleId"`
	RevisionNo    int       `json:"revisionNo"`
	ArticleTitle  string    `json:"articleTitle"`
	ArticlePath   string    `json:"articlePath"`
	Actor         string    `json:"actor"`
	Rules         []string  `json:"rules"`
	Justification string    `json:"justification"`
	CreatedAt     time.Time `json:"createdAt"`
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s *Service) Health(ctx context.Context) (Health, error) {
	settings, err := repo.GetEditorialSettings(ctx, s.Pool)
	if err != nil {
		return Health{}, err
	}
	h := Health{TakenAt: s.now(), Threshold: settings.StaleThresholdPct, ByOwner: []HealthGroup{}, ByCategory: []HealthGroup{}, LinkFailures: []LinkFailure{}, StaleTranslations: []HealthArticle{}}
	if err = s.Pool.QueryRow(ctx, `SELECT count(*),count(*) FILTER(WHERE review_due_on<CURRENT_DATE) FROM marketing.content_articles WHERE status='published' AND deleted_at IS NULL`).Scan(&h.Total, &h.Overdue); err != nil {
		return h, err
	}
	if h.Total > 0 {
		h.Percent = float64(h.Overdue) * 100 / float64(h.Total)
	}
	h.AboveThreshold = h.Percent > h.Threshold
	rows, err := s.Pool.Query(ctx, `SELECT a.id,a.title,a.path,a.author_slug,COALESCE(c.slug,'uncategorized'),a.review_due_on FROM marketing.content_articles a LEFT JOIN marketing.content_categories c ON c.id=a.category_id WHERE a.status='published' AND a.deleted_at IS NULL AND a.review_due_on<CURRENT_DATE ORDER BY a.author_slug,c.slug,a.review_due_on`)
	if err != nil {
		return h, err
	}
	defer rows.Close()
	owner := map[string]int{}
	category := map[string]int{}
	for rows.Next() {
		var a HealthArticle
		if err = rows.Scan(&a.ID, &a.Title, &a.Path, &a.Owner, &a.Category, &a.ReviewDueOn); err != nil {
			return h, err
		}
		oi, ok := owner[a.Owner]
		if !ok {
			oi = len(h.ByOwner)
			owner[a.Owner] = oi
			h.ByOwner = append(h.ByOwner, HealthGroup{Key: a.Owner, Label: a.Owner, Articles: []HealthArticle{}})
		}
		h.ByOwner[oi].Count++
		h.ByOwner[oi].Articles = append(h.ByOwner[oi].Articles, a)
		ci, ok := category[a.Category]
		if !ok {
			ci = len(h.ByCategory)
			category[a.Category] = ci
			h.ByCategory = append(h.ByCategory, HealthGroup{Key: a.Category, Label: a.Category, Articles: []HealthArticle{}})
		}
		h.ByCategory[ci].Count++
		h.ByCategory[ci].Articles = append(h.ByCategory[ci].Articles, a)
	}
	lf, err := s.Pool.Query(ctx, `SELECT l.article_id,a.title,a.path,l.url,l.status_code,COALESCE(l.error,''),l.checked_at FROM marketing.content_link_health l JOIN marketing.content_articles a ON a.id=l.article_id WHERE l.consecutive_failures>=2 AND a.deleted_at IS NULL ORDER BY l.checked_at DESC LIMIT 500`)
	if err != nil {
		return h, err
	}
	defer lf.Close()
	for lf.Next() {
		var v LinkFailure
		if err = lf.Scan(&v.ArticleID, &v.Title, &v.Path, &v.URL, &v.StatusCode, &v.Error, &v.CheckedAt); err != nil {
			return h, err
		}
		h.LinkFailures = append(h.LinkFailures, v)
	}
	if err = lf.Err(); err != nil {
		return h, err
	}
	stale, err := repo.ListStaleTranslations(ctx, s.Pool)
	if err != nil {
		return h, err
	}
	for _, a := range stale {
		h.StaleTranslations = append(h.StaleTranslations, HealthArticle{ID: a.ID, Title: a.Title, Path: a.Path, Owner: a.AuthorSlug, Locale: a.Locale})
	}
	return h, nil
}

func (s *Service) Snapshot(ctx context.Context) (Health, error) {
	h, err := s.Health(ctx)
	if err != nil {
		return h, err
	}
	payload, _ := json.Marshal(h)
	_, err = s.Pool.Exec(ctx, `INSERT INTO marketing.content_health_snapshots(payload) VALUES($1)`, payload)
	return h, err
}

// ReviewSweep emits one due reminder per article/day and one weekly overdue digest
// per owner. The notification log makes retries idempotent.
func (s *Service) ReviewSweep(ctx context.Context) (Health, error) {
	h, err := s.Snapshot(ctx)
	if err != nil {
		return h, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT a.id,a.title,au.user_id FROM marketing.content_articles a JOIN marketing.content_authors au ON au.slug=a.author_slug WHERE a.status='published' AND a.deleted_at IS NULL AND a.review_due_on=CURRENT_DATE+14 AND au.user_id IS NOT NULL`)
	if err != nil {
		return h, err
	}
	type target struct {
		id, user uuid.UUID
		title    string
	}
	due := []target{}
	for rows.Next() {
		var v target
		if err = rows.Scan(&v.id, &v.title, &v.user); err != nil {
			rows.Close()
			return h, err
		}
		due = append(due, v)
	}
	rows.Close()
	day := s.now().Format("2006-01-02")
	for _, v := range due {
		tx, e := s.Pool.Begin(ctx)
		if e != nil {
			return h, e
		}
		key := "review_due:" + v.id.String() + ":" + day
		e = repo.NotifyOnce(ctx, tx, key, v.id, v.user, "marketing_content_review_due", "Content review due soon", v.title, "/admin/marketing-content/"+v.id.String())
		if e == nil {
			e = tx.Commit(ctx)
		} else {
			_ = tx.Rollback(ctx)
		}
		if e != nil {
			return h, e
		}
	}
	// Monday's digest is one message per owner, not one notification per article.
	if s.now().Weekday() == time.Monday {
		for _, g := range h.ByOwner {
			var user uuid.UUID
			if e := s.Pool.QueryRow(ctx, `SELECT user_id FROM marketing.content_authors WHERE slug=$1`, g.Key).Scan(&user); e != nil {
				continue
			}
			tx, e := s.Pool.Begin(ctx)
			if e != nil {
				return h, e
			}
			key := "review_overdue_digest:" + user.String() + ":" + day
			articleID := g.Articles[0].ID
			e = repo.NotifyOnce(ctx, tx, key, articleID, user, "marketing_content_review_overdue", "Overdue content review digest", fmt.Sprintf("%d articles need review", g.Count), "/admin/marketing-content/editorial")
			if e == nil {
				e = tx.Commit(ctx)
			} else {
				_ = tx.Rollback(ctx)
			}
			if e != nil {
				return h, e
			}
		}
	}
	return h, nil
}

func (s *Service) Calendar(ctx context.Context, from, to time.Time) ([]CalendarItem, []repo.Brief, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id,title,path,scheduled_for FROM marketing.content_articles WHERE status='scheduled' AND deleted_at IS NULL AND scheduled_for >= $1 AND scheduled_for < $2 ORDER BY scheduled_for`, from, to)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	items := []CalendarItem{}
	for rows.Next() {
		var v CalendarItem
		v.Type = "scheduled"
		if err = rows.Scan(&v.ID, &v.Title, &v.Path, &v.Date); err != nil {
			return nil, nil, err
		}
		id := v.ID
		v.ArticleID = &id
		items = append(items, v)
	}
	briefs, err := repo.ListBriefs(ctx, s.Pool, from, to)
	return items, briefs, err
}

func (s *Service) Pillars(ctx context.Context) ([]Pillar, error) {
	settings, err := repo.GetEditorialSettings(ctx, s.Pool)
	if err != nil {
		return nil, err
	}
	var out []Pillar
	if err = json.Unmarshal(settings.Pillars, &out); err != nil {
		return nil, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT pillar,count(*) FROM marketing.content_articles WHERE status='published' AND deleted_at IS NULL GROUP BY pillar`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := map[string]int{}
	for rows.Next() {
		var id string
		var n int
		if err = rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		counts[id] = n
	}
	for i := range out {
		out[i].Count = counts[out[i].ID]
		out[i].Remaining = max(0, out[i].Floor-out[i].Count)
		out[i].Gap = out[i].Remaining > 0
	}
	return out, rows.Err()
}

func (s *Service) Overrides(ctx context.Context) ([]Override, error) {
	rows, err := s.Pool.Query(ctx, `SELECT o.id,o.article_id,o.revision_no,a.title,a.path,COALESCE(NULLIF(trim(u.display_name),''),u.email,'Deleted user'),o.rules,o.justification,o.created_at FROM marketing.content_overrides o JOIN marketing.content_articles a ON a.id=o.article_id LEFT JOIN "user".users u ON u.id=o.actor_id ORDER BY o.created_at DESC LIMIT 500`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Override{}
	for rows.Next() {
		var v Override
		if err = rows.Scan(&v.ID, &v.ArticleID, &v.RevisionNo, &v.ArticleTitle, &v.ArticlePath, &v.Actor, &v.Rules, &v.Justification, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Service) MarkReviewed(ctx context.Context, id, actor uuid.UUID) (repo.Article, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return repo.Article{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	a, err := repo.GetArticleByID(ctx, tx, id)
	if err != nil {
		return repo.Article{}, err
	}
	settings, err := repo.GetEditorialSettings(ctx, tx)
	if err != nil {
		return repo.Article{}, err
	}
	days := settings.ReviewIntervalBlogDays
	if a.Kind == "doc" {
		days = settings.ReviewIntervalDocDays
	}
	now := s.now()
	due := now.AddDate(0, 0, days)
	in := articleInput(a, actor)
	in.ReviewedAt = &now
	in.ReviewDueOn = &due
	in.ChangeNote = "reviewed, no content change"
	out, err := repo.UpdateArticle(ctx, tx, repo.ArticleUpdate{ID: id, ExpectedRevisionNo: a.RevisionNo, Article: in})
	if err != nil {
		return repo.Article{}, err
	}
	var reviewerID *uuid.UUID
	if a.ReviewerSlug != nil {
		var rid uuid.UUID
		if e := tx.QueryRow(ctx, `SELECT user_id FROM marketing.content_authors WHERE slug=$1`, *a.ReviewerSlug).Scan(&rid); e == nil {
			reviewerID = &rid
		}
	}
	if err = repo.InsertReview(ctx, tx, id, out.RevisionNo, "reviewed", reviewerID, &actor, ""); err != nil {
		return repo.Article{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return repo.Article{}, err
	}
	return *out, nil
}

func articleInput(a *repo.Article, actor uuid.UUID) repo.NewArticle {
	return repo.NewArticle{Kind: a.Kind, Slug: a.Slug, Locale: a.Locale, Title: a.Title, Description: a.Description, BodyMD: a.BodyMD, Status: a.Status, TranslationGroupID: a.TranslationGroupID, CategoryID: a.CategoryID, AuthorSlug: a.AuthorSlug, ReviewerSlug: a.ReviewerSlug, PublishedAt: a.PublishedAt, FirstPublishedAt: a.FirstPublishedAt, ScheduledFor: a.ScheduledFor, ContentUpdatedAt: a.ContentUpdatedAt, ReviewedAt: a.ReviewedAt, ReviewDueOn: a.ReviewDueOn, PrimaryQuestion: a.PrimaryQuestion, Cluster: a.Cluster, Pillar: a.Pillar, BriefRef: a.BriefRef, VerifiedAgainst: a.VerifiedAgainst, Keywords: a.Keywords, RelatedTo: a.RelatedTo, Roles: a.Roles, Segments: a.Segments, Citations: a.Citations, HeroMediaID: a.HeroMediaID, QualityScore: a.QualityScore, QualityReport: a.QualityReport, Noindex: a.Noindex, CanonicalOverride: a.CanonicalOverride, Extra: a.Extra, ActorID: actor}
}

func (s *Service) PruneRevisions(ctx context.Context) (int64, error) {
	settings, err := repo.GetEditorialSettings(ctx, s.Pool)
	if err != nil {
		return 0, err
	}
	tag, err := s.Pool.Exec(ctx, `WITH ranked AS (SELECT id,status_after,created_at,row_number() OVER(PARTITION BY article_id ORDER BY revision_no DESC) n FROM marketing.content_revisions) DELETE FROM marketing.content_revisions r USING ranked x WHERE r.id=x.id AND x.n>20 AND x.status_after<>'published' AND x.created_at < now()-make_interval(months=>$1)`, settings.RevisionRetentionMonths)
	return tag.RowsAffected(), err
}

var externalLink = regexp.MustCompile(`https?://[^\s<>\])}"']+`)

func publicURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return nil, errors.New("invalid URL")
	}
	ips, err := net.LookupIP(u.Hostname())
	if err != nil {
		return nil, err
	}
	for _, ip := range ips {
		if ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
			return nil, errors.New("non-public destination blocked")
		}
	}
	return u, nil
}
func (s *Service) CheckLinks(ctx context.Context) error {
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		_, err := publicURL(req.URL.String())
		if err != nil {
			return err
		}
		if len(via) >= 5 {
			return http.ErrUseLastResponse
		}
		return nil
	}
	rows, err := s.Pool.Query(ctx, `SELECT id,body_md FROM marketing.content_articles WHERE status='published' AND deleted_at IS NULL ORDER BY id LIMIT 500`)
	if err != nil {
		return err
	}
	defer rows.Close()
	checked := 0
	for rows.Next() {
		var id uuid.UUID
		var body string
		if err = rows.Scan(&id, &body); err != nil {
			return err
		}
		for _, raw := range externalLink.FindAllString(body, -1) {
			if checked >= 500 {
				return nil
			}
			checked++
			code := 0
			failure := ""
			u, e := publicURL(raw)
			if e != nil {
				failure = e.Error()
			} else {
				req, _ := http.NewRequestWithContext(ctx, http.MethodHead, u.String(), nil)
				req.Header.Set("User-Agent", "Lextures-Content-Link-Health/1.0")
				resp, e := client.Do(req)
				if e == nil && resp.StatusCode == http.StatusMethodNotAllowed {
					_ = resp.Body.Close()
					req, _ = http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
					req.Header.Set("User-Agent", "Lextures-Content-Link-Health/1.0")
					resp, e = client.Do(req)
				}
				if e != nil {
					failure = e.Error()
				} else {
					code = resp.StatusCode
					_ = resp.Body.Close()
					if code >= 400 {
						failure = fmt.Sprintf("HTTP %d", code)
					}
				}
			}
			_, err = s.Pool.Exec(ctx, `INSERT INTO marketing.content_link_health(article_id,url,status_code,error,consecutive_failures,checked_at) VALUES($1,$2,NULLIF($3,0),NULLIF($4,''),CASE WHEN $4='' THEN 0 ELSE 1 END,now()) ON CONFLICT(article_id,url) DO UPDATE SET status_code=EXCLUDED.status_code,error=EXCLUDED.error,checked_at=now(),consecutive_failures=CASE WHEN EXCLUDED.error IS NULL THEN 0 ELSE marketing.content_link_health.consecutive_failures+1 END`, id, raw, code, failure)
			if err != nil {
				return err
			}
		}
	}
	return rows.Err()
}

func ParseRange(fromRaw, toRaw string) (time.Time, time.Time, error) {
	from, err := time.Parse("2006-01-02", fromRaw)
	if err != nil {
		return from, time.Time{}, err
	}
	to, err := time.Parse("2006-01-02", toRaw)
	if err != nil {
		return from, to, err
	}
	if !to.After(from) || to.Sub(from) > 370*24*time.Hour {
		return from, to, errors.New("invalid calendar range")
	}
	return from, to.Add(24 * time.Hour), nil
}
