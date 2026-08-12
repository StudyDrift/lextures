package httpserver

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/objectcache"
	mcrepo "github.com/lextures/lextures/server/internal/repos/marketingcontent"
)

type publicContentAuthor struct {
	Slug       string   `json:"slug"`
	Name       string   `json:"name"`
	JobTitle   string   `json:"jobTitle"`
	Bio        string   `json:"bio,omitempty"`
	KnowsAbout []string `json:"knowsAbout,omitempty"`
}
type publicContentArticle struct {
	Path               string               `json:"path"`
	Kind               string               `json:"kind"`
	Slug               string               `json:"slug"`
	Locale             string               `json:"locale"`
	TranslationGroupID string               `json:"translationGroupId"`
	CategorySlug       *string              `json:"categorySlug"`
	CategoryTitle      *string              `json:"categoryTitle"`
	Title              string               `json:"title"`
	Description        string               `json:"description"`
	BodyMD             string               `json:"bodyMd,omitempty"`
	Author             *publicContentAuthor `json:"author"`
	Reviewer           *publicContentAuthor `json:"reviewer"`
	PublishedAt        *time.Time           `json:"publishedAt"`
	UpdatedAt          time.Time            `json:"updatedAt"`
	ContentUpdatedAt   *time.Time           `json:"contentUpdatedAt"`
	ReviewedAt         *time.Time           `json:"reviewedAt"`
	PrimaryQuestion    string               `json:"primaryQuestion"`
	Cluster            string               `json:"cluster"`
	Pillar             string               `json:"pillar"`
	Keywords           []string             `json:"keywords"`
	RelatedTo          []string             `json:"relatedTo"`
	Roles              []string             `json:"roles"`
	Segments           []string             `json:"segments"`
	Citations          []string             `json:"citations"`
	Tags               []string             `json:"tags"`
	HeroImageURL       *string              `json:"heroImageUrl"`
	Noindex            bool                 `json:"noindex"`
	CanonicalOverride  *string              `json:"canonicalOverride"`
	ContentHash        string               `json:"contentHash,omitempty"`
	Media              []publicContentMedia `json:"media,omitempty"`
}
type publicContentMedia struct {
	ID         uuid.UUID               `json:"id"`
	Alt        string                  `json:"alt"`
	Decorative bool                    `json:"decorative"`
	Width      *int                    `json:"width"`
	Height     *int                    `json:"height"`
	Checksum   string                  `json:"checksum"`
	Renditions []mcrepo.MediaRendition `json:"renditions"`
}
type publicContentCategory struct {
	Slug         string `json:"slug"`
	Locale       string `json:"locale"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	SortOrder    int    `json:"sortOrder"`
	PlatformPath string `json:"platformPath"`
	ArticleCount int    `json:"articleCount"`
}
type publicContentRedirect struct {
	From       string `json:"from"`
	To         string `json:"to"`
	StatusCode int    `json:"statusCode"`
}
type publicContentIndex struct {
	GeneratedAt time.Time               `json:"generatedAt"`
	Articles    []publicContentArticle  `json:"articles"`
	Categories  []publicContentCategory `json:"categories"`
	Authors     []publicContentAuthor   `json:"authors"`
	Redirects   []publicContentRedirect `json:"redirects"`
}

func (d Deps) registerPublicMarketingContentRoutes(r chi.Router) {
	r.Get("/api/v1/public/content/index", d.handlePublicContentIndex())
	r.Get("/api/v1/public/content/articles", d.handlePublicContentArticles())
	r.Get("/api/v1/public/content/articles/blog/{slug}", d.handlePublicContentDetail("blog"))
	r.Get("/api/v1/public/content/articles/docs/{category}/{slug}", d.handlePublicContentDetail("doc"))
	r.Get("/api/v1/public/content/categories", d.handlePublicContentCategories())
	r.Get("/api/v1/public/content/authors", d.handlePublicContentAuthors())
	r.Get("/api/v1/public/content/authors/{slug}", d.handlePublicContentAuthor())
	r.Get("/api/v1/public/content/redirects", d.handlePublicContentRedirects())
	r.Get("/api/v1/public/content/search", d.handlePublicContentSearch())
	r.Get("/api/v1/public/content/media/{id}/{file}", d.handlePublicMarketingMedia())
}
func (d Deps) publicContentOff(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFMarketingContent || d.Pool == nil {
		apierr.WriteJSON(w, 404, apierr.CodeNotFound, "Content not found.")
		return true
	}
	return false
}
func publicContentHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "public, max-age=60, stale-while-revalidate=300")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	appendVaryAcceptEncoding(w.Header())
}
func publicAuthor(a *mcrepo.Author) *publicContentAuthor {
	if a == nil {
		return nil
	}
	return &publicContentAuthor{Slug: a.Slug, Name: a.Name, JobTitle: a.JobTitle, Bio: a.Bio, KnowsAbout: a.KnowsAbout}
}
func mapPublicArticle(a mcrepo.PublicArticle, body, hash bool) publicContentArticle {
	p := publicContentArticle{Path: a.Path, Kind: a.Kind, Slug: a.Slug, Locale: a.Locale, TranslationGroupID: a.TranslationGroupID.String(), CategorySlug: a.CategorySlug, CategoryTitle: a.CategoryTitle, Title: a.Title, Description: a.Description, Author: publicAuthor(a.Author), Reviewer: publicAuthor(a.Reviewer), PublishedAt: a.PublishedAt, UpdatedAt: a.UpdatedAt, ContentUpdatedAt: a.ContentUpdatedAt, ReviewedAt: a.ReviewedAt, PrimaryQuestion: a.PrimaryQuestion, Cluster: a.Cluster, Pillar: a.Pillar, Keywords: a.Keywords, RelatedTo: a.RelatedTo, Roles: a.Roles, Segments: a.Segments, Citations: a.Citations, Tags: a.Tags, Noindex: a.Noindex, CanonicalOverride: a.CanonicalOverride}
	if body {
		p.BodyMD = a.BodyMD
	}
	if hash {
		sum := sha256.Sum256([]byte(strings.Join([]string{
			a.BodyMD, a.Kind, a.Slug, a.Locale, a.Path, a.Title, a.Description,
			a.TranslationGroupID.String(), valueString(a.CategorySlug), valueString(a.CategoryTitle),
			publicAuthorDigest(a.Author), publicAuthorDigest(a.Reviewer), instantDigest(a.PublishedAt),
			instantDigest(a.ContentUpdatedAt), instantDigest(a.ReviewedAt), a.PrimaryQuestion, a.Cluster,
			a.Pillar, strings.Join(a.Keywords, "\x00"), strings.Join(a.RelatedTo, "\x00"),
			strings.Join(a.Roles, "\x00"), strings.Join(a.Segments, "\x00"), strings.Join(a.Citations, "\x00"),
			strings.Join(a.Tags, "\x00"), uuidDigest(a.HeroMediaID), strconv.FormatBool(a.Noindex),
			valueString(a.CanonicalOverride),
		}, "\x1f")))
		p.ContentHash = hex.EncodeToString(sum[:])[:16]
	}
	return p
}
func publicAuthorDigest(a *mcrepo.Author) string {
	if a == nil {
		return ""
	}
	return strings.Join([]string{a.Slug, a.Name, a.JobTitle, a.Bio, strings.Join(a.KnowsAbout, "\x00")}, "\x00")
}
func instantDigest(v *time.Time) string {
	if v == nil {
		return ""
	}
	return v.UTC().Format(time.RFC3339Nano)
}
func uuidDigest(v *uuid.UUID) string {
	if v == nil {
		return ""
	}
	return v.String()
}
func valueString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
func (d Deps) contentCached(w http.ResponseWriter, r *http.Request, dest any, load func() error) {
	version, e := mcrepo.PublicContentVersion(r.Context(), d.Pool)
	if e != nil {
		apierr.WriteInternal(w, r, "Failed to load content.", e)
		return
	}
	key := objectcache.MarketingContentKey(r.URL.Path, r.URL.RawQuery, version)
	if c := d.objectCache(); c != nil {
		if hit, _ := c.GetJSON(r.Context(), key, objectcache.ResourceMarketingContent, dest); hit {
			publicContentHeaders(w)
			writeJSONWithETag(w, r, 200, dest)
			return
		}
	}
	if e = load(); e != nil {
		apierr.WriteInternal(w, r, "Failed to load content.", e)
		return
	}
	if c := d.objectCache(); c != nil {
		_ = c.SetJSON(r.Context(), key, dest, time.Minute)
	}
	publicContentHeaders(w)
	writeJSONWithETag(w, r, 200, dest)
}
func (d Deps) handlePublicContentIndex() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.publicContentOff(w) {
			return
		}
		res := publicContentIndex{Articles: []publicContentArticle{}, Categories: []publicContentCategory{}, Authors: []publicContentAuthor{}, Redirects: []publicContentRedirect{}}
		d.contentCached(w, r, &res, func() error {
			var latest time.Time
			cursor := ""
			for {
				as, next, e := mcrepo.ListPublishedArticles(r.Context(), d.Pool, mcrepo.PublicArticleFilter{Limit: 200, Cursor: cursor})
				if e != nil {
					return e
				}
				for _, a := range as {
					res.Articles = append(res.Articles, mapPublicArticle(a, false, true))
					if a.UpdatedAt.After(latest) {
						latest = a.UpdatedAt
					}
				}
				if next == "" {
					break
				}
				cursor = next
			}
			cs, e := mcrepo.ListPublicCategories(r.Context(), d.Pool, "")
			if e != nil {
				return e
			}
			for _, c := range cs {
				res.Categories = append(res.Categories, publicContentCategory{c.Slug, c.Locale, c.Title, c.Description, c.SortOrder, c.PlatformPath, c.ArticleCount})
			}
			aus, e := mcrepo.ListPublicAuthors(r.Context(), d.Pool)
			if e != nil {
				return e
			}
			for i := range aus {
				res.Authors = append(res.Authors, *publicAuthor(&aus[i]))
			}
			redirects, e := mcrepo.ListRedirects(r.Context(), d.Pool)
			for _, redirect := range redirects {
				res.Redirects = append(res.Redirects, publicContentRedirect{From: redirect.FromPath, To: redirect.ToPath, StatusCode: redirect.StatusCode})
			}
			if latest.IsZero() {
				latest = time.Unix(0, 0).UTC()
			}
			res.GeneratedAt = latest
			return e
		})
	}
}
func parseLimit(r *http.Request, def, max int) (int, error) {
	s := r.URL.Query().Get("limit")
	if s == "" {
		return def, nil
	}
	n, e := strconv.Atoi(s)
	if e != nil || n < 1 {
		return 0, errors.New("invalid limit")
	}
	if n > max {
		n = max
	}
	return n, nil
}
func (d Deps) handlePublicContentArticles() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.publicContentOff(w) {
			return
		}
		limit, e := parseLimit(r, 50, 200)
		if e != nil {
			apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, "Invalid limit.")
			return
		}
		q := r.URL.Query()
		f := mcrepo.PublicArticleFilter{Kind: q.Get("kind"), Locale: q.Get("locale"), CategorySlug: q.Get("category"), AuthorSlug: q.Get("author"), Tag: q.Get("tag"), Q: q.Get("q"), Cursor: q.Get("cursor"), Limit: limit}
		res := struct {
			Articles   []publicContentArticle `json:"articles"`
			NextCursor string                 `json:"nextCursor"`
		}{Articles: []publicContentArticle{}}
		d.contentCached(w, r, &res, func() error {
			as, next, e := mcrepo.ListPublishedArticles(r.Context(), d.Pool, f)
			if e != nil {
				return e
			}
			for _, a := range as {
				res.Articles = append(res.Articles, mapPublicArticle(a, false, false))
			}
			res.NextCursor = next
			return nil
		})
	}
}
func (d Deps) handlePublicContentDetail(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.publicContentOff(w) {
			return
		}
		path := "/blog/" + chi.URLParam(r, "slug")
		if kind == "doc" {
			path = "/docs/" + chi.URLParam(r, "category") + "/" + chi.URLParam(r, "slug")
		}
		token := r.URL.Query().Get("preview_token")
		var a *mcrepo.PublicArticle
		var e error
		if token == "" {
			a, e = mcrepo.GetPublishedArticleByPath(r.Context(), d.Pool, path)
		} else {
			a, e = mcrepo.GetPreviewArticleByPath(r.Context(), d.Pool, path)
			if e == nil {
				e = d.marketingService().VerifyPreviewToken(token, a.ID, a.RevisionNo)
				if e != nil {
					code := "preview_token_invalid"
					if strings.Contains(e.Error(), "expired") {
						code = "preview_token_expired"
					}
					apierr.WriteJSON(w, 403, code, "Preview token is invalid or expired.")
					return
				}
			}
		}
		if errors.Is(e, pgx.ErrNoRows) {
			apierr.WriteJSON(w, 404, apierr.CodeNotFound, "Article not found.")
			return
		}
		if e != nil {
			apierr.WriteInternal(w, r, "Failed to load article.", e)
			return
		}
		res := mapPublicArticle(*a, true, false)
		media, mediaErr := mcrepo.ListArticleMedia(r.Context(), d.Pool, a.ID)
		if mediaErr != nil {
			apierr.WriteInternal(w, r, "Failed to load article media.", mediaErr)
			return
		}
		res.Media = make([]publicContentMedia, 0, len(media))
		for _, m := range media {
			res.Media = append(res.Media, publicContentMedia{ID: m.ID, Alt: m.AltText, Decorative: m.Decorative, Width: m.Width, Height: m.Height, Checksum: m.Checksum, Renditions: m.Renditions})
		}
		w.Header().Set("Content-Language", a.Locale)
		if token != "" {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("X-Robots-Tag", "noindex, nofollow")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			writeJSONWithETag(w, r, 200, res)
			return
		}
		d.contentCached(w, r, &res, func() error { return nil })
	}
}
func (d Deps) handlePublicContentCategories() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.publicContentOff(w) {
			return
		}
		res := struct {
			Categories []publicContentCategory `json:"categories"`
		}{[]publicContentCategory{}}
		d.contentCached(w, r, &res, func() error {
			cs, e := mcrepo.ListPublicCategories(r.Context(), d.Pool, r.URL.Query().Get("locale"))
			for _, c := range cs {
				res.Categories = append(res.Categories, publicContentCategory{c.Slug, c.Locale, c.Title, c.Description, c.SortOrder, c.PlatformPath, c.ArticleCount})
			}
			return e
		})
	}
}
func (d Deps) handlePublicContentAuthors() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.publicContentOff(w) {
			return
		}
		res := struct {
			Authors []publicContentAuthor `json:"authors"`
		}{[]publicContentAuthor{}}
		d.contentCached(w, r, &res, func() error {
			as, e := mcrepo.ListPublicAuthors(r.Context(), d.Pool)
			for i := range as {
				res.Authors = append(res.Authors, *publicAuthor(&as[i]))
			}
			return e
		})
	}
}
func (d Deps) handlePublicContentAuthor() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.publicContentOff(w) {
			return
		}
		a, e := mcrepo.GetPublicAuthor(r.Context(), d.Pool, chi.URLParam(r, "slug"))
		if errors.Is(e, pgx.ErrNoRows) {
			apierr.WriteJSON(w, 404, apierr.CodeNotFound, "Author not found.")
			return
		}
		if e != nil {
			apierr.WriteInternal(w, r, "Failed to load author.", e)
			return
		}
		res := publicAuthor(a)
		d.contentCached(w, r, res, func() error { return nil })
	}
}
func (d Deps) handlePublicContentRedirects() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.publicContentOff(w) {
			return
		}
		res := struct {
			Redirects []publicContentRedirect `json:"redirects"`
		}{[]publicContentRedirect{}}
		d.contentCached(w, r, &res, func() error {
			rows, e := mcrepo.ListRedirects(r.Context(), d.Pool)
			for _, redirect := range rows {
				res.Redirects = append(res.Redirects, publicContentRedirect{From: redirect.FromPath, To: redirect.ToPath, StatusCode: redirect.StatusCode})
			}
			return e
		})
	}
}
func (d Deps) handlePublicContentSearch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.publicContentOff(w) {
			return
		}
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if query == "" {
			apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, "q is required.")
			return
		}
		limit, e := parseLimit(r, 10, 50)
		if e != nil {
			apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, "Invalid limit.")
			return
		}
		res := struct {
			Results []mcrepo.SearchResult `json:"results"`
		}{[]mcrepo.SearchResult{}}
		d.contentCached(w, r, &res, func() error {
			var e error
			res.Results, e = mcrepo.SearchPublished(r.Context(), d.Pool, query, r.URL.Query().Get("kind"), limit)
			return e
		})
	}
}
