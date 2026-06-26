package httpserver

import (
	"encoding/json"
	"math"
	"net/http"
	"sort"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/courseroles"
	"github.com/lextures/lextures/server/internal/models/coursemodulequiz"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursegrades"
	"github.com/lextures/lextures/server/internal/repos/coursemodulequizzes"
	"github.com/lextures/lextures/server/internal/repos/quizattempts"
	"github.com/lextures/lextures/server/internal/service/quizattemptgrading"
)

func (d Deps) registerQuizGradingRoutes(r chi.Router) {
	r.Get("/api/v1/courses/{course_code}/quizzes/{item_id}/attempts/{attempt_id}/grading", d.handleQuizAttemptGradingGet())
	r.Put("/api/v1/courses/{course_code}/quizzes/{item_id}/attempts/{attempt_id}/grading", d.handleQuizAttemptGradingPut())
}

func (d Deps) requireQuizGrader(w http.ResponseWriter, r *http.Request) (viewer uuid.UUID, ok bool) {
	courseCode, viewer, ok := d.requireCourseAccess(w, r)
	if !ok {
		return uuid.Nil, false
	}
	has, err := courseroles.UserHasPermission(r.Context(), d.Pool, viewer, "course:"+courseCode+":gradebook:view")
	if err != nil {
		apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to verify permissions.")
		return uuid.Nil, false
	}
	if !has {
		apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "You do not have permission to grade quiz attempts.")
		return uuid.Nil, false
	}
	return viewer, true
}

func (d Deps) handleQuizAttemptGradingGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode := chi.URLParam(r, "course_code")
		if _, ok := d.requireQuizGrader(w, r); !ok {
			return
		}
		itemID, attemptID, attempt, quizRow, cid, ok := d.loadQuizAttemptForGrading(w, r, courseCode)
		if !ok {
			return
		}
		ctx := r.Context()
		responses, err := quizattempts.ListResponses(ctx, d.Pool, attemptID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load quiz responses.")
			return
		}

		needsManual, err := quizattempts.AttemptNeedsManualGrading(ctx, d.Pool, attemptID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to check grading status.")
			return
		}

		var studentName string
		err = d.Pool.QueryRow(ctx, `
SELECT COALESCE(NULLIF(TRIM(display_name), ''), email, 'Student')
FROM "user".users WHERE id = $1
`, attempt.StudentUserID).Scan(&studentName)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load student.")
			return
		}

		quizQuestions := []coursemodulequiz.QuizQuestion{}
		if quizRow != nil {
			quizQuestions = quizRow.Questions
		}
		questions := buildQuizGradingQuestions(responses, quizQuestions)

		out := coursemodulequiz.QuizAttemptGradingResponse{
			AttemptID:          attempt.ID,
			StudentUserID:      attempt.StudentUserID,
			StudentName:        studentName,
			AttemptNumber:      attempt.AttemptNumber,
			SubmittedAt:        attempt.SubmittedAt,
			NeedsManualGrading: needsManual,
			Questions:          questions,
		}
		if attempt.PointsEarned != nil && attempt.PointsPossible != nil && attempt.ScorePercent != nil {
			out.Score = &coursemodulequiz.QuizResultsScoreSummary{
				PointsEarned:   *attempt.PointsEarned,
				PointsPossible: *attempt.PointsPossible,
				ScorePercent:   *attempt.ScorePercent,
			}
		}
		_ = cid
		_ = itemID

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

func (d Deps) handleQuizAttemptGradingPut() http.HandlerFunc {
	type questionGrade struct {
		QuestionIndex int32   `json:"questionIndex"`
		PointsAwarded float64 `json:"pointsAwarded"`
	}
	type body struct {
		Questions []questionGrade `json:"questions"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.Header().Set("Allow", http.MethodPut)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		courseCode := chi.URLParam(r, "course_code")
		if _, ok := d.requireQuizGrader(w, r); !ok {
			return
		}
		itemID, attemptID, attempt, quizRow, cid, ok := d.loadQuizAttemptForGrading(w, r, courseCode)
		if !ok {
			return
		}
		if attempt.Status != "submitted" {
			apierr.WriteJSON(w, http.StatusConflict, apierr.CodeConflict, "Only submitted attempts can be graded.")
			return
		}

		var b body
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		if len(b.Questions) == 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "At least one question score is required.")
			return
		}

		ctx := r.Context()
		responses, err := quizattempts.ListResponses(ctx, d.Pool, attemptID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to load quiz responses.")
			return
		}
		byIndex := make(map[int32]quizattempts.ResponseRow, len(responses))
		for _, resp := range responses {
			byIndex[resp.QuestionIndex] = resp
		}

		tx, err := d.Pool.Begin(ctx)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save grades.")
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()

		for _, g := range b.Questions {
			resp, ok := byIndex[g.QuestionIndex]
			if !ok {
				if quizRow == nil || int(g.QuestionIndex) < 0 || int(g.QuestionIndex) >= len(quizRow.Questions) {
					apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Unknown question index.")
					return
				}
				q := quizRow.Questions[g.QuestionIndex]
				maxPts := float64(q.Points)
				if maxPts <= 0 {
					maxPts = 1
				}
				resp = quizattempts.ResponseRow{
					QuestionIndex:  g.QuestionIndex,
					QuestionID:     q.ID,
					QuestionType:   q.QuestionType,
					PromptSnapshot: q.Prompt,
					ResponseJSON:   json.RawMessage(`{}`),
					MaxPoints:      maxPts,
				}
			}
			pts := g.PointsAwarded
			if pts < 0 {
				pts = 0
			}
			if resp.MaxPoints > 0 && pts > resp.MaxPoints {
				pts = resp.MaxPoints
			}
			if math.IsNaN(pts) || math.IsInf(pts, 0) {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid points value.")
				return
			}
			if err := quizattempts.UpsertResponseManualGrade(ctx, tx, attemptID, resp, pts); err != nil {
				apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Could not update question score.")
				return
			}
		}

		earned, possible, score, err := quizattempts.UpdateAttemptScoreTotals(ctx, tx, attemptID)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to update attempt score.")
			return
		}

		if err := tx.Commit(ctx); err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to save grades.")
			return
		}

		policyPoints, ready, err := quizattempts.PolicyPointsForStudent(
			ctx, d.Pool, *cid, itemID, attempt.StudentUserID, quizRow.GradeAttemptPolicy,
		)
		if err != nil {
			apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to sync gradebook.")
			return
		}
		if ready {
			if err := coursegrades.UpsertCellWithFlags(
				ctx, d.Pool, *cid, attempt.StudentUserID, itemID, policyPoints, nil, nil, nil, "manual", false,
			); err != nil {
				apierr.WriteJSON(w, http.StatusInternalServerError, apierr.CodeInternal, "Failed to sync gradebook.")
				return
			}
		}

		needsManual, _ := quizattempts.AttemptNeedsManualGrading(ctx, d.Pool, attemptID)
		out := coursemodulequiz.QuizAttemptGradingSaveResponse{
			PointsEarned:       earned,
			PointsPossible:     possible,
			ScorePercent:       score,
			NeedsManualGrading: needsManual,
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(out)
	}
}

func buildQuizGradingQuestions(
	responses []quizattempts.ResponseRow,
	quizQuestions []coursemodulequiz.QuizQuestion,
) []coursemodulequiz.QuizGradingQuestion {
	byIndex := make(map[int32]quizattempts.ResponseRow, len(responses))
	for _, resp := range responses {
		byIndex[resp.QuestionIndex] = resp
	}

	indices := make([]int32, 0, len(responses)+len(quizQuestions))
	seen := make(map[int32]struct{}, len(responses)+len(quizQuestions))
	for idx := range byIndex {
		if _, ok := seen[idx]; !ok {
			indices = append(indices, idx)
			seen[idx] = struct{}{}
		}
	}
	for i := range quizQuestions {
		idx := int32(i)
		if _, ok := seen[idx]; ok {
			continue
		}
		indices = append(indices, idx)
		seen[idx] = struct{}{}
	}
	sort.Slice(indices, func(i, j int) bool { return indices[i] < indices[j] })

	out := make([]coursemodulequiz.QuizGradingQuestion, 0, len(indices))
	for _, idx := range indices {
		var quizDef *coursemodulequiz.QuizQuestion
		if int(idx) >= 0 && int(idx) < len(quizQuestions) {
			q := quizQuestions[idx]
			quizDef = &q
		}
		if resp, ok := byIndex[idx]; ok {
			needsQ := quizattemptgrading.ResponseNeedsManualGrading(
				resp.QuestionType, resp.IsCorrect, resp.PointsAwarded, resp.MaxPoints,
			)
			qid := resp.QuestionID
			choices := gradingChoicesForQuestion(quizDef)
			out = append(out, coursemodulequiz.QuizGradingQuestion{
				QuestionIndex:  resp.QuestionIndex,
				QuestionID:     &qid,
				QuestionType:   resp.QuestionType,
				PromptSnapshot: optionalNonEmptyString(resp.PromptSnapshot),
				Choices:        choices,
				ResponseJSON:   resp.ResponseJSON,
				IsCorrect:      resp.IsCorrect,
				PointsAwarded:  optionalFiniteFloat(resp.PointsAwarded),
				MaxPoints:      resp.MaxPoints,
				NeedsGrading:   needsQ,
			})
			continue
		}
		if quizDef == nil {
			continue
		}
		q := *quizDef
		qid := q.ID
		maxPts := float64(q.Points)
		if maxPts <= 0 {
			maxPts = 1
		}
		out = append(out, coursemodulequiz.QuizGradingQuestion{
			QuestionIndex:  idx,
			QuestionID:     &qid,
			QuestionType:   q.QuestionType,
			PromptSnapshot: optionalNonEmptyString(q.Prompt),
			Choices:        gradingChoicesForQuestion(&q),
			ResponseJSON:   json.RawMessage(`{}`),
			MaxPoints:      maxPts,
			NeedsGrading:   false,
		})
	}
	return out
}

func gradingChoicesForQuestion(q *coursemodulequiz.QuizQuestion) []string {
	if q == nil || len(q.Choices) == 0 {
		return nil
	}
	return q.Choices
}

func (d Deps) loadQuizAttemptForGrading(
	w http.ResponseWriter,
	r *http.Request,
	courseCode string,
) (itemID, attemptID uuid.UUID, attempt *quizattempts.AttemptResultRow, quizRow *coursemodulequizzes.CourseItemQuizRow, cid *uuid.UUID, ok bool) {
	itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
		return
	}
	attemptID, err = uuid.Parse(chi.URLParam(r, "attempt_id"))
	if err != nil {
		apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid attempt id.")
		return
	}
	ctx := r.Context()
	cid, err = course.GetIDByCourseCode(ctx, d.Pool, courseCode)
	if err != nil || cid == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Course not found.")
		return
	}
	attempt, err = quizattempts.GetAttemptResult(ctx, d.Pool, attemptID)
	if err != nil || attempt == nil || attempt.StructureItemID != itemID || attempt.CourseID != *cid {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Attempt not found.")
		return
	}
	quizRow, err = coursemodulequizzes.GetForCourseItem(ctx, d.Pool, *cid, itemID)
	if err != nil || quizRow == nil {
		apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Quiz not found.")
		return
	}
	return itemID, attemptID, attempt, quizRow, cid, true
}