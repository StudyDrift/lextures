package quizattemptgrading

import (
	"encoding/json"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/lextures/lextures/server/internal/models/coursemodulequiz"
)

// GradedResponse is one stored quiz_responses row before insert.
type GradedResponse struct {
	QuestionIndex  int32
	QuestionID     string
	QuestionType   string
	PromptSnapshot string
	ResponseJSON   json.RawMessage
	IsCorrect      *bool
	PointsAwarded  float64
	MaxPoints      float64
	Locked         bool
}

func maxPointsForQuestion(q coursemodulequiz.QuizQuestion) float64 {
	if q.Points <= 0 {
		return 1
	}
	return float64(q.Points)
}

func responseJSONFromItem(item coursemodulequiz.QuizQuestionResponseItem) json.RawMessage {
	b, err := json.Marshal(item)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return b
}

// GradeResponseItem auto-grades one learner response against a quiz question definition.
func GradeResponseItem(q coursemodulequiz.QuizQuestion, item coursemodulequiz.QuizQuestionResponseItem) GradedResponse {
	maxPts := maxPointsForQuestion(q)
	respJSON := responseJSONFromItem(item)
	gr := GradedResponse{
		QuestionID:     q.ID,
		QuestionType:   q.QuestionType,
		PromptSnapshot: q.Prompt,
		ResponseJSON:   respJSON,
		MaxPoints:      maxPts,
	}

	switch q.QuestionType {
	case "multiple_choice", "true_false":
		if q.MultipleAnswer && len(item.SelectedChoiceIndices) > 0 {
			correct := gradeMultipleAnswerIndices(q, item.SelectedChoiceIndices)
			gr.IsCorrect = &correct
			if correct {
				gr.PointsAwarded = maxPts
			}
			break
		}
		if item.SelectedChoiceIndex != nil && q.CorrectChoiceIndex != nil {
			correct := *item.SelectedChoiceIndex == *q.CorrectChoiceIndex
			gr.IsCorrect = &correct
			if correct {
				gr.PointsAwarded = maxPts
			}
		}
	case "numeric":
		if item.NumericValue != nil {
			correct := numericAnswerCorrect(q.TypeConfig, *item.NumericValue)
			gr.IsCorrect = &correct
			if correct {
				gr.PointsAwarded = maxPts
			}
		}
	case "ordering":
		if len(item.OrderingSequence) > 0 {
			correct := orderingAnswerCorrect(q, item.OrderingSequence)
			gr.IsCorrect = &correct
			if correct {
				gr.PointsAwarded = maxPts
			}
		}
	case "matching":
		if len(item.MatchingPairs) > 0 {
			correct := matchingAnswerCorrect(q, item.MatchingPairs)
			gr.IsCorrect = &correct
			if correct {
				gr.PointsAwarded = maxPts
			}
		}
	case "short_answer", "fill_in_blank":
		if item.TextAnswer != nil {
			correct := textAnswerCorrect(q.TypeConfig, *item.TextAnswer)
			if correct {
				gr.IsCorrect = boolPtr(true)
				gr.PointsAwarded = maxPts
			} else {
				gr.IsCorrect = boolPtr(false)
			}
		}
	}

	return gr
}

func boolPtr(v bool) *bool { return &v }

func gradeMultipleAnswerIndices(q coursemodulequiz.QuizQuestion, selected []uint) bool {
	expected := correctChoiceIndicesFromConfig(q)
	if len(expected) == 0 && q.CorrectChoiceIndex != nil {
		expected = []uint{*q.CorrectChoiceIndex}
	}
	if len(expected) == 0 {
		return false
	}
	if len(selected) != len(expected) {
		return false
	}
	sel := append([]uint(nil), selected...)
	exp := append([]uint(nil), expected...)
	slices.Sort(sel)
	slices.Sort(exp)
	return slices.Equal(sel, exp)
}

func correctChoiceIndicesFromConfig(q coursemodulequiz.QuizQuestion) []uint {
	var cfg struct {
		CorrectChoiceIndices []uint `json:"correctChoiceIndices"`
	}
	if len(q.TypeConfig) == 0 || json.Unmarshal(q.TypeConfig, &cfg) != nil {
		return nil
	}
	return cfg.CorrectChoiceIndices
}

func numericAnswerCorrect(typeConfig json.RawMessage, value float64) bool {
	var cfg struct {
		Correct      *float64 `json:"correct"`
		ToleranceAbs *float64 `json:"toleranceAbs"`
		TolerancePct *float64 `json:"tolerancePct"`
	}
	if len(typeConfig) == 0 || json.Unmarshal(typeConfig, &cfg) != nil || cfg.Correct == nil {
		return false
	}
	target := *cfg.Correct
	if cfg.ToleranceAbs != nil {
		return math.Abs(value-target) <= *cfg.ToleranceAbs+0.000001
	}
	if cfg.TolerancePct != nil && *cfg.TolerancePct > 0 {
		return math.Abs(value-target) <= math.Abs(target)*(*cfg.TolerancePct)/100.0+0.000001
	}
	return math.Abs(value-target) <= 0.000001
}

func textAnswerCorrect(typeConfig json.RawMessage, answer string) bool {
	var cfg struct {
		AcceptableAnswers []string `json:"acceptableAnswers"`
		Correct           *string  `json:"correct"`
	}
	if len(typeConfig) == 0 || json.Unmarshal(typeConfig, &cfg) != nil {
		return false
	}
	trimmed := normalizeAnswerText(answer)
	if trimmed == "" {
		return false
	}
	if cfg.Correct != nil && normalizeAnswerText(*cfg.Correct) == trimmed {
		return true
	}
	for _, a := range cfg.AcceptableAnswers {
		if normalizeAnswerText(a) == trimmed {
			return true
		}
	}
	return false
}

func normalizeAnswerText(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func orderingItemsFromConfig(q coursemodulequiz.QuizQuestion) []string {
	var cfg struct {
		Items []string `json:"items"`
	}
	if len(q.TypeConfig) > 0 && json.Unmarshal(q.TypeConfig, &cfg) == nil && len(cfg.Items) > 0 {
		out := make([]string, 0, len(cfg.Items))
		for _, it := range cfg.Items {
			if t := strings.TrimSpace(it); t != "" {
				out = append(out, t)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	out := make([]string, 0, len(q.Choices))
	for _, c := range q.Choices {
		if t := strings.TrimSpace(c); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func orderingAnswerCorrect(q coursemodulequiz.QuizQuestion, got []string) bool {
	expected := orderingItemsFromConfig(q)
	if len(expected) == 0 || len(got) != len(expected) {
		return false
	}
	for i := range expected {
		if strings.TrimSpace(got[i]) != expected[i] {
			return false
		}
	}
	return true
}

func matchingAnswerCorrect(q coursemodulequiz.QuizQuestion, got []coursemodulequiz.QuizMatchingPairResponse) bool {
	expected := expectedMatchingPairs(q)
	if len(expected) == 0 {
		return false
	}
	if len(got) != len(expected) {
		return false
	}
	expMap := make(map[string]string, len(expected))
	for _, p := range expected {
		expMap[p.LeftID] = p.RightID
	}
	for _, p := range got {
		if expMap[p.LeftID] != p.RightID {
			return false
		}
	}
	return true
}

func expectedMatchingPairs(q coursemodulequiz.QuizQuestion) []coursemodulequiz.QuizMatchingPairResponse {
	var cfg struct {
		Pairs []struct {
			LeftID  string `json:"leftId"`
			RightID string `json:"rightId"`
			Left    string `json:"left"`
			Right   string `json:"right"`
		} `json:"pairs"`
	}
	if len(q.TypeConfig) == 0 || json.Unmarshal(q.TypeConfig, &cfg) != nil {
		return nil
	}
	out := make([]coursemodulequiz.QuizMatchingPairResponse, 0, len(cfg.Pairs))
	for i, p := range cfg.Pairs {
		leftID := strings.TrimSpace(p.LeftID)
		rightID := strings.TrimSpace(p.RightID)
		if leftID == "" {
			leftID = "left-" + strconv.Itoa(i)
		}
		if rightID == "" {
			rightID = "right-" + strconv.Itoa(i)
		}
		if leftID != "" && rightID != "" {
			out = append(out, coursemodulequiz.QuizMatchingPairResponse{LeftID: leftID, RightID: rightID})
		}
	}
	return out
}

// GradeStaticResponses grades a full static quiz submission in question order.
func GradeStaticResponses(
	questions []coursemodulequiz.QuizQuestion,
	responses []coursemodulequiz.QuizQuestionResponseItem,
) []GradedResponse {
	byID := make(map[string]coursemodulequiz.QuizQuestionResponseItem, len(responses))
	for _, r := range responses {
		if strings.TrimSpace(r.QuestionID) != "" {
			byID[r.QuestionID] = r
		}
	}
	out := make([]GradedResponse, 0, len(questions))
	for i, q := range questions {
		item, ok := byID[q.ID]
		if !ok {
			item = coursemodulequiz.QuizQuestionResponseItem{QuestionID: q.ID}
		}
		gr := GradeResponseItem(q, item)
		gr.QuestionIndex = int32(i)
		out = append(out, gr)
	}
	return out
}

// GradeAdaptiveHistory grades adaptive turns in order.
func GradeAdaptiveHistory(history []coursemodulequiz.AdaptiveQuizHistoryTurn) []GradedResponse {
	out := make([]GradedResponse, 0, len(history))
	for i, turn := range history {
		maxPts := AdaptiveTurnMaxPoints(&turn)
		correct := AdaptiveTurnIsCorrect(&turn)
		var pts float64
		if correct {
			pts = maxPts
		}
		resp := map[string]any{
			"selectedChoiceIndex": turn.SelectedChoiceIndex,
		}
		respJSON, _ := json.Marshal(resp)
		qid := ""
		if turn.QuestionID != nil {
			qid = *turn.QuestionID
		}
		out = append(out, GradedResponse{
			QuestionIndex:  int32(i),
			QuestionID:     qid,
			QuestionType:   turn.QuestionType,
			PromptSnapshot: turn.Prompt,
			ResponseJSON:   respJSON,
			IsCorrect:      &correct,
			PointsAwarded:  pts,
			MaxPoints:      maxPts,
		})
	}
	return out
}

func SumGradedPoints(rows []GradedResponse) (earned, possible float64) {
	for _, r := range rows {
		possible += r.MaxPoints
		earned += r.PointsAwarded
	}
	return earned, possible
}

func ScorePercent(earned, possible float64) float32 {
	if possible <= 0 {
		return 0
	}
	pct := float32(math.Min(100, math.Max(0, (earned/possible)*100)))
	return pct
}
