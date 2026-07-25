package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	acmodel "github.com/lextures/lextures/server/internal/models/adaptivecontent"
	acrepo "github.com/lextures/lextures/server/internal/repos/adaptivecontent"
	"github.com/lextures/lextures/server/internal/repos/course"
	acsvc "github.com/lextures/lextures/server/internal/service/adaptivecontent"
)

func (d Deps) registerAdaptiveContentRoutes(r chi.Router) {
	r.Get("/api/v1/courses/{course_code}/adaptive-content/settings", d.handleAdaptiveContentSettingsGet())
	r.Put("/api/v1/courses/{course_code}/adaptive-content/settings", d.handleAdaptiveContentSettingsPut())
	r.Patch("/api/v1/courses/{course_code}/adaptive-content/settings", d.handleAdaptiveContentSettingsPatch())
	r.Get("/api/v1/courses/{course_code}/adaptive-content/budget", d.handleAdaptiveContentBudgetGet())
	r.Get("/api/v1/courses/{course_code}/adaptive-content/units", d.handleAdaptiveContentUnitsList())
	r.Post("/api/v1/courses/{course_code}/adaptive-content/units", d.handleAdaptiveContentUnitsCreate())
	r.Patch("/api/v1/courses/{course_code}/adaptive-content/units/{unit_id}", d.handleAdaptiveContentUnitPatch())
	r.Delete("/api/v1/courses/{course_code}/adaptive-content/units/{unit_id}", d.handleAdaptiveContentUnitDelete())
	// AC.2
	r.Get("/api/v1/courses/{course_code}/adaptive-content/units/{unit_id}/profile", d.handleAdaptiveContentUnitProfileGet())
	r.Get("/api/v1/courses/{course_code}/adaptive-content/units/{unit_id}/profiles", d.handleAdaptiveContentUnitProfilesGet())
	r.Post("/api/v1/courses/{course_code}/adaptive-content/units/{unit_id}/pre-check/generate", d.handleAdaptiveContentPreCheckGenerate())
	// AC.3
	d.registerAdaptiveContentGenerateRoutes(r)
	// AC.4
	d.registerAdaptiveContentPipelineRoutes(r)
	// AC.5
	d.registerAdaptiveContentAuthoringRoutes(r)
	// AC.6
	d.registerAdaptiveContentServingRoutes(r)
	// AC.7
	d.registerAdaptiveContentEffectivenessRoutes(r)
	// AC.8
	d.registerAdaptiveContentGovernanceRoutes(r)
}

func writeACEKillSwitch(w http.ResponseWriter) {
	acsvc.RefreshKillSwitchMetric()
	apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeServiceUnavailable, acsvc.ErrKillSwitchEngaged.Error())
}

func settingsToAPI(r acrepo.SettingsRow) acmodel.Settings {
	axes := r.AllowedAxes
	if axes == nil {
		axes = []string{}
	}
	maxPrewarm := r.MaxPrewarmVariants
	if maxPrewarm <= 0 {
		maxPrewarm = 12
	}
	return acmodel.Settings{
		AllowedAxes:               axes,
		DefaultStrategy:           r.DefaultStrategy,
		HoldoutPercent:            r.HoldoutPercent,
		MonthlyTokenBudget:        r.MonthlyTokenBudget,
		RequireInstructorApproval: r.RequireInstructorApproval,
		StudentOptoutAllowed:      r.StudentOptoutAllowed,
		UpdatedAt:                 r.UpdatedAt,
		GenerationPaused:          r.GenerationPaused,
		MaxPrewarmVariants:        maxPrewarm,
	}
}

func unitToAPI(u acrepo.UnitRow) acmodel.Unit {
	axes := u.AllowedAxes
	if axes == nil {
		axes = []string{}
	}
	trigger := u.TriggerMode
	if trigger == "" {
		trigger = acsvc.TriggerPreQuiz
	}
	return acmodel.Unit{
		ID:                   u.ID,
		CourseID:             u.CourseID,
		TargetKind:           u.TargetKind,
		TargetModuleItemID:   u.TargetModuleItemID,
		TargetOutcomeID:      u.TargetOutcomeID,
		BaseContentItemID:    u.BaseContentItemID,
		PreAssessmentItemID:  u.PreAssessmentItemID,
		PostAssessmentItemID: u.PostAssessmentItemID,
		AllowedAxes:          axes,
		Status:               u.Status,
		CreatedBy:            u.CreatedBy,
		CreatedAt:            u.CreatedAt,
		UpdatedAt:            u.UpdatedAt,
		TriggerMode:          trigger,
		MasteryFreshnessDays: u.MasteryFreshnessDays,
		ContentVersion:       u.ContentVersion,
		MinFidelity:          u.MinFidelity,
		Quarantined:          u.Quarantined,
		QuarantinedReason:    u.QuarantinedReason,
	}
}

func unitToAPIWithConcepts(u acrepo.UnitRow, conceptIDs []uuid.UUID) acmodel.Unit {
	out := unitToAPI(u)
	if conceptIDs == nil {
		conceptIDs = []uuid.UUID{}
	}
	out.ConceptIDs = conceptIDs
	return out
}

// handleAdaptiveContentSettingsGet is GET .../adaptive-content/settings (course member).
func (d Deps) handleAdaptiveContentSettingsGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		courseCode, _, ok := d.requireCourseAccess(w, r)
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
		row, err := acrepo.GetSettings(r.Context(), d.Pool, *cid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load adaptive content settings.")
			return
		}
		if row == nil {
			def := acrepo.DefaultSettings(*cid)
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(settingsToAPI(def))
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(settingsToAPI(*row))
	}
}

// handleAdaptiveContentSettingsPut is PUT .../adaptive-content/settings (instructor).
func (d Deps) handleAdaptiveContentSettingsPut() http.HandlerFunc {
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
		var body acmodel.Settings
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		axes := acsvc.NormalizeAxes(body.AllowedAxes)
		strategy := body.DefaultStrategy
		if strategy == "" {
			strategy = "balanced"
		}
		if err := acsvc.ValidateSettings(axes, strategy, body.HoldoutPercent, body.MonthlyTokenBudget); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
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
		// Preserve AC.4 pipeline fields when legacy clients omit them (zero-value).
		// Prefer PATCH for generationPaused / maxPrewarmVariants.
		maxPrewarm := body.MaxPrewarmVariants
		generationPaused := body.GenerationPaused
		if existing, _ := acrepo.GetSettings(r.Context(), d.Pool, *cid); existing != nil {
			if maxPrewarm <= 0 {
				maxPrewarm = existing.MaxPrewarmVariants
				// Also preserve pause when maxPrewarm was omitted (legacy PUT).
				generationPaused = existing.GenerationPaused
			}
		}
		if maxPrewarm <= 0 {
			maxPrewarm = 12
		}
		if maxPrewarm > 100 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "maxPrewarmVariants must be between 0 and 100.")
			return
		}
		in := acrepo.SettingsRow{
			CourseID:                  *cid,
			AllowedAxes:               axes,
			DefaultStrategy:           strategy,
			HoldoutPercent:            body.HoldoutPercent,
			MonthlyTokenBudget:        body.MonthlyTokenBudget,
			RequireInstructorApproval: body.RequireInstructorApproval,
			StudentOptoutAllowed:      body.StudentOptoutAllowed,
			GenerationPaused:          generationPaused,
			MaxPrewarmVariants:        maxPrewarm,
		}
		out, err := acrepo.UpsertSettings(r.Context(), d.Pool, *cid, in, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save adaptive content settings.")
			return
		}
		actor := viewer
		_ = acrepo.InsertEvent(r.Context(), d.Pool, *cid, nil, &actor, nil, acsvc.EventSettingsUpdated, map[string]any{
			"allowedAxes":               out.AllowedAxes,
			"defaultStrategy":           out.DefaultStrategy,
			"holdoutPercent":            out.HoldoutPercent,
			"monthlyTokenBudget":        out.MonthlyTokenBudget,
			"requireInstructorApproval": out.RequireInstructorApproval,
			"studentOptoutAllowed":      out.StudentOptoutAllowed,
			"generationPaused":          out.GenerationPaused,
			"maxPrewarmVariants":        out.MaxPrewarmVariants,
		})
		acsvc.IncSettingsUpdated()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(settingsToAPI(*out))
	}
}

// handleAdaptiveContentUnitsList is GET .../adaptive-content/units (instructor|reviewer).
func (d Deps) handleAdaptiveContentUnitsList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		courseCode, _, _, ok := d.requireAdaptiveContentReview(w, r)
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
		rows, err := acrepo.ListUnits(r.Context(), d.Pool, *cid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list adaptive content units.")
			return
		}
		coverage, err := acrepo.CountVariantCoverageByCourse(r.Context(), d.Pool, *cid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load variant coverage.")
			return
		}
		units := make([]acmodel.Unit, 0, len(rows))
		for _, row := range rows {
			concepts, err := acrepo.ListUnitConceptIDs(r.Context(), d.Pool, row.ID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list unit concepts.")
				return
			}
			u := unitToAPIWithConcepts(row, concepts)
			if c, ok := coverage[row.ID]; ok {
				total, approved, pending, rejected, auto := c.Total, c.Approved, c.PendingReview, c.Rejected, c.AutoServed
				u.VariantTotal = &total
				u.VariantApproved = &approved
				u.VariantPendingReview = &pending
				u.VariantRejected = &rejected
				u.VariantAutoServed = &auto
			} else {
				z := int64(0)
				u.VariantTotal = &z
				u.VariantApproved = &z
				u.VariantPendingReview = &z
				u.VariantRejected = &z
				u.VariantAutoServed = &z
			}
			units = append(units, u)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(acmodel.UnitsListResponse{Units: units})
	}
}

// handleAdaptiveContentUnitsCreate is POST .../adaptive-content/units (instructor).
func (d Deps) handleAdaptiveContentUnitsCreate() http.HandlerFunc {
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
		var body acmodel.CreateUnitRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if body.BaseContentItemID == uuid.Nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, acsvc.ErrBaseContentRequired.Error())
			return
		}
		if err := acsvc.ValidateUnitTargetShape(body.TargetKind, body.TargetModuleItemID, body.TargetOutcomeID); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		status := body.Status
		if status == "" {
			status = "draft"
		}
		if err := acsvc.ValidateUnitStatus(status); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		axes := acsvc.NormalizeAxes(body.AllowedAxes)
		if err := acsvc.ValidateAxesList(axes); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
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
		if err := d.validateUnitRefs(r, *cid, body.BaseContentItemID, body.TargetModuleItemID, body.TargetOutcomeID, body.PreAssessmentItemID, body.PostAssessmentItemID); err != nil {
			if errors.Is(err, acsvc.ErrItemNotInCourse) || errors.Is(err, acsvc.ErrOutcomeNotInCourse) ||
				errors.Is(err, acsvc.ErrPreAssessmentNotQuiz) || errors.Is(err, acsvc.ErrPostAssessmentNotQuiz) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to validate unit references.")
			return
		}
		trigger := acsvc.NormalizeTriggerMode(body.TriggerMode)
		if err := acsvc.ValidateTriggerMode(trigger); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		freshness := int16(acsvc.DefaultMasteryFreshnessDays)
		if body.MasteryFreshnessDays != nil {
			if *body.MasteryFreshnessDays < 0 {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, acsvc.ErrInvalidFreshnessDays.Error())
				return
			}
			freshness = *body.MasteryFreshnessDays
		}
		if len(body.ConceptIDs) > 0 {
			if err := acsvc.ValidateConceptsBelongToCourse(r.Context(), d.Pool, *cid, body.ConceptIDs); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
		}
		in := acrepo.UnitRow{
			CourseID:             *cid,
			TargetKind:           body.TargetKind,
			TargetModuleItemID:   body.TargetModuleItemID,
			TargetOutcomeID:      body.TargetOutcomeID,
			BaseContentItemID:    body.BaseContentItemID,
			PreAssessmentItemID:  body.PreAssessmentItemID,
			PostAssessmentItemID: body.PostAssessmentItemID,
			AllowedAxes:          axes,
			Status:               status,
			CreatedBy:            viewer,
			TriggerMode:          trigger,
			MasteryFreshnessDays: freshness,
		}
		// Clear opposite target for clean shape.
		if body.TargetKind == "module" {
			in.TargetOutcomeID = nil
		} else {
			in.TargetModuleItemID = nil
		}
		out, err := acrepo.InsertUnit(r.Context(), d.Pool, in)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create adaptive content unit.")
			return
		}
		if len(body.ConceptIDs) > 0 {
			if err := acrepo.ReplaceUnitConcepts(r.Context(), d.Pool, out.ID, body.ConceptIDs); err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save unit concepts.")
				return
			}
		}
		concepts, _ := acrepo.ListUnitConceptIDs(r.Context(), d.Pool, out.ID)
		actor := viewer
		uid := out.ID
		_ = acrepo.InsertEvent(r.Context(), d.Pool, *cid, &uid, &actor, nil, acsvc.EventUnitCreated, map[string]any{
			"unitId":            out.ID,
			"targetKind":        out.TargetKind,
			"baseContentItemId": out.BaseContentItemID,
			"status":            out.Status,
			"triggerMode":       out.TriggerMode,
		})
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(unitToAPIWithConcepts(*out, concepts))
	}
}

// handleAdaptiveContentUnitPatch is PATCH .../units/{unit_id} (instructor).
func (d Deps) handleAdaptiveContentUnitPatch() http.HandlerFunc {
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
		var body acmodel.PatchUnitRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
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
		existing, err := acrepo.GetUnit(r.Context(), d.Pool, *cid, unitID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load unit.")
			return
		}
		if existing == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Unit not found.")
			return
		}
		next := *existing
		if body.TargetKind != nil {
			next.TargetKind = *body.TargetKind
		}
		if body.TargetModuleItemID != nil {
			next.TargetModuleItemID = body.TargetModuleItemID
		}
		if body.TargetOutcomeID != nil {
			next.TargetOutcomeID = body.TargetOutcomeID
		}
		if body.BaseContentItemID != nil {
			next.BaseContentItemID = *body.BaseContentItemID
		}
		if body.ClearPreAssessment {
			next.PreAssessmentItemID = nil
		} else if body.PreAssessmentItemID != nil {
			next.PreAssessmentItemID = body.PreAssessmentItemID
		}
		if body.ClearPostAssessment {
			next.PostAssessmentItemID = nil
		} else if body.PostAssessmentItemID != nil {
			next.PostAssessmentItemID = body.PostAssessmentItemID
		}
		if body.AllowedAxes != nil {
			next.AllowedAxes = acsvc.NormalizeAxes(body.AllowedAxes)
			if err := acsvc.ValidateAxesList(next.AllowedAxes); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
		}
		if body.Status != nil {
			if err := acsvc.ValidateUnitStatus(*body.Status); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			next.Status = *body.Status
		}
		if body.TriggerMode != nil {
			trigger := acsvc.NormalizeTriggerMode(*body.TriggerMode)
			if err := acsvc.ValidateTriggerMode(trigger); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			next.TriggerMode = trigger
		}
		if body.MasteryFreshnessDays != nil {
			if *body.MasteryFreshnessDays < 0 {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, acsvc.ErrInvalidFreshnessDays.Error())
				return
			}
			next.MasteryFreshnessDays = *body.MasteryFreshnessDays
		}
		if body.MinFidelity != nil {
			if *body.MinFidelity < 0 || *body.MinFidelity > 1 {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "minFidelity must be between 0 and 1.")
				return
			}
			next.MinFidelity = *body.MinFidelity
		}
		baseChanged := body.BaseContentItemID != nil && *body.BaseContentItemID != existing.BaseContentItemID
		// Enforce target shape after merges.
		switch next.TargetKind {
		case "module":
			next.TargetOutcomeID = nil
		case "outcome":
			next.TargetModuleItemID = nil
		}
		if err := acsvc.ValidateUnitTargetShape(next.TargetKind, next.TargetModuleItemID, next.TargetOutcomeID); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		if next.BaseContentItemID == uuid.Nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, acsvc.ErrBaseContentRequired.Error())
			return
		}
		if err := d.validateUnitRefs(r, *cid, next.BaseContentItemID, next.TargetModuleItemID, next.TargetOutcomeID, next.PreAssessmentItemID, next.PostAssessmentItemID); err != nil {
			if errors.Is(err, acsvc.ErrItemNotInCourse) || errors.Is(err, acsvc.ErrOutcomeNotInCourse) ||
				errors.Is(err, acsvc.ErrPreAssessmentNotQuiz) || errors.Is(err, acsvc.ErrPostAssessmentNotQuiz) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to validate unit references.")
			return
		}
		out, err := acrepo.UpdateUnit(r.Context(), d.Pool, next)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update unit.")
			return
		}
		if out == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Unit not found.")
			return
		}
		// AC.3: changing the bound base content invalidates cached variants.
		if baseChanged {
			if bumped, err := acrepo.BumpUnitContentVersion(r.Context(), d.Pool, *cid, out.ID); err == nil && bumped > 0 {
				out.ContentVersion = bumped
			}
			// AC.4: re-enqueue needed signatures for the new content version.
			if out.Status == "active" {
				_, _ = acsvc.EnqueueRegenForUnit(r.Context(), d.Pool, *out)
			}
		}
		// Concept set updates (AC.2).
		if body.ClearConceptIDs {
			if err := acrepo.ReplaceUnitConcepts(r.Context(), d.Pool, out.ID, nil); err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to clear unit concepts.")
				return
			}
		} else if body.ConceptIDs != nil {
			if err := acsvc.ValidateConceptsBelongToCourse(r.Context(), d.Pool, *cid, body.ConceptIDs); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			if err := acrepo.ReplaceUnitConcepts(r.Context(), d.Pool, out.ID, body.ConceptIDs); err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save unit concepts.")
				return
			}
		}
		// AC.4: unit activation → pre-warm top signatures.
		becameActive := body.Status != nil && *body.Status == "active" && existing.Status != "active"
		if becameActive && out.Status == "active" {
			_, _ = acsvc.PrewarmUnit(r.Context(), d.Pool, *out, 0, acsvc.PriorityActivation)
		}
		concepts, _ := acrepo.ListUnitConceptIDs(r.Context(), d.Pool, out.ID)
		actor := viewer
		uid := out.ID
		_ = acrepo.InsertEvent(r.Context(), d.Pool, *cid, &uid, &actor, nil, acsvc.EventUnitUpdated, map[string]any{
			"unitId":      out.ID,
			"status":      out.Status,
			"triggerMode": out.TriggerMode,
		})
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(unitToAPIWithConcepts(*out, concepts))
	}
}

// handleAdaptiveContentUnitDelete is DELETE .../units/{unit_id} (instructor).
func (d Deps) handleAdaptiveContentUnitDelete() http.HandlerFunc {
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
		deleted, err := acrepo.DeleteUnit(r.Context(), d.Pool, *cid, unitID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to delete unit.")
			return
		}
		if !deleted {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Unit not found.")
			return
		}
		actor := viewer
		uid := unitID
		_ = acrepo.InsertEvent(r.Context(), d.Pool, *cid, &uid, &actor, nil, acsvc.EventUnitDeleted, map[string]any{
			"unitId": unitID,
		})
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) validateUnitRefs(
	r *http.Request,
	courseID, baseID uuid.UUID,
	moduleItemID, outcomeID, preID, postID *uuid.UUID,
) error {
	ok, err := acrepo.StructureItemBelongsToCourse(r.Context(), d.Pool, courseID, baseID)
	if err != nil {
		return err
	}
	if !ok {
		return acsvc.ErrItemNotInCourse
	}
	checkItem := func(id *uuid.UUID) error {
		if id == nil || *id == uuid.Nil {
			return nil
		}
		ok, err := acrepo.StructureItemBelongsToCourse(r.Context(), d.Pool, courseID, *id)
		if err != nil {
			return err
		}
		if !ok {
			return acsvc.ErrItemNotInCourse
		}
		return nil
	}
	if err := checkItem(moduleItemID); err != nil {
		return err
	}
	if err := checkItem(preID); err != nil {
		return err
	}
	// FR-1: pre-assessment must be a quiz-kind structure item.
	if preID != nil && *preID != uuid.Nil {
		isQuiz, err := acrepo.StructureItemIsQuiz(r.Context(), d.Pool, courseID, *preID)
		if err != nil {
			return err
		}
		if !isQuiz {
			return acsvc.ErrPreAssessmentNotQuiz
		}
	}
	if err := checkItem(postID); err != nil {
		return err
	}
	// FR-1 (AC.7): post-assessment must be a quiz-kind structure item.
	if postID != nil && *postID != uuid.Nil {
		isQuiz, err := acrepo.StructureItemIsQuiz(r.Context(), d.Pool, courseID, *postID)
		if err != nil {
			return err
		}
		if !isQuiz {
			return acsvc.ErrPostAssessmentNotQuiz
		}
	}
	if outcomeID != nil && *outcomeID != uuid.Nil {
		ok, err := acrepo.OutcomeBelongsToCourse(r.Context(), d.Pool, courseID, *outcomeID)
		if err != nil {
			return err
		}
		if !ok {
			return acsvc.ErrOutcomeNotInCourse
		}
	}
	return nil
}
