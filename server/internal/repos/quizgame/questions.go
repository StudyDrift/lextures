package quizgame

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrVersionConflict indicates If-Match / version stamp mismatch.
var ErrVersionConflict = errors.New("quizgame: version conflict")

// Question is one quizgame.questions row.
type Question struct {
	ID               string          `json:"id"`
	KitID            string          `json:"kitId"`
	Position         int             `json:"position"`
	QuestionType     string          `json:"questionType"`
	Prompt           string          `json:"prompt"`
	PromptMediaRef   *string         `json:"promptMediaRef"`
	PromptMediaAlt   *string         `json:"promptMediaAlt"`
	Options          json.RawMessage `json:"options"`
	CorrectAnswer    json.RawMessage `json:"correctAnswer"`
	TimeLimitSeconds int             `json:"timeLimitSeconds"`
	PointsStyle      string          `json:"pointsStyle"`
	AnswerShuffle    bool            `json:"answerShuffle"`
	Explanation      *string         `json:"explanation"`
	SourceQuestionID *string         `json:"sourceQuestionId"`
	Version          int             `json:"version"`
	CreatedAt        time.Time       `json:"createdAt"`
	UpdatedAt        time.Time       `json:"updatedAt"`
}

// CreateQuestionInput is used for create and as the merge base for patch.
type CreateQuestionInput struct {
	QuestionType     string
	Prompt           string
	PromptMediaRef   *string
	PromptMediaAlt   *string
	Options          []Option
	CorrectAnswer    json.RawMessage
	TimeLimitSeconds int
	PointsStyle      string
	AnswerShuffle    *bool
	Explanation      *string
	SourceQuestionID *string
}

// PatchQuestionInput is a partial update with optimistic concurrency.
type PatchQuestionInput struct {
	ExpectedVersion  int
	QuestionType     *string
	Prompt           *string
	PromptMediaRef   **string // nil = omit; non-nil pointer to nil = clear
	PromptMediaAlt   **string
	Options          *[]Option
	CorrectAnswer    *json.RawMessage
	TimeLimitSeconds *int
	PointsStyle      *string
	AnswerShuffle    *bool
	Explanation      **string
}

// ReorderItem is one entry in a bulk reorder.
type ReorderItem struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
}

// ValidateResult is the kit readiness payload.
type ValidateResult struct {
	IsReady bool         `json:"isReady"`
	Issues  []ValidIssue `json:"issues"`
}

// BankCandidate is a lightweight bank row for the import drawer.
type BankCandidate struct {
	ID           string `json:"id"`
	QuestionType string `json:"questionType"`
	Stem         string `json:"stem"`
	Status       string `json:"status"`
}

func selectQuestionCols() string {
	return `q.id, q.kit_id, q.position, q.question_type::text, q.prompt,
		q.prompt_media_ref, q.prompt_media_alt, q.options, q.correct_answer,
		q.time_limit_seconds, q.points_style::text, q.answer_shuffle, q.explanation,
		q.source_question_id, q.version, q.created_at, q.updated_at`
}

func scanQuestion(row pgx.Row) (Question, error) {
	var q Question
	var id, kitID uuid.UUID
	var source uuid.NullUUID
	var opts, corr []byte
	if err := row.Scan(
		&id, &kitID, &q.Position, &q.QuestionType, &q.Prompt,
		&q.PromptMediaRef, &q.PromptMediaAlt, &opts, &corr,
		&q.TimeLimitSeconds, &q.PointsStyle, &q.AnswerShuffle, &q.Explanation,
		&source, &q.Version, &q.CreatedAt, &q.UpdatedAt,
	); err != nil {
		return Question{}, err
	}
	q.ID = id.String()
	q.KitID = kitID.String()
	if len(opts) == 0 {
		opts = []byte("[]")
	}
	q.Options = json.RawMessage(opts)
	if len(corr) > 0 {
		q.CorrectAnswer = json.RawMessage(corr)
	}
	if source.Valid {
		s := source.UUID.String()
		q.SourceQuestionID = &s
	}
	return q, nil
}

func kitBelongsToCourse(ctx context.Context, pool *pgxpool.Pool, courseCode, kitID string) (uuid.UUID, uuid.UUID, error) {
	kid, err := uuid.Parse(kitID)
	if err != nil {
		return uuid.Nil, uuid.Nil, nil
	}
	var courseID uuid.UUID
	err = pool.QueryRow(ctx, `
		SELECT c.id, k.id
		FROM quizgame.kits k
		INNER JOIN course.courses c ON c.id = k.course_id
		WHERE c.course_code = $1 AND k.id = $2
	`, courseCode, kid).Scan(&courseID, &kid)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, uuid.Nil, nil
	}
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	return courseID, kid, nil
}

// ListQuestions returns ordered questions for a kit.
func ListQuestions(ctx context.Context, pool *pgxpool.Pool, courseCode, kitID string) ([]Question, error) {
	_, kid, err := kitBelongsToCourse(ctx, pool, courseCode, kitID)
	if err != nil {
		return nil, err
	}
	if kid == uuid.Nil {
		return nil, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT `+selectQuestionCols()+`
		FROM quizgame.questions q
		WHERE q.kit_id = $1
		ORDER BY q.position ASC
	`, kid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Question, 0)
	for rows.Next() {
		q, err := scanQuestion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, rows.Err()
}

// GetQuestion returns one question scoped to course+kit.
func GetQuestion(ctx context.Context, pool *pgxpool.Pool, courseCode, kitID, questionID string) (*Question, error) {
	_, kid, err := kitBelongsToCourse(ctx, pool, courseCode, kitID)
	if err != nil {
		return nil, err
	}
	if kid == uuid.Nil {
		return nil, nil
	}
	qid, err := uuid.Parse(questionID)
	if err != nil {
		return nil, nil
	}
	row := pool.QueryRow(ctx, `
		SELECT `+selectQuestionCols()+`
		FROM quizgame.questions q
		WHERE q.kit_id = $1 AND q.id = $2
	`, kid, qid)
	q, err := scanQuestion(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &q, nil
}

func nextPosition(ctx context.Context, tx pgx.Tx, kitID uuid.UUID) (int, error) {
	var pos int
	err := tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(position), -1) + 1 FROM quizgame.questions WHERE kit_id = $1
	`, kitID).Scan(&pos)
	return pos, err
}

// CreateQuestion appends a question to a kit.
func CreateQuestion(ctx context.Context, pool *pgxpool.Pool, courseCode, kitID string, in CreateQuestionInput) (*Question, error) {
	if err := NormalizeCreateInput(&in); err != nil {
		return nil, err
	}
	_, kid, err := kitBelongsToCourse(ctx, pool, courseCode, kitID)
	if err != nil {
		return nil, err
	}
	if kid == uuid.Nil {
		return nil, nil
	}
	optsJSON, err := marshalOptions(in.Options)
	if err != nil {
		return nil, err
	}
	shuffle := true
	if in.AnswerShuffle != nil {
		shuffle = *in.AnswerShuffle
	}
	var source any
	if in.SourceQuestionID != nil {
		sid, err := uuid.Parse(*in.SourceQuestionID)
		if err != nil {
			return nil, fmt.Errorf("quizgame: invalid source_question_id")
		}
		source = sid
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	pos, err := nextPosition(ctx, tx, kid)
	if err != nil {
		return nil, err
	}
	var corr any
	if len(in.CorrectAnswer) > 0 {
		corr = []byte(in.CorrectAnswer)
	}
	row := tx.QueryRow(ctx, `
		INSERT INTO quizgame.questions (
			kit_id, position, question_type, prompt, prompt_media_ref, prompt_media_alt,
			options, correct_answer, time_limit_seconds, points_style, answer_shuffle,
			explanation, source_question_id
		) VALUES (
			$1, $2, $3::quizgame.question_type, $4, $5, $6,
			$7::jsonb, $8::jsonb, $9, $10::quizgame.points_style, $11,
			$12, $13
		)
		RETURNING id, kit_id, position, question_type::text, prompt,
			prompt_media_ref, prompt_media_alt, options, correct_answer,
			time_limit_seconds, points_style::text, answer_shuffle, explanation,
			source_question_id, version, created_at, updated_at
	`, kid, pos, in.QuestionType, in.Prompt, in.PromptMediaRef, in.PromptMediaAlt,
		[]byte(optsJSON), corr, in.TimeLimitSeconds, in.PointsStyle, shuffle,
		in.Explanation, source)
	q, err := scanQuestion(row)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &q, nil
}

// PatchQuestion updates a question with optimistic concurrency (expected version).
func PatchQuestion(ctx context.Context, pool *pgxpool.Pool, courseCode, kitID, questionID string, in PatchQuestionInput) (*Question, error) {
	existing, err := GetQuestion(ctx, pool, courseCode, kitID, questionID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, nil
	}
	if in.ExpectedVersion > 0 && existing.Version != in.ExpectedVersion {
		return nil, ErrVersionConflict
	}

	merged := CreateQuestionInput{
		QuestionType:     existing.QuestionType,
		Prompt:           existing.Prompt,
		PromptMediaRef:   existing.PromptMediaRef,
		PromptMediaAlt:   existing.PromptMediaAlt,
		CorrectAnswer:    existing.CorrectAnswer,
		TimeLimitSeconds: existing.TimeLimitSeconds,
		PointsStyle:      existing.PointsStyle,
		Explanation:      existing.Explanation,
	}
	opts, err := unmarshalOptions(existing.Options)
	if err != nil {
		return nil, err
	}
	merged.Options = opts
	shuffle := existing.AnswerShuffle
	merged.AnswerShuffle = &shuffle

	if in.QuestionType != nil {
		merged.QuestionType = *in.QuestionType
	}
	if in.Prompt != nil {
		merged.Prompt = *in.Prompt
	}
	if in.PromptMediaRef != nil {
		merged.PromptMediaRef = *in.PromptMediaRef
	}
	if in.PromptMediaAlt != nil {
		merged.PromptMediaAlt = *in.PromptMediaAlt
	}
	if in.Options != nil {
		merged.Options = *in.Options
	}
	if in.CorrectAnswer != nil {
		merged.CorrectAnswer = *in.CorrectAnswer
	}
	if in.TimeLimitSeconds != nil {
		merged.TimeLimitSeconds = *in.TimeLimitSeconds
	}
	if in.PointsStyle != nil {
		merged.PointsStyle = *in.PointsStyle
	}
	if in.AnswerShuffle != nil {
		merged.AnswerShuffle = in.AnswerShuffle
	}
	if in.Explanation != nil {
		merged.Explanation = *in.Explanation
	}
	if err := NormalizeCreateInput(&merged); err != nil {
		return nil, err
	}
	optsJSON, err := marshalOptions(merged.Options)
	if err != nil {
		return nil, err
	}
	ansShuffle := true
	if merged.AnswerShuffle != nil {
		ansShuffle = *merged.AnswerShuffle
	}
	var corr any
	if len(merged.CorrectAnswer) > 0 {
		corr = []byte(merged.CorrectAnswer)
	}
	qid, _ := uuid.Parse(questionID)
	expected := existing.Version
	if in.ExpectedVersion > 0 {
		expected = in.ExpectedVersion
	}

	row := pool.QueryRow(ctx, `
		UPDATE quizgame.questions
		SET question_type = $3::quizgame.question_type,
			prompt = $4,
			prompt_media_ref = $5,
			prompt_media_alt = $6,
			options = $7::jsonb,
			correct_answer = $8::jsonb,
			time_limit_seconds = $9,
			points_style = $10::quizgame.points_style,
			answer_shuffle = $11,
			explanation = $12,
			version = version + 1,
			updated_at = NOW()
		WHERE kit_id = $1 AND id = $2 AND version = $13
		RETURNING id, kit_id, position, question_type::text, prompt,
			prompt_media_ref, prompt_media_alt, options, correct_answer,
			time_limit_seconds, points_style::text, answer_shuffle, explanation,
			source_question_id, version, created_at, updated_at
	`, existing.KitID, qid, merged.QuestionType, merged.Prompt, merged.PromptMediaRef, merged.PromptMediaAlt,
		[]byte(optsJSON), corr, merged.TimeLimitSeconds, merged.PointsStyle, ansShuffle,
		merged.Explanation, expected)
	q, err := scanQuestion(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrVersionConflict
	}
	if err != nil {
		return nil, err
	}
	return &q, nil
}

// DeleteQuestion removes a question and recompacts positions.
func DeleteQuestion(ctx context.Context, pool *pgxpool.Pool, courseCode, kitID, questionID string) (bool, error) {
	_, kid, err := kitBelongsToCourse(ctx, pool, courseCode, kitID)
	if err != nil {
		return false, err
	}
	if kid == uuid.Nil {
		return false, nil
	}
	qid, err := uuid.Parse(questionID)
	if err != nil {
		return false, nil
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `DELETE FROM quizgame.questions WHERE kit_id = $1 AND id = $2`, kid, qid)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}
	if err := recompactPositions(ctx, tx, kid); err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

// DuplicateQuestion copies a question to the end of the kit.
func DuplicateQuestion(ctx context.Context, pool *pgxpool.Pool, courseCode, kitID, questionID string) (*Question, error) {
	src, err := GetQuestion(ctx, pool, courseCode, kitID, questionID)
	if err != nil || src == nil {
		return src, err
	}
	opts, err := unmarshalOptions(src.Options)
	if err != nil {
		return nil, err
	}
	shuffle := src.AnswerShuffle
	return CreateQuestion(ctx, pool, courseCode, kitID, CreateQuestionInput{
		QuestionType:     src.QuestionType,
		Prompt:           src.Prompt,
		PromptMediaRef:   src.PromptMediaRef,
		PromptMediaAlt:   src.PromptMediaAlt,
		Options:          opts,
		CorrectAnswer:    src.CorrectAnswer,
		TimeLimitSeconds: src.TimeLimitSeconds,
		PointsStyle:      src.PointsStyle,
		AnswerShuffle:    &shuffle,
		Explanation:      src.Explanation,
	})
}

func recompactPositions(ctx context.Context, tx pgx.Tx, kitID uuid.UUID) error {
	rows, err := tx.Query(ctx, `
		SELECT id FROM quizgame.questions WHERE kit_id = $1 ORDER BY position ASC, created_at ASC
	`, kitID)
	if err != nil {
		return err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	// Move to temporary high positions to avoid unique conflicts, then set final.
	for i, id := range ids {
		if _, err := tx.Exec(ctx, `UPDATE quizgame.questions SET position = $3 WHERE kit_id = $1 AND id = $2`,
			kitID, id, 100000+i); err != nil {
			return err
		}
	}
	for i, id := range ids {
		if _, err := tx.Exec(ctx, `UPDATE quizgame.questions SET position = $3, updated_at = NOW() WHERE kit_id = $1 AND id = $2`,
			kitID, id, i); err != nil {
			return err
		}
	}
	return nil
}

// ReorderQuestions applies a bulk position update in a transaction.
func ReorderQuestions(ctx context.Context, pool *pgxpool.Pool, courseCode, kitID string, items []ReorderItem) ([]Question, error) {
	_, kid, err := kitBelongsToCourse(ctx, pool, courseCode, kitID)
	if err != nil {
		return nil, err
	}
	if kid == uuid.Nil {
		return nil, nil
	}
	if len(items) == 0 {
		return ListQuestions(ctx, pool, courseCode, kitID)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Stage to high positions first.
	for i, it := range items {
		id, err := uuid.Parse(it.ID)
		if err != nil {
			return nil, fmt.Errorf("quizgame: invalid question id in reorder")
		}
		if _, err := tx.Exec(ctx, `
			UPDATE quizgame.questions SET position = $3 WHERE kit_id = $1 AND id = $2
		`, kid, id, 200000+i); err != nil {
			return nil, err
		}
	}
	for _, it := range items {
		id, _ := uuid.Parse(it.ID)
		if it.Position < 0 {
			return nil, fmt.Errorf("quizgame: position must be >= 0")
		}
		if _, err := tx.Exec(ctx, `
			UPDATE quizgame.questions SET position = $3, updated_at = NOW() WHERE kit_id = $1 AND id = $2
		`, kid, id, it.Position); err != nil {
			return nil, err
		}
	}
	if err := recompactPositions(ctx, tx, kid); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return ListQuestions(ctx, pool, courseCode, kitID)
}

// ValidateKit computes readiness for hosting.
func ValidateKit(ctx context.Context, pool *pgxpool.Pool, courseCode, kitID string) (*ValidateResult, error) {
	qs, err := ListQuestions(ctx, pool, courseCode, kitID)
	if err != nil {
		return nil, err
	}
	if qs == nil {
		return nil, nil
	}
	var issues []ValidIssue
	if len(qs) == 0 {
		issues = append(issues, ValidIssue{
			QuestionID: "",
			Code:       "empty_kit",
			Message:    "Add at least one question before hosting.",
		})
	}
	for _, q := range qs {
		issues = append(issues, ValidateQuestionReady(q)...)
	}
	return &ValidateResult{IsReady: len(issues) == 0, Issues: issues}, nil
}

// ListBankCandidates returns course bank questions for the import drawer.
func ListBankCandidates(ctx context.Context, pool *pgxpool.Pool, courseCode, query string, limit int) ([]BankCandidate, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	q := strings.TrimSpace(query)
	args := []any{courseCode}
	where := `WHERE c.course_code = $1 AND c.question_bank_enabled = TRUE`
	if q != "" {
		args = append(args, "%"+q+"%")
		where += fmt.Sprintf(` AND q.stem ILIKE $%d`, len(args))
	}
	args = append(args, limit)
	rows, err := pool.Query(ctx, `
		SELECT q.id, q.question_type::text, q.stem, q.status::text
		FROM course.questions q
		INNER JOIN course.courses c ON c.id = q.course_id
		`+where+`
		AND q.question_type::text IN ('mc_single','mc_multiple','true_false','short_answer','numeric','ordering')
		ORDER BY q.updated_at DESC
		LIMIT $`+fmt.Sprintf("%d", len(args))+`
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]BankCandidate, 0)
	for rows.Next() {
		var id uuid.UUID
		var bc BankCandidate
		if err := rows.Scan(&id, &bc.QuestionType, &bc.Stem, &bc.Status); err != nil {
			return nil, err
		}
		bc.ID = id.String()
		out = append(out, bc)
	}
	return out, rows.Err()
}

// ImportBankQuestions copies bank items into the kit (copy-with-link).
func ImportBankQuestions(ctx context.Context, pool *pgxpool.Pool, courseCode, kitID string, questionIDs []string) ([]Question, error) {
	courseID, kid, err := kitBelongsToCourse(ctx, pool, courseCode, kitID)
	if err != nil {
		return nil, err
	}
	if kid == uuid.Nil {
		return nil, nil
	}
	var bankOn bool
	if err := pool.QueryRow(ctx, `
		SELECT question_bank_enabled FROM course.courses WHERE id = $1
	`, courseID).Scan(&bankOn); err != nil {
		return nil, err
	}
	if !bankOn {
		return nil, fmt.Errorf("quizgame: question bank is not enabled for this course")
	}
	created := make([]Question, 0, len(questionIDs))
	for _, idStr := range questionIDs {
		bqID, err := uuid.Parse(idStr)
		if err != nil {
			return nil, fmt.Errorf("quizgame: invalid question id")
		}
		var stem, qtype string
		var opts, corr []byte
		var expl *string
		err = pool.QueryRow(ctx, `
			SELECT question_type::text, stem, options, correct_answer, explanation
			FROM course.questions WHERE course_id = $1 AND id = $2
		`, courseID, bqID).Scan(&qtype, &stem, &opts, &corr, &expl)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("quizgame: bank question not found")
		}
		if err != nil {
			return nil, err
		}
		mapped, ok := MapBankQuestionType(qtype)
		if !ok {
			return nil, fmt.Errorf("quizgame: bank question type %q is not importable", qtype)
		}
		in, err := bankRowToCreateInput(mapped, stem, opts, corr, expl, bqID.String())
		if err != nil {
			return nil, err
		}
		q, err := CreateQuestion(ctx, pool, courseCode, kitID, in)
		if err != nil {
			return nil, err
		}
		if q != nil {
			created = append(created, *q)
		}
	}
	return created, nil
}

func bankRowToCreateInput(mappedType, stem string, opts, corr []byte, expl *string, sourceID string) (CreateQuestionInput, error) {
	in := CreateQuestionInput{
		QuestionType:     mappedType,
		Prompt:           stem,
		TimeLimitSeconds: defTimeLimit,
		PointsStyle:      PointsStandard,
		SourceQuestionID: &sourceID,
		Explanation:      expl,
	}
	switch mappedType {
	case QTypeMCSingle, QTypeMCMultiple, QTypeTrueFalse, QTypePoll, QTypeOrdering:
		options, correctIdx := parseBankOptions(opts, corr, mappedType)
		in.Options = options
		if mappedType == QTypeOrdering {
			order := make([]string, 0, len(options))
			for _, o := range options {
				order = append(order, o.ID)
			}
			b, _ := json.Marshal(OrderingCorrect{Order: order})
			in.CorrectAnswer = b
		}
		_ = correctIdx
	case QTypeTypeAnswer:
		in.Options = nil
		accepted := parseBankShortAnswers(corr)
		b, _ := json.Marshal(TypeAnswerCorrect{Accepted: accepted})
		in.CorrectAnswer = b
	case QTypeNumeric:
		in.Options = nil
		nc := parseBankNumeric(corr)
		b, _ := json.Marshal(nc)
		in.CorrectAnswer = b
	}
	return in, nil
}

func parseBankOptions(opts, corr []byte, mappedType string) ([]Option, int) {
	correctIdx := -1
	if len(corr) > 0 {
		var m map[string]any
		if json.Unmarshal(corr, &m) == nil {
			if v, ok := m["correctChoiceIndex"].(float64); ok {
				correctIdx = int(v)
			}
		}
	}
	var raw []any
	if len(opts) > 0 {
		_ = json.Unmarshal(opts, &raw)
	}
	if len(raw) == 0 {
		return defaultOptionsForType(mappedType), correctIdx
	}
	out := make([]Option, 0, len(raw))
	for i, entry := range raw {
		text := ""
		id := fmt.Sprintf("opt-%d", i+1)
		switch v := entry.(type) {
		case string:
			text = v
		case map[string]any:
			if t, ok := v["text"].(string); ok {
				text = t
			} else if t, ok := v["label"].(string); ok {
				text = t
			}
			if sid, ok := v["id"].(string); ok && sid != "" {
				id = sid
			}
		}
		isCorrect := i == correctIdx
		if mappedType == QTypeMCMultiple {
			if m, ok := entry.(map[string]any); ok {
				if c, ok := m["isCorrect"].(bool); ok {
					isCorrect = c
				}
			}
		}
		out = append(out, Option{ID: id, Text: text, IsCorrect: isCorrect})
	}
	if mappedType == QTypeTrueFalse && len(out) == 2 && correctIdx < 0 {
		out[0].IsCorrect = true
	}
	return out, correctIdx
}

func parseBankShortAnswers(corr []byte) []AcceptedAnswer {
	if len(corr) == 0 {
		return []AcceptedAnswer{{Text: "", MatchMode: "case_insensitive"}}
	}
	var m map[string]any
	if json.Unmarshal(corr, &m) == nil {
		if arr, ok := m["accepted"].([]any); ok {
			out := make([]AcceptedAnswer, 0, len(arr))
			for _, a := range arr {
				out = append(out, AcceptedAnswer{Text: fmt.Sprint(a), MatchMode: "case_insensitive"})
			}
			return out
		}
		if s, ok := m["answer"].(string); ok {
			return []AcceptedAnswer{{Text: s, MatchMode: "case_insensitive"}}
		}
	}
	var s string
	if json.Unmarshal(corr, &s) == nil {
		return []AcceptedAnswer{{Text: s, MatchMode: "case_insensitive"}}
	}
	return []AcceptedAnswer{{Text: string(corr), MatchMode: "case_insensitive"}}
}

func parseBankNumeric(corr []byte) NumericCorrect {
	var nc NumericCorrect
	if len(corr) == 0 {
		return nc
	}
	_ = json.Unmarshal(corr, &nc)
	if nc.Tolerance == 0 {
		var m map[string]any
		if json.Unmarshal(corr, &m) == nil {
			if v, ok := m["value"].(float64); ok {
				nc.Value = v
			}
			if t, ok := m["tolerance"].(float64); ok {
				nc.Tolerance = t
			}
		}
	}
	return nc
}

// PushQuestionToBank creates a course.questions row from a kit question.
func PushQuestionToBank(ctx context.Context, pool *pgxpool.Pool, courseCode, kitID, questionID string, createdBy uuid.UUID) (string, error) {
	courseID, kid, err := kitBelongsToCourse(ctx, pool, courseCode, kitID)
	if err != nil {
		return "", err
	}
	if kid == uuid.Nil {
		return "", fmt.Errorf("quizgame: kit not found")
	}
	var bankOn bool
	if err := pool.QueryRow(ctx, `
		SELECT question_bank_enabled FROM course.courses WHERE id = $1
	`, courseID).Scan(&bankOn); err != nil {
		return "", err
	}
	if !bankOn {
		return "", fmt.Errorf("quizgame: question bank is not enabled for this course")
	}
	q, err := GetQuestion(ctx, pool, courseCode, kitID, questionID)
	if err != nil {
		return "", err
	}
	if q == nil {
		return "", fmt.Errorf("quizgame: question not found")
	}
	bankType, ok := MapToBankQuestionType(q.QuestionType)
	if !ok {
		return "", fmt.Errorf("quizgame: cannot push question type %q to bank", q.QuestionType)
	}
	opts := q.Options
	corr := q.CorrectAnswer
	if q.QuestionType == QTypeMCSingle || q.QuestionType == QTypeTrueFalse {
		parsed, _ := unmarshalOptions(q.Options)
		idx := -1
		for i, o := range parsed {
			if o.IsCorrect {
				idx = i
				break
			}
		}
		texts := make([]string, len(parsed))
		for i, o := range parsed {
			texts[i] = o.Text
		}
		opts, _ = json.Marshal(texts)
		if idx >= 0 {
			corr, _ = json.Marshal(map[string]any{"correctChoiceIndex": idx})
		}
	}
	meta, _ := json.Marshal(map[string]string{
		"source":       "live_quiz_kit",
		"kitId":        kitID,
		"kitQuestionId": questionID,
	})
	var newID uuid.UUID
	err = pool.QueryRow(ctx, `
		INSERT INTO course.questions (
			course_id, question_type, stem, options, correct_answer, explanation,
			points, status, shared, source, metadata, created_by, is_published
		) VALUES (
			$1, $2::course.question_type, $3, $4, $5, $6,
			1.0, 'active'::course.question_status, false, 'live_quiz', $7, $8, TRUE
		)
		RETURNING id
	`, courseID, bankType, q.Prompt, nullableJSON(opts), nullableJSON(corr), q.Explanation, meta, createdBy).Scan(&newID)
	if err != nil {
		return "", err
	}
	// Link kit question to the new bank row.
	_, _ = pool.Exec(ctx, `
		UPDATE quizgame.questions SET source_question_id = $2, updated_at = NOW()
		WHERE id = $1
	`, q.ID, newID)
	return newID.String(), nil
}

func nullableJSON(r json.RawMessage) any {
	if len(r) == 0 {
		return nil
	}
	return []byte(r)
}
