package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	acmodel "github.com/lextures/lextures/server/internal/models/adaptivecontent"
	acrepo "github.com/lextures/lextures/server/internal/repos/adaptivecontent"
	"github.com/lextures/lextures/server/internal/repos/course"
	acsvc "github.com/lextures/lextures/server/internal/service/adaptivecontent"
)

func (d Deps) registerAdaptiveContentEffectivenessRoutes(r chi.Router) {
	r.Get("/api/v1/courses/{course_code}/adaptive-content/units/{unit_id}/effectiveness", d.handleAdaptiveContentUnitEffectivenessGet())
	r.Get("/api/v1/courses/{course_code}/adaptive-content/effectiveness", d.handleAdaptiveContentEffectivenessList())
	r.Post("/api/v1/courses/{course_code}/adaptive-content/effectiveness/refresh", d.handleAdaptiveContentEffectivenessRefresh())
}

func (d Deps) effectivenessNotifyDeps() *acsvc.EffectivenessNotifyDeps {
	return &acsvc.EffectivenessNotifyDeps{
		Pool:   d.Pool,
		Config: d.effectiveConfig(),
		SSEHub: d.NotifHub,
	}
}

func mapUnitEffectiveness(
	cache *acrepo.EffectivenessCacheRow,
	modes []acrepo.ModeEffectivenessRow,
	variants []acrepo.VariantEffectivenessRow,
	unitID uuid.UUID,
) acmodel.UnitEffectiveness {
	out := acmodel.UnitEffectiveness{
		UnitID:        unitID,
		Verdict:       acsvc.VerdictInsufficientData,
		ByMode:        []acmodel.ModeEffectiveness{},
		ByVariant:     []acmodel.VariantEffectiveness{},
		SmallCellMinN: acsvc.SmallCellMinN,
		MinNPerArm:    acsvc.MinNPerArm,
	}
	if cache != nil {
		out.NTreatment = cache.NTreatment
		out.NHoldout = cache.NHoldout
		out.MeanLiftTreatment = cache.MeanLiftTreatment
		out.MeanLiftHoldout = cache.MeanLiftHoldout
		out.TreatmentMinusHoldout = cache.TreatmentMinusHoldout
		out.DiffStdError = cache.DiffStdError
		out.MeanMasteryDeltaTreatment = cache.MeanMasteryDeltaTreatment
		out.MeanMasteryDeltaHoldout = cache.MeanMasteryDeltaHoldout
		out.Verdict = cache.Verdict
		t := cache.RefreshedAt
		out.RefreshedAt = &t
	}
	for _, m := range modes {
		out.ByMode = append(out.ByMode, acmodel.ModeEffectiveness{
			EmphasisMode: m.EmphasisMode,
			N:            m.N,
			MeanLift:     m.MeanLift,
		})
	}
	for _, v := range variants {
		out.ByVariant = append(out.ByVariant, acmodel.VariantEffectiveness{
			VariantID: v.VariantID,
			N:         v.N,
			MeanLift:  v.MeanLift,
		})
	}
	return out
}

// handleAdaptiveContentUnitEffectivenessGet is GET .../units/{id}/effectiveness (instructor).
func (d Deps) handleAdaptiveContentUnitEffectivenessGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if acsvc.KillSwitchEngaged() {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeServiceUnavailable, acsvc.ErrKillSwitchEngaged.Error())
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
		unitID, err := uuid.Parse(chi.URLParam(r, "unit_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid unit id.")
			return
		}
		unit, err := acrepo.GetUnit(r.Context(), d.Pool, *cid, unitID)
		if err != nil || unit == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Unit not found.")
			return
		}
		cache, err := acrepo.GetEffectivenessCache(r.Context(), d.Pool, unitID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load effectiveness.")
			return
		}
		modes, err := acrepo.ListModeEffectiveness(r.Context(), d.Pool, unitID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load mode effectiveness.")
			return
		}
		variants, err := acrepo.ListVariantEffectiveness(r.Context(), d.Pool, unitID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load variant effectiveness.")
			return
		}
		out := mapUnitEffectiveness(cache, modes, variants, unitID)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

// handleAdaptiveContentEffectivenessList is GET .../adaptive-content/effectiveness (instructor).
func (d Deps) handleAdaptiveContentEffectivenessList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if acsvc.KillSwitchEngaged() {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeServiceUnavailable, acsvc.ErrKillSwitchEngaged.Error())
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
		caches, err := acrepo.ListEffectivenessForCourse(r.Context(), d.Pool, *cid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load effectiveness.")
			return
		}
		units := make([]acmodel.UnitEffectiveness, 0, len(caches))
		for i := range caches {
			c := caches[i]
			modes, err := acrepo.ListModeEffectiveness(r.Context(), d.Pool, c.UnitID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load mode effectiveness.")
				return
			}
			variants, err := acrepo.ListVariantEffectiveness(r.Context(), d.Pool, c.UnitID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load variant effectiveness.")
				return
			}
			units = append(units, mapUnitEffectiveness(&c, modes, variants, c.UnitID))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(acmodel.CourseEffectivenessResponse{Units: units})
	}
}

// handleAdaptiveContentEffectivenessRefresh is POST .../effectiveness/refresh (instructor).
func (d Deps) handleAdaptiveContentEffectivenessRefresh() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if acsvc.KillSwitchEngaged() {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeServiceUnavailable, acsvc.ErrKillSwitchEngaged.Error())
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
		n, err := acsvc.RefreshCourse(r.Context(), d.Pool, *cid, d.effectivenessNotifyDeps())
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to refresh effectiveness.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(acmodel.EffectivenessRefreshResponse{RefreshedUnits: n})
	}
}
