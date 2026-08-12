package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	mcrepo "github.com/lextures/lextures/server/internal/repos/marketingcontent"
	mcservice "github.com/lextures/lextures/server/internal/service/marketingcontent"
	"github.com/lextures/lextures/server/internal/service/marketingeditorial"
)

func (d Deps) registerMarketingEditorialRoutes(r chi.Router) {
	r.Get("/api/v1/admin/marketing/reviews/queue", d.handleMarketingReviewQueue())
	r.Post("/api/v1/admin/marketing/articles/{id}/review", d.handleMarketingReviewAction())
	r.Get("/api/v1/admin/marketing/articles/{id}/reviews", d.handleMarketingReviewHistory())
	r.Post("/api/v1/admin/marketing/articles/{id}/mark-reviewed", d.handleMarketingMarkReviewed())
	r.Get("/api/v1/admin/marketing/health", d.handleMarketingHealth())
	r.Get("/api/v1/admin/marketing/calendar", d.handleMarketingCalendar())
	r.Get("/api/v1/admin/marketing/briefs", d.handleMarketingBriefsList())
	r.Post("/api/v1/admin/marketing/briefs", d.handleMarketingBriefCreate())
	r.Patch("/api/v1/admin/marketing/briefs/{id}", d.handleMarketingBriefPatch())
	r.Delete("/api/v1/admin/marketing/briefs/{id}", d.handleMarketingBriefDelete())
	r.Get("/api/v1/admin/marketing/pillars", d.handleMarketingPillars())
	r.Get("/api/v1/admin/marketing/overrides", d.handleMarketingOverrides())
	r.Get("/api/v1/admin/marketing/link-health", d.handleMarketingLinkHealth())
	r.Get("/api/v1/admin/marketing/settings", d.handleMarketingEditorialSettings())
	r.Patch("/api/v1/admin/marketing/settings", d.handleMarketingEditorialSettingsPatch())
	r.Post("/api/v1/admin/marketing/authors/{slug}/retire", d.handleMarketingAuthorRetire())
}

func (d Deps) marketingEditorial() *marketingeditorial.Service {
	return &marketingeditorial.Service{Pool: d.Pool}
}
func (d Deps) handleMarketingReviewQueue() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := d.marketingAccess(w, r, marketingReview)
		if !ok {
			return
		}
		items, err := mcrepo.ListReviewQueue(r.Context(), d.Pool, actor, r.URL.Query().Get("assignedToMe") == "true")
		if err != nil {
			apierr.WriteInternal(w, r, "Failed to load the review queue.", err)
			return
		}
		writeJSON(w, 200, map[string]any{"items": items})
	}
}
func (d Deps) handleMarketingReviewAction() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := d.marketingAccess(w, r, marketingReview)
		if !ok {
			return
		}
		id, ok := marketingID(w, r)
		if !ok {
			return
		}
		var b struct {
			Action             string     `json:"action"`
			Note               string     `json:"note"`
			ReviewerID         *uuid.UUID `json:"reviewerId"`
			ExpectedRevisionNo int        `json:"expectedRevisionNo"`
		}
		if !readMarketingJSON(w, r, &b) {
			return
		}
		action := mcservice.ActionApprove
		if b.Action == "changes_requested" || b.Action == "request_changes" {
			action = mcservice.ActionRequestChanges
		} else if b.Action != "approved" && b.Action != "approve" {
			apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, "action must be approved or changes_requested")
			return
		}
		a, err := d.marketingService().Transition(r.Context(), id, actor, mcservice.TransitionInput{Action: action, Note: b.Note, ExpectedRevisionNo: b.ExpectedRevisionNo})
		if err != nil {
			d.writeMarketingConflict(w, r, id, err)
			return
		}
		writeJSON(w, 200, a)
	}
}
func (d Deps) handleMarketingReviewHistory() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingView); !ok {
			return
		}
		id, ok := marketingID(w, r)
		if !ok {
			return
		}
		items, err := mcrepo.ListReviews(r.Context(), d.Pool, id)
		if err != nil {
			apierr.WriteInternal(w, r, "Failed to load review history.", err)
			return
		}
		writeJSON(w, 200, map[string]any{"items": items})
	}
}
func (d Deps) handleMarketingMarkReviewed() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := d.marketingAccess(w, r, marketingReview)
		if !ok {
			return
		}
		id, ok := marketingID(w, r)
		if !ok {
			return
		}
		a, err := d.marketingEditorial().MarkReviewed(r.Context(), id, actor)
		if err != nil {
			writeMarketingError(w, r, err)
			return
		}
		d.recordMarketingAudit(r, actor, "mark_reviewed", &a)
		writeJSON(w, 200, a)
	}
}
func (d Deps) handleMarketingHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingView); !ok {
			return
		}
		h, err := d.marketingEditorial().Health(r.Context())
		if err != nil {
			apierr.WriteInternal(w, r, "Failed to load content health.", err)
			return
		}
		writeJSON(w, 200, h)
	}
}
func editorialRange(r *http.Request) (time.Time, time.Time, error) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" {
		now := time.Now().UTC()
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	}
	if to == "" {
		start, _ := time.Parse("2006-01-02", from)
		to = start.AddDate(0, 1, 0).Format("2006-01-02")
	}
	return marketingeditorial.ParseRange(from, to)
}
func (d Deps) handleMarketingCalendar() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingView); !ok {
			return
		}
		from, to, err := editorialRange(r)
		if err != nil {
			apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, err.Error())
			return
		}
		items, briefs, err := d.marketingEditorial().Calendar(r.Context(), from, to)
		if err != nil {
			apierr.WriteInternal(w, r, "Failed to load the editorial calendar.", err)
			return
		}
		writeJSON(w, 200, map[string]any{"scheduled": items, "briefs": briefs})
	}
}
func (d Deps) handleMarketingBriefsList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingView); !ok {
			return
		}
		from, to, err := editorialRange(r)
		if err != nil {
			apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, err.Error())
			return
		}
		items, err := mcrepo.ListBriefs(r.Context(), d.Pool, from, to)
		if err != nil {
			apierr.WriteInternal(w, r, "Failed to load briefs.", err)
			return
		}
		writeJSON(w, 200, map[string]any{"items": items})
	}
}
func (d Deps) handleMarketingBriefCreate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, ok := d.marketingAccess(w, r, marketingAuthor)
		if !ok {
			return
		}
		var b mcrepo.Brief
		if !readMarketingJSON(w, r, &b) {
			return
		}
		if strings.TrimSpace(b.Title) == "" || (b.Kind != "blog" && b.Kind != "doc") {
			apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, "title and a valid kind are required")
			return
		}
		if b.Status == "" {
			b.Status = "planned"
		}
		tx, err := d.Pool.Begin(r.Context())
		if err != nil {
			apierr.WriteInternal(w, r, "Failed to create brief.", err)
			return
		}
		defer tx.Rollback(r.Context())
		out, err := mcrepo.InsertBrief(r.Context(), tx, b)
		if err == nil {
			err = tx.Commit(r.Context())
		}
		if err != nil {
			apierr.WriteInternal(w, r, "Failed to create brief.", err)
			return
		}
		writeJSON(w, 201, out)
	}
}
func briefID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, "Invalid brief id.")
		return uuid.Nil, false
	}
	return id, true
}
func (d Deps) handleMarketingBriefPatch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingAuthor); !ok {
			return
		}
		id, ok := briefID(w, r)
		if !ok {
			return
		}
		var b mcrepo.Brief
		if !readMarketingJSON(w, r, &b) {
			return
		}
		b.ID = id
		affected, err := mcrepo.UpdateBrief(r.Context(), d.Pool, b)
		if err != nil {
			apierr.WriteInternal(w, r, "Failed to update brief.", err)
			return
		}
		if affected == 0 {
			apierr.WriteJSON(w, 404, apierr.CodeNotFound, "Brief not found.")
			return
		}
		w.WriteHeader(204)
	}
}
func (d Deps) handleMarketingBriefDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingAuthor); !ok {
			return
		}
		id, ok := briefID(w, r)
		if !ok {
			return
		}
		err := mcrepo.DeleteBrief(r.Context(), d.Pool, id)
		if err != nil {
			apierr.WriteInternal(w, r, "Failed to delete brief.", err)
			return
		}
		w.WriteHeader(204)
	}
}
func (d Deps) handleMarketingPillars() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingView); !ok {
			return
		}
		items, err := d.marketingEditorial().Pillars(r.Context())
		if err != nil {
			apierr.WriteInternal(w, r, "Failed to load pillar coverage.", err)
			return
		}
		writeJSON(w, 200, map[string]any{"items": items})
	}
}
func (d Deps) handleMarketingOverrides() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingReview); !ok {
			return
		}
		items, err := d.marketingEditorial().Overrides(r.Context())
		if err != nil {
			apierr.WriteInternal(w, r, "Failed to load governance report.", err)
			return
		}
		writeJSON(w, 200, map[string]any{"items": items})
	}
}
func (d Deps) handleMarketingLinkHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingView); !ok {
			return
		}
		h, err := d.marketingEditorial().Health(r.Context())
		if err != nil {
			apierr.WriteInternal(w, r, "Failed to load link health.", err)
			return
		}
		writeJSON(w, 200, map[string]any{"items": h.LinkFailures})
	}
}
func (d Deps) handleMarketingEditorialSettings() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingView); !ok {
			return
		}
		s, err := mcrepo.GetEditorialSettings(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteInternal(w, r, "Failed to load editorial settings.", err)
			return
		}
		writeJSON(w, 200, s)
	}
}
func (d Deps) handleMarketingEditorialSettingsPatch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, ok := d.marketingAccess(w, r, marketingAdmin)
		if !ok {
			return
		}
		var b mcrepo.EditorialSettings
		if !readMarketingJSON(w, r, &b) {
			return
		}
		if b.ReviewIntervalDocDays < 1 || b.ReviewIntervalBlogDays < 1 || b.RevisionRetentionMonths < 1 || b.StaleThresholdPct < 0 || b.StaleThresholdPct > 100 || !json.Valid(b.Pillars) {
			apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, "Invalid editorial settings.")
			return
		}
		err := mcrepo.UpdateEditorialSettings(r.Context(), d.Pool, b)
		if err != nil {
			apierr.WriteInternal(w, r, "Failed to save editorial settings.", err)
			return
		}
		writeJSON(w, 200, b)
	}
}
func (d Deps) handleMarketingAuthorRetire() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := d.marketingAccess(w, r, marketingAdmin)
		if !ok {
			return
		}
		slug := chi.URLParam(r, "slug")
		retired, err := mcrepo.RetireAuthor(r.Context(), d.Pool, slug, actor)
		if err != nil {
			apierr.WriteInternal(w, r, "Failed to retire author.", err)
			return
		}
		if !retired {
			apierr.WriteJSON(w, 404, apierr.CodeNotFound, "Active author not found.")
			return
		}
		items, _, err := mcrepo.ListArticles(r.Context(), d.Pool, mcrepo.ArticleFilter{AuthorSlug: slug, Limit: 100})
		if err != nil {
			apierr.WriteInternal(w, r, "Author retired, but reassignment list failed.", err)
			return
		}
		writeJSON(w, 200, map[string]any{"slug": slug, "status": "retired", "articlesForReassignment": items, "count": len(items)})
	}
}
