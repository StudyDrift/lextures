package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	lessongenerationjobs "github.com/lextures/lextures/server/internal/repos/lessongenerationjobs"
	"github.com/lextures/lextures/server/internal/repos/organization"
	"github.com/lextures/lextures/server/internal/repos/systemprompts"
	"github.com/lextures/lextures/server/internal/repos/userai"
	aigateway "github.com/lextures/lextures/server/internal/service/aigateway"
	"github.com/lextures/lextures/server/internal/service/lessonplanai"
)

func (d Deps) lessonGeneratorEnabled() bool {
	return d.effectiveConfig().FFLessonGenerator
}

func (d Deps) requireLessonGenerator(w http.ResponseWriter) bool {
	if !d.lessonGeneratorEnabled() {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Lesson generator is not enabled.")
		return false
	}
	return true
}

func (d Deps) loadLessonPrompts(ctx context.Context) lessonplanai.Prompts {
	fallback := lessonplanai.Prompts{
		LessonPlan: lessonplanai.DefaultLessonPlanPrompt,
		Activity:   lessonplanai.DefaultActivityPrompt,
		Quiz:       lessonplanai.DefaultQuizPrompt,
		Rubric:     lessonplanai.DefaultRubricPrompt,
	}
	if d.Pool == nil {
		return fallback
	}
	if s, err := systemprompts.GetByKey(ctx, d.Pool, "lesson_generation_plan"); err == nil && strings.TrimSpace(s) != "" {
		fallback.LessonPlan = s
	}
	if s, err := systemprompts.GetByKey(ctx, d.Pool, "lesson_generation_activity"); err == nil && strings.TrimSpace(s) != "" {
		fallback.Activity = s
	}
	if s, err := systemprompts.GetByKey(ctx, d.Pool, "quiz_generation"); err == nil && strings.TrimSpace(s) != "" {
		fallback.Quiz = s
	}
	if s, err := systemprompts.GetByKey(ctx, d.Pool, "lesson_generation_rubric"); err == nil && strings.TrimSpace(s) != "" {
		fallback.Rubric = s
	}
	return fallback
}

// handlePostLessonGenerator is POST /api/v1/courses/{course_code}/lesson-generator
func (d Deps) handlePostLessonGenerator() http.HandlerFunc {
	type resp struct {
		JobID string `json:"job_id"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireLessonGenerator(w) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		isStaff, err := enrollment.UserIsCourseStaff(r.Context(), d.Pool, courseCode, viewer)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify access.")
			return
		}
		if !isStaff {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Forbidden.")
			return
		}
		var input lessonplanai.InputParams
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if err := lessonplanai.ValidateInput(input); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		inputRaw, err := json.Marshal(lessonplanai.RedactInput(input))
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to encode input.")
			return
		}
		jobID, err := lessongenerationjobs.Create(r.Context(), d.Pool, viewer, *cid, inputRaw)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create generation job.")
			return
		}
		go d.runLessonGenerationJob(jobID, viewer, *cid, courseCode, input)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(resp{JobID: jobID.String()})
	}
}

func (d Deps) runLessonGenerationJob(jobID, viewer, courseID uuid.UUID, courseCode string, input lessonplanai.InputParams) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_ = lessongenerationjobs.MarkProcessing(ctx, d.Pool, jobID)

	or := d.openRouterClient()
	if or == nil {
		_ = lessongenerationjobs.MarkFailed(ctx, d.Pool, jobID, "AI generation is not configured.")
		return
	}
	model, err := userai.GetCourseSetupModelID(ctx, d.Pool, viewer)
	if err != nil {
		model = userai.DefaultCourseSetupModelID
	}
	promptMaterial := input.LearningObjective + input.Subject + input.GradeLevel
	if msg, blocked := d.evaluateAIGatewayBlock(ctx, viewer, aigateway.FeatureLessonGeneration, model, promptMaterial); blocked {
		_ = lessongenerationjobs.MarkFailed(ctx, d.Pool, jobID, msg)
		return
	}

	keys := lessonplanai.BuildComponentKeys(input.DifferentiationLevels)
	pkg := lessonplanai.NewPackage(jobID.String(), keys)
	pkg = lessonplanai.Generate(ctx, or, input, pkg, lessonplanai.GenerateOptions{
		ModelID: model,
		Prompts: d.loadLessonPrompts(ctx),
	})

	d.logAIInferenceAllowedBackground(ctx, viewer, aigateway.FeatureLessonGeneration, model, promptMaterial)

	raw, err := lessonplanai.MarshalPackage(pkg)
	if err != nil {
		_ = lessongenerationjobs.MarkFailed(ctx, d.Pool, jobID, "Failed to store generation result.")
		return
	}
	_ = lessongenerationjobs.SaveResult(ctx, d.Pool, jobID, raw)
}

func (d Deps) logAIInferenceAllowedBackground(ctx context.Context, userID uuid.UUID, feature, modelID, contentForHash string) {
	if d.Pool == nil || !d.effectiveConfig().AiDisclosureEnabled {
		return
	}
	var orgID *uuid.UUID
	if oid, err := organization.OrgIDForUser(ctx, d.Pool, userID); err == nil {
		orgID = &oid
	}
	dec := aigateway.Decision{
		UserIDHash:     aigateway.UserIDHash(d.aiGatewayConfig().HMACSecret, userID),
		OptInConfirmed: true,
	}
	_ = aigateway.LogInference(ctx, d.Pool, orgID, dec, feature, modelID, aigateway.ProviderOpenRouter, aigateway.ContentHash(contentForHash), false)
}

// handleGetLessonGeneratorJob is GET /api/v1/courses/{course_code}/lesson-generator/{job_id}
func (d Deps) handleGetLessonGeneratorJob() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireLessonGenerator(w) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		jobID, err := uuid.Parse(chi.URLParam(r, "job_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid job id.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		row, err := lessongenerationjobs.GetByID(r.Context(), d.Pool, viewer, *cid, jobID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load job.")
			return
		}
		if row == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Job not found.")
			return
		}
		var pkg lessonplanai.PackageResult
		if len(row.Result) > 0 {
			pkg, err = lessonplanai.UnmarshalPackage(row.Result)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to parse job result.")
				return
			}
		} else {
			var input lessonplanai.InputParams
			_ = json.Unmarshal(row.InputParams, &input)
			keys := lessonplanai.BuildComponentKeys(input.DifferentiationLevels)
			pkg = lessonplanai.NewPackage(jobID.String(), keys)
			pkg.Status = row.Status
		}
		if pkg.JobID == "" {
			pkg.JobID = jobID.String()
		}
		if pkg.Status == "" {
			pkg.Status = row.Status
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(pkg)
	}
}

// handlePostLessonGeneratorRegenerate is POST .../regenerate-component
func (d Deps) handlePostLessonGeneratorRegenerate() http.HandlerFunc {
	type body struct {
		Component string `json:"component"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireLessonGenerator(w) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		isStaff, err := enrollment.UserIsCourseStaff(r.Context(), d.Pool, courseCode, viewer)
		if err != nil || !isStaff {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Forbidden.")
			return
		}
		jobID, err := uuid.Parse(chi.URLParam(r, "job_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid job id.")
			return
		}
		var req body
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		component := strings.TrimSpace(req.Component)
		if component == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "component is required.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		row, err := lessongenerationjobs.GetByID(r.Context(), d.Pool, viewer, *cid, jobID)
		if err != nil || row == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Job not found.")
			return
		}
		var input lessonplanai.InputParams
		if err := json.Unmarshal(row.InputParams, &input); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid stored input.")
			return
		}
		pkg, err := lessonplanai.UnmarshalPackage(row.Result)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to parse job result.")
			return
		}
		or := d.openRouterClient()
		if or == nil {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeInternal, "AI generation is not configured.")
			return
		}
		model, mErr := userai.GetCourseSetupModelID(r.Context(), d.Pool, viewer)
		if mErr != nil {
			model = userai.DefaultCourseSetupModelID
		}
		promptMaterial := input.LearningObjective + input.Subject
		if !d.enforceAIGateway(w, r, viewer, aigateway.FeatureLessonGeneration, model, promptMaterial) {
			return
		}
		pkg = lessonplanai.Generate(r.Context(), or, input, pkg, lessonplanai.GenerateOptions{
			ModelID:  model,
			Prompts:  d.loadLessonPrompts(r.Context()),
			OnlyKeys: []string{component},
		})
		raw, err := lessonplanai.MarshalPackage(pkg)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to store result.")
			return
		}
		_ = lessongenerationjobs.SaveResult(r.Context(), d.Pool, jobID, raw)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(pkg)
	}
}

// handlePostLessonGeneratorSave is POST .../save-to-course
func (d Deps) handlePostLessonGeneratorSave() http.HandlerFunc {
	type body struct {
		AcceptedComponents []string                       `json:"accepted_components"`
		ModuleTitle        string                         `json:"module_title"`
		ComponentEdits     map[string]json.RawMessage     `json:"component_edits"`
	}
	type resp struct {
		ModuleID string `json:"module_id"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.requireLessonGenerator(w) {
			return
		}
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		isStaff, err := enrollment.UserIsCourseStaff(r.Context(), d.Pool, courseCode, viewer)
		if err != nil || !isStaff {
			apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Forbidden.")
			return
		}
		jobID, err := uuid.Parse(chi.URLParam(r, "job_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid job id.")
			return
		}
		var req body
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		cid, err := course.GetIDByCourseCode(r.Context(), d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		row, err := lessongenerationjobs.GetByID(r.Context(), d.Pool, viewer, *cid, jobID)
		if err != nil || row == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Job not found.")
			return
		}
		pkg, err := lessonplanai.UnmarshalPackage(row.Result)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to parse job result.")
			return
		}
		result, err := lessonplanai.SaveToCourse(r.Context(), d.Pool, lessonplanai.SaveAcceptedOptions{
			CourseID:       *cid,
			ModuleTitle:    req.ModuleTitle,
			AcceptedKeys:   req.AcceptedComponents,
			Components:     pkg.Components,
			ComponentEdits: req.ComponentEdits,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp{ModuleID: result.ModuleID.String()})
	}
}
