package marketingcontent

import (
	"context"
	"strings"

	repo "github.com/lextures/lextures/server/internal/repos/marketingcontent"
)

// ContextualArticle is the response shape for the in-app help widget's
// contextual-articles lookup (MC.13 FR-3/FR-4).
type ContextualArticle struct {
	Title        string  `json:"title"`
	URL          string  `json:"url"`
	Slug         string  `json:"slug"`
	CategorySlug *string `json:"categorySlug"`
	Summary      string  `json:"summary"`
	Tier         string  `json:"tier"`
	Locale       string  `json:"locale,omitempty"`
	IsFallback   bool    `json:"isFallback,omitempty"`
}

const helpCenterBase = "https://lextures.com"

// ContextualArticles resolves help articles for an app route using the tiered
// precedence in FR-4: explicit route hints, then related_to paths, then category
// platform-path prefixes, then full-text search on route-derived keywords. The
// first tier that yields any article visible to the viewer's roles wins. A nil,
// nil result means no tier matched and callers should fall back to a static list.
func (s *Service) ContextualArticles(ctx context.Context, route string, roles []string, locale string, limit int) ([]ContextualArticle, error) {
	if limit <= 0 || limit > 8 {
		limit = 5
	}
	route = strings.TrimSpace(route)
	if route == "" {
		return nil, nil
	}
	if locale == "" {
		locale = repo.DefaultLocale
	}
	keywords := routeKeywords(route)
	tiers := []struct {
		tier string
		load func() ([]repo.PublicArticle, error)
	}{
		{"hint", func() ([]repo.PublicArticle, error) { return repo.ArticlesByRouteHint(ctx, s.Pool, route, limit*2) }},
		{"related", func() ([]repo.PublicArticle, error) { return repo.ArticlesByRelatedRoute(ctx, s.Pool, route, limit*2) }},
		{"category", func() ([]repo.PublicArticle, error) { return repo.ArticlesByCategoryPath(ctx, s.Pool, route, limit*2) }},
		{"search", func() ([]repo.PublicArticle, error) {
			if keywords == "" {
				return nil, nil
			}
			arts, _, err := repo.ListPublishedArticles(ctx, s.Pool, repo.PublicArticleFilter{Q: keywords, Limit: limit * 2})
			return arts, err
		}},
	}
	for _, t := range tiers {
		arts, err := t.load()
		if err != nil {
			return nil, err
		}
		if filtered := preferLocale(filterByRole(arts, roles), locale); len(filtered) > 0 {
			return toContextual(filtered, t.tier, limit), nil
		}
	}
	return nil, nil
}

// routeKeywords turns an app route path into search terms, e.g.
// "/courses/123/assignments" -> "courses assignments" (numeric/ID-shaped
// segments are dropped since they never match article text).
func routeKeywords(route string) string {
	route = strings.Trim(route, "/")
	if route == "" {
		return ""
	}
	segments := strings.FieldsFunc(route, func(r rune) bool { return r == '/' || r == '-' || r == '_' })
	kept := make([]string, 0, len(segments))
	for _, seg := range segments {
		if seg == "" || looksLikeID(seg) {
			continue
		}
		kept = append(kept, seg)
	}
	return strings.Join(kept, " ")
}

func looksLikeID(seg string) bool {
	digits := 0
	for _, r := range seg {
		if r >= '0' && r <= '9' {
			digits++
		}
	}
	return digits > 0 && (digits == len(seg) || len(seg) >= 20)
}

// filterByRole keeps articles with no role restriction, or whose declared
// roles intersect the viewer's effective role set (FR-5).
func filterByRole(arts []repo.PublicArticle, roles []string) []repo.PublicArticle {
	out := make([]repo.PublicArticle, 0, len(arts))
	for _, a := range arts {
		if len(a.Roles) == 0 {
			out = append(out, a)
			continue
		}
		for _, want := range a.Roles {
			if containsFold(roles, want) {
				out = append(out, a)
				break
			}
		}
	}
	return out
}

func containsFold(list []string, v string) bool {
	for _, s := range list {
		if strings.EqualFold(s, v) {
			return true
		}
	}
	return false
}

func toContextual(arts []repo.PublicArticle, tier string, limit int) []ContextualArticle {
	if len(arts) > limit {
		arts = arts[:limit]
	}
	out := make([]ContextualArticle, 0, len(arts))
	for _, a := range arts {
		out = append(out, ContextualArticle{
			Title:        a.Title,
			URL:          helpCenterBase + a.Path,
			Slug:         a.Slug,
			CategorySlug: a.CategorySlug,
			Summary:      a.Description,
			Tier:         tier,
			Locale:       a.Locale,
			IsFallback:   a.IsFallback,
		})
	}
	return out
}

func preferLocale(arts []repo.PublicArticle, locale string) []repo.PublicArticle {
	if locale == "" || locale == repo.DefaultLocale {
		matched := make([]repo.PublicArticle, 0, len(arts))
		for _, a := range arts {
			if a.Locale == repo.DefaultLocale || a.Locale == "" {
				matched = append(matched, a)
			}
		}
		if len(matched) > 0 {
			return matched
		}
		return arts
	}
	matched := make([]repo.PublicArticle, 0)
	fallback := make([]repo.PublicArticle, 0)
	for _, a := range arts {
		switch a.Locale {
		case locale:
			matched = append(matched, a)
		case repo.DefaultLocale, "":
			a.IsFallback = true
			fallback = append(fallback, a)
		}
	}
	if len(matched) > 0 {
		return matched
	}
	return fallback
}
