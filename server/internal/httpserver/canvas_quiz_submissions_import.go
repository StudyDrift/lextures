package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/lextures/lextures/server/internal/models/coursemodulequiz"
	"github.com/lextures/lextures/server/internal/service/quizattemptgrading"
)

type canvasQuizSubmissionAnswer struct {
	CanvasQuestionID int64
	Answer           any
	Points           *float64
	Correct          *bool
	AttachmentIDs    []int64
}

// canvasAttachmentIDsFromMap collects Canvas attachment ids from a submission-data entry,
// question row, or quiz event payload. Canvas exposes file-upload answers as attachment_ids
// (and sometimes an inline attachments array), so we accept ids, id strings, and file objects.
func canvasAttachmentIDsFromMap(m map[string]any) []int64 {
	if m == nil {
		return nil
	}
	var out []int64
	seen := make(map[int64]struct{})
	add := func(id int64) {
		if id <= 0 {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	for _, key := range []string{"attachment_ids", "attachment_id", "attachments"} {
		switch v := m[key].(type) {
		case []any:
			for _, it := range v {
				switch iv := it.(type) {
				case map[string]any:
					add(int64At(iv, "id"))
				default:
					add(canvasAttachmentIDFromAny(iv))
				}
			}
		case map[string]any:
			add(int64At(v, "id"))
		default:
			add(canvasAttachmentIDFromAny(v))
		}
	}
	return out
}

func canvasAttachmentIDFromAny(v any) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case int64:
		return t
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		if err != nil {
			return 0
		}
		return n
	default:
		return 0
	}
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
	attempt int32,
) (map[string]any, error) {
	path := fmt.Sprintf("courses/%d/quizzes/%d/submissions/%d", canvasCourseID, canvasQuizID, quizSubmissionID)
	q := url.Values{}
	if attempt > 0 {
		q.Set("attempt", strconv.Itoa(int(attempt)))
	}
	raw, err := canvasGetObject(ctx, client, canvasBase, accessToken, path, q)
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
	attempt int32,
) ([]map[string]any, error) {
	path := fmt.Sprintf("quiz_submissions/%d/questions", quizSubmissionID)
	q := url.Values{}
	q.Add("include[]", "quiz_question")
	if attempt > 0 {
		q.Set("quiz_submission_attempt", strconv.Itoa(int(attempt)))
	}
	v, err := canvasGetJSON(ctx, client, canvasBase, accessToken, path, q)
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

func canvasGetQuizSubmissionEvents(
	ctx context.Context,
	client *http.Client,
	canvasBase, accessToken string,
	canvasCourseID, canvasQuizID, quizSubmissionID int64,
	attempt int32,
) ([]map[string]any, error) {
	path := fmt.Sprintf("courses/%d/quizzes/%d/submissions/%d/events", canvasCourseID, canvasQuizID, quizSubmissionID)
	q := url.Values{}
	if attempt > 0 {
		q.Set("attempt", strconv.Itoa(int(attempt)))
	}
	v, err := canvasGetJSON(ctx, client, canvasBase, accessToken, path, q)
	if err != nil {
		return nil, err
	}
	switch t := v.(type) {
	case map[string]any:
		raw, ok := t["quiz_submission_events"].([]any)
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

func canvasCanvasQuestionIDFromSubmissionDataKey(key string) (int64, bool) {
	key = strings.TrimSpace(key)
	if key == "" {
		return 0, false
	}
	switch key {
	case "attempt", "cnt", "validation_token":
		return 0, false
	}
	if strings.HasPrefix(key, "_question_") {
		return 0, false
	}
	if strings.HasPrefix(key, "question_") {
		rest := strings.TrimPrefix(key, "question_")
		if idx := strings.Index(rest, "_"); idx >= 0 {
			rest = rest[:idx]
		}
		qid, err := strconv.ParseInt(rest, 10, 64)
		return qid, err == nil && qid > 0
	}
	qid, err := strconv.ParseInt(key, 10, 64)
	return qid, err == nil && qid > 0
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
		qid, ok := canvasCanvasQuestionIDFromSubmissionDataKey(key)
		if !ok || qid <= 0 {
			continue
		}
		switch v := value.(type) {
		case map[string]any:
			if v == nil {
				continue
			}
			entry := cloneStringKeyMap(v)
			if entry["question_id"] == nil {
				entry["question_id"] = float64(qid)
			}
			out = append(out, canvasParseSubmissionDataEntry(entry)...)
		case string:
			if text := strings.TrimSpace(v); text != "" {
				out = append(out, canvasQuizSubmissionAnswer{
					CanvasQuestionID: qid,
					Answer:           text,
				})
			}
		default:
			if text := canvasAnswerAsString(v); text != "" {
				out = append(out, canvasQuizSubmissionAnswer{
					CanvasQuestionID: qid,
					Answer:           text,
				})
			}
		}
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
		case "undefined":
			// Canvas uses "undefined" for manual-grading items before review.
		}
	}
	answer := canvasAnswerValueFromMap(m)
	return []canvasQuizSubmissionAnswer{{
		CanvasQuestionID: qid,
		Answer:           answer,
		Points:           pts,
		Correct:          correct,
		AttachmentIDs:    canvasAttachmentIDsFromMap(m),
	}}
}

func canvasAnswerValueFromMap(m map[string]any) any {
	if m == nil {
		return nil
	}
	for _, key := range []string{"text", "answer", "answer_id", "answers", "student_answer", "student_answers"} {
		v := m[key]
		if v == nil {
			continue
		}
		if s, ok := v.(string); ok && strings.TrimSpace(s) == "" {
			continue
		}
		return v
	}
	return nil
}

func canvasQuizQuestionIDFromRow(row map[string]any) int64 {
	// QuizSubmissionQuestion.id is the Canvas quiz question id (per Canvas API).
	if qid := int64At(row, "id"); qid > 0 {
		return qid
	}
	return int64At(row, "question_id")
}

func canvasAnswerFromQuestionRow(row map[string]any) any {
	return canvasAnswerValueFromMap(row)
}

func canvasMergeAnswersFromQuizEvents(events []map[string]any) map[int64]canvasQuizSubmissionAnswer {
	out := make(map[int64]canvasQuizSubmissionAnswer)
	for _, ev := range events {
		if strings.TrimSpace(strAt(ev, "event_type", "")) != "question_answered" {
			continue
		}
		data, ok := ev["event_data"].(map[string]any)
		if !ok || data == nil {
			continue
		}
		qid := int64At(data, "question_id")
		if qid <= 0 {
			continue
		}
		answer := canvasAnswerValueFromMap(data)
		attachmentIDs := canvasAttachmentIDsFromMap(data)
		if answer == nil && len(attachmentIDs) == 0 {
			continue
		}
		out[qid] = canvasMergeQuizSubmissionAnswer(out[qid], canvasQuizSubmissionAnswer{
			CanvasQuestionID: qid,
			Answer:           answer,
			AttachmentIDs:    attachmentIDs,
		})
	}
	return out
}

func canvasMergeSubmissionAnswers(sources []map[string]any, questionRows []map[string]any, eventRows []map[string]any) map[int64]canvasQuizSubmissionAnswer {
	out := make(map[int64]canvasQuizSubmissionAnswer)
	for _, src := range sources {
		if src == nil {
			continue
		}
		for _, sd := range canvasCollectSubmissionDataBlobs(src) {
			for _, a := range canvasParseSubmissionData(sd) {
				out[a.CanvasQuestionID] = canvasMergeQuizSubmissionAnswer(out[a.CanvasQuestionID], a)
			}
		}
	}
	for qid, ans := range canvasMergeAnswersFromQuizEvents(eventRows) {
		out[qid] = canvasMergeQuizSubmissionAnswer(out[qid], ans)
	}
	for _, row := range questionRows {
		qid := canvasQuizQuestionIDFromRow(row)
		if qid <= 0 {
			continue
		}
		prev := out[qid]
		if prev.Answer == nil {
			prev.Answer = canvasAnswerFromQuestionRow(row)
		}
		if len(prev.AttachmentIDs) == 0 {
			prev.AttachmentIDs = canvasAttachmentIDsFromMap(row)
		}
		if prev.Points == nil {
			if v, ok := coerceCanvasJSONNumber(row["points"]); ok {
				prev.Points = &v
			} else if v, ok := coerceCanvasJSONNumber(row["points_possible"]); ok {
				prev.Points = &v
			}
		}
		if prev.Correct == nil {
			switch c := row["correct"].(type) {
			case bool:
				prev.Correct = &c
			case string:
				switch strings.ToLower(strings.TrimSpace(c)) {
				case "true", "1":
					b := true
					prev.Correct = &b
				case "false", "0":
					b := false
					prev.Correct = &b
				}
			}
		}
		prev.CanvasQuestionID = qid
		out[qid] = prev
	}
	return out
}

func canvasCollectSubmissionDataBlobs(src map[string]any) []any {
	if src == nil {
		return nil
	}
	out := make([]any, 0, 4)
	if sd := src["submission_data"]; sd != nil {
		out = append(out, sd)
	}
	for _, hist := range arrAt(src, "submission_history") {
		if sd := hist["submission_data"]; sd != nil {
			out = append(out, sd)
		}
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
		if len(incoming.AttachmentIDs) > 0 {
			existing.AttachmentIDs = incoming.AttachmentIDs
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

func canvasSupplementQuestionsFromSubmissionRows(
	questions []coursemodulequiz.QuizQuestion,
	rows []map[string]any,
) []coursemodulequiz.QuizQuestion {
	if len(rows) == 0 {
		return questions
	}
	seen := make(map[string]struct{}, len(questions))
	for _, q := range questions {
		seen[q.ID] = struct{}{}
	}
	out := append([]coursemodulequiz.QuizQuestion(nil), questions...)
	for _, row := range rows {
		qid := canvasQuizQuestionIDFromRow(row)
		if qid <= 0 {
			continue
		}
		localID := fmt.Sprintf("canvas-%d", qid)
		if _, ok := seen[localID]; ok {
			continue
		}
		payload := map[string]any{
			"id":              row["id"],
			"question_name":   row["question_name"],
			"question_text":   row["question_text"],
			"question_type":   row["question_type"],
			"points_possible": row["points_possible"],
		}
		if qq, ok := canvasQuestionToQuizQuestion(payload); ok {
			out = append(out, qq)
			seen[localID] = struct{}{}
		}
	}
	return out
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
		if text := canvasAnswerTextToMarkdown(canvasAnswerAsString(answer)); text != "" {
			b, _ := json.Marshal(map[string]any{"textAnswer": text})
			return b
		}
	case "fill_in_blank":
		switch v := answer.(type) {
		case map[string]any:
			b, _ := json.Marshal(map[string]any{"blanks": v})
			return b
		case string:
			if text := canvasAnswerTextToMarkdown(v); text != "" {
				b, _ := json.Marshal(map[string]any{"textAnswer": text})
				return b
			}
		default:
			if text := canvasAnswerTextToMarkdown(canvasAnswerAsString(answer)); text != "" {
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

// canvasAnswerTextToMarkdown converts a Canvas free-text answer to Markdown. Canvas stores essay
// and other rich answers as HTML, so when the value contains markup we run it through the same
// HTML→Markdown converter used elsewhere in the importer; plain-text answers pass through trimmed.
func canvasAnswerTextToMarkdown(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if htmlAnyTagRe.MatchString(s) {
		if mdText := markdownFromHTML(s); mdText != "" {
			return mdText
		}
	}
	return s
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

	clampPoints := func(p float64) float64 {
		if p < 0 {
			return 0
		}
		if p > maxPoints {
			return maxPoints
		}
		return p
	}

	// Subjective question types are never auto-scored as correct/incorrect on import. Canvas can
	// emit a default 0/undefined score before an instructor reviews them; import any existing
	// Canvas points but leave correctness unset so they surface as "needs grading", not "wrong".
	if quizattemptgrading.IsManualGradingQuestionType(q.QuestionType) {
		if answer.Points != nil {
			pointsAwarded = clampPoints(*answer.Points)
		}
		return responseJSON, nil, pointsAwarded, maxPoints
	}

	// Objective questions can only be graded when the imported question has a correct-answer key.
	// Reflection/survey items (e.g. a numeric question with no defined answer) have none, so
	// Canvas's "incorrect" flag is meaningless — leave them ungraded instead of marking them wrong.
	if !canvasQuestionHasAnswerKey(q, correctIDs) {
		if answer.Points != nil {
			pointsAwarded = clampPoints(*answer.Points)
		}
		return responseJSON, nil, pointsAwarded, maxPoints
	}

	if answer.Points != nil {
		pointsAwarded = clampPoints(*answer.Points)
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

// canvasQuestionHasAnswerKey reports whether an imported objective question carries a
// correct-answer key that the importer can grade against. Without one (e.g. a numeric reflection
// question with no defined answer), Canvas's correctness flag is meaningless and the response is
// left ungraded rather than marked wrong.
func canvasQuestionHasAnswerKey(q coursemodulequiz.QuizQuestion, correctIDs map[int64]struct{}) bool {
	switch q.QuestionType {
	case "multiple_choice", "true_false":
		return q.CorrectChoiceIndex != nil || len(correctIDs) > 0
	case "numeric":
		return canvasNumericConfigHasCorrect(q.TypeConfig)
	default:
		return false
	}
}

func canvasNumericConfigHasCorrect(typeConfig json.RawMessage) bool {
	if len(typeConfig) == 0 {
		return false
	}
	var cfg struct {
		Correct *float64 `json:"correct"`
	}
	if json.Unmarshal(typeConfig, &cfg) != nil {
		return false
	}
	return cfg.Correct != nil
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

// canvasImportedQuizFile is the per-file reference stored in a quiz response's response_json so
// the grading UI can render images inline and link other uploads. ContentPath points at the
// course-file content endpoint where the downloaded blob is served.
type canvasImportedQuizFile struct {
	FileID      uuid.UUID `json:"fileId"`
	Filename    string    `json:"filename"`
	MimeType    string    `json:"mimeType"`
	ContentPath string    `json:"contentPath"`
}

func canvasGetFile(
	ctx context.Context,
	client *http.Client,
	canvasBase, accessToken string,
	fileID int64,
) (map[string]any, error) {
	return canvasGetObject(ctx, client, canvasBase, accessToken, fmt.Sprintf("files/%d", fileID), nil)
}

// canvasImportQuizResponseAttachments downloads each Canvas file-upload attachment into the course
// file store and returns references for embedding in the response JSON. Individual failures are
// logged and skipped so one bad attachment never aborts the whole import.
func canvasImportQuizResponseAttachments(
	ctx context.Context,
	tx pgx.Tx,
	client *http.Client,
	canvasBase, accessToken string,
	deps *canvasAssignmentSubmissionImportDeps,
	courseID uuid.UUID,
	attachmentIDs []int64,
) []canvasImportedQuizFile {
	if deps == nil || len(attachmentIDs) == 0 {
		return nil
	}
	out := make([]canvasImportedQuizFile, 0, len(attachmentIDs))
	for _, attID := range attachmentIDs {
		if attID <= 0 {
			continue
		}
		fileObj, err := canvasGetFile(ctx, client, canvasBase, accessToken, attID)
		if err != nil || fileObj == nil {
			log.Printf("canvas-import: quiz response attachment %d metadata fetch failed: %v", attID, err)
			continue
		}
		fileID, err := canvasStreamAndStoreSubmissionAttachment(ctx, tx, client, accessToken, *deps, courseID, fileObj)
		if err != nil {
			log.Printf("canvas-import: quiz response attachment %d download failed: %v", attID, err)
			continue
		}
		if fileID == nil {
			continue
		}
		out = append(out, canvasImportedQuizFile{
			FileID:      *fileID,
			Filename:    canvasSubmissionAttachmentFilename(fileObj),
			MimeType:    canvasSubmissionAttachmentMimeType(fileObj, ""),
			ContentPath: fmt.Sprintf("/api/v1/courses/%s/course-files/%s/content", deps.CourseCode, fileID.String()),
		})
	}
	return out
}

func canvasInjectFilesIntoResponseJSON(responseJSON json.RawMessage, files []canvasImportedQuizFile) json.RawMessage {
	if len(files) == 0 {
		return responseJSON
	}
	obj := map[string]any{}
	if len(responseJSON) > 0 {
		_ = json.Unmarshal(responseJSON, &obj)
	}
	if obj == nil {
		obj = map[string]any{}
	}
	obj["files"] = files
	b, err := json.Marshal(obj)
	if err != nil {
		return responseJSON
	}
	return b
}

func canvasReplaceImportedQuizResponses(
	ctx context.Context,
	tx pgx.Tx,
	client *http.Client,
	canvasBase, accessToken string,
	deps *canvasAssignmentSubmissionImportDeps,
	courseID uuid.UUID,
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
		if files := canvasImportQuizResponseAttachments(ctx, tx, client, canvasBase, accessToken, deps, courseID, answer.AttachmentIDs); len(files) > 0 {
			responseJSON = canvasInjectFilesIntoResponseJSON(responseJSON, files)
		}
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
	eventRows    []map[string]any
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

// canvasFetchQuizAssignmentSubmissionsByUser loads the assignment-submission view of each quiz,
// keyed by Canvas assignment id then Canvas user id. Canvas only exposes other learners' quiz
// answers to graders through the assignment submission's submission_history[].submission_data
// (the same source Canvas SpeedGrader reads); the quiz-submission detail and
// quiz_submissions/{id}/questions endpoints return answers only for the requesting user.
func canvasFetchQuizAssignmentSubmissionsByUser(
	ctx context.Context,
	client *http.Client,
	canvasBase, accessToken string,
	canvasCourseID int64,
	assignmentIDs []int64,
) map[int64]map[int64]map[string]any {
	out := make(map[int64]map[int64]map[string]any, len(assignmentIDs))
	if len(assignmentIDs) == 0 {
		return out
	}
	q := url.Values{}
	q.Add("include[]", "submission_history")
	q.Add("include[]", "user")

	var mu sync.Mutex
	g, gctx := canvasImportParallelGroup(ctx, len(assignmentIDs))
	for _, aid := range assignmentIDs {
		aid := aid
		g.Go(func() error {
			subs, err := canvasGetArrayPaginated(gctx, client, canvasBase, accessToken,
				fmt.Sprintf("courses/%d/assignments/%d/submissions", canvasCourseID, aid), q)
			if err != nil {
				log.Printf("canvas-import: assignment %d submissions fetch failed: %v", aid, err)
				return nil
			}
			byUser := make(map[int64]map[string]any, len(subs))
			for _, s := range subs {
				if uid := canvasCanvasUserIDFromMap(s); uid > 0 {
					byUser[uid] = s
				}
			}
			mu.Lock()
			out[aid] = byUser
			mu.Unlock()
			return nil
		})
	}
	_ = g.Wait()
	return out
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
		canvasUserID := canvasCanvasUserIDFromMap(raw)
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

	g, gctx := canvasImportParallelGroup(ctx, len(candidates))
	for i, raw := range candidates {
		i, raw := i, raw
		g.Go(func() error {
			submissionID := int64At(raw, "id")
			attempt := int32(int64At(raw, "attempt"))
			if attempt < 1 {
				attempt = 1
			}
			// Fetch per-question answers first while Canvas may still expose hash submission_data.
			questionRows, err := canvasGetQuizSubmissionQuestions(gctx, client, canvasBase, accessToken, submissionID, attempt)
			if err != nil {
				log.Printf("canvas-import: quiz submission %d questions fetch failed (quiz %d): %v", submissionID, canvasQuizID, err)
				questionRows = nil
			}
			detail, err := canvasGetQuizSubmissionDetail(gctx, client, canvasBase, accessToken, canvasCourseID, canvasQuizID, submissionID, attempt)
			if err != nil {
				log.Printf("canvas-import: quiz submission %d detail fetch failed (quiz %d): %v", submissionID, canvasQuizID, err)
				detail = raw
			}
			eventRows, err := canvasGetQuizSubmissionEvents(gctx, client, canvasBase, accessToken, canvasCourseID, canvasQuizID, submissionID, attempt)
			if err != nil {
				log.Printf("canvas-import: quiz submission %d events fetch failed (quiz %d): %v", submissionID, canvasQuizID, err)
				eventRows = nil
			}
			out[i] = canvasQuizAttemptPayload{raw: raw, detail: detail, questionRows: questionRows, eventRows: eventRows}
			return nil
		})
	}
	_ = g.Wait()
	filtered := make([]canvasQuizAttemptPayload, 0, len(out))
	for _, p := range out {
		if p.raw != nil {
			filtered = append(filtered, p)
		}
	}
	return filtered, nil
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
	canvasQuizToAssignmentID map[int64]int64,
	canvasUserToLocal map[int64]uuid.UUID,
	quizSubsByQuiz map[int64][]map[string]any,
	submissionDeps *canvasAssignmentSubmissionImportDeps,
) error {
	quizIDs := canvasQuizIDsFromMap(canvasQuizToItem)
	answerMetaByQuiz, err := canvasFetchQuizAnswerMetadataParallel(ctx, client, canvasBase, accessToken, canvasCourseID, quizIDs)
	if err != nil {
		return err
	}

	// Canvas exposes other learners' quiz answers to graders via the assignment submission's
	// submission_history[].submission_data, so prefetch those keyed by assignment + user.
	assignmentIDs := make([]int64, 0, len(canvasQuizToAssignmentID))
	for _, aid := range canvasQuizToAssignmentID {
		if aid > 0 {
			assignmentIDs = append(assignmentIDs, aid)
		}
	}
	assignmentSubsByUser := canvasFetchQuizAssignmentSubmissionsByUser(ctx, client, canvasBase, accessToken, canvasCourseID, assignmentIDs)

	for canvasQID, itemID := range canvasQuizToItem {
		questions := canvasQuizToQuestions[canvasQID]
		if len(questions) == 0 {
			var qErr error
			questions, qErr = canvasImportQuizQuestions(ctx, client, canvasBase, accessToken, canvasCourseID, canvasQID)
			if qErr != nil {
				log.Printf("canvas-import: quiz %d question refetch failed: %v", canvasQID, qErr)
			}
		}
		meta := answerMetaByQuiz[canvasQID]
		choiceMaps := meta.choiceMaps
		correctAnswerIDs := meta.correctAnswerIDs
		subs := quizSubsByQuiz[canvasQID]
		assignmentSubsForUser := assignmentSubsByUser[canvasQuizToAssignmentID[canvasQID]]

		payloads, err := canvasFetchQuizAttemptPayloadsParallel(ctx, client, canvasBase, accessToken, canvasCourseID, canvasQID, subs, canvasUserToLocal)
		if err != nil {
			return err
		}
		for _, payload := range payloads {
			questions = canvasSupplementQuestionsFromSubmissionRows(questions, payload.questionRows)
		}
		if len(questions) == 0 && len(payloads) == 0 {
			if aid, ok := canvasQuizToAssignmentID[canvasQID]; ok && aid > 0 {
				log.Printf("canvas-import: quiz %d had no Canvas quiz submissions (assignment_id=%d)", canvasQID, aid)
			}
			continue
		}
		pointsPossibleQuiz := canvasQuizPointsPossible(questions, nil)

		for _, payload := range payloads {
			raw := payload.raw
			canvasUserID := canvasCanvasUserIDFromMap(raw)
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

			// The assignment submission is listed last so its grader-visible submission_data
			// takes precedence over the (usually empty-for-graders) quiz-submission blobs.
			answers := canvasMergeSubmissionAnswers(
				[]map[string]any{payload.detail, raw, assignmentSubsForUser[canvasUserID]},
				payload.questionRows, payload.eventRows,
			)
			score, hasScore := canvasQuizSubmissionScore(raw)
			if len(answers) == 0 && !hasScore {
				if !canvasQuizSubmissionImportable(raw) {
					continue
				}
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
			if err := canvasReplaceImportedQuizResponses(ctx, tx, client, canvasBase, accessToken, submissionDeps, courseID, attemptID, questions, answers, choiceMaps, correctAnswerIDs); err != nil {
				return fmt.Errorf("save quiz responses canvas submission %d: %w", submissionID, err)
			}
		}
	}
	return nil
}
