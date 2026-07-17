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

const (
	GenStatusQueued    = "queued"
	GenStatusRunning   = "running"
	GenStatusSucceeded = "succeeded"
	GenStatusFailed    = "failed"
	GenStatusCanceled  = "canceled"

	GenSourceTopic            = "topic"
	GenSourcePassage          = "passage"
	GenSourceCourseContentRef = "course_content_ref"

	QuestionSourceAuthored    = "authored"
	QuestionSourceAIGenerated = "ai_generated"
	QuestionSourceBankImport  = "bank_import"

	MaxActiveGenJobsPerCourse = 2
)

// GenerationParams is the instructor-facing generation request.
type GenerationParams struct {
	Count               int      `json:"count"`
	Types               []string `json:"types"`
	Difficulty          string   `json:"difficulty"`
	GradeBand           string   `json:"gradeBand"`
	Language            string   `json:"language"`
	IncludeExplanations bool     `json:"includeExplanations"`
	LikeQuestionID      string   `json:"likeQuestionId,omitempty"`
	ReplaceQuestionID   string   `json:"replaceQuestionId,omitempty"`
}

// GenerationJob is one quizgame.generation_jobs row.
type GenerationJob struct {
	ID            string          `json:"id"`
	KitID         string          `json:"kitId"`
	CourseID      string          `json:"courseId"`
	RequestedBy   *string         `json:"requestedBy"`
	SourceType    string          `json:"sourceType"`
	SourceRef     json.RawMessage `json:"sourceRef"`
	Params        json.RawMessage `json:"params"`
	Status        string          `json:"status"`
	Provider      *string         `json:"provider"`
	Model         *string         `json:"model"`
	UsageID       *string         `json:"usageId"`
	Error         *string         `json:"error"`
	ResultSummary json.RawMessage `json:"resultSummary"`
	Progress      int             `json:"progress"`
	CreatedAt     time.Time       `json:"createdAt"`
	StartedAt     *time.Time      `json:"startedAt"`
	CompletedAt   *time.Time      `json:"completedAt"`
}

// ResultSummary is stored on succeeded/failed jobs.
type ResultSummary struct {
	Inserted    int      `json:"inserted"`
	Repaired    int      `json:"repaired"`
	Dropped     int      `json:"dropped"`
	QuestionIDs []string `json:"questionIds,omitempty"`
}

// CreateGenerationJobInput is the enqueue payload.
type CreateGenerationJobInput struct {
	KitID       uuid.UUID
	CourseID    uuid.UUID
	RequestedBy uuid.UUID
	SourceType  string
	SourceRef   json.RawMessage
	Params      json.RawMessage
}

func isValidGenSource(s string) bool {
	switch s {
	case GenSourceTopic, GenSourcePassage, GenSourceCourseContentRef:
		return true
	default:
		return false
	}
}

// NormalizeGenerationParams validates and fills defaults.
func NormalizeGenerationParams(p *GenerationParams) error {
	if p.Count <= 0 {
		p.Count = 5
	}
	if p.Count > 25 {
		return fmt.Errorf("quizgame: count must be between 1 and 25")
	}
	if len(p.Types) == 0 {
		p.Types = []string{QTypeMCSingle, QTypeTrueFalse}
	}
	cleaned := make([]string, 0, len(p.Types))
	seen := map[string]bool{}
	for _, t := range p.Types {
		t = strings.TrimSpace(strings.ToLower(t))
		if !isValidQuestionType(t) {
			return fmt.Errorf("quizgame: invalid question type %q", t)
		}
		if !seen[t] {
			seen[t] = true
			cleaned = append(cleaned, t)
		}
	}
	p.Types = cleaned
	p.Difficulty = strings.TrimSpace(strings.ToLower(p.Difficulty))
	if p.Difficulty == "" {
		p.Difficulty = "medium"
	}
	switch p.Difficulty {
	case "easy", "medium", "hard":
	default:
		return fmt.Errorf("quizgame: invalid difficulty")
	}
	p.GradeBand = strings.TrimSpace(p.GradeBand)
	p.Language = strings.TrimSpace(p.Language)
	if p.Language == "" {
		p.Language = "en"
	}
	return nil
}

// CountActiveGenerationJobs returns queued+running jobs for a course.
func CountActiveGenerationJobs(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (int, error) {
	if pool == nil {
		return 0, errors.New("quizgame: nil pool")
	}
	var n int
	err := pool.QueryRow(ctx, `
SELECT COUNT(*)::int FROM quizgame.generation_jobs
WHERE course_id = $1 AND status IN ('queued', 'running')
`, courseID).Scan(&n)
	return n, err
}

// CreateGenerationJob inserts a queued job.
func CreateGenerationJob(ctx context.Context, pool *pgxpool.Pool, in CreateGenerationJobInput) (*GenerationJob, error) {
	if pool == nil {
		return nil, errors.New("quizgame: nil pool")
	}
	if !isValidGenSource(in.SourceType) {
		return nil, fmt.Errorf("quizgame: invalid source_type")
	}
	if len(in.SourceRef) == 0 {
		in.SourceRef = []byte("{}")
	}
	if len(in.Params) == 0 {
		in.Params = []byte("{}")
	}
	row := pool.QueryRow(ctx, `
INSERT INTO quizgame.generation_jobs (
  kit_id, course_id, requested_by, source_type, source_ref, params, status, progress
) VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7, 0)
RETURNING id, kit_id, course_id, requested_by, source_type, source_ref, params, status,
  provider, model, usage_id, error, result_summary, progress, created_at, started_at, completed_at
`, in.KitID, in.CourseID, in.RequestedBy, in.SourceType, []byte(in.SourceRef), []byte(in.Params), GenStatusQueued)
	return scanGenerationJob(row)
}

// GetGenerationJob returns a job scoped to course+kit, or nil.
func GetGenerationJob(ctx context.Context, pool *pgxpool.Pool, courseCode, kitID, jobID string) (*GenerationJob, error) {
	_, kid, err := kitBelongsToCourse(ctx, pool, courseCode, kitID)
	if err != nil {
		return nil, err
	}
	if kid == uuid.Nil {
		return nil, nil
	}
	jid, err := uuid.Parse(jobID)
	if err != nil {
		return nil, nil
	}
	row := pool.QueryRow(ctx, `
SELECT id, kit_id, course_id, requested_by, source_type, source_ref, params, status,
  provider, model, usage_id, error, result_summary, progress, created_at, started_at, completed_at
FROM quizgame.generation_jobs
WHERE id = $1 AND kit_id = $2
`, jid, kid)
	j, err := scanGenerationJob(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return j, err
}

// MarkGenerationRunning sets status=running.
func MarkGenerationRunning(ctx context.Context, pool *pgxpool.Pool, jobID uuid.UUID, provider, model string) error {
	if pool == nil {
		return errors.New("quizgame: nil pool")
	}
	_, err := pool.Exec(ctx, `
UPDATE quizgame.generation_jobs
SET status = $2, started_at = COALESCE(started_at, NOW()), provider = $3, model = $4, progress = GREATEST(progress, 5)
WHERE id = $1 AND status = $5
`, jobID, GenStatusRunning, nullIfEmpty(provider), nullIfEmpty(model), GenStatusQueued)
	return err
}

// SetGenerationProgress updates progress 0–100 for a running job.
func SetGenerationProgress(ctx context.Context, pool *pgxpool.Pool, jobID uuid.UUID, progress int) error {
	if pool == nil {
		return errors.New("quizgame: nil pool")
	}
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	_, err := pool.Exec(ctx, `
UPDATE quizgame.generation_jobs SET progress = $2
WHERE id = $1 AND status = $3
`, jobID, progress, GenStatusRunning)
	return err
}

// CompleteGenerationJob marks succeeded with summary.
func CompleteGenerationJob(ctx context.Context, pool *pgxpool.Pool, jobID uuid.UUID, summary ResultSummary, usageID *uuid.UUID) error {
	if pool == nil {
		return errors.New("quizgame: nil pool")
	}
	raw, err := json.Marshal(summary)
	if err != nil {
		return err
	}
	var usage any
	if usageID != nil {
		usage = *usageID
	}
	_, err = pool.Exec(ctx, `
UPDATE quizgame.generation_jobs
SET status = $2, result_summary = $3::jsonb, usage_id = $4, progress = 100, completed_at = NOW(), error = NULL
WHERE id = $1 AND status IN ($5, $6)
`, jobID, GenStatusSucceeded, raw, usage, GenStatusQueued, GenStatusRunning)
	return err
}

// FailGenerationJob marks failed.
func FailGenerationJob(ctx context.Context, pool *pgxpool.Pool, jobID uuid.UUID, message string) error {
	if pool == nil {
		return errors.New("quizgame: nil pool")
	}
	_, err := pool.Exec(ctx, `
UPDATE quizgame.generation_jobs
SET status = $2, error = $3, completed_at = NOW(), progress = 100
WHERE id = $1 AND status IN ($4, $5)
`, jobID, GenStatusFailed, strings.TrimSpace(message), GenStatusQueued, GenStatusRunning)
	return err
}

// CancelGenerationJob cancels a queued/running job if requested by the actor (or any instructor).
func CancelGenerationJob(ctx context.Context, pool *pgxpool.Pool, courseCode, kitID, jobID string) (*GenerationJob, error) {
	j, err := GetGenerationJob(ctx, pool, courseCode, kitID, jobID)
	if err != nil || j == nil {
		return j, err
	}
	if j.Status != GenStatusQueued && j.Status != GenStatusRunning {
		return j, nil
	}
	jid, _ := uuid.Parse(j.ID)
	_, err = pool.Exec(ctx, `
UPDATE quizgame.generation_jobs
SET status = $2, completed_at = NOW(), error = 'Canceled by requester.', progress = 100
WHERE id = $1 AND status IN ($3, $4)
`, jid, GenStatusCanceled, GenStatusQueued, GenStatusRunning)
	if err != nil {
		return nil, err
	}
	return GetGenerationJob(ctx, pool, courseCode, kitID, jobID)
}

// IsGenerationCanceled reports whether the job was canceled.
func IsGenerationCanceled(ctx context.Context, pool *pgxpool.Pool, jobID uuid.UUID) (bool, error) {
	if pool == nil {
		return false, errors.New("quizgame: nil pool")
	}
	var status string
	err := pool.QueryRow(ctx, `SELECT status FROM quizgame.generation_jobs WHERE id = $1`, jobID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return status == GenStatusCanceled, nil
}

func scanGenerationJob(row pgx.Row) (*GenerationJob, error) {
	var j GenerationJob
	var id, kitID, courseID uuid.UUID
	var requestedBy uuid.NullUUID
	var usageID uuid.NullUUID
	var provider, model, errMsg *string
	var sourceRef, params, summary []byte
	var started, completed *time.Time
	if err := row.Scan(
		&id, &kitID, &courseID, &requestedBy, &j.SourceType, &sourceRef, &params, &j.Status,
		&provider, &model, &usageID, &errMsg, &summary, &j.Progress, &j.CreatedAt, &started, &completed,
	); err != nil {
		return nil, err
	}
	j.ID = id.String()
	j.KitID = kitID.String()
	j.CourseID = courseID.String()
	if requestedBy.Valid {
		s := requestedBy.UUID.String()
		j.RequestedBy = &s
	}
	if len(sourceRef) == 0 {
		sourceRef = []byte("{}")
	}
	j.SourceRef = json.RawMessage(sourceRef)
	if len(params) == 0 {
		params = []byte("{}")
	}
	j.Params = json.RawMessage(params)
	j.Provider = provider
	j.Model = model
	if usageID.Valid {
		s := usageID.UUID.String()
		j.UsageID = &s
	}
	j.Error = errMsg
	if len(summary) > 0 {
		j.ResultSummary = json.RawMessage(summary)
	}
	j.StartedAt = started
	j.CompletedAt = completed
	return &j, nil
}

func nullIfEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}
