package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/board"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/telemetry"
)

// handleAdminBoardPolicies is GET/PATCH /api/v1/admin/boards/policies (VC.10).
func (d Deps) handleAdminBoardPolicies() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, actor)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to resolve organization.")
			return
		}
		if q := r.URL.Query().Get("orgId"); q != "" {
			parsed, perr := uuid.Parse(q)
			if perr != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid orgId.")
				return
			}
			orgID = parsed
		}

		switch r.Method {
		case http.MethodGet:
			pol, err := board.ResolveOrgPolicies(r.Context(), d.Pool, orgID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load board policies.")
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(pol)
		case http.MethodPatch:
			var body struct {
				ExternalSharing      *bool   `json:"externalSharing"`
				MinorModerationFloor *bool   `json:"minorModerationFloor"`
				DefaultAttribution   *string `json:"defaultAttribution"`
				BoardCapPerCourse    *int    `json:"boardCapPerCourse"`
				ClearBoardCap        bool    `json:"clearBoardCap"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
			pol, err := board.UpsertOrgPolicies(r.Context(), d.Pool, orgID, board.PatchOrgPoliciesInput{
				ExternalSharing:      body.ExternalSharing,
				MinorModerationFloor: body.MinorModerationFloor,
				DefaultAttribution:   body.DefaultAttribution,
				BoardCapPerCourse:    body.BoardCapPerCourse,
				ClearBoardCap:        body.ClearBoardCap,
			})
			if err != nil {
				if err.Error() == "board: invalid attribution" ||
					err.Error() == "board: board_cap_per_course must be >= 0" {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
					return
				}
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update board policies.")
				return
			}
			telemetry.RecordBusinessEvent("board.admin.policies_updated")
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(pol)
		default:
			w.Header().Set("Allow", "GET, PATCH")
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	}
}

// handleAdminBoardsOverview is GET /api/v1/admin/boards/overview (VC.10).
func (d Deps) handleAdminBoardsOverview() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		orgID, err := organization.OrgIDForUser(r.Context(), d.Pool, actor)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to resolve organization.")
			return
		}
		if q := r.URL.Query().Get("orgId"); q != "" {
			parsed, perr := uuid.Parse(q)
			if perr != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid orgId.")
				return
			}
			orgID = parsed
		}
		days := 30
		if raw := r.URL.Query().Get("activeDays"); raw != "" {
			if n, perr := strconv.Atoi(raw); perr == nil && n > 0 && n <= 365 {
				days = n
			}
		}
		overview, err := board.GetAdminOverview(r.Context(), d.Pool, orgID, days)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load boards overview.")
			return
		}
		telemetry.RecordBusinessEvent("board.admin.overview_viewed")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(overview)
	}
}

// handleGetBoardAnalytics is GET .../boards/{board_id}/analytics (VC.10).
func (d Deps) handleGetBoardAnalytics() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.visualBoardsFeatureOff(w, r, courseCode) {
			return
		}
		boardID := chi.URLParam(r, "board_id")
		b, caps, ok := d.loadBoardWithAccess(w, r, courseCode, viewer, boardID)
		if !ok {
			return
		}
		if !caps.CanManage {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Only board managers can view analytics.")
			return
		}
		days := 14
		if raw := r.URL.Query().Get("days"); raw != "" {
			if n, perr := strconv.Atoi(raw); perr == nil && n > 0 && n <= 90 {
				days = n
			}
		}
		sum, err := board.GetBoardAnalytics(r.Context(), d.Pool, courseCode, b.ID, days)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load board analytics.")
			return
		}
		if sum == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Board not found.")
			return
		}
		telemetry.RecordBusinessEvent("board.analytics.viewed")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(sum)
	}
}
