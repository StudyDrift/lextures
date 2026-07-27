package contenttools

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lextures/lextures/server/internal/service/contenttools/tools/inline_questions"
)

func init() {
	RegisterActionHandler(inline_questions.ID, "submit", handleInlineQuestionsSubmit)
	RegisterActionHandler(inline_questions.ID, "reveal", handleInlineQuestionsReveal)
}

func handleInlineQuestionsSubmit(ctx ActionContext) (*ActionResult, error) {
	cfg := inline_questions.ParseConfig(ctx.ConfigJSON)
	st := inline_questions.ParseState(ctx.StateJSON)

	var in struct {
		QuestionID string `json:"questionId"`
		Value      any    `json:"value"`
	}
	if len(ctx.Input) > 0 {
		if err := json.Unmarshal(ctx.Input, &in); err != nil {
			return nil, fmt.Errorf("invalid submit input: %w", err)
		}
	}
	in.QuestionID = strings.TrimSpace(in.QuestionID)
	if in.QuestionID == "" {
		return nil, fmt.Errorf("questionId is required")
	}
	q := inline_questions.FindQuestion(cfg, in.QuestionID)
	if q == nil {
		return nil, fmt.Errorf("unknown questionId")
	}
	if !inline_questions.QuestionUnlocked(cfg, st, in.QuestionID) {
		ObserveInlineQuestionsSubmit("sequential_locked")
		return &ActionResult{
			Result: map[string]any{
				"error":   "sequential_locked",
				"message": "Answer the previous question first.",
			},
		}, nil
	}

	remaining := inline_questions.AttemptsRemaining(cfg, st, in.QuestionID)
	if remaining == 0 {
		ObserveInlineQuestionsSubmit("max_attempts")
		return &ActionResult{
			Result: map[string]any{
				"error":             "max_attempts",
				"message":           "No attempts remaining for this question.",
				"attemptsRemaining": 0,
			},
		}, nil
	}

	value := in.Value
	if value == nil && st.Drafts != nil {
		value = st.Drafts[in.QuestionID]
	}
	if value == nil {
		return nil, fmt.Errorf("value is required")
	}

	// CT.8 — screen short_text before storage (FR-15).
	if q.Type == inline_questions.TypeShortText {
		text, ok := value.(string)
		if !ok {
			if b, err := json.Marshal(value); err == nil {
				text = string(b)
			}
		}
		screen := ScreenFreeText(text, FilterActionFlag, true)
		if screen.Action == FilterActionBlock {
			ObserveInlineQuestionsSubmit("filtered")
			return &ActionResult{
				Result: map[string]any{
					"error":         "filtered",
					"message":       screen.Guidance,
					"preserveInput": true,
				},
			}, nil
		}
		if screen.Crisis {
			ObserveInlineQuestionsSubmit("crisis")
			return &ActionResult{
				Result: map[string]any{
					"error":         "filtered",
					"message":       screen.Guidance,
					"crisis":        true,
					"preserveInput": true,
				},
			}, nil
		}
	}

	grade := inline_questions.GradeQuestion(*q, value)
	ans := st.Answers[in.QuestionID]
	ans.Attempts = append(ans.Attempts, inline_questions.Attempt{
		Value:   value,
		Correct: grade.Correct,
		At:      inline_questions.NowRFC3339(),
		Points:  grade.PointsAwarded,
	})
	st.Answers[in.QuestionID] = ans
	reveal := inline_questions.ShouldReveal(cfg, st, in.QuestionID, grade.Correct)
	if reveal {
		ans.Revealed = true
		st.Answers[in.QuestionID] = ans
	}
	if st.Drafts != nil {
		delete(st.Drafts, in.QuestionID)
		if len(st.Drafts) == 0 {
			st.Drafts = nil
		}
	}

	raw, max := inline_questions.ComputeScore(cfg, st)
	st.ScoreRaw = &raw
	st.ScoreMax = &max
	status := StatusInProgress
	if inline_questions.AllQuestionsExhaustedOrCorrect(cfg, st) {
		status = StatusCompleted
		st.CompletedAt = inline_questions.NowRFC3339()
	} else if len(st.Answers) > 0 {
		status = StatusSubmitted
	}

	result := map[string]any{
		"correct":           grade.Correct,
		"attemptsRemaining": inline_questions.AttemptsRemaining(cfg, st, in.QuestionID),
		"questionId":        in.QuestionID,
	}
	if grade.Feedback != "" {
		result["feedback"] = grade.Feedback
	}
	if reveal {
		if grade.Explanation != "" {
			result["explanation"] = grade.Explanation
		}
		if cfg.RevealCorrectAfter != inline_questions.RevealNever {
			result["correctAnswer"] = grade.CorrectAnswer
		}
	}

	patch, err := json.Marshal(st)
	if err != nil {
		return nil, err
	}
	outcome := "incorrect"
	if grade.Correct {
		outcome = "correct"
		ObserveInlineQuestionsCorrect()
	}
	ObserveInlineQuestionsSubmit(outcome)
	ObserveInlineQuestionsAttemptCount(len(ans.Attempts))

	return &ActionResult{
		Result:     result,
		StatePatch: patch,
		Status:     status,
		ScoreRaw:   &raw,
		ScoreMax:   &max,
	}, nil
}

func handleInlineQuestionsReveal(ctx ActionContext) (*ActionResult, error) {
	cfg := inline_questions.ParseConfig(ctx.ConfigJSON)
	st := inline_questions.ParseState(ctx.StateJSON)

	var in struct {
		QuestionID string `json:"questionId"`
	}
	if len(ctx.Input) > 0 {
		if err := json.Unmarshal(ctx.Input, &in); err != nil {
			return nil, fmt.Errorf("invalid reveal input: %w", err)
		}
	}
	in.QuestionID = strings.TrimSpace(in.QuestionID)
	if in.QuestionID == "" {
		return nil, fmt.Errorf("questionId is required")
	}
	q := inline_questions.FindQuestion(cfg, in.QuestionID)
	if q == nil {
		return nil, fmt.Errorf("unknown questionId")
	}
	if cfg.RevealCorrectAfter == inline_questions.RevealNever {
		return &ActionResult{
			Result: map[string]any{
				"error":   "reveal_forbidden",
				"message": "Correct answers are not revealed for this check.",
			},
		}, nil
	}
	if !inline_questions.ShouldReveal(cfg, st, in.QuestionID, false) {
		max := inline_questions.MaxAttempts(cfg)
		used := inline_questions.AttemptsUsed(st, in.QuestionID)
		if max > 0 && used < max && cfg.RevealCorrectAfter == inline_questions.RevealLastAttempt {
			return &ActionResult{
				Result: map[string]any{
					"error":   "reveal_not_ready",
					"message": "Finish your attempts before revealing the answer.",
				},
			}, nil
		}
		if used == 0 && cfg.RevealCorrectAfter == inline_questions.RevealFirstAttempt {
			return &ActionResult{
				Result: map[string]any{
					"error":   "reveal_not_ready",
					"message": "Submit an answer before revealing.",
				},
			}, nil
		}
	}

	ans := st.Answers[in.QuestionID]
	ans.Revealed = true
	st.Answers[in.QuestionID] = ans

	var correctAnswer any
	switch q.Type {
	case inline_questions.TypeSingle, inline_questions.TypeTrueFalse, inline_questions.TypeMulti:
		var ids []string
		for _, o := range q.Options {
			if o.Correct {
				ids = append(ids, o.ID)
			}
		}
		correctAnswer = ids
	case inline_questions.TypeShortText:
		correctAnswer = append([]string{}, q.AcceptedAnswers...)
	case inline_questions.TypeNumeric:
		if q.CorrectValue != nil {
			correctAnswer = *q.CorrectValue
		}
	}

	patch, err := json.Marshal(st)
	if err != nil {
		return nil, err
	}
	result := map[string]any{
		"questionId":    in.QuestionID,
		"correctAnswer": correctAnswer,
		"revealed":      true,
	}
	if q.Explanation != "" {
		result["explanation"] = q.Explanation
	}
	return &ActionResult{
		Result:     result,
		StatePatch: patch,
		Status:     ctx.Status,
	}, nil
}
