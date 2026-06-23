// Package gradingagent persists grading-agent configs, runs, and per-submission results.
package gradingagent

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Status string

const (
	StatusDraft    Status = "draft"
	StatusAccepted Status = "accepted"
	StatusArchived Status = "archived"
)

type RunScope string

const (
	RunScopeCurrent  RunScope = "current"
	RunScopeUngraded RunScope = "ungraded"
	RunScopeAll      RunScope = "all"
	RunScopeAuto     RunScope = "auto"
)

type ItemStatus string

const (
	ItemSuggested  ItemStatus = "suggested"
	ItemApplied    ItemStatus = "applied"
	ItemSkipped    ItemStatus = "skipped"
	ItemFailed     ItemStatus = "failed"
	ItemOverridden ItemStatus = "overridden"
)

type ConfigRow struct {
	ID                       uuid.UUID
	CourseID                 uuid.UUID
	ModuleItemID             uuid.UUID
	Status                   Status
	Prompt                   string
	IncludeAssignmentContent bool
	IncludeRubric            bool
	ModelID                  *string
	WorkflowGraph            []byte
	AutoGradeNew             bool
	PostPolicy               string
	ConfidenceFloor          *float64
	CreatedBy                uuid.UUID
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

type RunRow struct {
	ID             uuid.UUID
	ConfigID       uuid.UUID
	Scope          RunScope
	InitiatedBy    *uuid.UUID
	AuthoredVia    *string
	TotalCount     int
	CompletedCount int
	FailedCount    int
	Status         string
	CreatedAt      time.Time
	FinishedAt     *time.Time
}

type ResultRow struct {
	ID               uuid.UUID
	RunID            *uuid.UUID
	ConfigID         uuid.UUID
	SubmissionID     uuid.UUID
	IsDryRun         bool
	SuggestedPoints  *float64
	SuggestedRubric  []byte
	Comment          *string
	Confidence       *float64
	Status           ItemStatus
	ModelID          *string
	PromptTokens     *int
	CompletionTokens *int
	CostUSD          *float64
	Error            *string
	CreatedAt        time.Time
}

type CourseConfigSummary struct {
	ID               uuid.UUID
	ModuleItemID     uuid.UUID
	AssignmentTitle  string
	AssignmentArchived bool
	Status           Status
	AutoGradeNew     bool
	HasWorkflowGraph bool
	UpdatedAt        time.Time
}

func ListConfigsByCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]CourseConfigSummary, error) {
	if pool == nil {
		return nil, errors.New("nil pool")
	}
	rows, err := pool.Query(ctx, `
SELECT g.id, g.module_item_id, csi.title, csi.archived, g.status::text, g.auto_grade_new,
       (g.workflow_graph IS NOT NULL) AS has_workflow_graph, g.updated_at
FROM assessment.grading_agent_configs g
INNER JOIN course.course_structure_items csi ON csi.id = g.module_item_id
WHERE g.course_id = $1
ORDER BY csi.title ASC
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]CourseConfigSummary, 0)
	for rows.Next() {
		var row CourseConfigSummary
		var status string
		if err := rows.Scan(
			&row.ID, &row.ModuleItemID, &row.AssignmentTitle, &row.AssignmentArchived, &status,
			&row.AutoGradeNew, &row.HasWorkflowGraph, &row.UpdatedAt,
		); err != nil {
			return nil, err
		}
		row.Status = Status(status)
		out = append(out, row)
	}
	return out, rows.Err()
}

func GetConfigByItem(ctx context.Context, pool *pgxpool.Pool, moduleItemID uuid.UUID) (*ConfigRow, error) {
	if pool == nil {
		return nil, errors.New("nil pool")
	}
	var r ConfigRow
	var status string
	var workflowGraph []byte
	err := pool.QueryRow(ctx, `
SELECT id, course_id, module_item_id, status::text, prompt,
       include_assignment_content, include_rubric, model_id, workflow_graph, auto_grade_new, post_policy,
       confidence_floor, created_by, created_at, updated_at
FROM assessment.grading_agent_configs
WHERE module_item_id = $1
`, moduleItemID).Scan(
		&r.ID, &r.CourseID, &r.ModuleItemID, &status, &r.Prompt,
		&r.IncludeAssignmentContent, &r.IncludeRubric, &r.ModelID, &workflowGraph, &r.AutoGradeNew, &r.PostPolicy,
		&r.ConfidenceFloor, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.Status = Status(status)
	r.WorkflowGraph = workflowGraph
	return &r, nil
}

type UpsertConfigInput struct {
	CourseID                 uuid.UUID
	ModuleItemID             uuid.UUID
	Status                   Status
	Prompt                   string
	IncludeAssignmentContent bool
	IncludeRubric            bool
	ModelID                  *string
	WorkflowGraph            []byte
	AutoGradeNew             bool
	CreatedBy                uuid.UUID
}

func UpsertConfig(ctx context.Context, pool *pgxpool.Pool, in UpsertConfigInput) (*ConfigRow, error) {
	if pool == nil {
		return nil, errors.New("nil pool")
	}
	var r ConfigRow
	var status string
	var workflowGraph []byte
	err := pool.QueryRow(ctx, `
INSERT INTO assessment.grading_agent_configs (
	course_id, module_item_id, status, prompt,
	include_assignment_content, include_rubric, model_id, workflow_graph, auto_grade_new, created_by, updated_at
) VALUES ($1, $2, $3::assessment.grading_agent_status, $4, $5, $6, $7, $8, $9, $10, NOW())
ON CONFLICT (module_item_id) DO UPDATE SET
	status = EXCLUDED.status,
	prompt = EXCLUDED.prompt,
	include_assignment_content = EXCLUDED.include_assignment_content,
	include_rubric = EXCLUDED.include_rubric,
	model_id = COALESCE(EXCLUDED.model_id, assessment.grading_agent_configs.model_id),
	workflow_graph = COALESCE(EXCLUDED.workflow_graph, assessment.grading_agent_configs.workflow_graph),
	auto_grade_new = EXCLUDED.auto_grade_new,
	updated_at = NOW()
RETURNING id, course_id, module_item_id, status::text, prompt,
          include_assignment_content, include_rubric, model_id, workflow_graph, auto_grade_new, post_policy,
          confidence_floor, created_by, created_at, updated_at
`, in.CourseID, in.ModuleItemID, string(in.Status), in.Prompt,
		in.IncludeAssignmentContent, in.IncludeRubric, in.ModelID, nullableJSON(in.WorkflowGraph), in.AutoGradeNew, in.CreatedBy,
	).Scan(
		&r.ID, &r.CourseID, &r.ModuleItemID, &status, &r.Prompt,
		&r.IncludeAssignmentContent, &r.IncludeRubric, &r.ModelID, &workflowGraph, &r.AutoGradeNew, &r.PostPolicy,
		&r.ConfidenceFloor, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	r.Status = Status(status)
	r.WorkflowGraph = workflowGraph
	return &r, nil
}

func CreateRun(ctx context.Context, pool *pgxpool.Pool, configID uuid.UUID, scope RunScope, initiatedBy *uuid.UUID, authoredVia *string, total int) (*RunRow, error) {
	if pool == nil {
		return nil, errors.New("nil pool")
	}
	var r RunRow
	var scopeStr string
	err := pool.QueryRow(ctx, `
INSERT INTO assessment.grading_agent_runs (config_id, scope, initiated_by, authored_via, total_count, status)
VALUES ($1, $2::assessment.grading_agent_run_scope, $3, $4, $5, 'queued')
RETURNING id, config_id, scope::text, initiated_by, authored_via, total_count, completed_count, failed_count, status, created_at, finished_at
`, configID, string(scope), initiatedBy, authoredVia, total).Scan(
		&r.ID, &r.ConfigID, &scopeStr, &r.InitiatedBy, &r.AuthoredVia, &r.TotalCount, &r.CompletedCount, &r.FailedCount,
		&r.Status, &r.CreatedAt, &r.FinishedAt,
	)
	if err != nil {
		return nil, err
	}
	r.Scope = RunScope(scopeStr)
	return &r, nil
}

func GetRun(ctx context.Context, pool *pgxpool.Pool, runID uuid.UUID) (*RunRow, error) {
	if pool == nil {
		return nil, errors.New("nil pool")
	}
	var r RunRow
	var scopeStr string
	err := pool.QueryRow(ctx, `
SELECT id, config_id, scope::text, initiated_by, authored_via, total_count, completed_count, failed_count, status, created_at, finished_at
FROM assessment.grading_agent_runs WHERE id = $1
`, runID).Scan(
		&r.ID, &r.ConfigID, &scopeStr, &r.InitiatedBy, &r.AuthoredVia, &r.TotalCount, &r.CompletedCount, &r.FailedCount,
		&r.Status, &r.CreatedAt, &r.FinishedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.Scope = RunScope(scopeStr)
	return &r, nil
}

func IncrementRunProgress(ctx context.Context, pool *pgxpool.Pool, runID uuid.UUID, failed bool) error {
	if pool == nil {
		return errors.New("nil pool")
	}
	if failed {
		_, err := pool.Exec(ctx, `
UPDATE assessment.grading_agent_runs
SET failed_count = failed_count + 1,
    completed_count = completed_count + 1,
    status = CASE WHEN completed_count + 1 >= total_count THEN 'done' ELSE status END,
    finished_at = CASE WHEN completed_count + 1 >= total_count THEN NOW() ELSE finished_at END
WHERE id = $1
`, runID)
		return err
	}
	_, err := pool.Exec(ctx, `
UPDATE assessment.grading_agent_runs
SET completed_count = completed_count + 1,
    status = CASE WHEN completed_count + 1 >= total_count THEN 'done' ELSE status END,
    finished_at = CASE WHEN completed_count + 1 >= total_count THEN NOW() ELSE finished_at END
WHERE id = $1
`, runID)
	return err
}

func MarkRunRunning(ctx context.Context, pool *pgxpool.Pool, runID uuid.UUID) error {
	_, err := pool.Exec(ctx, `UPDATE assessment.grading_agent_runs SET status = 'running' WHERE id = $1`, runID)
	return err
}

type InsertResultInput struct {
	RunID            *uuid.UUID
	ConfigID         uuid.UUID
	SubmissionID     uuid.UUID
	IsDryRun         bool
	SuggestedPoints  *float64
	SuggestedRubric  map[string]any
	Comment          *string
	Confidence       *float64
	Status           ItemStatus
	ModelID          *string
	PromptTokens     *int
	CompletionTokens *int
	CostUSD          *float64
	Error            *string
}

func InsertResult(ctx context.Context, pool *pgxpool.Pool, in InsertResultInput) (*ResultRow, error) {
	if pool == nil {
		return nil, errors.New("nil pool")
	}
	var rubricJSON []byte
	if len(in.SuggestedRubric) > 0 {
		rubricJSON, _ = json.Marshal(in.SuggestedRubric)
	}
	var r ResultRow
	var status string
	err := pool.QueryRow(ctx, `
INSERT INTO assessment.grading_agent_results (
	run_id, config_id, submission_id, is_dry_run, suggested_points, suggested_rubric,
	comment, confidence, status, model_id, prompt_tokens, completion_tokens, cost_usd, error
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::assessment.grading_agent_item_status, $10, $11, $12, $13, $14)
RETURNING id, run_id, config_id, submission_id, is_dry_run, suggested_points, suggested_rubric,
          comment, confidence, status::text, model_id, prompt_tokens, completion_tokens, cost_usd, error, created_at
`, in.RunID, in.ConfigID, in.SubmissionID, in.IsDryRun, in.SuggestedPoints, nullableJSON(rubricJSON),
		in.Comment, in.Confidence, string(in.Status), in.ModelID, in.PromptTokens, in.CompletionTokens, in.CostUSD, in.Error,
	).Scan(
		&r.ID, &r.RunID, &r.ConfigID, &r.SubmissionID, &r.IsDryRun, &r.SuggestedPoints, &r.SuggestedRubric,
		&r.Comment, &r.Confidence, &status, &r.ModelID, &r.PromptTokens, &r.CompletionTokens, &r.CostUSD, &r.Error, &r.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	r.Status = ItemStatus(status)
	return &r, nil
}

func ListResultsForRun(ctx context.Context, pool *pgxpool.Pool, runID uuid.UUID) ([]ResultRow, error) {
	rows, err := pool.Query(ctx, `
SELECT id, run_id, config_id, submission_id, is_dry_run, suggested_points, suggested_rubric,
       comment, confidence, status::text, model_id, prompt_tokens, completion_tokens, cost_usd, error, created_at
FROM assessment.grading_agent_results
WHERE run_id = $1
ORDER BY created_at ASC
`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ResultRow, 0)
	for rows.Next() {
		var r ResultRow
		var status string
		if err := rows.Scan(
			&r.ID, &r.RunID, &r.ConfigID, &r.SubmissionID, &r.IsDryRun, &r.SuggestedPoints, &r.SuggestedRubric,
			&r.Comment, &r.Confidence, &status, &r.ModelID, &r.PromptTokens, &r.CompletionTokens, &r.CostUSD, &r.Error, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		r.Status = ItemStatus(status)
		out = append(out, r)
	}
	return out, rows.Err()
}

func nullableJSON(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return b
}