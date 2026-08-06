package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	acmodel "github.com/lextures/lextures/server/internal/models/adaptivecontent"
	acrepo "github.com/lextures/lextures/server/internal/repos/adaptivecontent"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursestructure"
	acsvc "github.com/lextures/lextures/server/internal/service/adaptivecontent"
)

// handleAdaptiveContentUnitProfileGet is GET .../units/{unit_id}/profile (student own).
func (d Deps) handleAdaptiveContentUnitProfileGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		// Prefer own profile by user id (works even if enrollment lookup fails for staff).
		row, err := acrepo.GetProfileForUser(r.Context(), d.Pool, unitID, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load profile.")
			return
		}

		// For mastery_snapshot / diagnostic without a profile yet, compute on read when ACE is active.
		if row == nil {
			enabled, err := acrepo.AdaptiveContentEnabledForCourse(r.Context(), d.Pool, *cid)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course flag.")
				return
			}
			mode := acsvc.NormalizeTriggerMode(unit.TriggerMode)
			if acsvc.ActiveForCourse(enabled) &&
				(mode == acsvc.TriggerMasterySnapshot || mode == acsvc.TriggerDiagnosticFirstVisit) {
				enrollmentID, err := acrepo.GetEnrollmentIDForUser(r.Context(), d.Pool, *cid, viewer)
				if err != nil {
					apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to resolve enrollment.")
					return
				}
				if enrollmentID != nil {
					row, err = acsvc.EnsureMasterySnapshotProfile(r.Context(), d.Pool, *unit, *enrollmentID, viewer)
					if err != nil {
						apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to compute profile.")
						return
					}
				}
			}
		}

		if row == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "No adaptation profile for this unit.")
			return
		}
		// Security: students may only read their own profile (already filtered by user id).
		if row.UserID != viewer {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You may only read your own adaptation profile.")
			return
		}

		out := profileRowToAPI(row)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

// handleAdaptiveContentUnitProfilesGet is GET .../units/{unit_id}/profiles (instructor cohort).
func (d Deps) handleAdaptiveContentUnitProfilesGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, _, ok := d.requireCourseItemCreate(w, r)
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
		dist, err := acrepo.ListCohortDistribution(r.Context(), d.Pool, unitID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load cohort profiles.")
			return
		}
		if n, err := acrepo.CountDistinctSignatures(r.Context(), d.Pool, unitID); err == nil {
			acsvc.SetDistinctSignaturesPerUnit(float64(n))
		}
		byEm := make([]acmodel.EmphasisBucket, 0, len(dist.ByEmphasis))
		for _, e := range dist.ByEmphasis {
			byEm = append(byEm, acmodel.EmphasisBucket{EmphasisMode: e.EmphasisMode, Count: e.Count})
		}
		bySig := make([]acmodel.SignatureBucket, 0, len(dist.BySignature))
		for _, s := range dist.BySignature {
			bySig = append(bySig, acmodel.SignatureBucket{
				ProfileSignature: s.ProfileSignature,
				EmphasisMode:     s.EmphasisMode,
				Count:            s.Count,
			})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(acmodel.CohortProfilesResponse{
			ByEmphasis:  byEm,
			BySignature: bySig,
		})
	}
}

// handleAdaptiveContentPreCheckGenerate is POST .../units/{unit_id}/pre-check/generate.
// Creates an adaptive quiz under the unit's module, seeded from base content, and binds it as pre-assessment.
func (d Deps) handleAdaptiveContentPreCheckGenerate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		var body acmodel.GeneratePreCheckRequest
		if r.Body != nil {
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
				// Empty body is fine (use defaults); other decode errors are invalid JSON.
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
				return
			}
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

		// Resolve parent module: prefer target module, else parent of base content.
		var moduleID *uuid.UUID
		if unit.TargetKind == "module" && unit.TargetModuleItemID != nil {
			moduleID = unit.TargetModuleItemID
		} else {
			moduleID, err = acrepo.ParentModuleID(r.Context(), d.Pool, *cid, unit.BaseContentItemID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to resolve module.")
				return
			}
		}
		if moduleID == nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput,
				"Could not determine a parent module for the pre-check quiz. Set targetModuleItemId on a module-target unit.")
			return
		}

		title := strings.TrimSpace(body.Title)
		if title == "" {
			title = "Pre-check"
		}
		qCount := body.QuestionCount
		if qCount <= 0 {
			qCount = 5
		}
		if qCount > 20 {
			qCount = 20
		}

		item, err := coursestructure.InsertQuizUnderModule(r.Context(), d.Pool, *cid, *moduleID, title)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create pre-check quiz.")
			return
		}

		// Configure as adaptive quiz seeded from the unit's base content (reuses adaptive_quiz path).
		prompt := "Generate a short diagnostic pre-check for this unit's base content. Prefer concept-tagged items."
		if err := acrepo.ConfigureQuizAsAdaptivePreCheck(
			r.Context(), d.Pool, item.ID,
			[]uuid.UUID{unit.BaseContentItemID},
			qCount,
			prompt,
		); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to configure adaptive pre-check.")
			return
		}

		// Bind as pre-assessment on the unit.
		next := *unit
		next.PreAssessmentItemID = &item.ID
		out, err := acrepo.UpdateUnit(r.Context(), d.Pool, next)
		if err != nil || out == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to bind pre-assessment to unit.")
			return
		}
		concepts, _ := acrepo.ListUnitConceptIDs(r.Context(), d.Pool, out.ID)
		actor := viewer
		uid := out.ID
		_ = acrepo.InsertEvent(r.Context(), d.Pool, *cid, &uid, &actor, nil, acsvc.EventUnitUpdated, map[string]any{
			"unitId":              out.ID,
			"preAssessmentItemId": item.ID,
			"preCheckGenerated":   true,
		})

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(acmodel.GeneratePreCheckResponse{
			PreAssessmentItemID: item.ID,
			Unit:                unitToAPIWithConcepts(*out, concepts),
		})
	}
}

func profileRowToAPI(row *acrepo.ProfileRow) acmodel.AdaptationProfile {
	emphasis := ""
	if row.EmphasisMode != nil {
		emphasis = *row.EmphasisMode
	}
	out := acmodel.AdaptationProfile{
		UnitID:           row.UnitID,
		EmphasisMode:     emphasis,
		TargetBloom:      row.TargetBloom,
		ProfileSignature: row.ProfileSignature,
		IsNeutral:        row.IsNeutral,
		ReadingLevelPref: row.ReadingLevelPref,
		ModalityPref:     row.ModalityPref,
		AxisSet:          row.AxisSet,
		SourceAttemptID:  row.SourceAttemptID,
		CreatedAt:        row.CreatedAt,
		ConceptGaps:      []acmodel.ConceptGap{},
		Misconceptions:   []string{},
	}
	// Decode payload for concept gaps / misconceptions.
	var payload struct {
		ConceptGaps []struct {
			ConceptID uuid.UUID `json:"conceptId"`
			Gap       float64   `json:"gap"`
		} `json:"conceptGaps"`
		Misconceptions []string `json:"misconceptions"`
	}
	if len(row.PayloadJSON) > 0 {
		_ = json.Unmarshal(row.PayloadJSON, &payload)
	}
	for _, g := range payload.ConceptGaps {
		out.ConceptGaps = append(out.ConceptGaps, acmodel.ConceptGap{ConceptID: g.ConceptID, Gap: g.Gap})
	}
	if payload.Misconceptions != nil {
		out.Misconceptions = payload.Misconceptions
	}
	return out
}
