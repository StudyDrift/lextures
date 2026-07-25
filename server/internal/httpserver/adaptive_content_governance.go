package httpserver

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	acmodel "github.com/lextures/lextures/server/internal/models/adaptivecontent"
	acrepo "github.com/lextures/lextures/server/internal/repos/adaptivecontent"
	"github.com/lextures/lextures/server/internal/repos/atrisk"
	"github.com/lextures/lextures/server/internal/repos/course"
	acsvc "github.com/lextures/lextures/server/internal/service/adaptivecontent"
	"github.com/lextures/lextures/server/internal/service/notifications"
)

func (d Deps) registerAdaptiveContentGovernanceRoutes(r chi.Router) {
	// Student contest + instructor inbox (AC.8 FR-6).
	r.Post("/api/v1/courses/{course_code}/adaptive-content/units/{unit_id}/contest", d.handleAdaptiveContentContestCreate())
	r.Get("/api/v1/courses/{course_code}/adaptive-content/contests", d.handleAdaptiveContentContestsList())
	r.Post("/api/v1/courses/{course_code}/adaptive-content/contests/{contest_id}/resolve", d.handleAdaptiveContentContestResolve())

	// Admin oversight / fairness / incident controls (AC.8 FR-9 / FR-10).
	r.Get("/api/v1/admin/adaptive-content/oversight", d.handleAdminAdaptiveContentOversight())
	r.Get("/api/v1/admin/adaptive-content/fairness", d.handleAdminAdaptiveContentFairness())
	r.Post("/api/v1/admin/adaptive-content/fairness/refresh", d.handleAdminAdaptiveContentFairnessRefresh())
	r.Post("/api/v1/admin/adaptive-content/quarantine", d.handleAdminAdaptiveContentQuarantine())
	r.Post("/api/v1/admin/adaptive-content/kill-switch", d.handleAdminAdaptiveContentKillSwitch())
}

func contestToAPI(c acrepo.ContestRow) acmodel.Contest {
	return acmodel.Contest{
		ID:            c.ID,
		CourseID:      c.CourseID,
		UnitID:        c.UnitID,
		ServingID:     c.ServingID,
		StudentUserID: c.StudentUserID,
		Reason:        c.Reason,
		Status:        c.Status,
		ResolvedBy:    c.ResolvedBy,
		CreatedAt:     c.CreatedAt,
		ResolvedAt:    c.ResolvedAt,
	}
}

// handleAdaptiveContentContestCreate is POST .../units/{id}/contest (student).
func (d Deps) handleAdaptiveContentContestCreate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		unitID, err := uuid.Parse(chi.URLParam(r, "unit_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid unit id.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		unit, err := acrepo.GetUnit(r.Context(), d.Pool, *cid, unitID)
		if err != nil || unit == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Unit not found.")
			return
		}
		var body acmodel.ContestRequest
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}
		row, err := acrepo.InsertContest(r.Context(), d.Pool, *cid, unitID, viewer, body.ServingID, body.Reason)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create contest.")
			return
		}
		uid := unitID
		subj := viewer
		_ = acrepo.InsertEvent(r.Context(), d.Pool, *cid, &uid, &subj, &subj, acsvc.EventContestOpened, map[string]any{
			"contestId": row.ID,
			"reason":    body.Reason,
		})
		acsvc.IncContestOpened()

		// Notify instructors for human review.
		instructors, _ := atrisk.ListInstructorUserIDs(r.Context(), d.Pool, *cid)
		actionURL := "/courses/" + courseCode + "/settings?tab=adaptive-content"
		push := &notifications.PushService{Pool: d.Pool, Config: d.Config, SSEHub: d.NotifHub}
		for _, iid := range instructors {
			if err := push.Enqueue(r.Context(), iid, notifications.EventAdaptiveContentContest,
				"Adaptation contest opened",
				"A student reported that an adapted section seems wrong. Please review.",
				actionURL); err != nil {
				slog.Warn("adaptivecontent: contest notify failed", "err", err)
			}
		}

		// Auto-pause unit when open contests exceed threshold (lean-yes open question).
		if n, err := acrepo.CountOpenContestsForUnit(r.Context(), d.Pool, unitID); err == nil && n >= acsvc.ContestAutoPauseThreshold {
			if unit.Status == "active" {
				unit.Status = "paused"
				_, _ = acrepo.UpdateUnit(r.Context(), d.Pool, *unit)
			}
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(contestToAPI(*row))
	}
}

// handleAdaptiveContentContestsList is GET .../adaptive-content/contests (instructor).
func (d Deps) handleAdaptiveContentContestsList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		courseCode, _, ok := d.requireCourseItemCreate(w, r)
		if !ok {
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		status := r.URL.Query().Get("status")
		rows, err := acrepo.ListContestsForCourse(r.Context(), d.Pool, *cid, status, 100, 0)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list contests.")
			return
		}
		out := make([]acmodel.Contest, 0, len(rows))
		for _, c := range rows {
			out = append(out, contestToAPI(c))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(acmodel.ContestsListResponse{Contests: out})
	}
}

// handleAdaptiveContentContestResolve is POST .../contests/{id}/resolve (instructor).
func (d Deps) handleAdaptiveContentContestResolve() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		courseCode, viewer, ok := d.requireCourseItemCreate(w, r)
		if !ok {
			return
		}
		contestID, err := uuid.Parse(chi.URLParam(r, "contest_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid contest id.")
			return
		}
		var body acmodel.ResolveContestRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if !acsvc.ValidContestResolveStatus(body.Status) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, acsvc.ErrInvalidContestStatus.Error())
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		existing, err := acrepo.GetContest(r.Context(), d.Pool, *cid, contestID)
		if err != nil || existing == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Contest not found.")
			return
		}
		if existing.Status != "open" {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeInvalidInput, acsvc.ErrContestNotOpen.Error())
			return
		}
		row, err := acrepo.ResolveContest(r.Context(), d.Pool, *cid, contestID, viewer, body.Status)
		if err != nil || row == nil {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeInvalidInput, "Could not resolve contest.")
			return
		}
		uid := row.UnitID
		actor := viewer
		_ = acrepo.InsertEvent(r.Context(), d.Pool, *cid, &uid, &actor, &row.StudentUserID, acsvc.EventContestResolved, map[string]any{
			"contestId": row.ID,
			"status":    body.Status,
		})
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(contestToAPI(*row))
	}
}

// handleAdminAdaptiveContentOversight is GET /api/v1/admin/adaptive-content/oversight.
func (d Deps) handleAdminAdaptiveContentOversight() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		acsvc.SyncDurableKillSwitchFromDB(r.Context(), d.Pool)
		paused, _ := acrepo.GetPlatformGenerationPaused(r.Context(), d.Pool)
		orgEnabled, _ := acrepo.GetOrgAdaptiveContentEnabled(r.Context(), d.Pool)
		pending, _ := acrepo.CountPendingJobs(r.Context(), d.Pool)
		inflight, _ := acrepo.CountGeneratingJobs(r.Context(), d.Pool)
		openContests, _ := acrepo.CountOpenContestsPlatform(r.Context(), d.Pool)
		disparity, _ := acrepo.CountDisparityFlags(r.Context(), d.Pool, nil)
		quarantined, _ := acrepo.CountQuarantinedUnits(r.Context(), d.Pool)
		regressing, _ := acrepo.CountRegressingUnits(r.Context(), d.Pool)
		gateBlocks, _ := acrepo.CountGateBlocksRecent(r.Context(), d.Pool)
		cost, _ := acrepo.SumAdaptiveContentCostUSD(r.Context(), d.Pool)

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(acmodel.OversightResponse{
			GenerationPaused:   paused,
			KillSwitch:         acsvc.KillSwitchEngaged(),
			OrgEnabled:         orgEnabled,
			QueueDepth:         pending,
			Inflight:           inflight,
			OpenContests:       openContests,
			DisparityFlags:     disparity,
			QuarantinedUnits:   quarantined,
			RegressingUnits:    regressing,
			GateBlocks7d:       gateBlocks,
			CostUSD30d:         cost,
			DPIADocPath:        "/docs/compliance/ace-dpia.md",
			AIActChecklistPath: "/docs/compliance/ace-eu-ai-act-checklist.md",
		})
	}
}

// handleAdminAdaptiveContentFairness is GET /api/v1/admin/adaptive-content/fairness?course=…
func (d Deps) handleAdminAdaptiveContentFairness() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		courseParam := r.URL.Query().Get("course")
		if courseParam == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "course query parameter is required (course id or code).")
			return
		}
		var courseID uuid.UUID
		if id, err := uuid.Parse(courseParam); err == nil {
			courseID = id
		} else {
			cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseParam)
			if err != nil || cid == nil {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
				return
			}
			courseID = *cid
		}
		rows, err := acrepo.ListFairnessForCourse(r.Context(), d.Pool, courseID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load fairness audit.")
			return
		}
		cells := make([]acmodel.FairnessCell, 0, len(rows))
		for _, row := range rows {
			cells = append(cells, acmodel.FairnessCell{
				ID:            row.ID,
				CourseID:      row.CourseID,
				Dimension:     row.Dimension,
				GroupLabel:    row.GroupLabel,
				N:             row.N,
				MeanFidelity:  row.MeanFidelity,
				CoveragePct:   row.CoveragePct,
				MeanLift:      row.MeanLift,
				DisparityFlag: row.DisparityFlag,
				ComputedAt:    row.ComputedAt,
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(acmodel.FairnessResponse{
			Cells:         cells,
			SmallCellMinN: acsvc.SmallCellMinN,
			FairnessMinN:  acsvc.FairnessMinN,
		})
	}
}

// handleAdminAdaptiveContentFairnessRefresh is POST /api/v1/admin/adaptive-content/fairness/refresh?course=…
func (d Deps) handleAdminAdaptiveContentFairnessRefresh() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		courseParam := r.URL.Query().Get("course")
		notify := &acsvc.FairnessNotifyDeps{Pool: d.Pool, Config: d.Config, SSEHub: d.NotifHub}
		if courseParam == "" {
			n, err := acsvc.RefreshFairnessAll(r.Context(), d.Pool, notify)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to refresh fairness.")
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(map[string]any{"refreshedCells": n})
			return
		}
		var courseID uuid.UUID
		if id, err := uuid.Parse(courseParam); err == nil {
			courseID = id
		} else {
			cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseParam)
			if err != nil || cid == nil {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
				return
			}
			courseID = *cid
		}
		n, err := acsvc.RefreshFairnessCourse(r.Context(), d.Pool, courseID, notify)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to refresh fairness.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"refreshedCells": n})
	}
}

// handleAdminAdaptiveContentQuarantine is POST /api/v1/admin/adaptive-content/quarantine.
func (d Deps) handleAdminAdaptiveContentQuarantine() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		actor, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		var body acmodel.QuarantineRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if body.UnitID == nil && body.CourseID == nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, acsvc.ErrQuarantineTarget.Error())
			return
		}
		if body.Clear {
			if body.UnitID != nil && body.CourseID != nil {
				okSet, err := acrepo.SetUnitQuarantine(r.Context(), d.Pool, *body.CourseID, *body.UnitID, false, "", actor)
				if err != nil || !okSet {
					apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Unit not found.")
					return
				}
				uid := *body.UnitID
				_ = acrepo.InsertEvent(r.Context(), d.Pool, *body.CourseID, &uid, &actor, nil, acsvc.EventUnquarantined, map[string]any{
					"unitId": *body.UnitID,
				})
			} else if body.CourseID != nil {
				if err := acrepo.SetCourseQuarantine(r.Context(), d.Pool, *body.CourseID, false, ""); err != nil {
					apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to clear course quarantine.")
					return
				}
				_ = acrepo.InsertEvent(r.Context(), d.Pool, *body.CourseID, nil, &actor, nil, acsvc.EventUnquarantined, map[string]any{
					"courseId": *body.CourseID,
					"scope":    "course",
				})
			} else {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "courseId is required to clear unit quarantine.")
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "cleared": true})
			return
		}

		if body.UnitID != nil {
			if body.CourseID == nil {
				// Look up course from unit.
				var courseID uuid.UUID
				err := d.Pool.QueryRow(r.Context(), `
SELECT course_id FROM course.adaptive_content_units WHERE id = $1
`, *body.UnitID).Scan(&courseID)
				if err != nil {
					apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Unit not found.")
					return
				}
				body.CourseID = &courseID
			}
			if err := acsvc.QuarantineUnit(r.Context(), d.Pool, *body.CourseID, *body.UnitID, actor, body.Reason); err != nil {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Unit not found.")
				return
			}
		} else if body.CourseID != nil {
			if err := acsvc.QuarantineCourse(r.Context(), d.Pool, *body.CourseID, actor, body.Reason); err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to quarantine course.")
				return
			}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "quarantined": true})
	}
}

// handleAdminAdaptiveContentKillSwitch is POST /api/v1/admin/adaptive-content/kill-switch.
func (d Deps) handleAdminAdaptiveContentKillSwitch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		actor, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		var body acmodel.KillSwitchRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if err := acsvc.EngageKillSwitch(r.Context(), d.Pool, actor, body.Engage); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update kill-switch.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"killSwitch": acsvc.KillSwitchEngaged(),
			"engage":     body.Engage,
		})
	}
}
