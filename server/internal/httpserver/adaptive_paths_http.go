package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	apmodel "github.com/lextures/lextures/server/internal/models/adaptivepath"
	"github.com/lextures/lextures/server/internal/repos/adaptivepath"
	"github.com/lextures/lextures/server/internal/repos/concepts"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursestructure"
	apsvc "github.com/lextures/lextures/server/internal/service/adaptivepath"
)

func (d Deps) registerAdaptivePathRoutes(r chi.Router) {
	r.Get("/api/v1/courses/{course_code}/structure/items/{item_id}/path-rules", d.handleListStructurePathRules())
	r.Post("/api/v1/courses/{course_code}/structure/items/{item_id}/path-rules", d.handleCreateStructurePathRule())
	r.Delete("/api/v1/courses/{course_code}/structure/items/{item_id}/path-rules/{rule_id}", d.handleDeleteStructurePathRule())
	r.Get("/api/v1/courses/{course_code}/concepts-for-path", d.handleCourseConceptsForPath())
}

func (d Deps) requireCourseItemCreate(w http.ResponseWriter, r *http.Request) (courseCode string, viewer uuid.UUID, ok bool) {
	courseCode, viewer, ok = d.requireCourseAccess(w, r)
	if !ok {
		return "", uuid.UUID{}, false
	}
	canEdit, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return "", uuid.UUID{}, false
	}
	if !canEdit {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Instructor permission required.")
		return "", uuid.UUID{}, false
	}
	return courseCode, viewer, true
}

func structurePathRuleToAPI(r adaptivepath.StructurePathRuleRow) apmodel.StructurePathRuleResponse {
	conceptIDs := r.ConceptIDs
	if conceptIDs == nil {
		conceptIDs = []uuid.UUID{}
	}
	return apmodel.StructurePathRuleResponse{
		ID:              r.ID,
		StructureItemID: r.StructureItemID,
		RuleType:        r.RuleType,
		ConceptIDs:      conceptIDs,
		Threshold:       r.Threshold,
		TargetItemID:    r.TargetItemID,
		Priority:        r.Priority,
		CreatedAt:       r.CreatedAt,
	}
}

// handleListStructurePathRules is GET /api/v1/courses/{course_code}/structure/items/{item_id}/path-rules
func (d Deps) handleListStructurePathRules() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		courseCode, _, ok := d.requireCourseItemCreate(w, r)
		if !ok {
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
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
		host, err := coursestructure.GetItemRow(r.Context(), d.Pool, *cid, itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load structure item.")
			return
		}
		if host == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Structure item not found.")
			return
		}
		rows, err := adaptivepath.ListRulesForStructureItem(r.Context(), d.Pool, *cid, itemID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list path rules.")
			return
		}
		out := make([]apmodel.StructurePathRuleResponse, 0, len(rows))
		for _, row := range rows {
			out = append(out, structurePathRuleToAPI(row))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

// handleCreateStructurePathRule is POST /api/v1/courses/{course_code}/structure/items/{item_id}/path-rules
func (d Deps) handleCreateStructurePathRule() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		courseCode, _, ok := d.requireCourseItemCreate(w, r)
		if !ok {
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}
		var body apmodel.CreateStructurePathRuleRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if err := apsvc.ValidateThreshold(body.Threshold); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		if err := apsvc.ValidateRuleType(body.RuleType); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		if err := apsvc.RequireTargetForRuleType(body.RuleType, body.TargetItemID); err != nil {
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
		if err := apsvc.ValidateConceptsForCourse(r.Context(), d.Pool, *cid, body.ConceptIDs); err != nil {
			if errors.Is(err, apsvc.ErrEmptyConceptIDs) || errors.Is(err, apsvc.ErrUnknownConcepts) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to validate concepts.")
			return
		}
		if err := apsvc.ValidateRuleTargetsInCourse(r.Context(), d.Pool, *cid, itemID, body.TargetItemID); err != nil {
			if errors.Is(err, apsvc.ErrHostNotInCourse) {
				apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Structure item not found.")
				return
			}
			if errors.Is(err, apsvc.ErrTargetNotInCourse) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to validate structure items.")
			return
		}
		priority := int16(0)
		if body.Priority != nil {
			priority = *body.Priority
		}
		row, err := adaptivepath.InsertRule(
			r.Context(), d.Pool, itemID, body.RuleType, body.ConceptIDs, body.Threshold, body.TargetItemID, priority,
		)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create path rule.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(structurePathRuleToAPI(*row))
	}
}

// handleDeleteStructurePathRule is DELETE /api/v1/courses/{course_code}/structure/items/{item_id}/path-rules/{rule_id}
func (d Deps) handleDeleteStructurePathRule() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		courseCode, _, ok := d.requireCourseItemCreate(w, r)
		if !ok {
			return
		}
		ruleID, err := uuid.Parse(chi.URLParam(r, "rule_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid rule id.")
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
		deleted, err := adaptivepath.DeleteRuleForCourse(r.Context(), d.Pool, *cid, ruleID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to delete path rule.")
			return
		}
		if !deleted {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Path rule not found.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

type pathConceptOption struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Slug string    `json:"slug"`
}

// handleCourseConceptsForPath is GET /api/v1/courses/{course_code}/concepts-for-path
func (d Deps) handleCourseConceptsForPath() http.HandlerFunc {
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
		rows, err := concepts.ListConceptsForCourse(r.Context(), d.Pool, *cid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to list concepts.")
			return
		}
		out := make([]pathConceptOption, 0, len(rows))
		for _, row := range rows {
			out = append(out, pathConceptOption{ID: row.ID, Name: row.Name, Slug: row.Slug})
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}
