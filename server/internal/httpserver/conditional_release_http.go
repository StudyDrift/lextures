package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/models/conditionalrelease"
	"github.com/lextures/lextures/server/internal/repos/adminaudit"
	"github.com/lextures/lextures/server/internal/repos/course"
	crrepo "github.com/lextures/lextures/server/internal/repos/conditionalrelease"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/repos/learnerprogress"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/service/competencygating"
)

func (d Deps) conditionalReleaseFeatureOff(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFConditionalRelease {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Conditional release is not enabled.")
		return true
	}
	return false
}

func (d Deps) gatingService() competencygating.Service {
	return competencygating.New(d.Pool)
}

type moduleRequirementsBody struct {
	CompletionMode      string   `json:"completionMode"`
	PrerequisiteModules []string `json:"prerequisiteModuleIds"`
	UnlockAt            *string  `json:"unlockAt"`
}

type itemCompletionRuleBody struct {
	RuleType  string   `json:"ruleType"`
	Threshold *float64 `json:"threshold"`
}

type unlockOverrideBody struct {
	EnrollmentID string `json:"enrollmentId"`
}

func (d Deps) registerConditionalReleaseRoutes(r chi.Router) {
	r.Put("/api/v1/courses/{course_code}/structure/modules/{module_id}/requirements", d.handlePutModuleRequirements())
	r.Put("/api/v1/courses/{course_code}/items/{item_id}/completion-rule", d.handlePutItemCompletionRule())
	r.Delete("/api/v1/courses/{course_code}/items/{item_id}/completion-rule", d.handleDeleteItemCompletionRule())
	r.Get("/api/v1/courses/{course_code}/modules/progress", d.handleGetModulesProgress())
	r.Get("/api/v1/courses/{course_code}/requirements/report", d.handleGetRequirementsReport())
	r.Post("/api/v1/courses/{course_code}/structure/modules/{module_id}/unlock-override", d.handlePostModuleUnlockOverride())
}

func (d Deps) handlePutModuleRequirements() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if d.conditionalReleaseFeatureOff(w) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		perm := "course:" + courseCode + ":item:create"
		canEdit, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, perm)
		if err != nil || !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Instructor permission required.")
			return
		}
		moduleID, err := uuid.Parse(chi.URLParam(r, "module_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid module id.")
			return
		}
		var body moduleRequirementsBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		mode := conditionalrelease.CompletionMode(body.CompletionMode)
		switch mode {
		case conditionalrelease.CompletionAllItems, conditionalrelease.CompletionOneItem, conditionalrelease.CompletionSequentialOrder:
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid completionMode.")
			return
		}
		var unlockAt *time.Time
		if body.UnlockAt != nil && *body.UnlockAt != "" {
			t, err := time.Parse(time.RFC3339, *body.UnlockAt)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid unlockAt.")
				return
			}
			utc := t.UTC()
			unlockAt = &utc
		}
		var prereqIDs []uuid.UUID
		for _, s := range body.PrerequisiteModules {
			id, err := uuid.Parse(s)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid prerequisiteModuleIds.")
				return
			}
			prereqIDs = append(prereqIDs, id)
		}
		svc := d.gatingService()
		if err := svc.SetModuleRequirements(r.Context(), moduleID, mode, unlockAt, prereqIDs); err != nil {
			switch {
			case errors.Is(err, competencygating.ErrSelfPrerequisite):
				apierr.WriteJSON(w, http.StatusUnprocessableEntity, apierr.CodeInvalidInput, err.Error())
			case errors.Is(err, competencygating.ErrCircularPrerequisite):
				apierr.WriteJSON(w, http.StatusUnprocessableEntity, apierr.CodeInvalidInput, err.Error())
			default:
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save module requirements.")
			}
			return
		}
		req, err := crrepo.GetModuleRequirement(r.Context(), d.Pool, moduleID)
		if err != nil || req == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load module requirements.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(req)
	}
}

func (d Deps) handlePutItemCompletionRule() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if d.conditionalReleaseFeatureOff(w) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		perm := "course:" + courseCode + ":item:create"
		canEdit, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, perm)
		if err != nil || !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Instructor permission required.")
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}
		var body itemCompletionRuleBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		ruleType := conditionalrelease.RuleType(body.RuleType)
		switch ruleType {
		case conditionalrelease.RuleMustView, conditionalrelease.RuleMustMarkDone, conditionalrelease.RuleMustSubmit,
			conditionalrelease.RuleMustScoreAtLeast, conditionalrelease.RuleMustContribute:
		default:
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid ruleType.")
			return
		}
		if ruleType == conditionalrelease.RuleMustScoreAtLeast && (body.Threshold == nil || *body.Threshold < 0 || *body.Threshold > 100) {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "threshold required for must_score_at_least.")
			return
		}
		if err := crrepo.UpsertItemRule(r.Context(), d.Pool, itemID, ruleType, body.Threshold); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save item rule.")
			return
		}
		rule, err := crrepo.GetItemRule(r.Context(), d.Pool, itemID)
		if err != nil || rule == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load item rule.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(rule)
	}
}

func (d Deps) handleDeleteItemCompletionRule() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if d.conditionalReleaseFeatureOff(w) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		perm := "course:" + courseCode + ":item:create"
		canEdit, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, perm)
		if err != nil || !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Instructor permission required.")
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}
		if err := crrepo.DeleteItemRule(r.Context(), d.Pool, itemID); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to delete item rule.")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) handleGetModulesProgress() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if d.conditionalReleaseFeatureOff(w) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		enrollmentID, err := enrollment.GetStudentEnrollmentID(r.Context(), d.Pool, *cid, viewer)
		if err != nil || enrollmentID == nil {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Student enrollment required.")
			return
		}
		snap, err := d.gatingService().BuildStudentProgress(r.Context(), *cid, *enrollmentID, time.Now().UTC())
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load progress.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(snap)
	}
}

func (d Deps) handleGetRequirementsReport() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if d.conditionalReleaseFeatureOff(w) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		perm := "course:" + courseCode + ":item:create"
		canEdit, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, perm)
		if err != nil || !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Instructor permission required.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		rows, err := crrepo.ListRequirementsReport(r.Context(), d.Pool, *cid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load report.")
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"rows": rows})
	}
}

func (d Deps) handlePostModuleUnlockOverride() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if d.conditionalReleaseFeatureOff(w) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		perm := "course:" + courseCode + ":item:create"
		canEdit, err := rbac.UserHasPermission(r.Context(), d.Pool, viewer, perm)
		if err != nil || !canEdit {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Instructor permission required.")
			return
		}
		moduleID, err := uuid.Parse(chi.URLParam(r, "module_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid module id.")
			return
		}
		var body unlockOverrideBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		enrollmentID, err := uuid.Parse(body.EnrollmentID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid enrollmentId.")
			return
		}
		svc := d.gatingService()
		if err := svc.GrantUnlockOverride(r.Context(), enrollmentID, moduleID, viewer); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to grant override.")
			return
		}
		if d.effectiveConfig().AdminAuditLogEnabled {
			targetType := "module_unlock_override"
			after, _ := json.Marshal(map[string]string{
				"enrollmentId": enrollmentID.String(),
				"moduleId":     moduleID.String(),
			})
			_, _, _ = adminaudit.Insert(r.Context(), d.Pool, adminaudit.InsertParams{
				EventType:   "conditional_release.override",
				ActorID:     viewer,
				TargetType:  &targetType,
				TargetID:    &moduleID,
				AfterValue:  after,
			})
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// enforceConditionalRelease blocks student access to locked items when the feature is enabled.
// Instructors (canEdit) bypass gating. Returns true when the request may proceed.
func (d Deps) enforceConditionalRelease(
	w http.ResponseWriter, r *http.Request, courseID uuid.UUID, viewer uuid.UUID, itemID uuid.UUID, canEdit bool,
) bool {
	if canEdit || !d.effectiveConfig().FFConditionalRelease {
		return true
	}
	hasReq, err := crrepo.CourseHasRequirements(r.Context(), d.Pool, courseID)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to check gating.")
		return false
	}
	if !hasReq {
		return true
	}
	enrollmentID, err := enrollment.GetStudentEnrollmentID(r.Context(), d.Pool, courseID, viewer)
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load enrollment.")
		return false
	}
	if enrollmentID == nil {
		return true
	}
	allowed, reason, err := d.gatingService().CheckItemAccess(r.Context(), courseID, *enrollmentID, viewer, itemID, time.Now().UTC())
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to check gating.")
		return false
	}
	if allowed {
		return true
	}
	payload := map[string]any{"reason": reason}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(payload)
	return false
}

// recordConditionalReleaseProgress evaluates item rules after learner actions.
func (d Deps) recordConditionalReleaseProgress(r *http.Request, courseID, userID, itemID uuid.UUID) {
	if !d.effectiveConfig().FFConditionalRelease {
		return
	}
	enrollmentID, err := enrollment.GetStudentEnrollmentID(r.Context(), d.Pool, courseID, userID)
	if err != nil || enrollmentID == nil {
		return
	}
	_, _ = d.gatingService().EvaluateAndPersistItemRule(r.Context(), courseID, *enrollmentID, userID, itemID)
}

// recordConditionalReleaseView marks must_view progress when a student opens content.
func (d Deps) recordConditionalReleaseView(r *http.Request, courseID, userID, itemID uuid.UUID) {
	if !d.effectiveConfig().FFConditionalRelease {
		return
	}
	enrollmentID, err := enrollment.GetStudentEnrollmentID(r.Context(), d.Pool, courseID, userID)
	if err != nil || enrollmentID == nil {
		return
	}
	_ = learnerprogress.MarkVisited(r.Context(), d.Pool, *enrollmentID, itemID)
	d.recordConditionalReleaseProgress(r, courseID, userID, itemID)
}
