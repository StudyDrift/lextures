package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/lextures/lextures/server/internal/models/coursemodulequiz"
)

type canvasQuizSubmissionAnswer struct {
	CanvasQuestionID int64
	Answer           any
	Points           *float64
	Correct          *bool
}

func canvasQuizSubmissionImportable(m map[string]any) bool {
	if m == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(strAt(m, "workflow_state", ""))) {
	case "complete", "pending_review":
		return true
	default:
		return false
	}
}

func canvasQuestionIDFromLocalID(id string) (int64, bool) {
	const prefix = "canvas-"
	if !strings.HasPrefix(id, prefix) {
		return 0, false
	}
	n, err := strconv.ParseInt(strings.TrimPrefix(id, prefix), 10, 64)
	return n, err == nil && n > 0
}

func canvasQuestionIndexByCanvasID(questions []coursemodulequiz.QuizQuestion) map[int64]int {
	out := make(map[int64]int, len(questions))
	for i, q := range questions {
		if cid, ok := canvasQuestionIDFromLocalID(q.ID); ok {
			out[cid] = i
		}
	}
	return out
}

func canvasUnwrapQuizSubmission(raw map[string]any) map[string]any {
	if raw == nil {
		return nil
	}
	if _, ok := raw["submission_data"]; ok {
		return raw
	}
	if _, ok := raw["workflow_state"]; ok {
		return raw
	}
	if subs, ok := raw["quiz_submissions"].([]any); ok && len(subs) > 0 {
		if m, ok := subs[0].(map[string]any); ok {
			return m
		}
	}
	return raw
}

func canvasLoadQuizAnswerMetadata(
	ctx context.Context,
	client *http.Client,
	canvasBase, accessToken string,
	canvasCourseID, canvasQuizID int64,
) (choiceMaps map[int64]map[int64]int, correctAnswerIDs map[int64]map[int64]struct{}, err error) {
	path := fmt.Sprintf("courses/%d/quizzes/%d/questions", canvasCourseID, canvasQuizID)
	rows, err := canvasGetArrayPaginated(ctx, client, canvasBase, accessToken, path, nil)
	if err != nil {
		return nil, nil, err
	}
	choiceMaps = make(map[int64]map[int64]int, len(rows))
	correctAnswerIDs = make(map[int64]map[int64]struct{}, len(rows))
	for _, row := range rows {
		qid := int64At(row, "id")
		if qid <= 0 {
			continue
		}
		choiceByAnswerID := make(map[int64]int)
		correctIDs := make(map[int64]struct{})
		choiceIdx := 0
		for _, a := range canvasAnswerMaps(row) {
			aid := int64At(a, "id")
			if aid <= 0 {
				continue
			}
			choiceByAnswerID[aid] = choiceIdx
			choiceIdx++
			if canvasAnswerWeight(a) > 0 {
				correctIDs[aid] = struct{}{}
			}
		}
		if len(choiceByAnswerID) > 0 {
			choiceMaps[qid] = choiceByAnswerID
		}
		if len(correctIDs) > 0 {
			correctAnswerIDs[qid] = correctIDs
		}
	}
	return choiceMaps, correctAnswerIDs, nil
}

func canvasGetQuizSubmissionDetail(
	ctx context.Context,
	client *http.Client,
	canvasBase, accessToken string,
	canvasCourseID, canvasQuizID, quizSubmissionID int64,
) (map[string]any, error) {
	path := fmt.Sprintf("courses/%d/quizzes/%d/submissions/%d", canvasCourseID, canvasQuizID, quizSubmissionID)
	raw, err := canvasGetObject(ctx, client, canvasBase, accessToken, path, nil)
	if err != nil {
		return nil, err
	}
	return canvasUnwrapQuizSubmission(raw), nil
}

func canvasGetQuizSubmissionQuestions(
	ctx context.Context,
	client *http.Client,
	canvasBase, accessToken string,
	quizSubmissionID int64,
) ([]map[string]any, error) {
	path := fmt.Sprintf("quiz_submissions/%d/questions", quizSubmissionID)
	v, err := canvasGetJSON(ctx, client, canvasBase, accessToken, path, nil)
	if err != nil {
		return nil, err
	}
	switch t := v.(type) {
	case map[string]any:
		raw, ok := t["quiz_submission_questions"].([]any)
		if !ok {
			return nil, nil
		}
		out := make([]map[string]any, 0, len(raw))
		for _, it := range raw {
			if m, ok := it.(map[string]any); ok && m != nil {
				out = append(out, m)
			}
		}
		return out, nil
	case []any:
		out := make([]map[string]any, 0, len(t))
		for _, it := range t {
			if m, ok := it.(map[string]any); ok && m != nil {
				out = append(out, m)
			}
		}
		return out, nil
	default:
		return nil, nil
	}
}

func canvasParseSubmissionData(raw any) []canvasQuizSubmissionAnswer {
	switch t := raw.(type) {
	case []any:
		return canvasParseSubmissionDataSlice(t)
	case map[string]any:
		return canvasParseSubmissionDataMap(t)
	case string:
		if strings.TrimSpace(t) == "" {
			return nil
		}
		var parsed any
		if err := json.Unmarshal([]byte(t), &parsed); err != nil {
			return nil
		}
		return canvasParseSubmissionData(parsed)
	default:
		return nil
	}
}

func canvasParseSubmissionDataMap(items map[string]any) []canvasQuizSubmissionAnswer {
	out := make([]canvasQuizSubmissionAnswer, 0, len(items))
	for key, value := range items {
		qid, err := strconv.ParseInt(strings.TrimSpace(key), 10, 64)
		if err != nil || qid <= 0 {
			continue
		}
		entry, ok := value.(map[string]any)
		if !ok || entry == nil {
			continue
		}
		entry = cloneStringKeyMap(entry)
		if entry["question_id"] == nil {
			entry["question_id"] = float64(qid)
		}
		out = append(out, canvasParseSubmissionDataEntry(entry)...)
	}
	return out
}

func cloneStringKeyMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func canvasParseSubmissionDataSlice(items []any) []canvasQuizSubmissionAnswer {
	out := make([]canvasQuizSubmissionAnswer, 0, len(items))
	for _, it := range items {
		m, ok := it.(map[string]any)
		if !ok || m == nil {
			continue
		}
		out = append(out, canvasParseSubmissionDataEntry(m)...)
	}
	return out
}

func canvasParseSubmissionDataEntry(m map[string]any) []canvasQuizSubmissionAnswer {
	qid := int64At(m, "question_id")
	if qid <= 0 {
		qid = int64At(m, "id")
	}
	if qid <= 0 {
		return nil
	}
	var pts *float64
	for _, key := range []string{"points", "score", "points_awarded"} {
		if v, ok := coerceCanvasJSONNumber(m[key]); ok {
			pts = &v
			break
		}
	}
	var correct *bool
	switch c := m["correct"].(type) {
	case bool:
		correct = &c
	case string:
		switch strings.ToLower(strings.TrimSpace(c)) {
		case "true", "1":
			b := true
			correct = &b
		case "false", "0":
			b := false
			correct = &b
		}
	}
	answer := m["answer"]
	if answer == nil {
		answer = m["answer_id"]
	}
	if answer == nil {
		answer = m["answers"]
	}
	if answer == nil {
		answer = m["text"]
	}
	return []canvasQuizSubmissionAnswer{{
		CanvasQuestionID: qid,
		Answer:           answer,
		Points:           pts,
		Correct:          correct,
	}}
}

func canvasMergeSubmissionAnswers(detail, listItem map[string]any, questionRows []map[string]any) map[int64]canvasQuizSubmissionAnswer {
	out := make(map[int64]canvasQuizSubmissionAnswer)
	for _, src := range []map[string]any{detail, listItem} {
		if src == nil {
			continue
		}
		for _, a := range canvasParseSubmissionData(src["submission_data"]) {
			out[a.CanvasQuestionID] = canvasMergeQuizSubmissionAnswer(out[a.CanvasQuestionID], a)
		}
	}
	for _, row := range questionRows {
		qid := int64At(row, "id")
		if qid <= 0 {
			continue
		}
		prev := out[qid]
		if prev.Answer == nil {
			prev.Answer = row["answer"]
		}
		if prev.Points == nil {
			if v, ok := coerceCanvasJSONNumber(row["points"]); ok {
				prev.Points = &v
			}
		}
		prev.CanvasQuestionID = qid
		out[qid] = prev
	}
	return out
}

func canvasMergeQuizSubmissionAnswer(existing, incoming canvasQuizSubmissionAnswer) canvasQuizSubmissionAnswer {
	if existing.CanvasQuestionID == 0 {
		existing = incoming
	} else {
		if incoming.Answer != nil {
			existing.Answer = incoming.Answer
		}
		if incoming.Points != nil {
			existing.Points = incoming.Points
		}
		if incoming.Correct != nil {
			existing.Correct = incoming.Correct
		}
	}
	return existing
}

func canvasQuizSubmissionScore(raw map[string]any) (score float64, hasScore bool) {
	if raw == nil {
		return 0, false
	}
	for _, key := range []string{"kept_score", "score", "score_before_regrade"} {
		if v, ok := coerceCanvasJSONNumber(raw[key]); ok {
			return v, true
		}
	}
	return 0, false
}

func canvasQuizPointsPossible(questions []coursemodulequiz.QuizQuestion, quizPointsWorth *int) float64 {
	var sum float64
	for _, q := range questions {
		if q.Points > 0 {
			sum += float64(q.Points)
		} else {
			sum += 1
		}
	}
	if sum > 0 {
		return sum
	}
	if quizPointsWorth != nil && *quizPointsWorth > 0 {
		return float64(*quizPointsWorth)
	}
	return 0
}

func canvasAnswerIDs(answer any) []int64 {
	switch v := answer.(type) {
	case float64:
		return []int64{int64(v)}
	case int64:
		return []int64{v}
	case string:
		if n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil {
			return []int64{n}
		}
	case []any:
		out := make([]int64, 0, len(v))
		for _, item := range v {
			out = append(out, canvasAnswerIDs(item)...)
		}
		return out
	}
	return nil
}

func canvasAnswerIDSet(answer any) map[int64]struct{} {
	ids := canvasAnswerIDs(answer)
	out := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		out[id] = struct{}{}
	}
	return out
}

func canvasResponseJSONForAnswer(q coursemodulequiz.QuizQuestion, answer any, choiceByAnswerID map[int64]int) json.RawMessage {
	switch q.QuestionType {
	case "multiple_choice", "true_false":
		if q.MultipleAnswer {
			if ids := canvasAnswerIDs(answer); len(ids) > 0 {
				indices := make([]uint, 0, len(ids))
				for _, id := range ids {
					if idx, ok := canvasAnswerToChoiceIndex(float64(id), choiceByAnswerID); ok {
						indices = append(indices, idx)
					}
				}
				if len(indices) > 0 {
					b, _ := json.Marshal(map[string]any{"selectedChoiceIndices": indices})
					return b
				}
			}
		}
		if idx, ok := canvasAnswerToChoiceIndex(answer, choiceByAnswerID); ok {
			b, _ := json.Marshal(map[string]any{"selectedChoiceIndex": idx})
			return b
		}
		if idx, ok := canvasAnswerAsChoiceIndex(answer, len(q.Choices)); ok {
			b, _ := json.Marshal(map[string]any{"selectedChoiceIndex": idx})
			return b
		}
	case "short_answer", "essay":
		if text := canvasAnswerAsString(answer); text != "" {
			b, _ := json.Marshal(map[string]any{"textAnswer": text})
			return b
		}
	case "fill_in_blank":
		switch v := answer.(type) {
		case map[string]any:
			b, _ := json.Marshal(map[string]any{"blanks": v})
			return b
		case string:
			if strings.TrimSpace(v) != "" {
				b, _ := json.Marshal(map[string]any{"textAnswer": strings.TrimSpace(v)})
				return b
			}
		default:
			if text := canvasAnswerAsString(answer); text != "" {
				b, _ := json.Marshal(map[string]any{"textAnswer": text})
				return b
			}
		}
	case "numeric":
		if v, ok := coerceCanvasJSONNumber(answer); ok {
			b, _ := json.Marshal(map[string]any{"numericValue": v})
			return b
		}
	}
	if answer != nil {
		b, _ := json.Marshal(map[string]any{"canvasAnswer": answer})
		return b
	}
	return json.RawMessage(`{}`)
}

func canvasAnswerAsChoiceIndex(answer any, choiceCount int) (uint, bool) {
	if choiceCount <= 0 {
		return 0, false
	}
	switch v := answer.(type) {
	case float64:
		idx := int(v)
		if idx >= 0 && idx < choiceCount {
			return uint(idx), true
		}
	case int64:
		if v >= 0 && int(v) < choiceCount {
			return uint(v), true
		}
	}
	return 0, false
}

func canvasAnswerAsString(answer any) string {
	switch v := answer.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int64:
		return strconv.FormatInt(v, 10)
	default:
		return ""
	}
}

func canvasAnswerToChoiceIndex(answer any, choiceByAnswerID map[int64]int) (uint, bool) {
	switch v := answer.(type) {
	case float64:
		if idx, ok := choiceByAnswerID[int64(v)]; ok {
			return uint(idx), true
		}
	case int64:
		if idx, ok := choiceByAnswerID[v]; ok {
			return uint(idx), true
		}
	case string:
		if n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil {
			if idx, ok := choiceByAnswerID[n]; ok {
				return uint(idx), true
			}
		}
	}
	return 0, false
}

func canvasGradeImportedQuestion(
	q coursemodulequiz.QuizQuestion,
	answer canvasQuizSubmissionAnswer,
	choiceByAnswerID map[int64]int,
	correctIDs map[int64]struct{},
) (responseJSON json.RawMessage, isCorrect *bool, pointsAwarded, maxPoints float64) {
	maxPoints = float64(q.Points)
	if maxPoints <= 0 {
		maxPoints = 1
	}
	responseJSON = canvasResponseJSONForAnswer(q, answer.Answer, choiceByAnswerID)

	if answer.Points != nil {
		pointsAwarded = *answer.Points
		if pointsAwarded < 0 {
			pointsAwarded = 0
		}
		if pointsAwarded > maxPoints {
			pointsAwarded = maxPoints
		}
		if answer.Correct != nil {
			isCorrect = answer.Correct
		} else if maxPoints > 0 {
			c := pointsAwarded >= maxPoints-0.0001
			isCorrect = &c
		}
		return responseJSON, isCorrect, pointsAwarded, maxPoints
	}

	if answer.Correct != nil {
		isCorrect = answer.Correct
		if *answer.Correct {
			pointsAwarded = maxPoints
		}
		return responseJSON, isCorrect, pointsAwarded, maxPoints
	}

	switch q.QuestionType {
	case "multiple_choice", "true_false":
		if q.MultipleAnswer && len(correctIDs) > 0 {
			selected := canvasAnswerIDSet(answer.Answer)
			c := canvasAnswerSetsEqual(selected, correctIDs)
			isCorrect = &c
			if c {
				pointsAwarded = maxPoints
			}
			break
		}
		if idx, ok := canvasAnswerToChoiceIndex(answer.Answer, choiceByAnswerID); ok && q.CorrectChoiceIndex != nil {
			c := idx == *q.CorrectChoiceIndex
			isCorrect = &c
			if c {
				pointsAwarded = maxPoints
			}
			break
		}
		if idx, ok := canvasAnswerAsChoiceIndex(answer.Answer, len(q.Choices)); ok && q.CorrectChoiceIndex != nil {
			c := idx == *q.CorrectChoiceIndex
			isCorrect = &c
			if c {
				pointsAwarded = maxPoints
			}
		}
	case "numeric":
		if v, ok := coerceCanvasJSONNumber(answer.Answer); ok {
			c := canvasNumericAnswerCorrect(q.TypeConfig, v)
			isCorrect = &c
			if c {
				pointsAwarded = maxPoints
			}
		}
	}

	return responseJSON, isCorrect, pointsAwarded, maxPoints
}

func canvasAnswerSetsEqual(selected, correct map[int64]struct{}) bool {
	if len(selected) != len(correct) {
		return false
	}
	for id := range correct {
		if _, ok := selected[id]; !ok {
			return false
		}
	}
	return true
}

func canvasNumericAnswerCorrect(typeConfig json.RawMessage, value float64) bool {
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

func canvasParseCanvasTime(raw any) *time.Time {
	s, _ := raw.(string)
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	utc := t.UTC()
	return &utc
}

func canvasUpsertImportedQuizAttempt(
	ctx context.Context,
	tx pgx.Tx,
	courseID, itemID, studentID uuid.UUID,
	attemptNumber int32,
	startedAt, submittedAt *time.Time,
	pointsEarned, pointsPossible float64,
) (uuid.UUID, error) {
	var scorePercent *float32
	if pointsPossible > 0 {
		pct := float32((pointsEarned / pointsPossible) * 100)
		if pct < 0 {
			pct = 0
		}
		if pct > 100 {
			pct = 100
		}
		scorePercent = &pct
	}
	start := time.Now().UTC()
	if startedAt != nil {
		start = *startedAt
	}
	submitted := start
	if submittedAt != nil {
		submitted = *submittedAt
	}
	var attemptID uuid.UUID
	err := tx.QueryRow(ctx, `
INSERT INTO course.quiz_attempts (
  course_id, structure_item_id, student_user_id, attempt_number, status,
  is_adaptive, started_at, submitted_at, points_earned, points_possible, score_percent
) VALUES ($1, $2, $3, $4, 'submitted', false, $5, $6, $7, $8, $9)
ON CONFLICT (structure_item_id, student_user_id, attempt_number) DO UPDATE SET
  status = 'submitted',
  started_at = EXCLUDED.started_at,
  submitted_at = EXCLUDED.submitted_at,
  points_earned = EXCLUDED.points_earned,
  points_possible = EXCLUDED.points_possible,
  score_percent = EXCLUDED.score_percent
RETURNING id
`, courseID, itemID, studentID, attemptNumber, start, submitted, pointsEarned, pointsPossible, scorePercent).Scan(&attemptID)
	return attemptID, err
}

func canvasReplaceImportedQuizResponses(
	ctx context.Context,
	tx pgx.Tx,
	attemptID uuid.UUID,
	questions []coursemodulequiz.QuizQuestion,
	answers map[int64]canvasQuizSubmissionAnswer,
	choiceMaps map[int64]map[int64]int,
	correctAnswerIDs map[int64]map[int64]struct{},
) error {
	if _, err := tx.Exec(ctx, `DELETE FROM course.quiz_responses WHERE attempt_id = $1`, attemptID); err != nil {
		return err
	}
	indexByCanvasID := canvasQuestionIndexByCanvasID(questions)
	for canvasQID, answer := range answers {
		qi, ok := indexByCanvasID[canvasQID]
		if !ok || qi < 0 || qi >= len(questions) {
			continue
		}
		q := questions[qi]
		responseJSON, isCorrect, pointsAwarded, maxPoints := canvasGradeImportedQuestion(
			q, answer, choiceMaps[canvasQID], correctAnswerIDs[canvasQID],
		)
		_, err := tx.Exec(ctx, `
INSERT INTO course.quiz_responses (
  attempt_id, question_index, question_id, question_type, prompt_snapshot,
  response_json, is_correct, points_awarded, max_points
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
`, attemptID, qi, q.ID, q.QuestionType, q.Prompt, responseJSON, isCorrect, pointsAwarded, maxPoints)
		if err != nil {
			return err
		}
	}
	return nil
}

type canvasQuizAnswerMetadata struct {
	choiceMaps       map[int64]map[int64]int
	correctAnswerIDs map[int64]map[int64]struct{}
}

type canvasQuizAttemptPayload struct {
	raw          map[string]any
	detail       map[string]any
	questionRows []map[string]any
}

func canvasFetchQuizAnswerMetadataParallel(
	ctx context.Context,
	client *http.Client,
	canvasBase, accessToken string,
	canvasCourseID int64,
	quizIDs []int64,
) (map[int64]canvasQuizAnswerMetadata, error) {
	out := make(map[int64]canvasQuizAnswerMetadata, len(quizIDs))
	if len(quizIDs) == 0 {
		return out, nil
	}
	var mu sync.Mutex
	var firstErr error
	var errOnce sync.Once

	g, gctx := canvasImportParallelGroup(ctx, len(quizIDs))
	for _, quizID := range quizIDs {
		quizID := quizID
		g.Go(func() error {
			choiceMaps, correctAnswerIDs, err := canvasLoadQuizAnswerMetadata(gctx, client, canvasBase, accessToken, canvasCourseID, quizID)
			if err != nil {
				errOnce.Do(func() { firstErr = fmt.Errorf("Canvas quiz %d answer map: %w", quizID, err) })
				return err
			}
			mu.Lock()
			out[quizID] = canvasQuizAnswerMetadata{choiceMaps: choiceMaps, correctAnswerIDs: correctAnswerIDs}
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		if firstErr != nil {
			return nil, firstErr
		}
		return nil, err
	}
	return out, nil
}

func canvasFetchQuizAttemptPayloadsParallel(
	ctx context.Context,
	client *http.Client,
	canvasBase, accessToken string,
	canvasCourseID, canvasQuizID int64,
	subs []map[string]any,
	canvasUserToLocal map[int64]uuid.UUID,
) ([]canvasQuizAttemptPayload, error) {
	candidates := make([]map[string]any, 0, len(subs))
	for _, raw := range subs {
		if !canvasQuizSubmissionImportable(raw) {
			continue
		}
		canvasUserID := int64At(raw, "user_id")
		if _, ok := canvasUserToLocal[canvasUserID]; !ok {
			continue
		}
		if int64At(raw, "id") <= 0 {
			continue
		}
		candidates = append(candidates, raw)
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	out := make([]canvasQuizAttemptPayload, len(candidates))
	var firstErr error
	var errOnce sync.Once

	g, gctx := canvasImportParallelGroup(ctx, len(candidates))
	for i, raw := range candidates {
		i, raw := i, raw
		g.Go(func() error {
			submissionID := int64At(raw, "id")
			detail, err := canvasGetQuizSubmissionDetail(gctx, client, canvasBase, accessToken, canvasCourseID, canvasQuizID, submissionID)
			if err != nil {
				errOnce.Do(func() { firstErr = fmt.Errorf("Canvas quiz submission %d detail: %w", submissionID, err) })
				return err
			}
			questionRows, err := canvasGetQuizSubmissionQuestions(gctx, client, canvasBase, accessToken, submissionID)
			if err != nil {
				errOnce.Do(func() { firstErr = fmt.Errorf("Canvas quiz submission %d questions: %w", submissionID, err) })
				return err
			}
			out[i] = canvasQuizAttemptPayload{raw: raw, detail: detail, questionRows: questionRows}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		if firstErr != nil {
			return nil, firstErr
		}
		return nil, err
	}
	return out, nil
}

func canvasImportQuizAttempts(
	ctx context.Context,
	tx pgx.Tx,
	client *http.Client,
	canvasBase, accessToken string,
	canvasCourseID int64,
	courseID uuid.UUID,
	canvasQuizToItem map[int64]uuid.UUID,
	canvasQuizToQuestions map[int64][]coursemodulequiz.QuizQuestion,
	canvasUserToLocal map[int64]uuid.UUID,
	quizSubsByQuiz map[int64][]map[string]any,
) error {
	quizIDs := canvasQuizIDsFromMap(canvasQuizToItem)
	answerMetaByQuiz, err := canvasFetchQuizAnswerMetadataParallel(ctx, client, canvasBase, accessToken, canvasCourseID, quizIDs)
	if err != nil {
		return err
	}

	for canvasQID, itemID := range canvasQuizToItem {
		questions := canvasQuizToQuestions[canvasQID]
		if len(questions) == 0 {
			continue
		}
		meta := answerMetaByQuiz[canvasQID]
		choiceMaps := meta.choiceMaps
		correctAnswerIDs := meta.correctAnswerIDs
		subs := quizSubsByQuiz[canvasQID]
		pointsPossibleQuiz := canvasQuizPointsPossible(questions, nil)

		payloads, err := canvasFetchQuizAttemptPayloadsParallel(ctx, client, canvasBase, accessToken, canvasCourseID, canvasQID, subs, canvasUserToLocal)
		if err != nil {
			return err
		}
		for _, payload := range payloads {
			raw := payload.raw
			canvasUserID := int64At(raw, "user_id")
			studentID, ok := canvasUserToLocal[canvasUserID]
			if !ok {
				continue
			}
			submissionID := int64At(raw, "id")
			if submissionID <= 0 {
				continue
			}
			attemptNum := int32(int64At(raw, "attempt"))
			if attemptNum < 1 {
				attemptNum = 1
			}

			answers := canvasMergeSubmissionAnswers(payload.detail, raw, payload.questionRows)
			score, hasScore := canvasQuizSubmissionScore(raw)
			if len(answers) == 0 && !hasScore {
				continue
			}

			var earned float64
			indexByCanvasID := canvasQuestionIndexByCanvasID(questions)
			for canvasQuestionID, ans := range answers {
				qi, ok := indexByCanvasID[canvasQuestionID]
				if !ok || qi >= len(questions) {
					continue
				}
				_, _, pts, _ := canvasGradeImportedQuestion(questions[qi], ans, choiceMaps[canvasQuestionID], correctAnswerIDs[canvasQuestionID])
				earned += pts
			}
			possible := pointsPossibleQuiz
			if possible <= 0 {
				possible = earned
			}
			if hasScore {
				earned = score
				if possible <= 0 {
					possible = score
				}
			}

			startedAt := canvasParseCanvasTime(raw["started_at"])
			submittedAt := canvasParseCanvasTime(raw["finished_at"])
			if submittedAt == nil {
				submittedAt = canvasParseCanvasTime(raw["submitted_at"])
			}

			attemptID, err := canvasUpsertImportedQuizAttempt(ctx, tx, courseID, itemID, studentID, attemptNum, startedAt, submittedAt, earned, possible)
			if err != nil {
				return fmt.Errorf("save quiz attempt canvas submission %d: %w", submissionID, err)
			}
			if err := canvasReplaceImportedQuizResponses(ctx, tx, attemptID, questions, answers, choiceMaps, correctAnswerIDs); err != nil {
				return fmt.Errorf("save quiz responses canvas submission %d: %w", submissionID, err)
			}
		}
	}
	return nil
}
