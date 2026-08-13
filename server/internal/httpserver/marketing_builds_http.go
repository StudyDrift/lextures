package httpserver

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	publish "github.com/lextures/lextures/server/internal/service/marketingpublish"
)

func (d Deps) marketingPublishService() *publish.Service {
	return &publish.Service{Pool: d.Pool, SecretsKey: d.effectiveConfig().PlatformSecretsKey}
}

func (d Deps) registerMarketingBuildRoutes(r interface {
	Get(string, http.HandlerFunc)
	Post(string, http.HandlerFunc)
	Put(string, http.HandlerFunc)
}) {
	r.Get("/api/v1/admin/marketing/builds", d.handleMarketingBuilds())
	r.Post("/api/v1/admin/marketing/builds", d.handleMarketingBuildCreate())
	r.Get("/api/v1/admin/marketing/publish-events", d.handleMarketingPublishEvents())
	r.Get("/api/v1/admin/marketing/build-settings", d.handleMarketingBuildSettings())
	r.Put("/api/v1/admin/marketing/build-settings", d.handleMarketingBuildSettingsUpdate())
}
func queryLimit(r *http.Request, def int) int {
	n, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if n <= 0 {
		return def
	}
	return n
}
func (d Deps) handleMarketingBuilds() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingView); !ok {
			return
		}
		v, e := d.marketingPublishService().ListBuilds(r.Context(), queryLimit(r, 50))
		if e != nil {
			apierr.WriteInternal(w, r, "Failed to list site builds.", e)
			return
		}
		writeJSON(w, 200, map[string]any{"items": v})
	}
}
func (d Deps) handleMarketingBuildCreate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := d.marketingAccess(w, r, marketingPublish)
		if !ok {
			return
		}
		v, e := d.marketingPublishService().EnqueueManual(r.Context(), actor)
		if errors.Is(e, publish.ErrRateLimited) {
			apierr.WriteJSON(w, 429, apierr.CodeRateLimited, "Manual rebuild limit exceeded (6 per hour).")
			return
		}
		if e != nil {
			apierr.WriteInternal(w, r, "Failed to request site build.", e)
			return
		}
		writeJSON(w, 202, v)
	}
}
func (d Deps) handleMarketingPublishEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingView); !ok {
			return
		}
		var article *uuid.UUID
		if raw := r.URL.Query().Get("articleId"); raw != "" {
			id, e := uuid.Parse(raw)
			if e != nil {
				apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, "Invalid article id.")
				return
			}
			article = &id
		}
		v, e := d.marketingPublishService().ListEvents(r.Context(), article, queryLimit(r, 100))
		if e != nil {
			apierr.WriteInternal(w, r, "Failed to list publish events.", e)
			return
		}
		writeJSON(w, 200, map[string]any{"items": v})
	}
}
func (d Deps) handleMarketingBuildSettings() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingAdmin); !ok {
			return
		}
		v, e := d.marketingPublishService().Settings(r.Context())
		if e != nil {
			apierr.WriteInternal(w, r, "Failed to load build settings.", e)
			return
		}
		writeJSON(w, 200, map[string]any{"provider": v.Provider, "repository": v.Repository, "workflowRef": v.WorkflowRef, "quietSeconds": int(v.QuietPeriod / time.Second), "maxWaitSeconds": int(v.MaxWait / time.Second), "tokenConfigured": v.Token != ""})
	}
}
func (d Deps) handleMarketingBuildSettingsUpdate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingAdmin); !ok {
			return
		}
		var b struct {
			Provider, Repository, WorkflowRef string
			QuietSeconds, MaxWaitSeconds      int
			Token                             *string
		}
		if !readMarketingJSON(w, r, &b) {
			return
		}
		e := d.marketingPublishService().UpdateSettings(r.Context(), publish.Settings{Provider: b.Provider, Repository: b.Repository, WorkflowRef: b.WorkflowRef, QuietPeriod: time.Duration(b.QuietSeconds) * time.Second, MaxWait: time.Duration(b.MaxWaitSeconds) * time.Second}, b.Token)
		if e != nil {
			apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, e.Error())
			return
		}
		writeJSON(w, 200, map[string]bool{"updated": true})
	}
}
