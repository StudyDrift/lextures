package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/lextures/lextures/server/internal/apierr"
	mcrepo "github.com/lextures/lextures/server/internal/repos/marketingcontent"
	mcservice "github.com/lextures/lextures/server/internal/service/marketingcontent"
)

func (d Deps) registerMarketingI18nRoutes(r chi.Router) {
	r.Get("/api/v1/admin/marketing/articles/{id}/translations", d.handleMarketingTranslationsList())
	r.Post("/api/v1/admin/marketing/articles/{id}/translations", d.handleMarketingTranslationCreate())
	r.Post("/api/v1/admin/marketing/articles/{id}/mark-synced", d.handleMarketingMarkSynced())
	r.Get("/api/v1/admin/marketing/locales", d.handleMarketingLocalesList())
	r.Post("/api/v1/admin/marketing/locales", d.handleMarketingLocaleCreate())
	r.Patch("/api/v1/admin/marketing/locales/{code}", d.handleMarketingLocalePatch())
}

func (d Deps) handleMarketingTranslationsList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingView); !ok {
			return
		}
		id, ok := marketingID(w, r)
		if !ok {
			return
		}
		items, err := d.marketingService().ListTranslations(r.Context(), id)
		if err != nil {
			writeMarketingError(w, r, err)
			return
		}
		writeJSON(w, 200, map[string]any{"items": items})
	}
}

func (d Deps) handleMarketingTranslationCreate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := d.marketingAccess(w, r, marketingAuthor)
		if !ok {
			return
		}
		id, ok := marketingID(w, r)
		if !ok {
			return
		}
		var body struct {
			Locale string `json:"locale"`
			Slug   string `json:"slug"`
		}
		if !readMarketingJSON(w, r, &body) {
			return
		}
		a, err := d.marketingService().CreateTranslation(r.Context(), id, mcservice.CreateTranslationInput{Locale: body.Locale, Slug: body.Slug, Actor: actor})
		if err != nil {
			writeTranslationError(w, r, err)
			return
		}
		d.recordMarketingAudit(r, actor, "create_translation", a)
		writeJSON(w, 201, a)
	}
}

func (d Deps) handleMarketingMarkSynced() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := d.marketingAccess(w, r, marketingAuthor)
		if !ok {
			return
		}
		id, ok := marketingID(w, r)
		if !ok {
			return
		}
		a, err := d.marketingService().MarkSynced(r.Context(), id, actor)
		if err != nil {
			writeMarketingError(w, r, err)
			return
		}
		d.recordMarketingAudit(r, actor, "mark_synced", a)
		writeJSON(w, 200, a)
	}
}

func (d Deps) handleMarketingLocalesList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingView); !ok {
			return
		}
		items, enabled, err := d.marketingService().ListLocales(r.Context(), false)
		if err != nil {
			apierr.WriteInternal(w, r, "Failed to list locales.", err)
			return
		}
		writeJSON(w, 200, map[string]any{"items": items, "localesEnabled": enabled})
	}
}

func (d Deps) handleMarketingLocaleCreate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingAdmin); !ok {
			return
		}
		var body mcrepo.Locale
		if !readMarketingJSON(w, r, &body) {
			return
		}
		body.Code = mcrepo.NormalizeLocaleCode(body.Code)
		if body.Code == "" {
			apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, "locale code is invalid.")
			return
		}
		out, err := d.marketingService().UpsertLocale(r.Context(), body)
		if err != nil {
			apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, err.Error())
			return
		}
		writeJSON(w, 200, out)
	}
}

func (d Deps) handleMarketingLocalePatch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := d.marketingAccess(w, r, marketingAdmin); !ok {
			return
		}
		code := mcrepo.NormalizeLocaleCode(chi.URLParam(r, "code"))
		if code == "" {
			apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, "locale code is invalid.")
			return
		}
		var raw map[string]json.RawMessage
		if !readMarketingJSON(w, r, &raw) {
			return
		}
		var enabled *bool
		var sortOrder *int
		var label *string
		if v, ok := raw["enabled"]; ok {
			var b bool
			if err := json.Unmarshal(v, &b); err != nil {
				apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, "enabled must be a boolean.")
				return
			}
			enabled = &b
		}
		if v, ok := raw["sortOrder"]; ok {
			var n int
			if err := json.Unmarshal(v, &n); err != nil {
				apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, "sortOrder must be an integer.")
				return
			}
			sortOrder = &n
		}
		if v, ok := raw["label"]; ok {
			var s string
			if err := json.Unmarshal(v, &s); err != nil {
				apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, "label must be a string.")
				return
			}
			label = &s
		}
		out, err := d.marketingService().PatchLocale(r.Context(), code, enabled, sortOrder, label)
		if err != nil {
			writeMarketingError(w, r, err)
			return
		}
		writeJSON(w, 200, out)
	}
}

func writeTranslationError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, mcservice.ErrLocalesDisabled):
		apierr.WriteJSON(w, 404, apierr.CodeNotFound, "Translated content is not enabled.")
	case errors.Is(err, mcservice.ErrUnsupportedLocale):
		apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, "That locale is not supported.")
	case errors.Is(err, mcservice.ErrTranslationExists):
		apierr.WriteJSON(w, 409, apierr.CodeConflict, "A translation already exists for that locale.")
	case errors.Is(err, mcservice.ErrCannotTranslateSelf):
		apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, "Pick a different locale than the source article.")
	default:
		writeMarketingError(w, r, err)
	}
}

func (d Deps) publicLocale(w http.ResponseWriter, r *http.Request) (string, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("locale"))
	if raw == "" {
		return mcrepo.DefaultLocale, true
	}
	code, err := d.marketingService().ResolvePublicLocale(r.Context(), raw)
	if err != nil {
		apierr.WriteJSON(w, 400, apierr.CodeInvalidInput, "Unsupported locale.")
		return "", false
	}
	return code, true
}
