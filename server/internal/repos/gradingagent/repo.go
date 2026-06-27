// Package gradingagent persists grading-agent configs, runs, and per-submission results.
package gradingagent

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
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

type RunMode string

const (
	RunModeSuggest RunMode = "suggest"
	RunModeApply   RunMode = "apply"
)

type ItemStatus string

const (
	ItemSuggested  ItemStatus = "suggested"
	ItemApplied    ItemStatus = "applied"
	ItemSkipped    ItemStatus = "skipped"
	ItemFailed     ItemStatus = "failed"
	ItemOverridden ItemStatus = "overridden"
	ItemFlagged    ItemStatus = "flagged"
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

const (
	RunStatusQueued    = "queued"
	RunStatusRunning   = "running"
	RunStatusDone      = "done"
	RunStatusFailed    = "failed"
	RunStatusCancelled = "cancelled"
)

type RunRow struct {
	ID             uuid.UUID
	ConfigID       uuid.UUID
	Scope          RunScope
	Mode           RunMode
	InitiatedBy    *uuid.UUID
	AuthoredVia    *string
	Filter         []byte
	BudgetUSD      *float64
	TotalCount     int
	CompletedCount int
	FailedCount    int
	Status         string
	CreatedAt      time.Time
	FinishedAt     *time.Time
	CancelledAt    *time.Time
	CancelledBy    *uuid.UUID
}

// RunFilter is the persisted / request filter for section, group, or explicit submissions (GA-M5).
type RunFilter struct {
	SectionID     *uuid.UUID
	GroupID       *uuid.UUID
	SubmissionIDs []uuid.UUID
}

func (f *RunFilter) IsEmpty() bool {
	if f == nil {
		return true
	}
	return f.SectionID == nil && f.GroupID == nil && len(f.SubmissionIDs) == 0
}

func ParseRunFilterJSON(raw []byte) (*RunFilter, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var payload struct {
		SectionID     *string  `json:"sectionId"`
		GroupID       *string  `json:"groupId"`
		SubmissionIDs []string `json:"submissionIds"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	out := &RunFilter{}
	if payload.SectionID != nil {
		s := strings.TrimSpace(*payload.SectionID)
		if s != "" {
			id, err := uuid.Parse(s)
			if err != nil {
				return nil, err
			}
			out.SectionID = &id
		}
	}
	if payload.GroupID != nil {
		s := strings.TrimSpace(*payload.GroupID)
		if s != "" {
			id, err := uuid.Parse(s)
			if err != nil {
				return nil, err
			}
			out.GroupID = &id
		}
	}
	for _, sid := range payload.SubmissionIDs {
		s := strings.TrimSpace(sid)
		if s == "" {
			continue
		}
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, err
		}
		out.SubmissionIDs = append(out.SubmissionIDs, id)
	}
	if out.IsEmpty() {
		return nil, nil
	}
	return out, nil
}

func (f *RunFilter) ToJSON() ([]byte, error) {
	if f == nil || f.IsEmpty() {
		return nil, nil
	}
	payload := map[string]any{}
	if f.SectionID != nil {
		payload["sectionId"] = f.SectionID.String()
	}
	if f.GroupID != nil {
		payload["groupId"] = f.GroupID.String()
	}
	if len(f.SubmissionIDs) > 0 {
		ids := make([]string, 0, len(f.SubmissionIDs))
		for _, id := range f.SubmissionIDs {
			ids = append(ids, id.String())
		}
		payload["submissionIds"] = ids
	}
	return json.Marshal(payload)
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
	FlagReason       *string
	FlagPriority     *string
	HeldReason       *string
	HeldAt           *time.Time
	HeldQueue        *string
	InputModality    *string
	ResolvedAt       *time.Time
	ResolvedBy       *uuid.UUID
	CreatedAt        time.Time
}

// RunSummary aggregates a batch run with optional cost totals for run history (GA-M1 / GA-M7).
type RunSummary struct {
	RunRow
	CostUSD          *float64
	PromptTokens     *int
	CompletionTokens *int
	ModelID          *string
}

// ReviewQueueItem is a deduped held or flagged result for the persistent review inbox (GA-M1).
type ReviewQueueItem struct {
	ResultRow
	RunCreatedAt *time.Time
}

type CourseConfigSummary struct {
	ID                 uuid.UUID
	ModuleItemID       uuid.UUID
	ItemKind           string
	AssignmentTitle    string
	AssignmentArchived bool
	Status             Status
	AutoGradeNew       bool
	HasWorkflowGraph   bool
	UpdatedAt          time.Time
}

func ListConfigsByCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]CourseConfigSummary, error) {
	if pool == nil {
		return nil, errors.New("nil pool")
	}
	rows, err := pool.Query(ctx, `
SELECT g.id, g.module_item_id, csi.kind, csi.title, csi.archived, g.status::text, g.auto_grade_new,
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
			&row.ID, &row.ModuleItemID, &row.ItemKind, &row.AssignmentTitle, &row.AssignmentArchived, &status,
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
	PostPolicy               string
	ConfidenceFloor          *float64
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
	include_assignment_content, include_rubric, model_id, workflow_graph, auto_grade_new, post_policy,
	confidence_floor, created_by, updated_at
) VALUES ($1, $2, $3::assessment.grading_agent_status, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
ON CONFLICT (module_item_id) DO UPDATE SET
	status = EXCLUDED.status,
	prompt = EXCLUDED.prompt,
	include_assignment_content = EXCLUDED.include_assignment_content,
	include_rubric = EXCLUDED.include_rubric,
	model_id = COALESCE(EXCLUDED.model_id, assessment.grading_agent_configs.model_id),
	workflow_graph = COALESCE(EXCLUDED.workflow_graph, assessment.grading_agent_configs.workflow_graph),
	auto_grade_new = EXCLUDED.auto_grade_new,
	post_policy = EXCLUDED.post_policy,
	confidence_floor = EXCLUDED.confidence_floor,
	updated_at = NOW()
RETURNING id, course_id, module_item_id, status::text, prompt,
          include_assignment_content, include_rubric, model_id, workflow_graph, auto_grade_new, post_policy,
          confidence_floor, created_by, created_at, updated_at
`, in.CourseID, in.ModuleItemID, string(in.Status), in.Prompt,
		in.IncludeAssignmentContent, in.IncludeRubric, in.ModelID, nullableJSON(in.WorkflowGraph), in.AutoGradeNew, in.PostPolicy, in.ConfidenceFloor, in.CreatedBy,
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

func CreateRun(ctx context.Context, pool *pgxpool.Pool, configID uuid.UUID, scope RunScope, mode RunMode, initiatedBy *uuid.UUID, authoredVia *string, total int, filterJSON []byte, budgetUSD *float64) (*RunRow, error) {
	if pool == nil {
		return nil, errors.New("nil pool")
	}
	if mode != RunModeSuggest {
		mode = RunModeApply
	}
	var r RunRow
	var scopeStr string
	var modeStr string
	err := pool.QueryRow(ctx, `
INSERT INTO assessment.grading_agent_runs (config_id, scope, mode, initiated_by, authored_via, total_count, filter, budget_usd, status)
VALUES ($1, $2::assessment.grading_agent_run_scope, $3, $4, $5, $6, $7, $8, 'queued')
RETURNING id, config_id, scope::text, mode, initiated_by, authored_via, filter, budget_usd, total_count, completed_count, failed_count, status, created_at, finished_at
`, configID, string(scope), string(mode), initiatedBy, authoredVia, total, filterJSON, budgetUSD).Scan(
		&r.ID, &r.ConfigID, &scopeStr, &modeStr, &r.InitiatedBy, &r.AuthoredVia, &r.Filter, &r.BudgetUSD, &r.TotalCount, &r.CompletedCount, &r.FailedCount,
		&r.Status, &r.CreatedAt, &r.FinishedAt,
	)
	if err != nil {
		return nil, err
	}
	r.Scope = RunScope(scopeStr)
	r.Mode = RunMode(modeStr)
	return &r, nil
}

func GetRun(ctx context.Context, pool *pgxpool.Pool, runID uuid.UUID) (*RunRow, error) {
	if pool == nil {
		return nil, errors.New("nil pool")
	}
	var r RunRow
	var scopeStr string
	var modeStr string
	err := pool.QueryRow(ctx, `
SELECT id, config_id, scope::text, mode, initiated_by, authored_via, filter, budget_usd, total_count, completed_count, failed_count, status, created_at, finished_at,
       cancelled_at, cancelled_by
FROM assessment.grading_agent_runs WHERE id = $1
`, runID).Scan(
		&r.ID, &r.ConfigID, &scopeStr, &modeStr, &r.InitiatedBy, &r.AuthoredVia, &r.Filter, &r.BudgetUSD, &r.TotalCount, &r.CompletedCount, &r.FailedCount,
		&r.Status, &r.CreatedAt, &r.FinishedAt, &r.CancelledAt, &r.CancelledBy,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.Scope = RunScope(scopeStr)
	r.Mode = RunMode(modeStr)
	if r.Mode != RunModeSuggest {
		r.Mode = RunModeApply
	}
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
    completed_count = LEAST(completed_count + 1, total_count),
    last_progress_at = NOW(),
    status = CASE
        WHEN completed_count + 1 >= total_count AND status = 'running' THEN 'done'
        WHEN completed_count + 1 >= total_count AND status = 'budget_exceeded' THEN 'budget_exceeded'
        ELSE status
    END,
    finished_at = CASE
        WHEN completed_count + 1 >= total_count AND status IN ('running', 'budget_exceeded', 'cancelled') THEN NOW()
        ELSE finished_at
    END
WHERE id = $1 AND completed_count < total_count
`, runID)
		return err
	}
	_, err := pool.Exec(ctx, `
UPDATE assessment.grading_agent_runs
SET completed_count = LEAST(completed_count + 1, total_count),
    last_progress_at = NOW(),
    status = CASE
        WHEN completed_count + 1 >= total_count AND status = 'running' THEN 'done'
        WHEN completed_count + 1 >= total_count AND status = 'budget_exceeded' THEN 'budget_exceeded'
        ELSE status
    END,
    finished_at = CASE
        WHEN completed_count + 1 >= total_count AND status IN ('running', 'budget_exceeded', 'cancelled') THEN NOW()
        ELSE finished_at
    END
WHERE id = $1 AND completed_count < total_count
`, runID)
	return err
}

// MarkRunBudgetExceeded marks a running batch as budget-limited; remaining items are skipped without LLM calls.
func MarkRunBudgetExceeded(ctx context.Context, pool *pgxpool.Pool, runID uuid.UUID) error {
	if pool == nil {
		return errors.New("nil pool")
	}
	_, err := pool.Exec(ctx, `
UPDATE assessment.grading_agent_runs
SET status = 'budget_exceeded', last_progress_at = NOW()
WHERE id = $1 AND status = 'running'
`, runID)
	return err
}

func SumRunCostUSD(ctx context.Context, pool *pgxpool.Pool, runID uuid.UUID) (float64, error) {
	if pool == nil {
		return 0, errors.New("nil pool")
	}
	var spent *float64
	err := pool.QueryRow(ctx, `
SELECT SUM(cost_usd)::float8
FROM assessment.grading_agent_results
WHERE run_id = $1 AND is_dry_run = false
`, runID).Scan(&spent)
	if err != nil {
		return 0, err
	}
	if spent == nil {
		return 0, nil
	}
	return *spent, nil
}

func SumRunUsage(ctx context.Context, pool *pgxpool.Pool, runID uuid.UUID) (RunUsageTotals, error) {
	if pool == nil {
		return RunUsageTotals{}, errors.New("nil pool")
	}
	var totals RunUsageTotals
	err := pool.QueryRow(ctx, `
SELECT COALESCE(SUM(prompt_tokens), 0)::int,
       COALESCE(SUM(completion_tokens), 0)::int,
       COALESCE(SUM(cost_usd), 0)::float8
FROM assessment.grading_agent_results
WHERE run_id = $1 AND is_dry_run = false
`, runID).Scan(&totals.PromptTokens, &totals.CompletionTokens, &totals.CostUSD)
	if err != nil {
		return RunUsageTotals{}, err
	}
	return totals, nil
}

func GetLatestDryRunSample(ctx context.Context, pool *pgxpool.Pool, configID uuid.UUID) (*DryRunCostSample, error) {
	if pool == nil {
		return nil, errors.New("nil pool")
	}
	var sample DryRunCostSample
	err := pool.QueryRow(ctx, `
SELECT prompt_tokens, completion_tokens, cost_usd, model_id
FROM assessment.grading_agent_results
WHERE config_id = $1 AND is_dry_run = true
ORDER BY created_at DESC
LIMIT 1
`, configID).Scan(&sample.PromptTokens, &sample.CompletionTokens, &sample.CostUSD, &sample.ModelID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &sample, nil
}

// GetRunStatus returns the current status string for a run (cheap indexed read).
func GetRunStatus(ctx context.Context, pool *pgxpool.Pool, runID uuid.UUID) (string, error) {
	if pool == nil {
		return "", errors.New("nil pool")
	}
	var status string
	err := pool.QueryRow(ctx, `SELECT status FROM assessment.grading_agent_runs WHERE id = $1`, runID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return status, err
}

// CancelRun marks a queued or running run as cancelled. Returns true when the run was
// transitioned; false when already terminal or not found (idempotent no-op).
func CancelRun(ctx context.Context, pool *pgxpool.Pool, runID uuid.UUID, cancelledBy uuid.UUID) (bool, error) {
	if pool == nil {
		return false, errors.New("nil pool")
	}
	tag, err := pool.Exec(ctx, `
UPDATE assessment.grading_agent_runs
SET status = 'cancelled', cancelled_at = NOW(), cancelled_by = $2, last_progress_at = NOW()
WHERE id = $1 AND status IN ('queued', 'running')
`, runID, cancelledBy)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// ResultExistsForRun returns true if a non-dry-run result already exists for the
// given (run_id, submission_id) pair. Used as an idempotency guard before grading.
func ResultExistsForRun(ctx context.Context, pool *pgxpool.Pool, runID, submissionID uuid.UUID) (bool, error) {
	if pool == nil {
		return false, errors.New("nil pool")
	}
	var exists bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1 FROM assessment.grading_agent_results
    WHERE run_id = $1 AND submission_id = $2 AND is_dry_run = false
)
`, runID, submissionID).Scan(&exists)
	return exists, err
}

func MarkRunRunning(ctx context.Context, pool *pgxpool.Pool, runID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
UPDATE assessment.grading_agent_runs
SET status = 'running', last_progress_at = NOW()
WHERE id = $1`, runID)
	return err
}

// MarkRunFailed transitions a run to the terminal failed state. It is a no-op
// if the run is already in a terminal state (done or failed).
func MarkRunFailed(ctx context.Context, pool *pgxpool.Pool, runID uuid.UUID) error {
	if pool == nil {
		return errors.New("nil pool")
	}
	_, err := pool.Exec(ctx, `
UPDATE assessment.grading_agent_runs
SET status = 'failed', finished_at = NOW()
WHERE id = $1 AND status NOT IN ('done', 'failed', 'cancelled')
`, runID)
	return err
}

// ReconcileStuckRuns marks running runs that have had no progress for longer
// than noProgressTimeout as failed. Returns the number of runs reconciled.
func ReconcileStuckRuns(ctx context.Context, pool *pgxpool.Pool, noProgressTimeout time.Duration) (int, error) {
	if pool == nil {
		return 0, errors.New("nil pool")
	}
	tag, err := pool.Exec(ctx, `
UPDATE assessment.grading_agent_runs
SET status = 'failed', finished_at = NOW()
WHERE status = 'running'
  AND last_progress_at < NOW() - $1::interval
`, noProgressTimeout.String())
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
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
	FlagReason       *string
	FlagPriority     *string
	HeldReason       *string
	HeldAt           *time.Time
	HeldQueue        *string
	InputModality    *string
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
	comment, confidence, status, model_id, prompt_tokens, completion_tokens, cost_usd, error,
	flag_reason, flag_priority, held_reason, held_at, held_queue, input_modality
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::assessment.grading_agent_item_status, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
RETURNING id, run_id, config_id, submission_id, is_dry_run, suggested_points, suggested_rubric,
          comment, confidence, status::text, model_id, prompt_tokens, completion_tokens, cost_usd, error,
          flag_reason, flag_priority, held_reason, held_at, held_queue, input_modality, created_at
`, in.RunID, in.ConfigID, in.SubmissionID, in.IsDryRun, in.SuggestedPoints, nullableJSON(rubricJSON),
		in.Comment, in.Confidence, string(in.Status), in.ModelID, in.PromptTokens, in.CompletionTokens, in.CostUSD, in.Error,
		in.FlagReason, in.FlagPriority, in.HeldReason, in.HeldAt, in.HeldQueue, in.InputModality,
	).Scan(
		&r.ID, &r.RunID, &r.ConfigID, &r.SubmissionID, &r.IsDryRun, &r.SuggestedPoints, &r.SuggestedRubric,
		&r.Comment, &r.Confidence, &status, &r.ModelID, &r.PromptTokens, &r.CompletionTokens, &r.CostUSD, &r.Error,
		&r.FlagReason, &r.FlagPriority, &r.HeldReason, &r.HeldAt, &r.HeldQueue, &r.InputModality, &r.CreatedAt,
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
       comment, confidence, status::text, model_id, prompt_tokens, completion_tokens, cost_usd, error,
       flag_reason, flag_priority, held_reason, held_at, held_queue, input_modality, created_at
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
			&r.Comment, &r.Confidence, &status, &r.ModelID, &r.PromptTokens, &r.CompletionTokens, &r.CostUSD, &r.Error,
			&r.FlagReason, &r.FlagPriority, &r.HeldReason, &r.HeldAt, &r.HeldQueue, &r.InputModality, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		r.Status = ItemStatus(status)
		out = append(out, r)
	}
	return out, rows.Err()
}

func ListRunsByConfig(ctx context.Context, pool *pgxpool.Pool, configID uuid.UUID, limit int) ([]RunSummary, error) {
	if pool == nil {
		return nil, errors.New("nil pool")
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := pool.Query(ctx, `
SELECT r.id, r.config_id, r.scope::text, r.mode, r.initiated_by, r.authored_via, r.filter, r.budget_usd, r.total_count, r.completed_count,
       r.failed_count, r.status, r.created_at, r.finished_at, r.cancelled_at, r.cancelled_by,
       (SELECT SUM(res.cost_usd) FROM assessment.grading_agent_results res WHERE res.run_id = r.id AND res.is_dry_run = false) AS cost_usd,
       (SELECT SUM(res.prompt_tokens)::int FROM assessment.grading_agent_results res WHERE res.run_id = r.id AND res.is_dry_run = false) AS prompt_tokens,
       (SELECT SUM(res.completion_tokens)::int FROM assessment.grading_agent_results res WHERE res.run_id = r.id AND res.is_dry_run = false) AS completion_tokens,
       (SELECT res.model_id FROM assessment.grading_agent_results res
        WHERE res.run_id = r.id AND res.model_id IS NOT NULL AND res.is_dry_run = false
        ORDER BY res.created_at ASC LIMIT 1) AS model_id
FROM assessment.grading_agent_runs r
WHERE r.config_id = $1
ORDER BY r.created_at DESC
LIMIT $2
`, configID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]RunSummary, 0)
	for rows.Next() {
		var summary RunSummary
		var scopeStr string
		var modeStr string
		if err := rows.Scan(
			&summary.ID, &summary.ConfigID, &scopeStr, &modeStr, &summary.InitiatedBy, &summary.AuthoredVia, &summary.Filter, &summary.BudgetUSD,
			&summary.TotalCount, &summary.CompletedCount, &summary.FailedCount, &summary.Status,
			&summary.CreatedAt, &summary.FinishedAt, &summary.CancelledAt, &summary.CancelledBy,
			&summary.CostUSD, &summary.PromptTokens, &summary.CompletionTokens, &summary.ModelID,
		); err != nil {
			return nil, err
		}
		summary.Scope = RunScope(scopeStr)
		summary.Mode = RunMode(modeStr)
		if summary.Mode != RunModeSuggest {
			summary.Mode = RunModeApply
		}
		out = append(out, summary)
	}
	return out, rows.Err()
}

const reviewQueueLatestCTE = `
WITH latest AS (
    SELECT DISTINCT ON (submission_id)
           id, run_id, config_id, submission_id, is_dry_run, suggested_points, suggested_rubric,
           comment, confidence, status::text, model_id, prompt_tokens, completion_tokens, cost_usd, error,
           flag_reason, flag_priority, held_reason, held_at, held_queue, resolved_at, resolved_by, created_at
    FROM assessment.grading_agent_results
    WHERE config_id = $1 AND is_dry_run = false
    ORDER BY submission_id, created_at DESC
)
`

func scanReviewQueueItem(rows pgx.Rows) (*ReviewQueueItem, error) {
	var item ReviewQueueItem
	var status string
	if err := rows.Scan(
		&item.ID, &item.RunID, &item.ConfigID, &item.SubmissionID, &item.IsDryRun, &item.SuggestedPoints, &item.SuggestedRubric,
		&item.Comment, &item.Confidence, &status, &item.ModelID, &item.PromptTokens, &item.CompletionTokens, &item.CostUSD, &item.Error,
		&item.FlagReason, &item.FlagPriority, &item.HeldReason, &item.HeldAt, &item.HeldQueue, &item.ResolvedAt, &item.ResolvedBy, &item.CreatedAt,
		&item.RunCreatedAt,
	); err != nil {
		return nil, err
	}
	item.Status = ItemStatus(status)
	return &item, nil
}

func ListReviewQueueByConfig(ctx context.Context, pool *pgxpool.Pool, configID uuid.UUID, limit int) (held, flagged []ReviewQueueItem, err error) {
	if pool == nil {
		return nil, nil, errors.New("nil pool")
	}
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	heldRows, err := pool.Query(ctx, reviewQueueLatestCTE+`
SELECT l.id, l.run_id, l.config_id, l.submission_id, l.is_dry_run, l.suggested_points, l.suggested_rubric,
       l.comment, l.confidence, l.status, l.model_id, l.prompt_tokens, l.completion_tokens, l.cost_usd, l.error,
       l.flag_reason, l.flag_priority, l.held_reason, l.held_at, l.held_queue, l.resolved_at, l.resolved_by, l.created_at,
       r.created_at AS run_created_at
FROM latest l
LEFT JOIN assessment.grading_agent_runs r ON r.id = l.run_id
WHERE l.status = 'suggested' AND l.held_at IS NOT NULL
ORDER BY l.held_at DESC
LIMIT $2
`, configID, limit)
	if err != nil {
		return nil, nil, err
	}
	defer heldRows.Close()
	held = make([]ReviewQueueItem, 0)
	for heldRows.Next() {
		item, scanErr := scanReviewQueueItem(heldRows)
		if scanErr != nil {
			return nil, nil, scanErr
		}
		held = append(held, *item)
	}
	if err := heldRows.Err(); err != nil {
		return nil, nil, err
	}

	flaggedRows, err := pool.Query(ctx, reviewQueueLatestCTE+`
SELECT l.id, l.run_id, l.config_id, l.submission_id, l.is_dry_run, l.suggested_points, l.suggested_rubric,
       l.comment, l.confidence, l.status, l.model_id, l.prompt_tokens, l.completion_tokens, l.cost_usd, l.error,
       l.flag_reason, l.flag_priority, l.held_reason, l.held_at, l.held_queue, l.resolved_at, l.resolved_by, l.created_at,
       r.created_at AS run_created_at
FROM latest l
LEFT JOIN assessment.grading_agent_runs r ON r.id = l.run_id
WHERE l.status = 'flagged'
ORDER BY l.created_at DESC
LIMIT $2
`, configID, limit)
	if err != nil {
		return nil, nil, err
	}
	defer flaggedRows.Close()
	flagged = make([]ReviewQueueItem, 0)
	for flaggedRows.Next() {
		item, scanErr := scanReviewQueueItem(flaggedRows)
		if scanErr != nil {
			return nil, nil, scanErr
		}
		flagged = append(flagged, *item)
	}
	return held, flagged, flaggedRows.Err()
}

func CountReviewQueueByConfig(ctx context.Context, pool *pgxpool.Pool, configID uuid.UUID) (int, error) {
	if pool == nil {
		return 0, errors.New("nil pool")
	}
	var count int
	err := pool.QueryRow(ctx, reviewQueueLatestCTE+`
SELECT COUNT(*)::int
FROM latest
WHERE (status = 'suggested' AND held_at IS NOT NULL) OR status = 'flagged'
`, configID).Scan(&count)
	return count, err
}

func CountReviewQueueByConfigs(ctx context.Context, pool *pgxpool.Pool, configIDs []uuid.UUID) (map[uuid.UUID]int, error) {
	out := make(map[uuid.UUID]int, len(configIDs))
	if pool == nil {
		return out, errors.New("nil pool")
	}
	if len(configIDs) == 0 {
		return out, nil
	}
	rows, err := pool.Query(ctx, `
WITH latest AS (
    SELECT DISTINCT ON (config_id, submission_id)
           config_id, status, held_at
    FROM assessment.grading_agent_results
    WHERE config_id = ANY($1) AND is_dry_run = false
    ORDER BY config_id, submission_id, created_at DESC
)
SELECT config_id, COUNT(*)::int
FROM latest
WHERE (status = 'suggested' AND held_at IS NOT NULL) OR status = 'flagged'
GROUP BY config_id
`, configIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var configID uuid.UUID
		var count int
		if err := rows.Scan(&configID, &count); err != nil {
			return nil, err
		}
		out[configID] = count
	}
	return out, rows.Err()
}

func UpdateResultStatus(ctx context.Context, pool *pgxpool.Pool, resultID uuid.UUID, status ItemStatus, reason *string, resolvedBy *uuid.UUID) (*ResultRow, error) {
	if pool == nil {
		return nil, errors.New("nil pool")
	}
	var r ResultRow
	var statusStr string
	err := pool.QueryRow(ctx, `
UPDATE assessment.grading_agent_results
SET status = $2::assessment.grading_agent_item_status,
    error = COALESCE($3, error),
    resolved_at = CASE
        WHEN $2::assessment.grading_agent_item_status IN ('applied', 'overridden', 'skipped')
        THEN COALESCE(resolved_at, NOW())
        ELSE resolved_at
    END,
    resolved_by = CASE
        WHEN $2::assessment.grading_agent_item_status IN ('applied', 'overridden', 'skipped')
        THEN COALESCE(resolved_by, $4)
        ELSE resolved_by
    END
WHERE id = $1
RETURNING id, run_id, config_id, submission_id, is_dry_run, suggested_points, suggested_rubric,
          comment, confidence, status::text, model_id, prompt_tokens, completion_tokens, cost_usd, error,
          flag_reason, flag_priority, held_reason, held_at, held_queue, resolved_at, resolved_by, created_at
`, resultID, string(status), reason, resolvedBy).Scan(
		&r.ID, &r.RunID, &r.ConfigID, &r.SubmissionID, &r.IsDryRun, &r.SuggestedPoints, &r.SuggestedRubric,
		&r.Comment, &r.Confidence, &statusStr, &r.ModelID, &r.PromptTokens, &r.CompletionTokens, &r.CostUSD, &r.Error,
		&r.FlagReason, &r.FlagPriority, &r.HeldReason, &r.HeldAt, &r.HeldQueue, &r.ResolvedAt, &r.ResolvedBy, &r.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.Status = ItemStatus(statusStr)
	return &r, nil
}

func nullableJSON(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return b
}
