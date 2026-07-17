package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/repos/aiusage"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursemodulecontent"
	"github.com/lextures/lextures/server/internal/repos/quizgame"
	"github.com/lextures/lextures/server/internal/repos/systemprompts"
	tutorrepo "github.com/lextures/lextures/server/internal/repos/tutor"
	"github.com/lextures/lextures/server/internal/repos/userai"
	aigateway "github.com/lextures/lextures/server/internal/service/aigateway"
	"github.com/lextures/lextures/server/internal/service/aiprovider"
	"github.com/lextures/lextures/server/internal/service/quizgameai"
	"github.com/lextures/lextures/server/internal/telemetry"
)

type generateKitRequest struct {
	SourceType string                    `json:"sourceType"`
	SourceRef  map[string]any            `json:"sourceRef"`
	Params     quizgame.GenerationParams `json:"params"`
}

func (d Deps) iqAiGenerationOff(w http.ResponseWriter) bool {
	if !d.effectiveConfig().FFIqAiGeneration {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "AI quiz generation is not enabled.")
		return true
	}
	return false
}

func generationJobJSON(j quizgame.GenerationJob) map[string]any {
	out := map[string]any{
		"id":            j.ID,
		"kitId":         j.KitID,
		"courseId":      j.CourseID,
		"requestedBy":   j.RequestedBy,
		"sourceType":    j.SourceType,
		"sourceRef":     json.RawMessage(j.SourceRef),
		"params":        json.RawMessage(j.Params),
		"status":        j.Status,
		"provider":      j.Provider,
		"model":         j.Model,
		"usageId":       j.UsageID,
		"error":         j.Error,
		"resultSummary": nil,
		"progress":      j.Progress,
		"createdAt":     j.CreatedAt.UTC().Format(time.RFC3339),
		"startedAt":     nil,
		"completedAt":   nil,
	}
	if len(j.ResultSummary) > 0 {
		out["resultSummary"] = json.RawMessage(j.ResultSummary)
	}
	if j.StartedAt != nil {
		out["startedAt"] = j.StartedAt.UTC().Format(time.RFC3339)
	}
	if j.CompletedAt != nil {
		out["completedAt"] = j.CompletedAt.UTC().Format(time.RFC3339)
	}
	return out
}

func (d Deps) loadLiveQuizGenPrompt(ctx context.Context) string {
	if d.Pool == nil {
		return quizgameai.DefaultPrompt
	}
	if s, err := systemprompts.GetByKey(ctx, d.Pool, quizgameai.PromptKey); err == nil && strings.TrimSpace(s) != "" {
		return s
	}
	return quizgameai.DefaultPrompt
}

func resolveSourceMaterial(ctx context.Context, d Deps, courseID uuid.UUID, sourceType string, ref map[string]any) (quizgameai.SourceMaterial, error) {
	src := quizgameai.SourceMaterial{SourceType: sourceType}
	str := func(key string) string {
		if ref == nil {
			return ""
		}
		v, ok := ref[key]
		if !ok || v == nil {
			return ""
		}
		s, _ := v.(string)
		return strings.TrimSpace(s)
	}
	switch sourceType {
	case quizgame.GenSourceTopic:
		src.Topic = str("topic")
		if src.Topic == "" {
			src.Topic = str("text")
		}
		if src.Topic == "" {
			return src, fmt.Errorf("topic is required")
		}
	case quizgame.GenSourcePassage:
		src.Passage = str("passage")
		if src.Passage == "" {
			src.Passage = str("text")
		}
		if src.Passage == "" {
			return src, fmt.Errorf("passage is required")
		}
		if len([]rune(src.Passage)) > 20000 {
			return src, fmt.Errorf("passage is too long")
		}
	case quizgame.GenSourceCourseContentRef:
		contentID := str("contentId")
		if contentID == "" {
			contentID = str("itemId")
		}
		if contentID == "" {
			return src, fmt.Errorf("contentId is required")
		}
		itemID, err := uuid.Parse(contentID)
		if err != nil {
			return src, fmt.Errorf("invalid contentId")
		}
		row, err := coursemodulecontent.GetForCourseItem(ctx, d.Pool, courseID, itemID)
		if err != nil {
			return src, fmt.Errorf("failed to load course content")
		}
		if row == nil || strings.TrimSpace(row.Markdown) == "" {
			return src, fmt.Errorf("course content not found or empty")
		}
		src.ContentID = contentID
		src.ContentTitle = row.Title
		src.Passage = row.Markdown
		if len([]rune(src.Passage)) > 20000 {
			runes := []rune(src.Passage)
			src.Passage = string(runes[:20000])
		}
	default:
		return src, fmt.Errorf("invalid sourceType")
	}
	return quizgameai.RedactSource(src), nil
}

func (d Deps) requireQuizKitAIWrite(w http.ResponseWriter, r *http.Request) (courseCode, kitID string, viewer uuid.UUID, ok bool) {
	courseCode, viewer, ok = d.requireCourseAccess(w, r)
	if !ok {
		return "", "", uuid.Nil, false
	}
	if d.interactiveQuizzesFeatureOff(w, r, courseCode) {
		return "", "", uuid.Nil, false
	}
	if d.iqAiGenerationOff(w) {
		return "", "", uuid.Nil, false
	}
	hasPerm, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":item:create")
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return "", "", uuid.Nil, false
	}
	if !hasPerm {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission for this action.")
		return "", "", uuid.Nil, false
	}
	kitID = chi.URLParam(r, "kit_id")
	if kitID == "" {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Missing kit id.")
		return "", "", uuid.Nil, false
	}
	return courseCode, kitID, viewer, true
}

// handlePostQuizKitGenerate is POST .../kits/{kit_id}/generate
func (d Deps) handlePostQuizKitGenerate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, kitID, viewer, ok := d.requireQuizKitAIWrite(w, r)
		if !ok {
			return
		}
		var body generateKitRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		body.SourceType = strings.TrimSpace(strings.ToLower(body.SourceType))
		if err := quizgame.NormalizeGenerationParams(&body.Params); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		cid, kid, err := quizgameKitCourseIDs(r.Context(), d, courseCode, kitID)
		if err != nil || kid == uuid.Nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Kit not found.")
			return
		}
		src, err := resolveSourceMaterial(r.Context(), d, cid, body.SourceType, body.SourceRef)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		if body.Params.LikeQuestionID != "" {
			lq, err := quizgame.GetQuestion(r.Context(), d.Pool, courseCode, kitID, body.Params.LikeQuestionID)
			if err != nil || lq == nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "likeQuestionId not found.")
				return
			}
			src.LikePrompt = lq.Prompt
			src.LikeType = lq.QuestionType
		}

		orgID := d.orgIDPtrForUser(r.Context(), viewer)
		if !d.aiConfigured(r.Context(), orgID) {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeAiNotConfigured, aiNotConfiguredMsg)
			return
		}
		if orgID != nil {
			budget, err := tutorrepo.GetTokenBudget(r.Context(), d.Pool, viewer, *orgID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load AI budget.")
				return
			}
			if budget.TokensUsed >= budget.TokenLimit {
				apierr.WriteJSON(w, http.StatusPaymentRequired, "BUDGET_EXCEEDED",
					fmt.Sprintf("You have reached your monthly AI interaction limit of %d tokens.", budget.TokenLimit))
				return
			}
		}
		active, err := quizgame.CountActiveGenerationJobs(r.Context(), d.Pool, cid)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to check generation queue.")
			return
		}
		if active >= quizgame.MaxActiveGenJobsPerCourse {
			apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeInvalidInput, "Too many active generation jobs for this course. Wait or cancel one.")
			return
		}
		if err := quizgame.CheckAIGenerationQuota(r.Context(), d.Pool, courseCode); err != nil {
			if errors.Is(err, quizgame.ErrAIGenerationQuota) {
				telemetry.RecordBusinessEvent("quizgame.quota.ai_generation_rejected")
				apierr.WriteJSON(w, http.StatusTooManyRequests, apierr.CodeRateLimited, "Daily AI generation budget for Live Quizzes has been reached.")
				return
			}
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Could not verify AI generation quota.")
			return
		}

		model, err := userai.GetCourseSetupModelID(r.Context(), d.Pool, viewer)
		if err != nil {
			model = userai.DefaultCourseSetupModelID
		}
		promptMaterial := src.Topic + src.Passage + src.ContentTitle
		if !d.enforceAIGateway(w, r, viewer, aigateway.FeatureLiveQuizKitGeneration, model, promptMaterial) {
			return
		}

		sourceRefRaw, _ := json.Marshal(map[string]any{
			"topic":        src.Topic,
			"passage":      truncateForStore(src.Passage, 4000),
			"contentId":    src.ContentID,
			"contentTitle": src.ContentTitle,
		})
		paramsRaw, _ := json.Marshal(body.Params)
		job, err := quizgame.CreateGenerationJob(r.Context(), d.Pool, quizgame.CreateGenerationJobInput{
			KitID:       kid,
			CourseID:    cid,
			RequestedBy: viewer,
			SourceType:  body.SourceType,
			SourceRef:   sourceRefRaw,
			Params:      paramsRaw,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create generation job.")
			return
		}
		telemetry.RecordBusinessEvent("quizgame.generation.created")
		go d.runQuizKitGenerationJob(job.ID, viewer, courseCode, kitID, src, body.Params)

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{"job": generationJobJSON(*job)})
	}
}

// handleGetQuizKitGenerateJob is GET .../kits/{kit_id}/generate/{job_id}
func (d Deps) handleGetQuizKitGenerateJob() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, _, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		if d.interactiveQuizzesFeatureOff(w, r, courseCode) {
			return
		}
		if d.iqAiGenerationOff(w) {
			return
		}
		kitID := chi.URLParam(r, "kit_id")
		jobID := chi.URLParam(r, "job_id")
		job, err := quizgame.GetGenerationJob(r.Context(), d.Pool, courseCode, kitID, jobID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load job.")
			return
		}
		if job == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Job not found.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"job": generationJobJSON(*job)})
	}
}

// handlePostQuizKitGenerateCancel is POST .../kits/{kit_id}/generate/{job_id}/cancel
func (d Deps) handlePostQuizKitGenerateCancel() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, kitID, _, ok := d.requireQuizKitAIWrite(w, r)
		if !ok {
			return
		}
		jobID := chi.URLParam(r, "job_id")
		job, err := quizgame.CancelGenerationJob(r.Context(), d.Pool, courseCode, kitID, jobID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to cancel job.")
			return
		}
		if job == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Job not found.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"job": generationJobJSON(*job)})
	}
}

// handlePostQuizQuestionRegenerate is POST .../questions/{qid}/regenerate
func (d Deps) handlePostQuizQuestionRegenerate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, kitID, viewer, ok := d.requireQuizKitAIWrite(w, r)
		if !ok {
			return
		}
		qid := chi.URLParam(r, "qid")
		existing, err := quizgame.GetQuestion(r.Context(), d.Pool, courseCode, kitID, qid)
		if err != nil || existing == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Question not found.")
			return
		}
		var body struct {
			SourceType string                    `json:"sourceType"`
			SourceRef  map[string]any            `json:"sourceRef"`
			Params     quizgame.GenerationParams `json:"params"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.SourceType == "" {
			body.SourceType = quizgame.GenSourceTopic
		}
		body.SourceType = strings.TrimSpace(strings.ToLower(body.SourceType))
		if body.SourceRef == nil {
			body.SourceRef = map[string]any{"topic": existing.Prompt}
		}
		body.Params.Count = 1
		body.Params.Types = []string{existing.QuestionType}
		body.Params.ReplaceQuestionID = qid
		body.Params.IncludeExplanations = true
		if body.Params.Difficulty == "" {
			body.Params.Difficulty = "medium"
		}
		if err := quizgame.NormalizeGenerationParams(&body.Params); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}
		cid, kid, err := quizgameKitCourseIDs(r.Context(), d, courseCode, kitID)
		if err != nil || kid == uuid.Nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Kit not found.")
			return
		}
		src, err := resolveSourceMaterial(r.Context(), d, cid, body.SourceType, body.SourceRef)
		if err != nil {
			// Fall back to topic from the existing prompt.
			src = quizgameai.RedactSource(quizgameai.SourceMaterial{
				SourceType: quizgame.GenSourceTopic,
				Topic:      existing.Prompt,
			})
		}
		src.LikePrompt = existing.Prompt
		src.LikeType = existing.QuestionType

		orgID := d.orgIDPtrForUser(r.Context(), viewer)
		if !d.aiConfigured(r.Context(), orgID) {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeAiNotConfigured, aiNotConfiguredMsg)
			return
		}
		if orgID != nil {
			budget, err := tutorrepo.GetTokenBudget(r.Context(), d.Pool, viewer, *orgID)
			if err == nil && budget.TokensUsed >= budget.TokenLimit {
				apierr.WriteJSON(w, http.StatusPaymentRequired, "BUDGET_EXCEEDED",
					fmt.Sprintf("You have reached your monthly AI interaction limit of %d tokens.", budget.TokenLimit))
				return
			}
		}
		model, _ := userai.GetCourseSetupModelID(r.Context(), d.Pool, viewer)
		if model == "" {
			model = userai.DefaultCourseSetupModelID
		}
		if !d.enforceAIGateway(w, r, viewer, aigateway.FeatureLiveQuizKitGeneration, model, existing.Prompt) {
			return
		}
		sourceRefRaw, _ := json.Marshal(map[string]any{"topic": src.Topic, "passage": truncateForStore(src.Passage, 4000)})
		paramsRaw, _ := json.Marshal(body.Params)
		job, err := quizgame.CreateGenerationJob(r.Context(), d.Pool, quizgame.CreateGenerationJobInput{
			KitID: kid, CourseID: cid, RequestedBy: viewer,
			SourceType: body.SourceType, SourceRef: sourceRefRaw, Params: paramsRaw,
		})
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to create generation job.")
			return
		}
		go d.runQuizKitGenerationJob(job.ID, viewer, courseCode, kitID, src, body.Params)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{"job": generationJobJSON(*job)})
	}
}

func (d Deps) runQuizKitGenerationJob(jobIDStr string, viewer uuid.UUID, courseCode, kitID string, src quizgameai.SourceMaterial, params quizgame.GenerationParams) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		return
	}
	orgID := d.orgIDPtrForUser(ctx, viewer)
	model, err := userai.GetCourseSetupModelID(ctx, d.Pool, viewer)
	if err != nil {
		model = userai.DefaultCourseSetupModelID
	}
	providerLabel := ""
	if providers := d.aiProvidersConfigured(ctx, orgID); len(providers) > 0 {
		providerLabel = providers[0]
	}
	_ = quizgame.MarkGenerationRunning(ctx, d.Pool, jobID, providerLabel, model)

	if canceled, _ := quizgame.IsGenerationCanceled(ctx, d.Pool, jobID); canceled {
		return
	}
	if !d.aiConfigured(ctx, orgID) {
		_ = quizgame.FailGenerationJob(ctx, d.Pool, jobID, aiNotConfiguredMsg)
		return
	}
	promptMaterial := src.Topic + src.Passage + src.ContentTitle
	if msg, blocked := d.evaluateAIGatewayBlock(ctx, viewer, aigateway.FeatureLiveQuizKitGeneration, model, promptMaterial); blocked {
		_ = quizgame.FailGenerationJob(ctx, d.Pool, jobID, msg)
		return
	}

	bound := aiprovider.BoundCompleter{Resolver: d.aiProviderResolver(), OrgID: orgID}
	_ = quizgame.SetGenerationProgress(ctx, d.Pool, jobID, 20)

	payload, meta, err := quizgameai.Generate(ctx, bound, src, params, quizgameai.GenerateOptions{
		ModelID: model,
		Prompt:  d.loadLiveQuizGenPrompt(ctx),
	})
	if err != nil {
		_ = quizgame.FailGenerationJob(ctx, d.Pool, jobID, "AI generation failed. Try again or author manually.")
		return
	}
	if canceled, _ := quizgame.IsGenerationCanceled(ctx, d.Pool, jobID); canceled {
		return
	}
	_ = quizgame.SetGenerationProgress(ctx, d.Pool, jobID, 70)

	parsed := quizgameai.ValidateAndFilter(payload.Questions, params.Types, params.IncludeExplanations, jobIDStr)
	summary := quizgame.ResultSummary{Repaired: parsed.Repaired, Dropped: parsed.Dropped}
	insertedIDs := make([]string, 0, len(parsed.Inputs))

	if params.ReplaceQuestionID != "" {
		if len(parsed.Inputs) == 0 {
			_ = quizgame.FailGenerationJob(ctx, d.Pool, jobID, "Generated question was invalid and was not applied.")
			return
		}
		q, err := quizgame.ReplaceQuestionContent(ctx, d.Pool, courseCode, kitID, params.ReplaceQuestionID, parsed.Inputs[0])
		if err != nil || q == nil {
			_ = quizgame.FailGenerationJob(ctx, d.Pool, jobID, "Failed to replace question.")
			return
		}
		summary.Inserted = 1
		insertedIDs = append(insertedIDs, q.ID)
	} else {
		for _, in := range parsed.Inputs {
			q, err := quizgame.CreateQuestion(ctx, d.Pool, courseCode, kitID, in)
			if err != nil || q == nil {
				summary.Dropped++
				continue
			}
			summary.Inserted++
			insertedIDs = append(insertedIDs, q.ID)
		}
	}
	summary.QuestionIDs = insertedIDs

	// Best-effort kit subject/grade tagging when the kit has none yet.
	if kid, err := uuid.Parse(kitID); err == nil {
		if s := strings.TrimSpace(payload.SuggestedSubject); s != "" {
			_, _ = d.Pool.Exec(ctx, `
UPDATE quizgame.kits SET subject = $2, updated_at = NOW()
WHERE id = $1 AND (subject IS NULL OR TRIM(subject) = '')
`, kid, s)
		}
		if g := strings.TrimSpace(payload.SuggestedGradeBand); g != "" {
			_, _ = d.Pool.Exec(ctx, `
UPDATE quizgame.kits SET grade_band = $2, updated_at = NOW()
WHERE id = $1 AND (grade_band IS NULL OR TRIM(grade_band) = '')
`, kid, g)
		}
	}

	entry := aiusage.EntryFromCallMeta(&viewer, courseIDPtr(courseCode, d, ctx), aigateway.FeatureLiveQuizKitGeneration, meta, meta.Usage, summary.Inserted > 0)
	_ = aiusage.Insert(ctx, d.Pool, entry)

	d.logAIInferenceAllowedBackground(ctx, viewer, aigateway.FeatureLiveQuizKitGeneration, model, promptMaterial, d.aiProvidersConfigured(ctx, orgID))

	if summary.Inserted == 0 {
		_ = quizgame.FailGenerationJob(ctx, d.Pool, jobID, "No valid questions could be generated. Try a clearer topic or passage.")
		return
	}
	_ = quizgame.CompleteGenerationJob(ctx, d.Pool, jobID, summary, nil)
}

func quizgameKitCourseIDs(ctx context.Context, d Deps, courseCode, kitID string) (courseID, kid uuid.UUID, err error) {
	cid, err := course.GetIDByCourseCode(ctx, d.Pool, courseCode)
	if err != nil || cid == nil {
		return uuid.Nil, uuid.Nil, err
	}
	kit, err := quizgame.Get(ctx, d.Pool, courseCode, kitID)
	if err != nil || kit == nil {
		return *cid, uuid.Nil, err
	}
	kid, err = uuid.Parse(kit.ID)
	return *cid, kid, err
}

func courseIDPtr(courseCode string, d Deps, ctx context.Context) *uuid.UUID {
	cid, err := course.GetIDByCourseCode(ctx, d.Pool, courseCode)
	if err != nil || cid == nil {
		return nil
	}
	return cid
}

func truncateForStore(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}
