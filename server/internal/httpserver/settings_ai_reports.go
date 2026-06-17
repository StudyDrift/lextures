package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	models "github.com/lextures/lextures/server/internal/models/aiusage"
	"github.com/lextures/lextures/server/internal/repos/aiusage"
)

// handleGetSettingsAIReports is GET /api/v1/settings/ai/reports
func (d Deps) handleGetSettingsAIReports() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet+","+http.MethodOptions)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		if d.Pool == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "Database unavailable.")
			return
		}

		from, to, err := parseAIReportsTimeRange(r.URL.Query(), timeNowUTC())
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}

		q := r.URL.Query()
		filters := aiusage.Filters{
			From:       from,
			To:         to,
			Feature:    strings.TrimSpace(q.Get("feature")),
			UserQuery:  strings.TrimSpace(q.Get("userQuery")),
			CourseCode: strings.TrimSpace(q.Get("courseCode")),
		}
		if s := strings.TrimSpace(q.Get("userId")); s != "" {
			uid, perr := uuid.Parse(s)
			if perr != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid userId.")
				return
			}
			filters.UserID = &uid
		}
		if s := strings.TrimSpace(q.Get("courseId")); s != "" {
			cid, perr := uuid.Parse(s)
			if perr != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid courseId.")
				return
			}
			filters.CourseID = &cid
		}

		ctx := r.Context()
		summary, err := aiusage.CostSummary(ctx, d.Pool, filters)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load AI cost report.")
			return
		}
		byDay, err := aiusage.CostByDay(ctx, d.Pool, filters)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load AI cost report.")
			return
		}
		byFeature, err := aiusage.CostByFeature(ctx, d.Pool, filters)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load AI cost report.")
			return
		}
		byUser, err := aiusage.UsageByUser(ctx, d.Pool, filters, 50)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load AI usage by user.")
			return
		}
		byCourse, err := aiusage.UsageByCourse(ctx, d.Pool, filters, 50)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load AI usage by course.")
			return
		}

		out := models.ReportsPayload{
			Range: models.DateRange{From: from, To: to},
			Cost: models.CostReport{
				Summary:   summary,
				ByDay:     byDay,
				ByFeature: byFeature,
			},
			ByUser:   byUser,
			ByCourse: byCourse,
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}