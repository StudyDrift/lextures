package httpserver

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/models/coursemodulequiz"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursemodulequizzes"
	"github.com/lextures/lextures/server/internal/repos/questionbank"
	"github.com/lextures/lextures/server/internal/repos/quizattempts"
	"github.com/lextures/lextures/server/internal/service/codeexecution"
)

func (d Deps) registerQuizCodeRunRoutes(r chi.Router) {
	r.Post(
		"/api/v1/courses/{course_code}/quizzes/{item_id}/attempts/{attempt_id}/questions/{question_id}/run",
		d.handleQuizQuestionRun(),
	)
}

type quizCodeRunRequest struct {
	Code       string `json:"code"`
	LanguageID *int   `json:"languageId"`
}

type quizCodeRunResultJSON struct {
	Status         string  `json:"status"`
	Passed         bool    `json:"passed"`
	ActualOutput   string  `json:"actualOutput"`
	ExpectedOutput string  `json:"expectedOutput"`
	Stderr         *string `json:"stderr,omitempty"`
	ExecutionMs    *int    `json:"executionMs,omitempty"`
	MemoryKb       *int    `json:"memoryKb,omitempty"`
}

type quizCodeRunResponseJSON struct {
	QuestionID     string                  `json:"questionId"`
	Results        []quizCodeRunResultJSON `json:"results"`
	PointsEarned   float64                 `json:"pointsEarned"`
	PointsPossible float64                 `json:"pointsPossible"`
}

func (d Deps) handleQuizQuestionRun() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		courseCode, viewer, ok := d.requireCourseAccess(w, r)
		if !ok {
			return
		}
		cfg := d.effectiveConfig()
		if !cfg.CodeExecutionEnabled {
			apierr.WriteJSON(w, http.StatusServiceUnavailable, apierr.CodeNotImplemented, "Code execution is not enabled.")
			return
		}
		itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid item id.")
			return
		}
		attemptID, err := uuid.Parse(chi.URLParam(r, "attempt_id"))
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid attempt id.")
			return
		}
		questionID := strings.TrimSpace(chi.URLParam(r, "question_id"))
		if questionID == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid question id.")
			return
		}

		var body quizCodeRunRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Invalid JSON body.")
			return
		}
		code := strings.TrimSpace(body.Code)
		if code == "" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "code is required.")
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
		attempt, err := quizattempts.GetAttempt(ctx, d.Pool, attemptID)
		if err != nil || attempt == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Attempt not found.")
			return
		}
		if attempt.StudentUserID != viewer || attempt.StructureItemID != itemID || attempt.CourseID != *cid {
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

		attemptPtr := &attemptID
		questions, _, err := questionbank.ResolveDeliveryQuestionsForGet(
			ctx, d.Pool, *cid, itemID, meta.QuestionBankEnabled, row.Questions, attemptPtr, false,
		)
		if err != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, err.Error())
			return
		}

		var question *coursemodulequiz.QuizQuestion
		for i := range questions {
			if questions[i].ID == questionID {
				question = &questions[i]
				break
			}
		}
		if question == nil {
			apierr.WriteJSON(w, http.StatusNotFound, apierr.CodeNotFound, "Question not found.")
			return
		}
		if question.QuestionType != "code" {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "Question is not a code question.")
			return
		}

		runtime, allTests := parseQuizCodeTypeConfig(question.TypeConfig)
		publicTests := filterPublicCodeTests(allTests)
		if len(publicTests) == 0 {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, "No public test cases are configured for this question.")
			return
		}

		runner := codeexecution.New()
		resp, runErr := runner.RunTests(ctx, codeexecution.RunRequest{
			Runtime: runtime,
			Code:    code,
			Tests:   publicTests,
		})
		if runErr != nil {
			apierr.WriteJSON(w, http.StatusBadRequest, apierr.CodeInvalidInput, runErr.Error())
			return
		}

		results := make([]quizCodeRunResultJSON, 0, len(resp.Results))
		var earned, possible float64
		points := float64(question.Points)
		if points <= 0 {
			points = 1
		}
		possible = points
		perTest := points / float64(len(publicTests))
		for _, tr := range resp.Results {
			item := quizCodeRunResultJSON{
				Status:         tr.Status,
				Passed:         tr.Passed,
				ActualOutput:   tr.ActualOutput,
				ExpectedOutput: tr.ExpectedOutput,
			}
			if tr.Stderr != "" {
				stderr := tr.Stderr
				item.Stderr = &stderr
			}
			if tr.ExecutionMs > 0 {
				ms := tr.ExecutionMs
				item.ExecutionMs = &ms
			}
			results = append(results, item)
			if tr.Passed {
				earned += perTest
			}
		}
		earned = math.Round(earned*100) / 100

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(quizCodeRunResponseJSON{
			QuestionID:     questionID,
			Results:        results,
			PointsEarned:   earned,
			PointsPossible: possible,
		})
	}
}

func parseQuizCodeTypeConfig(raw json.RawMessage) (runtime string, tests []codeexecution.TestCase) {
	runtime = "python3"
	if len(raw) == 0 {
		return runtime, nil
	}
	var cfg map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return runtime, nil
	}
	if lang, ok := cfg["language"].(string); ok && strings.TrimSpace(lang) != "" {
		runtime = strings.TrimSpace(lang)
	}
	tests = parseQuizCodeTestCases(cfg["testCases"])
	return runtime, tests
}

func parseQuizCodeTestCases(raw any) []codeexecution.TestCase {
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]codeexecution.TestCase, 0, len(list))
	for i, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		tc := codeexecution.TestCase{ID: "t" + strconv.Itoa(i+1)}
		if id, ok := m["id"].(string); ok && strings.TrimSpace(id) != "" {
			tc.ID = strings.TrimSpace(id)
		}
		if v, ok := m["input"].(string); ok {
			tc.Input = v
		}
		if v, ok := m["expectedOutput"].(string); ok {
			tc.ExpectedOutput = v
		}
		if v, ok := m["isHidden"].(bool); ok {
			tc.IsHidden = v
		}
		if v, ok := m["timeLimitMs"].(float64); ok && v > 0 {
			tc.TimeLimitMs = int(v)
		}
		if v, ok := m["memoryLimitKb"].(float64); ok && v > 0 {
			tc.MemoryLimitKb = int(v)
		}
		out = append(out, tc)
	}
	return out
}

func filterPublicCodeTests(tests []codeexecution.TestCase) []codeexecution.TestCase {
	out := make([]codeexecution.TestCase, 0, len(tests))
	for _, tc := range tests {
		if !tc.IsHidden {
			out = append(out, tc)
		}
	}
	return out
}
