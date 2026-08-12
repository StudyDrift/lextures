package httpserver

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/objectcache"
	mcrepo "github.com/lextures/lextures/server/internal/repos/marketingcontent"
	"github.com/lextures/lextures/server/internal/repos/supportwidget"
	mcservice "github.com/lextures/lextures/server/internal/service/marketingcontent"
)

// GET /api/v1/orgs/{orgId}/settings/support-widget
// PUT /api/v1/orgs/{orgId}/settings/support-widget
func (d Deps) handleOrgSupportWidgetItem() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgStr := strings.TrimSpace(chi.URLParam(r, "orgId"))
		orgID, err := uuid.Parse(orgStr)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid organization id.")
			return
		}
		if _, _, ok := d.adminOrgOrUnitAccess(w, r, orgID); !ok {
			return
		}
		ctx := r.Context()
		switch r.Method {
		case http.MethodGet:
			row, err := supportwidget.Get(ctx, d.Pool, orgID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load support widget config.")
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(toWidgetJSON(orgID, row))

		case http.MethodPut:
			var body supportWidgetPutBody
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
			if err := validateWidgetBody(body); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}

			cur, err := supportwidget.Get(ctx, d.Pool, orgID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load support widget config.")
				return
			}

			enabled := mergeWidgetBool(cur, func(r *supportwidget.Row) bool { return r.Enabled }, body.Enabled, true)
			provider := mergeWidgetStr(cur, func(r *supportwidget.Row) string { return r.Provider }, body.Provider, "crisp")
			var websiteID *string
			if body.WebsiteID != nil {
				s := strings.TrimSpace(*body.WebsiteID)
				if s != "" {
					websiteID = &s
				}
			} else if cur != nil {
				websiteID = cur.WebsiteID
			}
			var dpaAt *time.Time
			if body.DPAConfirm != nil && *body.DPAConfirm {
				now := time.Now().UTC()
				dpaAt = &now
			} else if cur != nil {
				dpaAt = cur.DPAConfirmedAt
			}

			if err := supportwidget.Upsert(ctx, d.Pool, orgID, enabled, provider, websiteID, dpaAt); err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save support widget config.")
				return
			}
			updated, err := supportwidget.Get(ctx, d.Pool, orgID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to reload support widget config.")
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(toWidgetJSON(orgID, updated))

		default:
			w.Header().Set("Allow", strings.Join([]string{http.MethodGet, http.MethodPut}, ", "))
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	}
}

type supportWidgetPutBody struct {
	Enabled    *bool   `json:"enabled"`
	Provider   *string `json:"provider"`
	WebsiteID  *string `json:"websiteId"`
	DPAConfirm *bool   `json:"dpaConfirm"`
}

func validateWidgetBody(b supportWidgetPutBody) error {
	if b.Provider != nil {
		switch *b.Provider {
		case "crisp", "intercom", "none":
		default:
			return &widgetErr{s: `provider must be one of: "crisp", "intercom", "none"`}
		}
	}
	return nil
}

type widgetErr struct{ s string }

func (e *widgetErr) Error() string { return e.s }

func mergeWidgetBool(cur *supportwidget.Row, get func(*supportwidget.Row) bool, override *bool, def bool) bool {
	if override != nil {
		return *override
	}
	if cur != nil {
		return get(cur)
	}
	return def
}

func mergeWidgetStr(cur *supportwidget.Row, get func(*supportwidget.Row) string, override *string, def string) string {
	if override != nil {
		if s := strings.TrimSpace(*override); s != "" {
			return s
		}
	}
	if cur != nil {
		if s := get(cur); s != "" {
			return s
		}
	}
	return def
}

func toWidgetJSON(orgID uuid.UUID, row *supportwidget.Row) map[string]any {
	if row == nil {
		return map[string]any{
			"orgId":          orgID.String(),
			"enabled":        true,
			"provider":       "crisp",
			"websiteId":      nil,
			"dpaConfirmedAt": nil,
		}
	}
	var dpaStr any
	if row.DPAConfirmedAt != nil {
		dpaStr = row.DPAConfirmedAt.UTC().Format(time.RFC3339)
	}
	var wsID any
	if row.WebsiteID != nil {
		wsID = *row.WebsiteID
	}
	return map[string]any{
		"orgId":          row.OrgID.String(),
		"enabled":        row.Enabled,
		"provider":       row.Provider,
		"websiteId":      wsID,
		"dpaConfirmedAt": dpaStr,
	}
}

// GET /api/v1/help/contextual-articles?route=<path>
//
// Returns help articles relevant to the current route (MC.13 FR-3). When
// ff_marketing_content is enabled, results come from a tiered resolution over the
// published help center (route hints → related articles → category → search),
// filtered by the viewer's roles and cached for 5 minutes (FR-4/FR-5/FR-11).
func (d Deps) handleHelpContextualArticles() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := d.meUserID(w, r)
		if !ok {
			return
		}
		route := strings.TrimSpace(r.URL.Query().Get("route"))
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		if !d.effectiveConfig().FFMarketingContent || d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeServiceUnavailable, "Marketing content is unavailable.")
			return
		}

		ctx := r.Context()
		roles, err := mcrepo.ViewerRoles(ctx, d.Pool, actor)
		if err != nil {
			apierr.WriteInternal(w, r, "Failed to resolve viewer roles.", err)
			return
		}
		roles = mcservice.NormalizeViewerRoles(roles)

		key := objectcache.HelpContextualKey(route, roles, r.URL.Query().Get("locale"))
		var cached []mcservice.ContextualArticle
		if c := d.objectCache(); c != nil {
			if hit, _ := c.GetJSON(ctx, key, objectcache.ResourceHelpContextual, &cached); hit {
				_ = json.NewEncoder(w).Encode(map[string]any{"articles": cached})
				return
			}
		}

		articles, err := d.marketingService().ContextualArticles(ctx, route, roles, r.URL.Query().Get("locale"), 5)
		if err != nil {
			apierr.WriteInternal(w, r, "Failed to resolve contextual help articles.", err)
			return
		}
		if articles == nil {
			articles = []mcservice.ContextualArticle{}
		}
		tier := "none"
		if len(articles) > 0 {
			tier = articles[0].Tier
		}
		slog.Info("help_contextual_requests_total", "tier", tier)
		if c := d.objectCache(); c != nil {
			_ = c.SetJSON(ctx, key, articles, 5*time.Minute)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"articles": articles})
	}
}
