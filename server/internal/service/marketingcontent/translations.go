package marketingcontent

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lextures/lextures/server/internal/l10n"
	repo "github.com/lextures/lextures/server/internal/repos/marketingcontent"
)

var (
	ErrLocalesDisabled     = errors.New("marketingcontent: locales are not enabled")
	ErrUnsupportedLocale   = errors.New("marketingcontent: unsupported locale")
	ErrTranslationExists   = errors.New("marketingcontent: translation already exists for locale")
	ErrCannotTranslateSelf = errors.New("marketingcontent: cannot create a translation in the source locale")
)

type CreateTranslationInput struct {
	Locale string
	Slug   string
	Actor  uuid.UUID
}

func (s *Service) CreateTranslation(ctx context.Context, sourceID uuid.UUID, in CreateTranslationInput) (*repo.Article, error) {
	locale := repo.NormalizeLocaleCode(in.Locale)
	if locale == "" {
		return nil, ErrUnsupportedLocale
	}
	if _, err := l10n.NormalizeLocale(locale); err != nil {
		return nil, ErrUnsupportedLocale
	}
	var out *repo.Article
	err := s.transaction(ctx, func(tx pgx.Tx) error {
		on, err := repo.LocalesEnabled(ctx, tx)
		if err != nil {
			return err
		}
		if !on {
			return ErrLocalesDisabled
		}
		if err := repo.EnsureLocaleAllowed(ctx, tx, locale); err != nil {
			return ErrUnsupportedLocale
		}
		source, err := repo.GetArticleByID(ctx, tx, sourceID)
		if err != nil {
			return err
		}
		if source.Locale == locale {
			return ErrCannotTranslateSelf
		}
		english := source
		if source.Locale != repo.DefaultLocale {
			if root, rootErr := repo.DefaultLocaleSource(ctx, tx, source.TranslationGroupID); rootErr == nil {
				english = root
			}
		}
		existingGroup, listErr := repo.ListTranslations(ctx, tx, source.ID)
		if listErr != nil {
			return listErr
		}
		for _, member := range existingGroup {
			if member.Locale == locale {
				return ErrTranslationExists
			}
		}
		now := s.now()
		rev := english.RevisionNo
		draft := repo.NewArticle{
			Kind: source.Kind, Slug: firstNonEmpty(in.Slug, source.Slug), Locale: locale,
			TranslationGroupID: source.TranslationGroupID, CategoryID: source.CategoryID,
			Title: source.Title, Description: source.Description, BodyMD: "", Status: "draft",
			AuthorSlug: source.AuthorSlug, ReviewerSlug: source.ReviewerSlug,
			PrimaryQuestion: source.PrimaryQuestion, Cluster: source.Cluster, Pillar: source.Pillar,
			BriefRef: source.BriefRef, VerifiedAgainst: source.VerifiedAgainst,
			Keywords: source.Keywords, RelatedTo: source.RelatedTo, Roles: source.Roles,
			Segments: source.Segments, Citations: source.Citations, HeroMediaID: source.HeroMediaID,
			Noindex: source.Noindex, Extra: source.Extra, ActorID: in.Actor,
			ChangeNote:      "Created translation from " + source.Locale,
			SourceArticleID: &english.ID, SourceSyncedRevision: &rev, SourceSyncedAt: &now,
		}
		if source.Kind == "doc" && source.CategoryID != nil {
			catID, catErr := matchingCategory(ctx, tx, *source.CategoryID, locale)
			if catErr != nil {
				return catErr
			}
			draft.CategoryID = catID
		}
		out, err = repo.InsertArticle(ctx, tx, draft)
		return err
	})
	return out, err
}

func matchingCategory(ctx context.Context, tx pgx.Tx, sourceCategory uuid.UUID, locale string) (*uuid.UUID, error) {
	var group uuid.UUID
	var slug string
	if err := tx.QueryRow(ctx, `SELECT category_group_id, slug FROM marketing.content_categories WHERE id=$1`, sourceCategory).Scan(&group, &slug); err != nil {
		return nil, err
	}
	var id uuid.UUID
	err := tx.QueryRow(ctx, `SELECT id FROM marketing.content_categories WHERE category_group_id=$1 AND locale=$2`, group, locale).Scan(&id)
	if err == nil {
		return &id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	// Fall back to the source category so the doc still satisfies the category constraint;
	// authors localize the category slug from the metadata panel.
	return &sourceCategory, nil
}

func (s *Service) ListTranslations(ctx context.Context, id uuid.UUID) ([]repo.TranslationLink, error) {
	return repo.ListTranslations(ctx, s.Pool, id)
}

func (s *Service) MarkSynced(ctx context.Context, id, actor uuid.UUID) (*repo.Article, error) {
	var out *repo.Article
	err := s.transaction(ctx, func(tx pgx.Tx) error {
		a, err := repo.GetArticleByID(ctx, tx, id)
		if err != nil {
			return err
		}
		sourceID := a.SourceArticleID
		if sourceID == nil {
			src, srcErr := repo.DefaultLocaleSource(ctx, tx, a.TranslationGroupID)
			if srcErr != nil {
				return srcErr
			}
			sourceID = &src.ID
			a = src
		} else {
			src, srcErr := repo.GetArticleByID(ctx, tx, *sourceID)
			if srcErr != nil {
				return srcErr
			}
			a = src
		}
		out, err = repo.MarkTranslationSynced(ctx, tx, id, actor, a.RevisionNo, s.now())
		return err
	})
	return out, err
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func (s *Service) ListLocales(ctx context.Context, enabledOnly bool) ([]repo.Locale, bool, error) {
	items, err := repo.ListLocales(ctx, s.Pool, enabledOnly)
	if err != nil {
		return nil, false, err
	}
	on, err := repo.LocalesEnabled(ctx, s.Pool)
	return items, on, err
}

func (s *Service) UpsertLocale(ctx context.Context, l repo.Locale) (*repo.Locale, error) {
	var out *repo.Locale
	err := s.transaction(ctx, func(tx pgx.Tx) error {
		var err error
		out, err = repo.UpsertLocale(ctx, tx, l)
		return err
	})
	return out, err
}

func (s *Service) PatchLocale(ctx context.Context, code string, enabled *bool, sortOrder *int, label *string) (*repo.Locale, error) {
	var out *repo.Locale
	err := s.transaction(ctx, func(tx pgx.Tx) error {
		var err error
		out, err = repo.PatchLocale(ctx, tx, code, enabled, sortOrder, label)
		return err
	})
	return out, err
}

func (s *Service) ResolvePublicLocale(ctx context.Context, raw string) (string, error) {
	return repo.ResolvePublicLocale(ctx, s.Pool, raw)
}
