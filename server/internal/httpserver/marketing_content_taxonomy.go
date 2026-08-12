package httpserver

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	repo "github.com/lextures/lextures/server/internal/repos/marketingcontent"
	"github.com/lextures/lextures/server/internal/service/adminaudit"
	"net/http"
)

func (d Deps) registerMarketingTaxonomyRoutes(r chi.Router) {
	r.Get("/api/v1/admin/marketing/categories", d.handleMarketingCategories())
	r.Post("/api/v1/admin/marketing/categories", d.handleMarketingCategorySave())
	r.Patch("/api/v1/admin/marketing/categories/{id}", d.handleMarketingCategorySave())
	r.Delete("/api/v1/admin/marketing/categories/{id}", d.handleMarketingCategoryDelete())
	r.Get("/api/v1/admin/marketing/authors", d.handleMarketingAuthors())
	r.Post("/api/v1/admin/marketing/authors", d.handleMarketingAuthorSave())
	r.Patch("/api/v1/admin/marketing/authors/{slug}", d.handleMarketingAuthorSave())
	r.Get("/api/v1/admin/marketing/tags", d.handleMarketingTags())
	r.Post("/api/v1/admin/marketing/tags", d.handleMarketingTagSave())
	r.Delete("/api/v1/admin/marketing/tags/{id}", d.handleMarketingTagDelete())
	r.Get("/api/v1/admin/marketing/redirects", d.handleMarketingRedirects())
	r.Post("/api/v1/admin/marketing/redirects", d.handleMarketingRedirectSave())
	r.Delete("/api/v1/admin/marketing/redirects/{id}", d.handleMarketingRedirectDelete())
}
func routeUUID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, e := uuid.Parse(chi.URLParam(r, "id"))
	if e != nil {
		apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, "Invalid resource id.")
		return uuid.Nil, false
	}
	return id, true
}
func (d Deps) handleMarketingCategories() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingView); !ok {
			return
		}
		v, e := repo.ListCategories(r.Context(), d.Pool, r.URL.Query().Get("locale"))
		if e != nil {
			writeMarketingError(w, r, e)
			return
		}
		writeJSON(w, 200, v)
	}
}
func (d Deps) handleMarketingCategorySave() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := d.marketingAccess(w, r, marketingAdmin)
		if !ok {
			return
		}
		var v repo.Category
		if !readMarketingJSON(w, r, &v) {
			return
		}
		if p := chi.URLParam(r, "id"); p != "" {
			id, e := uuid.Parse(p)
			if e != nil {
				apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, "Invalid category id.")
				return
			}
			v.ID = id
		}
		out, e := d.marketingService().SaveCategory(r.Context(), v, actor)
		if e != nil {
			writeMarketingError(w, r, e)
			return
		}
		d.recordMarketingTaxonomyAudit(r, actor, "category_write", out.ID)
		writeJSON(w, 200, out)
	}
}
func (d Deps) handleMarketingCategoryDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := d.marketingAccess(w, r, marketingAdmin)
		if !ok {
			return
		}
		id, ok := routeUUID(w, r)
		if !ok {
			return
		}
		if e := d.marketingService().DeleteCategory(r.Context(), id); e != nil {
			writeMarketingError(w, r, e)
			return
		}
		d.recordMarketingTaxonomyAudit(r, actor, "category_delete", id)
		w.WriteHeader(204)
	}
}
func (d Deps) handleMarketingAuthors() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingView); !ok {
			return
		}
		v, e := repo.ListAuthors(r.Context(), d.Pool)
		if e != nil {
			writeMarketingError(w, r, e)
			return
		}
		writeJSON(w, 200, v)
	}
}
func (d Deps) handleMarketingAuthorSave() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := d.marketingAccess(w, r, marketingAdmin)
		if !ok {
			return
		}
		var v repo.Author
		if !readMarketingJSON(w, r, &v) {
			return
		}
		if slug := chi.URLParam(r, "slug"); slug != "" {
			v.Slug = slug
		}
		out, e := d.marketingService().SaveAuthor(r.Context(), v, actor)
		if e != nil {
			writeMarketingError(w, r, e)
			return
		}
		d.recordMarketingTaxonomyAudit(r, actor, "author_write", uuid.Nil)
		writeJSON(w, 200, out)
	}
}
func (d Deps) handleMarketingTags() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingView); !ok {
			return
		}
		v, e := repo.ListTags(r.Context(), d.Pool)
		if e != nil {
			writeMarketingError(w, r, e)
			return
		}
		writeJSON(w, 200, v)
	}
}
func (d Deps) handleMarketingTagSave() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := d.marketingAccess(w, r, marketingAdmin)
		if !ok {
			return
		}
		var v repo.Tag
		if !readMarketingJSON(w, r, &v) {
			return
		}
		out, e := d.marketingService().SaveTag(r.Context(), v, actor)
		if e != nil {
			writeMarketingError(w, r, e)
			return
		}
		d.recordMarketingTaxonomyAudit(r, actor, "tag_write", out.ID)
		writeJSON(w, 200, out)
	}
}
func (d Deps) handleMarketingTagDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := d.marketingAccess(w, r, marketingAdmin)
		if !ok {
			return
		}
		id, ok := routeUUID(w, r)
		if !ok {
			return
		}
		if e := d.marketingService().DeleteTag(r.Context(), id); e != nil {
			writeMarketingError(w, r, e)
			return
		}
		d.recordMarketingTaxonomyAudit(r, actor, "tag_delete", id)
		w.WriteHeader(204)
	}
}
func (d Deps) handleMarketingRedirects() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingView); !ok {
			return
		}
		v, e := repo.ListRedirects(r.Context(), d.Pool)
		if e != nil {
			writeMarketingError(w, r, e)
			return
		}
		writeJSON(w, 200, v)
	}
}
func (d Deps) handleMarketingRedirectSave() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := d.marketingAccess(w, r, marketingAdmin)
		if !ok {
			return
		}
		var v repo.Redirect
		if !readMarketingJSON(w, r, &v) {
			return
		}
		if v.StatusCode == 0 {
			v.StatusCode = 301
		}
		if v.Source == "" {
			v.Source = "manual"
		}
		out, e := d.marketingService().SaveRedirect(r.Context(), v, actor)
		if e != nil {
			writeMarketingError(w, r, e)
			return
		}
		d.recordMarketingTaxonomyAudit(r, actor, "redirect_write", out.ID)
		writeJSON(w, 201, out)
	}
}
func (d Deps) handleMarketingRedirectDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := d.marketingAccess(w, r, marketingAdmin)
		if !ok {
			return
		}
		id, ok := routeUUID(w, r)
		if !ok {
			return
		}
		if e := d.marketingService().DeleteRedirect(r.Context(), id); e != nil {
			writeMarketingError(w, r, e)
			return
		}
		d.recordMarketingTaxonomyAudit(r, actor, "redirect_delete", id)
		w.WriteHeader(204)
	}
}
func (d Deps) recordMarketingAudit(r *http.Request, actor uuid.UUID, action string, a *repo.Article) {
	target := "marketing_article"
	before, _ := json.Marshal(map[string]any{"action": action, "path": a.Path, "revisionNo": a.RevisionNo})
	_, _ = adminaudit.Record(r.Context(), d.Pool, adminaudit.RecordParams{EventType: adminaudit.EventMarketingContent, ActorID: actor, TargetType: &target, TargetID: &a.ID, AfterValue: before})
}
func (d Deps) recordMarketingTaxonomyAudit(r *http.Request, actor uuid.UUID, action string, id uuid.UUID) {
	target := "marketing_taxonomy"
	after, _ := json.Marshal(map[string]string{"action": action})
	var targetID *uuid.UUID
	if id != uuid.Nil {
		targetID = &id
	}
	_, _ = adminaudit.Record(r.Context(), d.Pool, adminaudit.RecordParams{EventType: adminaudit.EventMarketingContent, ActorID: actor, TargetType: &target, TargetID: targetID, AfterValue: after})
}
