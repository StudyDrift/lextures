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
// platform-path prefixes, then full-text search on route-derived keywords, then a
// role-filtered published-doc fallback. The first tier that yields any article
// visible to the viewer's roles wins.
func (s *Service) ContextualArticles(ctx context.Context, route string, roles []string, locale string, limit int) ([]ContextualArticle, error) {
	if limit <= 0 || limit > 8 {
		limit = 5
	}
	route = strings.TrimSpace(route)
	roles = NormalizeViewerRoles(roles)
	if locale == "" {
		locale = repo.DefaultLocale
	}

	pick := func(arts []repo.PublicArticle, tier string) []ContextualArticle {
		filtered := preferLocale(filterByRole(arts, roles), locale)
		if len(filtered) == 0 {
			return nil
		}
		return toContextual(filtered, tier, limit)
	}

	if route != "" {
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
			if out := pick(arts, t.tier); len(out) > 0 {
				return out, nil
			}
		}
	}

	// FR-4 fallback tier: recent published docs visible to this viewer (also covers
	// empty/unknown routes so the widget is never an empty panel when content exists).
	arts, _, err := repo.ListPublishedArticles(ctx, s.Pool, repo.PublicArticleFilter{
		Kind:   "doc",
		Locale: locale,
		Limit:  limit * 2,
	})
	if err != nil {
		return nil, err
	}
	if out := pick(arts, "fallback"); len(out) > 0 {
		return out, nil
	}
	return []ContextualArticle{}, nil
}

// NormalizeViewerRoles maps LMS enrollment/app role names onto the vocabulary used
// in marketing content frontmatter (learner/instructor/admin/parent) and ensures
// authenticated viewers without enrollments still see learner-facing help (FR-5).
func NormalizeViewerRoles(roles []string) []string {
	seen := make(map[string]struct{}, len(roles)+4)
	out := make([]string, 0, len(roles)+4)
	add := func(role string) {
		role = strings.TrimSpace(role)
		if role == "" {
			return
		}
		key := strings.ToLower(role)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, role)
	}
	for _, role := range roles {
		add(role)
		switch strings.ToLower(strings.TrimSpace(role)) {
		case "student":
			add("learner")
		case "teacher", "ta":
			add("instructor")
		case "global admin", "org admin", "administrator", "platform admin":
			add("admin")
		}
	}
	if len(out) == 0 {
		add("learner")
	}
	return out
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
