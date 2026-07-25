package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	acmodel "github.com/lextures/lextures/server/internal/models/adaptivecontent"
	acrepo "github.com/lextures/lextures/server/internal/repos/adaptivecontent"
	"github.com/lextures/lextures/server/internal/repos/course"
	acsvc "github.com/lextures/lextures/server/internal/service/adaptivecontent"
)

// registerAdaptiveContentPipelineRoutes is AC.4 prewarm + called from registerAdaptiveContentRoutes.
func (d Deps) registerAdaptiveContentPipelineRoutes(r chi.Router) {
	r.Post("/api/v1/courses/{course_code}/adaptive-content/units/{unit_id}/prewarm", d.handleAdaptiveContentUnitPrewarm())
	// Admin platform generation pause (distinct from kill-switch).
	r.Get("/api/v1/admin/adaptive-content", d.handleAdminAdaptiveContentGet())
	r.Patch("/api/v1/admin/adaptive-content", d.handleAdminAdaptiveContentPatch())
}

// handleAdaptiveContentSettingsPatch is PATCH .../adaptive-content/settings (instructor).
// Accepts generationPaused and/or maxPrewarmVariants without requiring full settings body.
func (d Deps) handleAdaptiveContentSettingsPatch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if acsvc.KillSwitchEngaged() {
			writeACEKillSwitch(w)
			return
		}
		courseCode, viewer, ok := d.requireCourseItemCreate(w, r)
		if !ok {
			return
		}
		var body acmodel.PatchSettingsRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if body.GenerationPaused == nil && body.MaxPrewarmVariants == nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Provide generationPaused and/or maxPrewarmVariants.")
			return
		}
		if body.MaxPrewarmVariants != nil && (*body.MaxPrewarmVariants < 0 || *body.MaxPrewarmVariants > 100) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "maxPrewarmVariants must be between 0 and 100.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		out, err := acrepo.PatchPipelineSettings(r.Context(), d.Pool, *cid, body.GenerationPaused, body.MaxPrewarmVariants, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to patch adaptive content settings.")
			return
		}
		actor := viewer
		_ = acrepo.InsertEvent(r.Context(), d.Pool, *cid, nil, &actor, nil, acsvc.EventSettingsUpdated, map[string]any{
			"generationPaused":   out.GenerationPaused,
			"maxPrewarmVariants": out.MaxPrewarmVariants,
			"partial":            true,
		})
		acsvc.IncSettingsUpdated()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(settingsToAPI(*out))
	}
}

// handleAdaptiveContentBudgetGet is GET .../adaptive-content/budget (instructor).
func (d Deps) handleAdaptiveContentBudgetGet() http.HandlerFunc {
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
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		st, err := acsvc.LoadBudgetStatus(r.Context(), d.Pool, *cid, time.Now().UTC())
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load budget.")
			return
		}
		out := acmodel.BudgetResponse{
			MonthlyTokenBudget: st.MonthlyTokenBudget,
			TokensUsedPeriod:   st.TokensUsedPeriod,
			BudgetRemaining:    st.BudgetRemaining,
			PeriodStart:        st.PeriodStart.Format("2006-01-02"),
			GenerationPaused:   st.GenerationPaused,
			Unlimited:          st.Unlimited,
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

// handleAdaptiveContentUnitPrewarm is POST .../units/{id}/prewarm (instructor).
func (d Deps) handleAdaptiveContentUnitPrewarm() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if acsvc.KillSwitchEngaged() {
			writeACEKillSwitch(w)
			return
		}
		courseCode, viewer, ok := d.requireCourseItemCreate(w, r)
		if !ok {
			return
		}
		unitID, err := uuid.Parse(chi.URLParam(r, "unit_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid unit id.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		if cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		unit, err := acrepo.GetUnit(r.Context(), d.Pool, *cid, unitID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load unit.")
			return
		}
		if unit == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Unit not found.")
			return
		}
		// Course flag + kill-switch.
		enabled, _ := acrepo.AdaptiveContentEnabledForCourse(r.Context(), d.Pool, *cid)
		if !acsvc.ActiveForCourse(enabled) {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeInvalidInput, "Adaptive content is not enabled for this course.")
			return
		}
		settings, _ := acrepo.GetSettings(r.Context(), d.Pool, *cid)
		if settings != nil && settings.GenerationPaused {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeInvalidInput, "Generation is paused for this course.")
			return
		}
		platformPaused, _ := acrepo.GetPlatformGenerationPaused(r.Context(), d.Pool)
		if platformPaused {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeInvalidInput, "Adaptive content generation is paused platform-wide.")
			return
		}

		enqueued, err := acsvc.PrewarmUnit(r.Context(), d.Pool, *unit, 0, acsvc.PriorityPrewarm)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to prewarm unit.")
			return
		}
		actor := viewer
		uid := unit.ID
		_ = acrepo.InsertEvent(r.Context(), d.Pool, *cid, &uid, &actor, nil, acsvc.EventPrewarmStarted, map[string]any{
			"enqueued": enqueued,
			"source":   "instructor",
		})
		pending, _ := acrepo.CountPendingJobs(r.Context(), d.Pool)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(acmodel.PrewarmResponse{
			Enqueued:   enqueued,
			QueueDepth: pending,
			UnitID:     unit.ID,
		})
	}
}

// handleAdminAdaptiveContentGet is GET /api/v1/admin/adaptive-content.
func (d Deps) handleAdminAdaptiveContentGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if _, ok := d.adminRbacUser(w, r); !ok {
			return
		}
		paused, err := acrepo.GetPlatformGenerationPaused(r.Context(), d.Pool)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load adaptive content admin state.")
			return
		}
		pending, _ := acrepo.CountPendingJobs(r.Context(), d.Pool)
		inflight, _ := acrepo.CountGeneratingJobs(r.Context(), d.Pool)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(acmodel.AdminAdaptiveContentResponse{
			GenerationPaused: paused,
			QueueDepth:       pending,
			Inflight:         inflight,
			KillSwitch:       acsvc.KillSwitchEngaged(),
		})
	}
}

// handleAdminAdaptiveContentPatch is PATCH /api/v1/admin/adaptive-content { generationPaused }.
func (d Deps) handleAdminAdaptiveContentPatch() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		actor, ok := d.adminRbacUser(w, r)
		if !ok {
			return
		}
		var body acmodel.AdminAdaptiveContentPatch
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if body.GenerationPaused == nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "generationPaused is required.")
			return
		}
		if err := acrepo.SetPlatformGenerationPaused(r.Context(), d.Pool, *body.GenerationPaused); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update platform generation pause.")
			return
		}
		// Best-effort audit on a synthetic platform course event is not available; log via events only if needed.
		_ = actor
		pending, _ := acrepo.CountPendingJobs(r.Context(), d.Pool)
		inflight, _ := acrepo.CountGeneratingJobs(r.Context(), d.Pool)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(acmodel.AdminAdaptiveContentResponse{
			GenerationPaused: *body.GenerationPaused,
			QueueDepth:       pending,
			Inflight:         inflight,
			KillSwitch:       acsvc.KillSwitchEngaged(),
		})
	}
}
