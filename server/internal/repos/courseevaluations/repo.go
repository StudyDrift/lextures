// Package courseevaluations persists anonymous end-of-term course evaluation data (plan 14.7).
package courseevaluations

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrTemplateNotFound = errors.New("evaluation template not found")
	ErrWindowNotFound   = errors.New("evaluation window not found")
	ErrAlreadySubmitted = errors.New("evaluation already submitted")
	ErrWindowClosed     = errors.New("evaluation window is closed")
	ErrWindowNotOpen    = errors.New("evaluation window is not yet open")
)

// QuestionType values for template question objects.
const (
	QuestionTypeRating         = "rating"
	QuestionTypeMultipleChoice = "multiple_choice"
	QuestionTypeOpenText       = "open_text"
)

// KAnonThreshold is the minimum response count before aggregate data is shown.
const KAnonThreshold = 5

// Template is an institution-defined evaluation question bank.
type Template struct {
	ID        uuid.UUID
	OrgID     uuid.UUID
	Name      string
	Questions json.RawMessage
	CreatedBy *uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Window is a scheduled evaluation period for a course.
type Window struct {
	ID            uuid.UUID
	CourseID      uuid.UUID
	TemplateID    uuid.UUID
	OpensAt       time.Time
	ClosesAt      time.Time
	EnrolledCount int
	ResponseCount int
	CreatedAt     time.Time
}

// CreateTemplateInput is input for CreateTemplate.
type CreateTemplateInput struct {
	OrgID     uuid.UUID
	Name      string
	Questions json.RawMessage
	CreatedBy *uuid.UUID
}

// UpdateTemplateInput is input for UpdateTemplate.
type UpdateTemplateInput struct {
	Name      string
	Questions json.RawMessage
}

// CreateWindowInput is input for CreateWindow.
type CreateWindowInput struct {
	CourseID      uuid.UUID
	TemplateID    uuid.UUID
	OpensAt       time.Time
	ClosesAt      time.Time
	EnrolledCount int
}

// CreateTemplate inserts a new evaluation template and returns it.
func CreateTemplate(ctx context.Context, pool *pgxpool.Pool, in CreateTemplateInput) (*Template, error) {
	questions := in.Questions
	if len(questions) == 0 {
		questions = json.RawMessage("[]")
	}
	var t Template
	err := pool.QueryRow(ctx, `
INSERT INTO course.evaluation_templates (org_id, name, questions, created_by)
VALUES ($1, $2, $3, $4)
RETURNING id, org_id, name, questions, created_by, created_at, updated_at
`, in.OrgID, in.Name, questions, in.CreatedBy).Scan(
		&t.ID, &t.OrgID, &t.Name, &t.Questions, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// GetTemplate returns a template by ID.
func GetTemplate(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Template, error) {
	var t Template
	err := pool.QueryRow(ctx, `
SELECT id, org_id, name, questions, created_by, created_at, updated_at
FROM course.evaluation_templates
WHERE id = $1
`, id).Scan(&t.ID, &t.OrgID, &t.Name, &t.Questions, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTemplateNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// ListTemplates returns all templates for an org ordered by creation time descending.
func ListTemplates(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) ([]Template, error) {
	rows, err := pool.Query(ctx, `
SELECT id, org_id, name, questions, created_by, created_at, updated_at
FROM course.evaluation_templates
WHERE org_id = $1
ORDER BY created_at DESC
`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Template
	for rows.Next() {
		var t Template
		if err := rows.Scan(&t.ID, &t.OrgID, &t.Name, &t.Questions, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// UpdateTemplate replaces name and questions of an existing template.
// Returns ErrTemplateNotFound if no row was updated.
func UpdateTemplate(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, in UpdateTemplateInput) (*Template, error) {
	questions := in.Questions
	if len(questions) == 0 {
		questions = json.RawMessage("[]")
	}
	var t Template
	err := pool.QueryRow(ctx, `
UPDATE course.evaluation_templates
SET name = $2, questions = $3, updated_at = NOW()
WHERE id = $1
RETURNING id, org_id, name, questions, created_by, created_at, updated_at
`, id, in.Name, questions).Scan(
		&t.ID, &t.OrgID, &t.Name, &t.Questions, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTemplateNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// DeleteTemplate removes a template by ID. Returns ErrTemplateNotFound if missing.
func DeleteTemplate(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) error {
	tag, err := pool.Exec(ctx, `DELETE FROM course.evaluation_templates WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrTemplateNotFound
	}
	return nil
}

// CreateWindow schedules an evaluation window for a course.
// Callers are responsible for locking templates before window opens.
func CreateWindow(ctx context.Context, pool *pgxpool.Pool, in CreateWindowInput) (*Window, error) {
	var w Window
	err := pool.QueryRow(ctx, `
INSERT INTO course.evaluation_windows (course_id, template_id, opens_at, closes_at, enrolled_count)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, course_id, template_id, opens_at, closes_at, enrolled_count, response_count, created_at
`, in.CourseID, in.TemplateID, in.OpensAt, in.ClosesAt, in.EnrolledCount).Scan(
		&w.ID, &w.CourseID, &w.TemplateID, &w.OpensAt, &w.ClosesAt,
		&w.EnrolledCount, &w.ResponseCount, &w.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

// GetActiveWindowByCourseID returns the open window for a course at the given time, or nil.
func GetActiveWindowByCourseID(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, now time.Time) (*Window, error) {
	var w Window
	err := pool.QueryRow(ctx, `
SELECT id, course_id, template_id, opens_at, closes_at, enrolled_count, response_count, created_at
FROM course.evaluation_windows
WHERE course_id = $1
  AND opens_at <= $2
  AND closes_at > $2
ORDER BY opens_at DESC
LIMIT 1
`, courseID, now).Scan(&w.ID, &w.CourseID, &w.TemplateID, &w.OpensAt, &w.ClosesAt,
		&w.EnrolledCount, &w.ResponseCount, &w.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}

// ListWindowsByCourseID returns all evaluation windows for a course, newest first.
func ListWindowsByCourseID(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]Window, error) {
	rows, err := pool.Query(ctx, `
SELECT id, course_id, template_id, opens_at, closes_at, enrolled_count, response_count, created_at
FROM course.evaluation_windows
WHERE course_id = $1
ORDER BY opens_at DESC
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Window
	for rows.Next() {
		var w Window
		if err := rows.Scan(&w.ID, &w.CourseID, &w.TemplateID, &w.OpensAt, &w.ClosesAt,
			&w.EnrolledCount, &w.ResponseCount, &w.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// HasUserSubmitted reports whether userID has already submitted for the given window.
func HasUserSubmitted(ctx context.Context, pool *pgxpool.Pool, windowID, userID uuid.UUID) (bool, error) {
	var ok bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS(
  SELECT 1 FROM course.evaluation_submissions
  WHERE window_id = $1 AND user_id = $2
)`, windowID, userID).Scan(&ok)
	return ok, err
}

// SubmitInput is input for SubmitResponse.
type SubmitInput struct {
	WindowID uuid.UUID
	UserID   uuid.UUID
	Answers  json.RawMessage
}

// SubmitResponse records an anonymous response and marks the user as submitted.
// Returns ErrAlreadySubmitted if the user already submitted, ErrWindowClosed / ErrWindowNotOpen
// if the window is not currently open.
func SubmitResponse(ctx context.Context, pool *pgxpool.Pool, in SubmitInput, now time.Time) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Lock and validate the window.
	var opensAt, closesAt time.Time
	err = tx.QueryRow(ctx, `
SELECT opens_at, closes_at FROM course.evaluation_windows WHERE id = $1 FOR UPDATE
`, in.WindowID).Scan(&opensAt, &closesAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrWindowNotFound
	}
	if err != nil {
		return err
	}
	if now.Before(opensAt) {
		return ErrWindowNotOpen
	}
	if !now.Before(closesAt) {
		return ErrWindowClosed
	}

	// Prevent double submission.
	var already bool
	if err = tx.QueryRow(ctx, `
SELECT EXISTS(SELECT 1 FROM course.evaluation_submissions WHERE window_id = $1 AND user_id = $2)
`, in.WindowID, in.UserID).Scan(&already); err != nil {
		return err
	}
	if already {
		return ErrAlreadySubmitted
	}

	answers := in.Answers
	if len(answers) == 0 {
		answers = json.RawMessage("{}")
	}

	// Insert anonymous response (no user_id).
	if _, err = tx.Exec(ctx, `
INSERT INTO course.evaluation_responses (window_id, answers) VALUES ($1, $2)
`, in.WindowID, answers); err != nil {
		return err
	}

	// Record submission (user_id stored here, not in responses).
	if _, err = tx.Exec(ctx, `
INSERT INTO course.evaluation_submissions (window_id, user_id) VALUES ($1, $2)
`, in.WindowID, in.UserID); err != nil {
		return err
	}

	// Increment response count.
	if _, err = tx.Exec(ctx, `
UPDATE course.evaluation_windows SET response_count = response_count + 1 WHERE id = $1
`, in.WindowID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// AggregateResults holds aggregate data for a closed evaluation window.
type AggregateResults struct {
	WindowID       uuid.UUID
	ResponseCount  int
	MeetsThreshold bool
	Questions      []QuestionResult
}

// QuestionResult holds aggregate data for one question.
type QuestionResult struct {
	QuestionIndex int
	Type          string
	Text          string
	// For rating questions: average and distribution map (rating -> count).
	Average      *float64
	Distribution map[string]int
	// For open-text questions: raw texts (only if meets k-anon threshold).
	OpenTexts []string
}

// GetAggregateResults computes aggregate results for a window.
// Returns nil if the window does not meet the k-anonymity threshold (MeetsThreshold=false still returned).
func GetAggregateResults(ctx context.Context, pool *pgxpool.Pool, window *Window, templateQuestions json.RawMessage) (*AggregateResults, error) {
	res := &AggregateResults{
		WindowID:       window.ID,
		ResponseCount:  window.ResponseCount,
		MeetsThreshold: window.ResponseCount >= KAnonThreshold,
	}

	if !res.MeetsThreshold {
		return res, nil
	}

	// Parse question schema to understand question types and labels.
	type questionDef struct {
		Type     string   `json:"type"`
		Text     string   `json:"text"`
		Options  []string `json:"options"`
		Required bool     `json:"required"`
	}
	var questions []questionDef
	if err := json.Unmarshal(templateQuestions, &questions); err != nil {
		return res, nil
	}

	// Load all responses for this window.
	rows, err := pool.Query(ctx, `
SELECT answers FROM course.evaluation_responses WHERE window_id = $1
`, window.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// answers is a map of question index (string) to answer value.
	type answerMap map[string]json.RawMessage
	var allAnswers []answerMap
	for rows.Next() {
		var raw json.RawMessage
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var am answerMap
		if err := json.Unmarshal(raw, &am); err != nil {
			continue
		}
		allAnswers = append(allAnswers, am)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	res.Questions = make([]QuestionResult, len(questions))
	for i, q := range questions {
		qr := QuestionResult{
			QuestionIndex: i,
			Type:          q.Type,
			Text:          q.Text,
		}
		key := string(rune('0' + i))
		if i >= 10 {
			key = string(rune(i + 48)) // simple key
		}

		switch q.Type {
		case QuestionTypeRating:
			dist := map[string]int{}
			total := 0
			count := 0
			for _, am := range allAnswers {
				raw, ok := am[key]
				if !ok {
					continue
				}
				var val string
				if err := json.Unmarshal(raw, &val); err != nil {
					continue
				}
				dist[val]++
				var numVal int
				if err := json.Unmarshal(raw, &numVal); err == nil {
					total += numVal
					count++
				}
			}
			qr.Distribution = dist
			if count > 0 {
				avg := float64(total) / float64(count)
				qr.Average = &avg
			}

		case QuestionTypeMultipleChoice:
			dist := map[string]int{}
			for _, am := range allAnswers {
				raw, ok := am[key]
				if !ok {
					continue
				}
				var val string
				if err := json.Unmarshal(raw, &val); err != nil {
					continue
				}
				dist[val]++
			}
			qr.Distribution = dist

		case QuestionTypeOpenText:
			var texts []string
			for _, am := range allAnswers {
				raw, ok := am[key]
				if !ok {
					continue
				}
				var val string
				if err := json.Unmarshal(raw, &val); err != nil {
					continue
				}
				if val != "" {
					texts = append(texts, val)
				}
			}
			qr.OpenTexts = texts
		}

		res.Questions[i] = qr
	}

	return res, nil
}

// AdminReportRow is one row in the cross-section evaluation report.
type AdminReportRow struct {
	CourseID      uuid.UUID
	CourseCode    string
	CourseTitle   string
	WindowID      uuid.UUID
	OpensAt       time.Time
	ClosesAt      time.Time
	EnrolledCount int
	ResponseCount int
	CompletionPct float64
	// AverageRating is the mean of all rating question averages across the window (nil if threshold not met).
	AverageRating *float64
}

// ListAdminReport returns cross-section evaluation data for an org.
// If closedOnly is true, only past windows are included.
func ListAdminReport(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, closedOnly bool) ([]AdminReportRow, error) {
	filter := ""
	if closedOnly {
		filter = "AND ew.closes_at < NOW()"
	}
	query := `
SELECT
    c.id,
    c.course_code,
    c.title,
    ew.id,
    ew.opens_at,
    ew.closes_at,
    ew.enrolled_count,
    ew.response_count
FROM course.evaluation_windows ew
JOIN course.courses c ON c.id = ew.course_id
WHERE c.org_id = $1
` + filter + `
ORDER BY ew.closes_at DESC, c.title
`
	rows, err := pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AdminReportRow
	for rows.Next() {
		var r AdminReportRow
		if err := rows.Scan(
			&r.CourseID, &r.CourseCode, &r.CourseTitle,
			&r.WindowID, &r.OpensAt, &r.ClosesAt,
			&r.EnrolledCount, &r.ResponseCount,
		); err != nil {
			return nil, err
		}
		if r.EnrolledCount > 0 {
			r.CompletionPct = float64(r.ResponseCount) / float64(r.EnrolledCount) * 100
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
