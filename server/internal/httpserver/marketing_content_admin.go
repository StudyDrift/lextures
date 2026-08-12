package httpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	mcrepo "github.com/lextures/lextures/server/internal/repos/marketingcontent"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	mcservice "github.com/lextures/lextures/server/internal/service/marketingcontent"
	validator "github.com/lextures/lextures/server/internal/service/marketingcontent/validate"
)

const (
	marketingView    = "global:app:marketing-content:view"
	marketingAuthor  = "global:app:marketing-content:author"
	marketingReview  = "global:app:marketing-content:review"
	marketingPublish = "global:app:marketing-content:publish"
	marketingAdmin   = "global:app:marketing-content:admin"
	marketingMaxBody = 1 << 20
)

func (d Deps) registerMarketingContentAdminRoutes(r chi.Router) {
	r.Get("/api/v1/admin/marketing/articles", d.handleMarketingArticlesList())
	r.Post("/api/v1/admin/marketing/articles", d.handleMarketingArticleCreate())
	r.Get("/api/v1/admin/marketing/articles/{id}", d.handleMarketingArticleGet())
	r.Patch("/api/v1/admin/marketing/articles/{id}", d.handleMarketingArticlePatch())
	r.Delete("/api/v1/admin/marketing/articles/{id}", d.handleMarketingArticleDelete())
	r.Post("/api/v1/admin/marketing/articles/{id}/transition", d.handleMarketingArticleTransition())
	r.Get("/api/v1/admin/marketing/articles/{id}/revisions", d.handleMarketingRevisionsList())
	r.Get("/api/v1/admin/marketing/articles/{id}/revisions/{no}", d.handleMarketingRevisionGet())
	r.Post("/api/v1/admin/marketing/articles/{id}/revisions/{no}/restore", d.handleMarketingRevisionRestore())
	r.Post("/api/v1/admin/marketing/articles/{id}/preview-token", d.handleMarketingPreviewToken())
	r.Post("/api/v1/admin/marketing/lint", d.handleMarketingLint())
	r.Get("/api/v1/admin/marketing/known-paths", d.handleMarketingKnownPathsList())
	r.Post("/api/v1/admin/marketing/known-paths", d.handleMarketingKnownPaths())
	d.registerMarketingTaxonomyRoutes(r)
	d.registerMarketingMediaRoutes(r)
	d.registerMarketingBuildRoutes(r)
	d.registerMarketingEditorialRoutes(r)
	d.registerMarketingRouteHintsRoutes(r)
	d.registerMarketingI18nRoutes(r)
}

func (d Deps) handleMarketingKnownPathsList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingView); !ok {
			return
		}
		paths, err := mcrepo.KnownPaths(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteInternal(w, r, "Failed to load known marketing paths.", err)
			return
		}
		items := make([]string, 0, len(paths))
		for path := range paths {
			items = append(items, path)
		}
		slices.Sort(items)
		writeJSON(w, 200, map[string]any{"items": items})
	}
}

func (d Deps) handleMarketingLint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingAuthor); !ok {
			return
		}
		var body struct {
			Kind     string             `json:"kind"`
			BodyMD   string             `json:"bodyMd"`
			Metadata validator.Metadata `json:"metadata"`
		}
		if !readMarketingJSON(w, r, &body) {
			return
		}
		writeJSON(w, 200, d.marketingService().Lint(r.Context(), body.Kind, body.BodyMD, body.Metadata))
	}
}

func (d Deps) handleMarketingKnownPaths() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingAdmin); !ok {
			return
		}
		var body struct {
			Paths []string `json:"paths"`
		}
		if !readMarketingJSON(w, r, &body) {
			return
		}
		if len(body.Paths) > 10000 {
			apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, "At most 10,000 paths may be submitted.")
			return
		}
		if err := d.marketingService().ReplaceStaticKnownPaths(r.Context(), body.Paths); err != nil {
			apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, err.Error())
			return
		}
		writeJSON(w, 200, map[string]int{"count": len(body.Paths)})
	}
}

func (d Deps) marketingAccess(w http.ResponseWriter, r *http.Request, permission string) (uuid.UUID, bool) {
	if !d.effectiveConfig().FFMarketingContent {
		apierr.WriteJSON(w, 404, apierr.CodeNotFound, "Marketing content is not enabled.")
		return uuid.Nil, false
	}
	actor, ok := d.meUserID(w, r)
	if !ok {
		return uuid.Nil, false
	}
	has, err := rbac.UserHasPermission(r.Context(), d.Pool, actor, permission)
	if err != nil {
		apierr.WriteInternal(w, r, "Failed to verify marketing content permission.", err)
		return uuid.Nil, false
	}
	if !has {
		apierr.WriteJSON(w, 403, apierr.CodeForbidden, "You do not have permission for this marketing content action.")
		return uuid.Nil, false
	}
	return actor, true
}

func (d Deps) marketingService() *mcservice.Service {
	return &mcservice.Service{Pool: d.Pool, PreviewSecret: []byte(d.effectiveConfig().JWTSecret)}
}
func marketingID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, "Invalid article id.")
		return uuid.Nil, false
	}
	return id, true
}
func readMarketingJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, marketingMaxBody)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, "Invalid request body: "+err.Error())
		return false
	}
	return true
}

func (d Deps) handleMarketingArticlesList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingView); !ok {
			return
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		items, next, err := mcrepo.ListArticles(r.Context(), d.Pool, mcrepo.ArticleFilter{Kind: r.URL.Query().Get("kind"), Status: r.URL.Query().Get("status"), Locale: r.URL.Query().Get("locale"), CategorySlug: r.URL.Query().Get("category"), AuthorSlug: r.URL.Query().Get("author"), Q: r.URL.Query().Get("q"), Sort: r.URL.Query().Get("sort"), Overdue: r.URL.Query().Get("overdue") == "true", Cursor: r.URL.Query().Get("cursor"), Limit: limit})
		if err != nil {
			apierr.WriteInternal(w, r, "Failed to list articles.", err)
			return
		}
		writeJSON(w, 200, map[string]any{"items": items, "nextCursor": next})
	}
}
func (d Deps) handleMarketingArticleGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingView); !ok {
			return
		}
		id, ok := marketingID(w, r)
		if !ok {
			return
		}
		a, err := mcrepo.GetArticleByID(r.Context(), d.Pool, id)
		if err != nil {
			writeMarketingError(w, r, err)
			return
		}
		latest, err := d.marketingPublishService().LatestForArticle(r.Context(), id)
		if err != nil {
			apierr.WriteInternal(w, r, "Failed to load publish status.", err)
			return
		}
		live := "draft"
		if a.Status == "scheduled" {
			live = "scheduled"
		} else if a.Status == "published" {
			live = "live"
		}
		if latest != nil {
			switch latest.Status {
			case "pending", "dispatched", "running":
				live = "publishing"
			case "failed", "timed_out":
				live = "publish_failed"
			}
		}
		settings, _ := d.marketingPublishService().Settings(r.Context())
		if a.Status == "published" && settings.Provider == "none" {
			live = "rebuild_not_configured"
		}
		a.LiveStatus = live
		a.LatestBuild = latest
		writeJSON(w, 200, a)
	}
}

type marketingArticleBody struct {
	Kind               string     `json:"kind"`
	Slug               string     `json:"slug"`
	Locale             string     `json:"locale"`
	CategoryID         *uuid.UUID `json:"categoryId"`
	Title              string     `json:"title"`
	Description        string     `json:"description"`
	BodyMD             string     `json:"bodyMd"`
	AuthorSlug         string     `json:"authorSlug"`
	ReviewerSlug       *string    `json:"reviewerSlug"`
	PrimaryQuestion    string     `json:"primaryQuestion"`
	Cluster            string     `json:"cluster"`
	Pillar             string     `json:"pillar"`
	BriefRef           string     `json:"briefRef"`
	VerifiedAgainst    string     `json:"verifiedAgainst"`
	ReviewDueOn        *time.Time `json:"reviewDueOn"`
	Keywords           []string   `json:"keywords"`
	RelatedTo          []string   `json:"relatedTo"`
	Roles              []string   `json:"roles"`
	Segments           []string   `json:"segments"`
	Citations          []string   `json:"citations"`
	HeroMediaID        *uuid.UUID `json:"heroMediaId"`
	Noindex            bool       `json:"noindex"`
	CanonicalOverride  *string    `json:"canonicalOverride"`
	ExpectedRevisionNo int        `json:"expectedRevisionNo"`
	ChangeNote         string     `json:"changeNote"`
}

func (b marketingArticleBody) input(actor uuid.UUID) mcrepo.NewArticle {
	return mcrepo.NewArticle{Kind: b.Kind, Slug: b.Slug, Locale: b.Locale, CategoryID: b.CategoryID, Title: b.Title, Description: b.Description, BodyMD: b.BodyMD, AuthorSlug: b.AuthorSlug, ReviewerSlug: b.ReviewerSlug, ReviewDueOn: b.ReviewDueOn, PrimaryQuestion: b.PrimaryQuestion, Cluster: b.Cluster, Pillar: b.Pillar, BriefRef: b.BriefRef, VerifiedAgainst: b.VerifiedAgainst, Keywords: b.Keywords, RelatedTo: b.RelatedTo, Roles: b.Roles, Segments: b.Segments, Citations: b.Citations, HeroMediaID: b.HeroMediaID, Noindex: b.Noindex, CanonicalOverride: b.CanonicalOverride, ActorID: actor, ChangeNote: b.ChangeNote}
}
func validateArticleBody(b marketingArticleBody) error {
	if b.Kind != "blog" && b.Kind != "doc" {
		return errors.New("kind must be blog or doc")
	}
	if strings.TrimSpace(b.Slug) == "" || strings.ContainsAny(b.Slug, " /_") {
		return errors.New("slug must be non-empty kebab-case")
	}
	if b.Kind == "doc" && b.CategoryID == nil {
		return errors.New("categoryId is required for doc articles")
	}
	if b.Title == "" || b.AuthorSlug == "" {
		return errors.New("title and authorSlug are required")
	}
	return nil
}
func (d Deps) handleMarketingArticleCreate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := d.marketingAccess(w, r, marketingAuthor)
		if !ok {
			return
		}
		var b marketingArticleBody
		if !readMarketingJSON(w, r, &b) {
			return
		}
		if err := validateArticleBody(b); err != nil {
			apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, err.Error())
			return
		}
		a, err := d.marketingService().Create(r.Context(), b.input(actor))
		if err != nil {
			writeMarketingError(w, r, err)
			return
		}
		d.recordMarketingAudit(r, actor, "create", a)
		writeJSON(w, 201, a)
	}
}

func mergeMarketingArticle(a *mcrepo.Article, b marketingArticleBody, raw map[string]json.RawMessage, actor uuid.UUID) mcrepo.NewArticle {
	in := mcArticleInput(a, actor)
	if _, ok := raw["kind"]; ok {
		in.Kind = b.Kind
	}
	if _, ok := raw["slug"]; ok {
		in.Slug = b.Slug
	}
	if _, ok := raw["categoryId"]; ok {
		in.CategoryID = b.CategoryID
	}
	if _, ok := raw["title"]; ok {
		in.Title = b.Title
	}
	if _, ok := raw["description"]; ok {
		in.Description = b.Description
	}
	if _, ok := raw["bodyMd"]; ok {
		in.BodyMD = b.BodyMD
	}
	if _, ok := raw["authorSlug"]; ok {
		in.AuthorSlug = b.AuthorSlug
	}
	if _, ok := raw["reviewerSlug"]; ok {
		in.ReviewerSlug = b.ReviewerSlug
	}
	if _, ok := raw["reviewDueOn"]; ok {
		in.ReviewDueOn = b.ReviewDueOn
	}
	if _, ok := raw["keywords"]; ok {
		in.Keywords = b.Keywords
	}
	if _, ok := raw["relatedTo"]; ok {
		in.RelatedTo = b.RelatedTo
	}
	if _, ok := raw["roles"]; ok {
		in.Roles = b.Roles
	}
	if _, ok := raw["segments"]; ok {
		in.Segments = b.Segments
	}
	if _, ok := raw["citations"]; ok {
		in.Citations = b.Citations
	}
	if _, ok := raw["noindex"]; ok {
		in.Noindex = b.Noindex
	}
	if _, ok := raw["canonicalOverride"]; ok {
		in.CanonicalOverride = b.CanonicalOverride
	}
	in.ChangeNote = b.ChangeNote
	return in
}
func mcArticleInput(a *mcrepo.Article, actor uuid.UUID) mcrepo.NewArticle {
	return mcrepo.NewArticle{Kind: a.Kind, Slug: a.Slug, Locale: a.Locale, TranslationGroupID: a.TranslationGroupID, CategoryID: a.CategoryID, Title: a.Title, Description: a.Description, BodyMD: a.BodyMD, Status: a.Status, AuthorSlug: a.AuthorSlug, ReviewerSlug: a.ReviewerSlug, PublishedAt: a.PublishedAt, FirstPublishedAt: a.FirstPublishedAt, ScheduledFor: a.ScheduledFor, ContentUpdatedAt: a.ContentUpdatedAt, ReviewedAt: a.ReviewedAt, ReviewDueOn: a.ReviewDueOn, PrimaryQuestion: a.PrimaryQuestion, Cluster: a.Cluster, Pillar: a.Pillar, BriefRef: a.BriefRef, VerifiedAgainst: a.VerifiedAgainst, Keywords: a.Keywords, RelatedTo: a.RelatedTo, Roles: a.Roles, Segments: a.Segments, Citations: a.Citations, HeroMediaID: a.HeroMediaID, QualityScore: a.QualityScore, QualityReport: a.QualityReport, Noindex: a.Noindex, CanonicalOverride: a.CanonicalOverride, Extra: a.Extra, ActorID: actor}
}
func (d Deps) handleMarketingArticlePatch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := d.marketingAccess(w, r, marketingAuthor)
		if !ok {
			return
		}
		id, ok := marketingID(w, r)
		if !ok {
			return
		}
		var raw map[string]json.RawMessage
		r.Body = http.MaxBytesReader(w, r.Body, marketingMaxBody)
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, "Invalid request body.")
			return
		}
		if _, bad := raw["status"]; bad {
			apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, "status can only be changed through the transition endpoint.")
			return
		}
		blob, _ := json.Marshal(raw)
		var b marketingArticleBody
		if err := json.Unmarshal(blob, &b); err != nil {
			apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, "Invalid request body.")
			return
		}
		if b.ExpectedRevisionNo < 1 {
			apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, "expectedRevisionNo is required.")
			return
		}
		old, err := mcrepo.GetArticleByID(r.Context(), d.Pool, id)
		if err != nil {
			writeMarketingError(w, r, err)
			return
		}
		a, redirect, err := d.marketingService().Update(r.Context(), mcrepo.ArticleUpdate{ID: id, ExpectedRevisionNo: b.ExpectedRevisionNo, Article: mergeMarketingArticle(old, b, raw, actor)})
		if err != nil {
			d.writeMarketingConflict(w, r, id, err)
			return
		}
		d.recordMarketingAudit(r, actor, "update", a)
		if redirect != nil {
			writeJSON(w, 200, map[string]any{"article": a, "createdRedirect": redirect})
			return
		}
		writeJSON(w, 200, a)
	}
}

type marketingTransitionBody struct {
	Action             mcservice.Action `json:"action"`
	ScheduledFor       *time.Time       `json:"scheduledFor"`
	Note               string           `json:"note"`
	ExpectedRevisionNo int              `json:"expectedRevisionNo"`
	LintOverride       bool             `json:"lintOverride"`
	ReviewerID         *uuid.UUID       `json:"reviewerId"`
}

func transitionPermission(a mcservice.Action) string {
	switch a {
	case mcservice.ActionSubmit, mcservice.ActionApprove, mcservice.ActionRequestChanges:
		return marketingReview
	case mcservice.ActionPublish, mcservice.ActionSchedule, mcservice.ActionUnpublish, mcservice.ActionArchive:
		return marketingPublish
	default:
		return marketingAuthor
	}
}
func (d Deps) handleMarketingArticleTransition() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.effectiveConfig().FFMarketingContent {
			apierr.WriteJSON(w, 404, apierr.CodeNotFound, "Marketing content is not enabled.")
			return
		}
		var b marketingTransitionBody
		if !readMarketingJSON(w, r, &b) {
			return
		}
		actor, ok := d.marketingAccess(w, r, transitionPermission(b.Action))
		if !ok {
			return
		}
		id, ok := marketingID(w, r)
		if !ok {
			return
		}
		a, err := d.marketingService().Transition(r.Context(), id, actor, mcservice.TransitionInput{Action: b.Action, ScheduledFor: b.ScheduledFor, Note: b.Note, ExpectedRevisionNo: b.ExpectedRevisionNo, LintOverride: b.LintOverride, ReviewerID: b.ReviewerID})
		if err != nil {
			d.writeMarketingConflict(w, r, id, err)
			return
		}
		d.recordMarketingAudit(r, actor, string(b.Action), a)
		writeJSON(w, 200, a)
	}
}

func writeMarketingError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		apierr.WriteJSON(w, 404, apierr.CodeNotFound, "Marketing content resource not found.")
	case errors.Is(err, mcrepo.ErrDuplicateSlug):
		apierr.WriteJSON(w, 409, "duplicate_slug", "An article or redirect already uses that path.")
	case errors.Is(err, mcservice.ErrInvalidTransition), errors.Is(err, mcservice.ErrScheduledInPast), errors.Is(err, mcservice.ErrReviewNoteTooShort), errors.Is(err, mcservice.ErrReviewerRequired), errors.Is(err, mcservice.ErrOverrideJustification):
		apierr.WriteJSON(w, 422, apierr.CodeUnprocessableEntity, err.Error())
	case errors.Is(err, mcservice.ErrLintBlocked):
		apierr.WriteJSON(w, 422, "content_validation_failed", "Publishing is blocked by content validation errors.")
	default:
		apierr.WriteInternal(w, r, "Marketing content operation failed.", err)
	}
}
func (d Deps) writeMarketingConflict(w http.ResponseWriter, r *http.Request, id uuid.UUID, err error) {
	if !errors.Is(err, mcrepo.ErrRevisionConflict) {
		writeMarketingError(w, r, err)
		return
	}
	a, getErr := mcrepo.GetArticleByID(r.Context(), d.Pool, id)
	if getErr != nil {
		writeMarketingError(w, r, getErr)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(409)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": "CONFLICT", "message": "A newer revision has already been saved."}, "currentRevisionNo": a.RevisionNo, "updatedBy": a.UpdatedBy, "updatedAt": a.UpdatedAt})
}

func (d Deps) handleMarketingRevisionsList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingView); !ok {
			return
		}
		id, ok := marketingID(w, r)
		if !ok {
			return
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		revs, err := mcrepo.ListRevisions(r.Context(), d.Pool, id, limit)
		if err != nil {
			writeMarketingError(w, r, err)
			return
		}
		type meta struct {
			RevisionNo  int        `json:"revisionNo"`
			ChangeNote  string     `json:"changeNote"`
			StatusAfter string     `json:"statusAfter"`
			ActorID     *uuid.UUID `json:"actorId"`
			CreatedAt   time.Time  `json:"createdAt"`
		}
		out := make([]meta, len(revs))
		for i, v := range revs {
			out[i] = meta{v.RevisionNo, v.ChangeNote, v.StatusAfter, v.ActorID, v.CreatedAt}
		}
		writeJSON(w, 200, map[string]any{"items": out})
	}
}
func (d Deps) handleMarketingRevisionGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingView); !ok {
			return
		}
		id, ok := marketingID(w, r)
		if !ok {
			return
		}
		no, _ := strconv.Atoi(chi.URLParam(r, "no"))
		rev, err := mcrepo.GetRevision(r.Context(), d.Pool, id, no)
		if err != nil {
			writeMarketingError(w, r, err)
			return
		}
		writeJSON(w, 200, rev)
	}
}
func (d Deps) handleMarketingRevisionRestore() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := d.marketingAccess(w, r, marketingAuthor)
		if !ok {
			return
		}
		id, ok := marketingID(w, r)
		if !ok {
			return
		}
		no, _ := strconv.Atoi(chi.URLParam(r, "no"))
		var b struct {
			ExpectedRevisionNo int    `json:"expectedRevisionNo"`
			Note               string `json:"note"`
		}
		if !readMarketingJSON(w, r, &b) {
			return
		}
		a, err := d.marketingService().Restore(r.Context(), id, actor, no, b.ExpectedRevisionNo, b.Note)
		if err != nil {
			d.writeMarketingConflict(w, r, id, err)
			return
		}
		d.recordMarketingAudit(r, actor, "restore", a)
		writeJSON(w, 200, a)
	}
}
func (d Deps) handleMarketingPreviewToken() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingView); !ok {
			return
		}
		id, ok := marketingID(w, r)
		if !ok {
			return
		}
		a, err := mcrepo.GetArticleByID(r.Context(), d.Pool, id)
		if err != nil {
			writeMarketingError(w, r, err)
			return
		}
		token, exp, err := d.marketingService().MintPreviewToken(id, a.RevisionNo, 30*time.Minute)
		if err != nil {
			writeMarketingError(w, r, err)
			return
		}
		writeJSON(w, 200, map[string]any{"token": token, "expiresAt": exp, "url": fmt.Sprintf("/preview/%s?token=%s", id, token)})
	}
}
func (d Deps) handleMarketingArticleDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := d.marketingAccess(w, r, marketingAuthor)
		if !ok {
			return
		}
		id, ok := marketingID(w, r)
		if !ok {
			return
		}
		a, err := mcrepo.GetArticleByID(r.Context(), d.Pool, id)
		if err != nil {
			writeMarketingError(w, r, err)
			return
		}
		if a.FirstPublishedAt != nil {
			has, e := rbac.UserHasPermission(r.Context(), d.Pool, actor, marketingPublish)
			if e != nil || !has {
				apierr.WriteJSON(w, 403, apierr.CodeForbidden, "Published articles require publish permission to delete.")
				return
			}
		}
		var b struct {
			RedirectTo string `json:"redirectTo"`
		}
		if r.Body != nil && r.ContentLength != 0 && !readMarketingJSON(w, r, &b) {
			return
		}
		if err := d.marketingService().Delete(r.Context(), id, actor, b.RedirectTo); err != nil {
			if strings.Contains(err.Error(), "requires redirectTo") {
				apierr.WriteJSON(w, 422, apierr.CodeUnprocessableEntity, err.Error())
				return
			}
			writeMarketingError(w, r, err)
			return
		}
		d.recordMarketingAudit(r, actor, "delete", a)
		w.WriteHeader(204)
	}
}
