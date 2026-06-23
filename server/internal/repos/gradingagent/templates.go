package gradingagent

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TemplateRow struct {
	ID                       uuid.UUID
	CourseID                 uuid.UUID
	Name                     string
	Prompt                   string
	IncludeAssignmentContent bool
	IncludeRubric            bool
	WorkflowGraph            []byte
	CreatedBy                uuid.UUID
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

type CreateTemplateInput struct {
	CourseID                 uuid.UUID
	Name                     string
	Prompt                   string
	IncludeAssignmentContent bool
	IncludeRubric            bool
	WorkflowGraph            []byte
	CreatedBy                uuid.UUID
}

type TemplateSummary struct {
	ID        uuid.UUID
	Name      string
	UpdatedAt time.Time
}

func GetTemplateByCourseAndID(ctx context.Context, pool *pgxpool.Pool, courseID, templateID uuid.UUID) (*TemplateRow, error) {
	if pool == nil {
		return nil, errors.New("nil pool")
	}
	var r TemplateRow
	var workflowGraph []byte
	err := pool.QueryRow(ctx, `
SELECT id, course_id, name, prompt, include_assignment_content, include_rubric, workflow_graph, created_by, created_at, updated_at
FROM assessment.grading_agent_templates
WHERE course_id = $1 AND id = $2
`, courseID, templateID).Scan(
		&r.ID, &r.CourseID, &r.Name, &r.Prompt, &r.IncludeAssignmentContent, &r.IncludeRubric, &workflowGraph,
		&r.CreatedBy, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	r.WorkflowGraph = workflowGraph
	return &r, nil
}

func ListTemplatesByCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]TemplateSummary, error) {
	if pool == nil {
		return nil, errors.New("nil pool")
	}
	rows, err := pool.Query(ctx, `
SELECT id, name, updated_at
FROM assessment.grading_agent_templates
WHERE course_id = $1
ORDER BY updated_at DESC, name ASC
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]TemplateSummary, 0)
	for rows.Next() {
		var row TemplateSummary
		if err := rows.Scan(&row.ID, &row.Name, &row.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

type UpdateTemplateInput struct {
	Name                     string
	Prompt                   string
	IncludeAssignmentContent bool
	IncludeRubric            bool
	WorkflowGraph            []byte
}

func UpdateTemplate(ctx context.Context, pool *pgxpool.Pool, courseID, templateID uuid.UUID, in UpdateTemplateInput) (*TemplateRow, error) {
	if pool == nil {
		return nil, errors.New("nil pool")
	}
	var r TemplateRow
	var workflowGraph []byte
	err := pool.QueryRow(ctx, `
UPDATE assessment.grading_agent_templates
SET name = $3,
    prompt = $4,
    include_assignment_content = $5,
    include_rubric = $6,
    workflow_graph = $7,
    updated_at = NOW()
WHERE course_id = $1 AND id = $2
RETURNING id, course_id, name, prompt, include_assignment_content, include_rubric, workflow_graph, created_by, created_at, updated_at
`, courseID, templateID, in.Name, in.Prompt, in.IncludeAssignmentContent, in.IncludeRubric, nullableJSON(in.WorkflowGraph),
	).Scan(
		&r.ID, &r.CourseID, &r.Name, &r.Prompt, &r.IncludeAssignmentContent, &r.IncludeRubric, &workflowGraph,
		&r.CreatedBy, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	r.WorkflowGraph = workflowGraph
	return &r, nil
}

func CreateTemplate(ctx context.Context, pool *pgxpool.Pool, in CreateTemplateInput) (*TemplateRow, error) {
	if pool == nil {
		return nil, errors.New("nil pool")
	}
	var r TemplateRow
	var workflowGraph []byte
	err := pool.QueryRow(ctx, `
INSERT INTO assessment.grading_agent_templates (
	course_id, name, prompt, include_assignment_content, include_rubric, workflow_graph, created_by, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
RETURNING id, course_id, name, prompt, include_assignment_content, include_rubric, workflow_graph, created_by, created_at, updated_at
`, in.CourseID, in.Name, in.Prompt, in.IncludeAssignmentContent, in.IncludeRubric, nullableJSON(in.WorkflowGraph), in.CreatedBy,
	).Scan(
		&r.ID, &r.CourseID, &r.Name, &r.Prompt, &r.IncludeAssignmentContent, &r.IncludeRubric, &workflowGraph,
		&r.CreatedBy, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	r.WorkflowGraph = workflowGraph
	return &r, nil
}
