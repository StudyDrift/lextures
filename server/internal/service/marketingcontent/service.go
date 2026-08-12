package marketingcontent

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	repo "github.com/lextures/lextures/server/internal/repos/marketingcontent"
	"github.com/lextures/lextures/server/internal/service/marketingcontent/render"
	validator "github.com/lextures/lextures/server/internal/service/marketingcontent/validate"
)

type Service struct {
	Pool          *pgxpool.Pool
	PreviewSecret []byte
	Now           func() time.Time
}
type TransitionInput struct {
	Action             Action
	ScheduledFor       *time.Time
	Note               string
	ExpectedRevisionNo int
	LintOverride       bool
}

var ErrLintBlocked = errors.New("marketingcontent: publish blocked by content validation")

func metadataFor(in repo.NewArticle) validator.Metadata {
	updated := ""
	if in.ContentUpdatedAt != nil {
		updated = in.ContentUpdatedAt.Format("2006-01-02")
	}
	return validator.Metadata{Title: in.Title, Description: in.Description, Updated: updated, Author: in.AuthorSlug, Cluster: in.Cluster, PrimaryQuestion: in.PrimaryQuestion, Keywords: in.Keywords}
}

func (s *Service) Lint(ctx context.Context, kind, body string, metadata validator.Metadata) validator.Report {
	paths, err := repo.KnownPaths(ctx, s.Pool)
	if err != nil {
		return validator.Report{Findings: []validator.Finding{{Rule: "validator_error", Severity: "warn", Message: "Known paths could not be loaded."}}, Stats: renderStats(body), ValidatorError: true}
	}
	return validator.Article(validator.Input{Kind: kind, BodyMD: body, Metadata: metadata, KnownPaths: paths})
}

func renderStats(body string) render.StatsResult { return render.Stats(body) }

func (s *Service) applyQuality(ctx context.Context, in *repo.NewArticle) validator.Report {
	if in.ContentUpdatedAt == nil {
		now := s.now()
		in.ContentUpdatedAt = &now
	}
	report := s.Lint(ctx, in.Kind, in.BodyMD, metadataFor(*in))
	data, err := json.Marshal(report)
	if err != nil {
		report.ValidatorError = true
		data = []byte(`{"validatorError":true}`)
	}
	in.QualityScore = &report.Score
	in.QualityReport = data
	return report
}

func blocksPublish(report validator.Report) bool {
	for _, f := range report.Findings {
		if f.Severity == "error" {
			return true
		}
	}
	return report.ValidatorError
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}
func (s *Service) transaction(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Service) Create(ctx context.Context, in repo.NewArticle) (*repo.Article, error) {
	s.applyQuality(ctx, &in)
	var out *repo.Article
	err := s.transaction(ctx, func(tx pgx.Tx) error {
		var err error
		out, err = repo.InsertArticle(ctx, tx, in)
		if err == nil {
			err = repo.SyncArticleMedia(ctx, tx, out.ID, out.BodyMD, out.HeroMediaID)
		}
		return err
	})
	return out, err
}

func (s *Service) Update(ctx context.Context, in repo.ArticleUpdate) (*repo.Article, *repo.Redirect, error) {
	s.applyQuality(ctx, &in.Article)
	var out *repo.Article
	var redirect *repo.Redirect
	err := s.transaction(ctx, func(tx pgx.Tx) error {
		before, err := repo.GetArticleByID(ctx, tx, in.ID)
		if err != nil {
			return err
		}
		out, err = repo.UpdateArticle(ctx, tx, in)
		if err != nil {
			return err
		}
		if err = repo.SyncArticleMedia(ctx, tx, out.ID, out.BodyMD, out.HeroMediaID); err != nil {
			return err
		}
		if before.Path != out.Path && (before.Status == "published" || out.Status == "published") {
			redirect = &repo.Redirect{FromPath: before.Path, ToPath: out.Path, StatusCode: 301, Source: "slug_change", ArticleID: &out.ID}
		}
		return nil
	})
	return out, redirect, err
}

func (s *Service) Transition(ctx context.Context, id, actor uuid.UUID, in TransitionInput) (*repo.Article, error) {
	var out *repo.Article
	err := s.transaction(ctx, func(tx pgx.Tx) error {
		current, err := repo.GetArticleByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if current.RevisionNo != in.ExpectedRevisionNo {
			return repo.ErrRevisionConflict
		}
		next, err := NextStatus(Status(current.Status), in.Action, in.ScheduledFor, s.now())
		if err != nil {
			return err
		}
		copy := articleAsInput(current, actor)
		report := s.applyQuality(ctx, &copy)
		if (in.Action == ActionPublish || in.Action == ActionSchedule) && blocksPublish(report) && !in.LintOverride {
			return ErrLintBlocked
		}
		copy.Status = string(next)
		copy.ChangeNote = in.Note
		now := s.now()
		switch in.Action {
		case ActionPublish:
			copy.PublishedAt = &now
			if copy.FirstPublishedAt == nil {
				copy.FirstPublishedAt = &now
			}
			copy.ScheduledFor = nil
		case ActionSchedule:
			copy.ScheduledFor = in.ScheduledFor
			copy.PublishedAt = nil
		case ActionUnpublish:
			copy.PublishedAt = nil
			copy.ScheduledFor = nil
		case ActionApprove:
			copy.ReviewedAt = &now
		}
		out, err = repo.UpdateArticle(ctx, tx, repo.ArticleUpdate{ID: id, ExpectedRevisionNo: in.ExpectedRevisionNo, Article: copy})
		return err
	})
	return out, err
}

func (s *Service) Restore(ctx context.Context, id, actor uuid.UUID, revisionNo, expected int, note string) (*repo.Article, error) {
	var out *repo.Article
	err := s.transaction(ctx, func(tx pgx.Tx) error {
		current, err := repo.GetArticleByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if current.RevisionNo != expected {
			return repo.ErrRevisionConflict
		}
		rev, err := repo.GetRevision(ctx, tx, id, revisionNo)
		if err != nil {
			return err
		}
		var snapshot repo.Article
		if err := json.Unmarshal(rev.Metadata, &snapshot); err != nil {
			return err
		}
		in := articleAsInput(&snapshot, actor)
		in.Status = current.Status
		in.PublishedAt = current.PublishedAt
		in.FirstPublishedAt = current.FirstPublishedAt
		in.ScheduledFor = current.ScheduledFor
		in.ChangeNote = note
		out, err = repo.UpdateArticle(ctx, tx, repo.ArticleUpdate{ID: id, ExpectedRevisionNo: expected, Article: in})
		return err
	})
	return out, err
}

func (s *Service) Delete(ctx context.Context, id, actor uuid.UUID, redirectTo string) error {
	return s.transaction(ctx, func(tx pgx.Tx) error {
		a, err := repo.GetArticleByID(ctx, tx, id)
		if err != nil {
			return err
		}
		if a.FirstPublishedAt != nil && strings.TrimSpace(redirectTo) == "" {
			return errors.New("published article requires redirectTo")
		}
		if err := repo.SoftDeleteArticle(ctx, tx, id, actor); err != nil {
			return err
		}
		if redirectTo != "" {
			_, err = repo.InsertRedirect(ctx, tx, repo.Redirect{FromPath: a.Path, ToPath: redirectTo, StatusCode: 301, Source: "manual", ArticleID: &id}, actor)
		}
		return err
	})
}

func articleAsInput(a *repo.Article, actor uuid.UUID) repo.NewArticle {
	return repo.NewArticle{Kind: a.Kind, Slug: a.Slug, Locale: a.Locale, Title: a.Title, Description: a.Description, BodyMD: a.BodyMD, Status: a.Status, TranslationGroupID: a.TranslationGroupID, CategoryID: a.CategoryID, AuthorSlug: a.AuthorSlug, ReviewerSlug: a.ReviewerSlug, PublishedAt: a.PublishedAt, FirstPublishedAt: a.FirstPublishedAt, ScheduledFor: a.ScheduledFor, ContentUpdatedAt: a.ContentUpdatedAt, ReviewedAt: a.ReviewedAt, ReviewDueOn: a.ReviewDueOn, PrimaryQuestion: a.PrimaryQuestion, Cluster: a.Cluster, Pillar: a.Pillar, BriefRef: a.BriefRef, VerifiedAgainst: a.VerifiedAgainst, Keywords: a.Keywords, RelatedTo: a.RelatedTo, Roles: a.Roles, Segments: a.Segments, Citations: a.Citations, HeroMediaID: a.HeroMediaID, QualityScore: a.QualityScore, QualityReport: a.QualityReport, Noindex: a.Noindex, CanonicalOverride: a.CanonicalOverride, Extra: a.Extra, ActorID: actor}
}

func (s *Service) MintPreviewToken(articleID uuid.UUID, revision int, ttl time.Duration) (string, time.Time, error) {
	if len(s.PreviewSecret) == 0 {
		return "", time.Time{}, errors.New("preview token secret is not configured")
	}
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	exp := s.now().Add(ttl)
	payload := articleID.String() + "." + strconv.Itoa(revision) + "." + strconv.FormatInt(exp.Unix(), 10)
	mac := hmac.New(sha256.New, s.PreviewSecret)
	_, _ = mac.Write([]byte(payload))
	token := base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return token, exp, nil
}

func (s *Service) VerifyPreviewToken(token string, articleID uuid.UUID, revision int) error {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return errors.New("invalid preview token")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return err
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return err
	}
	mac := hmac.New(sha256.New, s.PreviewSecret)
	_, _ = mac.Write(payload)
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return errors.New("invalid preview token")
	}
	fields := strings.Split(string(payload), ".")
	if len(fields) != 3 || fields[0] != articleID.String() || fields[1] != strconv.Itoa(revision) {
		return errors.New("preview token scope mismatch")
	}
	exp, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil || !s.now().Before(time.Unix(exp, 0)) {
		return fmt.Errorf("preview token expired")
	}
	return nil
}
