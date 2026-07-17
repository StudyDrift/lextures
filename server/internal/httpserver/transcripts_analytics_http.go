package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	transcriptsrepo "github.com/lextures/lextures/server/internal/repos/transcripts"
	"github.com/lextures/lextures/server/internal/telemetry"
)

const (
	permTranscriptsConsoleManage   = "org:transcripts:console:manage"
	permTranscriptsFinanceView     = "org:transcripts:finance:view"
	permTranscriptsConfigManage    = "org:transcripts:config:manage"
	permTranscriptsAnalyticsView   = "org:transcripts:analytics:view"
	permTranscriptsAnalyticsExport = "org:transcripts:analytics:export"
)

func (d Deps) registerTranscriptAnalyticsRoutes(r chi.Router) {
	r.Get("/api/v1/admin/transcripts/dashboard", d.handleAdminTranscriptDashboard())
	r.Get("/api/v1/admin/transcripts/health", d.handleAdminTranscriptHealth())
	r.Get("/api/v1/admin/transcripts/reports/export", d.handleAdminTranscriptReportExport())
	r.Get("/api/v1/admin/transcripts/dashboard/drilldown", d.handleAdminTranscriptDashboardDrilldown())
}

func (d Deps) transcriptsConsoleAccess(w http.ResponseWriter, r *http.Request) (uuid.UUID, transcriptsrepo.ConsolePanels, bool) {
	if d.transcriptsFeatureOff(w) {
		return uuid.Nil, transcriptsrepo.ConsolePanels{}, false
	}
	if d.JWTSigner == nil {
		apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "Sign in required.")
		return uuid.Nil, transcriptsrepo.ConsolePanels{}, false
	}
	u, err := auth.UserFromRequest(r, d.JWTSigner)
	if err != nil {
		apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "Sign in required.")
		return uuid.Nil, transcriptsrepo.ConsolePanels{}, false
	}
	userID, err := uuid.Parse(u.UserID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusUnauthorized, apierr.CodeUnauthorized, "Sign in required.")
		return uuid.Nil, transcriptsrepo.ConsolePanels{}, false
	}
	if d.Pool == nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
		return uuid.Nil, transcriptsrepo.ConsolePanels{}, false
	}
	ctx := r.Context()
	globalOK, err := rbac.UserHasPermission(ctx, d.Pool, userID, permGlobalRBACManage)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Server misconfiguration.")
		return uuid.Nil, transcriptsrepo.ConsolePanels{}, false
	}
	panels := transcriptsrepo.ConsolePanels{}
	if globalOK {
		panels = transcriptsrepo.ConsolePanels{
			Queue: true, Holds: true, Fees: true, Delivery: true, Recipients: true,
			Settings: true, Analytics: true, Finance: true, Export: true,
		}
		return userID, panels, true
	}
	has := func(perm string) bool {
		ok, err := rbac.UserHasPermission(ctx, d.Pool, userID, perm)
		return err == nil && ok
	}
	panels.Queue = has(permTranscriptsConsoleManage)
	panels.Holds = panels.Queue || has(permTranscriptsFinanceView)
	panels.Fees = has(permTranscriptsConfigManage) || has(permTranscriptsFinanceView)
	panels.Delivery = has(permTranscriptsConfigManage) || panels.Queue
	panels.Recipients = has(permTranscriptsConfigManage)
	panels.Settings = has(permTranscriptsConfigManage)
	panels.Analytics = has(permTranscriptsAnalyticsView) || has(permTranscriptsFinanceView)
	panels.Finance = has(permTranscriptsFinanceView) || has(permTranscriptsConfigManage)
	panels.Export = has(permTranscriptsAnalyticsExport) || (panels.Analytics && has(permTranscriptsConfigManage))

	if !panels.Queue && !panels.Holds && !panels.Fees && !panels.Analytics && !panels.Settings && !panels.Recipients {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
		return uuid.Nil, transcriptsrepo.ConsolePanels{}, false
	}
	return userID, panels, true
}

func (d Deps) resolveTranscriptsAdminOrg(w http.ResponseWriter, r *http.Request, actor uuid.UUID) (uuid.UUID, bool) {
	orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, actor)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to resolve organization.")
		return uuid.Nil, false
	}
	if q := r.URL.Query().Get("orgId"); q != "" {
		parsed, perr := uuid.Parse(q)
		if perr != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid orgId.")
			return uuid.Nil, false
		}
		orgID = parsed
	}
	if orgID == uuid.Nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Organization required.")
		return uuid.Nil, false
	}
	return orgID, true
}

func parseDashboardRange(r *http.Request) (time.Time, time.Time) {
	to := time.Now().UTC()
	from := to.AddDate(0, 0, -30)
	if raw := r.URL.Query().Get("from"); raw != "" {
		if t, err := time.Parse("2006-01-02", raw); err == nil {
			from = t.UTC()
		}
	}
	if raw := r.URL.Query().Get("to"); raw != "" {
		if t, err := time.Parse("2006-01-02", raw); err == nil {
			to = t.UTC()
		}
	}
	return from, to
}

// GET /api/v1/admin/transcripts/dashboard
func (d Deps) handleAdminTranscriptDashboard() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, panels, ok := d.transcriptsConsoleAccess(w, r)
		if !ok {
			return
		}
		if !panels.Analytics {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Analytics access required.")
			return
		}
		orgID, ok := d.resolveTranscriptsAdminOrg(w, r, actor)
		if !ok {
			return
		}
		from, to := parseDashboardRange(r)
		sum, err := transcriptsrepo.GetDashboard(r.Context(), d.Pool, orgID, from, to, panels)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load transcript analytics.")
			return
		}
		telemetry.RecordBusinessEvent("transcripts.admin.analytics_viewed")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(sum)
	}
}

// GET /api/v1/admin/transcripts/health
func (d Deps) handleAdminTranscriptHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, panels, ok := d.transcriptsConsoleAccess(w, r)
		if !ok {
			return
		}
		if !panels.Queue && !panels.Analytics {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Queue or analytics access required.")
			return
		}
		orgID, ok := d.resolveTranscriptsAdminOrg(w, r, actor)
		if !ok {
			return
		}
		sum, err := transcriptsrepo.GetHealth(r.Context(), d.Pool, orgID, panels)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load transcript health.")
			return
		}
		if sum.AnyAlert {
			telemetry.RecordBusinessEvent("transcripts.admin.sla_alert")
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(sum)
	}
}

// GET /api/v1/admin/transcripts/reports/export
func (d Deps) handleAdminTranscriptReportExport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, panels, ok := d.transcriptsConsoleAccess(w, r)
		if !ok {
			return
		}
		if !panels.Export && !panels.Analytics {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Export access required.")
			return
		}
		orgID, ok := d.resolveTranscriptsAdminOrg(w, r, actor)
		if !ok {
			return
		}
		reportType := strings.TrimSpace(r.URL.Query().Get("type"))
		if reportType == "" {
			reportType = "dashboard"
		}
		if reportType != "dashboard" && reportType != "summary" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "type must be dashboard or summary.")
			return
		}
		from, to := parseDashboardRange(r)
		sum, err := transcriptsrepo.GetDashboard(r.Context(), d.Pool, orgID, from, to, panels)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to build report.")
			return
		}
		filename := "transcript-analytics-" + sum.From + "_" + sum.To + ".csv"
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
		telemetry.RecordBusinessEvent("transcripts.admin.analytics_exported")
		if err := transcriptsrepo.WriteDashboardCSV(w, sum); err != nil {
			return
		}
	}
}

// GET /api/v1/admin/transcripts/dashboard/drilldown
func (d Deps) handleAdminTranscriptDashboardDrilldown() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, panels, ok := d.transcriptsConsoleAccess(w, r)
		if !ok {
			return
		}
		if !panels.Analytics {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Analytics access required.")
			return
		}
		orgID, ok := d.resolveTranscriptsAdminOrg(w, r, actor)
		if !ok {
			return
		}
		metric := strings.TrimSpace(r.URL.Query().Get("metric"))
		if metric == "" {
			metric = "orders"
		}
		from, to := parseDashboardRange(r)
		orders, err := transcriptsrepo.ListDrillDownOrders(r.Context(), d.Pool, orgID, metric, from, to, 50)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"metric": metric,
			"orders": orders,
		})
	}
}
