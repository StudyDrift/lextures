package httpserver

import (
	"encoding/json"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/models/coursemodulequiz"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursemodulequizzes"
	"github.com/lextures/lextures/server/internal/repos/questionbank"
	"github.com/lextures/lextures/server/internal/repos/quizattempts"
	"github.com/lextures/lextures/server/internal/service/gamification"
	"github.com/lextures/lextures/server/internal/service/learningevents"
	"github.com/lextures/lextures/server/internal/service/quizattemptgrading"
)

func (d Deps) registerQuizSubmitRoutes(r chi.Router) {
	r.Post("/api/v1/courses/{course_code}/quizzes/{item_id}/submit", d.handleQuizSubmit())
	r.Get("/api/v1/courses/{course_code}/quizzes/{item_id}/results", d.handleQuizResults())
	r.Post("/api/v1/courses/{course_code}/quizzes/{item_id}/attempts/{attempt_id}/advance", d.handleQuizAdvance())
}

func (d Deps) handleQuizSubmit() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}
		var body coursemodulequiz.QuizSubmitRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if body.AttemptID == uuid.Nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "attemptId is required.")
			return
		}

		ctx := r.Context()
		cid, err := course.GetIDByCourseCode(ctx, d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}
		if !d.enforceConditionalReleaseForLearner(w, r, courseCode, *cid, viewer, itemID) {
			return
		}
		attempt, err := quizattempts.GetAttempt(ctx, d.Pool, body.AttemptID)
		if err != nil || attempt == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Attempt not found.")
			return
		}
		if attempt.StudentUserID != viewer || attempt.StructureItemID != itemID {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Attempt not found.")
			return
		}
		if attempt.CourseID != *cid {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Attempt not found.")
			return
		}
		if attempt.Status != "in_progress" {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Attempt is not in progress.")
			return
		}

		row, err := coursemodulequizzes.GetForCourseItem(ctx, d.Pool, *cid, itemID)
		if err != nil || row == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		meta, err := course.GetCourseQuizMeta(ctx, d.Pool, *cid)
		if err != nil || meta == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}

		lockdown := effectiveLockdownMode(meta.LockdownModeEnabled, row.LockdownMode)
		isAdaptive := row.IsAdaptive || len(body.AdaptiveHistory) > 0

		var graded []quizattemptgrading.GradedResponse
		var adaptiveJSON json.RawMessage
		if isAdaptive {
			graded = quizattemptgrading.GradeAdaptiveHistory(body.AdaptiveHistory)
			adaptiveJSON, _ = json.Marshal(body.AdaptiveHistory)
		} else if len(body.Responses) > 0 {
			attemptPtr := &body.AttemptID
			questions, _, err := questionbank.ResolveDeliveryQuestionsForGet(
				ctx, d.Pool, *cid, itemID, meta.QuestionBankEnabled, row.Questions, attemptPtr, false,
			)
			if err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
				return
			}
			graded = quizattemptgrading.GradeStaticResponses(questions, body.Responses)
		} else if lockdown != "standard" {
			existing, err := quizattempts.ListResponses(ctx, d.Pool, body.AttemptID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load responses.")
				return
			}
			for _, ex := range existing {
				graded = append(graded, quizattemptgrading.GradedResponse{
					QuestionIndex:  ex.QuestionIndex,
					QuestionID:     ex.QuestionID,
					QuestionType:   ex.QuestionType,
					PromptSnapshot: ex.PromptSnapshot,
					ResponseJSON:   ex.ResponseJSON,
					IsCorrect:      ex.IsCorrect,
					PointsAwarded:  ex.PointsAwarded,
					MaxPoints:      ex.MaxPoints,
					Locked:         ex.Locked,
				})
			}
		} else {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "responses are required.")
			return
		}

		earned, possible := quizattemptgrading.SumGradedPoints(graded)
		score := quizattemptgrading.ScorePercent(earned, possible)
		now := time.Now().UTC()

		integrityFlag := false
		if lockdown == "kiosk" && row.FocusLossThreshold != nil {
			count, err := quizattempts.CountFocusLossEvents(ctx, d.Pool, body.AttemptID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to check focus events.")
				return
			}
			if count > int64(*row.FocusLossThreshold) {
				integrityFlag = true
			}
		}

		responseRows := make([]quizattempts.ResponseRow, len(graded))
		for i, g := range graded {
			responseRows[i] = quizattempts.ResponseRow{
				QuestionIndex:  g.QuestionIndex,
				QuestionID:     g.QuestionID,
				QuestionType:   g.QuestionType,
				PromptSnapshot: g.PromptSnapshot,
				ResponseJSON:   g.ResponseJSON,
				IsCorrect:      g.IsCorrect,
				PointsAwarded:  g.PointsAwarded,
				MaxPoints:      g.MaxPoints,
				Locked:         g.Locked,
			}
		}

		tx, err := d.Pool.Begin(ctx)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to submit attempt.")
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()

		if err := quizattempts.ReplaceResponses(ctx, tx, body.AttemptID, responseRows); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save responses.")
			return
		}

		okFinalize, err := quizattempts.FinalizeAttemptSubmitted(ctx, tx, quizattempts.FinalizeSubmitParams{
			AttemptID:             body.AttemptID,
			SubmittedAt:           now,
			PointsEarned:          earned,
			PointsPossible:        possible,
			ScorePercent:          score,
			AcademicIntegrityFlag: integrityFlag,
			IsAdaptive:            isAdaptive,
			AdaptiveHistoryJSON:   adaptiveJSON,
		})
		if err != nil || !okFinalize {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Attempt could not be submitted.")
			return
		}
		if err := tx.Commit(ctx); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to submit attempt.")
			return
		}

		learningevents.EmitQuizGradedAsync(d.Pool, d.effectiveConfig(), body.AttemptID)
		if score >= 60 && cid != nil {
			gamification.EmitQuizPassed(d.Pool, d.effectiveConfig(), viewer, *cid, body.AttemptID)
		}

		out := coursemodulequiz.QuizSubmitResponse{
			AttemptID:      body.AttemptID,
			PointsEarned:   earned,
			PointsPossible: possible,
			ScorePercent:   score,
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

func (d Deps) handleQuizResults() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}
		ctx := r.Context()
		cid, err := course.GetIDByCourseCode(ctx, d.Pool, courseCode)
		if err != nil || cid == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
			return
		}

		attemptIDStr := strings.TrimSpace(r.URL.Query().Get("attemptId"))
		if attemptIDStr == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "attemptId is required.")
			return
		}
		attemptID, err := uuid.Parse(attemptIDStr)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid attempt id.")
			return
		}

		attempt, err := quizattempts.GetAttemptResult(ctx, d.Pool, attemptID)
		if err != nil || attempt == nil || attempt.StructureItemID != itemID || attempt.CourseID != *cid {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Attempt not found.")
			return
		}
		if attempt.StudentUserID != viewer {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Attempt not found.")
			return
		}

		row, err := coursemodulequizzes.GetForCourseItem(ctx, d.Pool, *cid, itemID)
		if err != nil || row == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}

		showScore := row.ShowScoreTiming == "immediate" || attempt.Status == "submitted"
		showQuestions := row.ReviewWhen == "always" || (row.ReviewWhen == "after_submit" && attempt.Status == "submitted")

		out := coursemodulequiz.QuizResultsResponse{
			AttemptID:             attempt.ID,
			AttemptNumber:         attempt.AttemptNumber,
			StartedAt:             attempt.StartedAt,
			AcademicIntegrityFlag: attempt.AcademicIntegrityFlag,
			ExtendedTimeActive:    attempt.ExtendedTimeApplied,
			SubmittedAt:           attempt.SubmittedAt,
			Status:                attempt.Status,
			IsAdaptive:            attempt.IsAdaptive,
		}

		if showScore && attempt.PointsEarned != nil && attempt.PointsPossible != nil && attempt.ScorePercent != nil {
			out.Score = &coursemodulequiz.QuizResultsScoreSummary{
				PointsEarned:   *attempt.PointsEarned,
				PointsPossible: *attempt.PointsPossible,
				ScorePercent:   *attempt.ScorePercent,
			}
		}

		if showQuestions {
			responses, err := quizattempts.ListResponses(ctx, d.Pool, attemptID)
			if err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load responses.")
				return
			}
			questions := make([]coursemodulequiz.QuizResultsQuestionResult, 0, len(responses))
			for _, resp := range responses {
				qid := resp.QuestionID
				questions = append(questions, coursemodulequiz.QuizResultsQuestionResult{
					QuestionIndex:  resp.QuestionIndex,
					QuestionID:     &qid,
					QuestionType:   resp.QuestionType,
					PromptSnapshot: optionalNonEmptyString(resp.PromptSnapshot),
					ResponseJSON:   resp.ResponseJSON,
					IsCorrect:      resp.IsCorrect,
					PointsAwarded:  optionalFiniteFloat(resp.PointsAwarded),
					MaxPoints:      resp.MaxPoints,
				})
			}
			out.Questions = questions
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

func optionalNonEmptyString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func optionalFiniteFloat(v float64) *float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return nil
	}
	return &v
}

func (d Deps) handleQuizAdvance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		itemID, attemptID, attempt, ok := d.parseQuizAttemptForViewer(w, r, courseCode, viewer)
		if !ok {
			return
		}
		if attempt.Status != "in_progress" {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Attempt is not in progress.")
			return
		}

		var body coursemodulequiz.QuizQuestionResponseItem
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if strings.TrimSpace(body.QuestionID) == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "questionId is required.")
			return
		}

		ctx := r.Context()
		cid, err := course.GetIDByCourseCode(ctx, d.Pool, courseCode)
		if err != nil || cid == nil || attempt.CourseID != *cid {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Attempt not found.")
			return
		}
		meta, err := course.GetCourseQuizMeta(ctx, d.Pool, *cid)
		if err != nil || meta == nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load course.")
			return
		}
		row, err := coursemodulequizzes.GetForCourseItem(ctx, d.Pool, *cid, itemID)
		if err != nil || row == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Not found.")
			return
		}
		lockdown := effectiveLockdownMode(meta.LockdownModeEnabled, row.LockdownMode)
		if lockdown == "standard" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "This quiz does not use lockdown delivery.")
			return
		}

		attemptPtr := &attemptID
		questions, _, err := questionbank.ResolveDeliveryQuestionsForGet(
			ctx, d.Pool, *cid, itemID, meta.QuestionBankEnabled, row.Questions, attemptPtr, false,
		)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}

		qIdx := int(attempt.CurrentQuestionIndex)
		if qIdx < 0 || qIdx >= len(questions) {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "No current question to advance.")
			return
		}
		q := questions[qIdx]
		if q.ID != body.QuestionID {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Question mismatch.")
			return
		}

		graded := quizattemptgrading.GradeResponseItem(q, body)
		graded.QuestionIndex = int32(qIdx)

		nextIdx := int32(qIdx + 1)
		completed := int(nextIdx) >= len(questions)

		tx, err := d.Pool.Begin(ctx)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save answer.")
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()

		if err := quizattempts.UpsertLockedResponse(ctx, tx, attemptID, int32(qIdx), quizattempts.ResponseRow{
			QuestionIndex:  graded.QuestionIndex,
			QuestionID:     graded.QuestionID,
			QuestionType:   graded.QuestionType,
			PromptSnapshot: graded.PromptSnapshot,
			ResponseJSON:   graded.ResponseJSON,
			IsCorrect:      graded.IsCorrect,
			PointsAwarded:  graded.PointsAwarded,
			MaxPoints:      graded.MaxPoints,
		}); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save answer.")
			return
		}
		if err := quizattempts.SetAttemptQuestionIndex(ctx, tx, attemptID, nextIdx); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to advance attempt.")
			return
		}
		if err := tx.Commit(ctx); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save answer.")
			return
		}

		out := coursemodulequiz.QuizAdvanceResponse{
			Locked:               true,
			CurrentQuestionIndex: nextIdx,
			Completed:            completed,
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}
