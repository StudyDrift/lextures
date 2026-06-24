package gradingagent

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	gradingagentsvc "github.com/lextures/lextures/server/internal/service/gradingagent"
)

type seedQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// SeedDefaultTemplates inserts built-in grading agent templates for a course when missing.
func SeedDefaultTemplates(ctx context.Context, q seedQuerier, courseID, createdBy uuid.UUID) error {
	if q == nil {
		return errors.New("nil querier")
	}
	for _, spec := range gradingagentsvc.DefaultTemplates() {
		var exists bool
		if err := q.QueryRow(ctx, `
SELECT EXISTS (
	SELECT 1 FROM assessment.grading_agent_templates
	WHERE course_id = $1 AND name = $2
)
`, courseID, spec.Name).Scan(&exists); err != nil {
			return err
		}
		if exists {
			continue
		}
		if err := gradingagentsvc.ValidateWorkflowGraph(&spec.Graph); err != nil {
			return err
		}
		raw, err := gradingagentsvc.WorkflowGraphToJSON(&spec.Graph)
		if err != nil {
			return err
		}
		prompt := gradingagentsvc.PersistencePrompt(&spec.Graph, spec.Prompt)
		includeContent := spec.IncludeAssignmentContent
		includeRubric := spec.IncludeRubric
		if !includeContent && !includeRubric {
			_, derivedContent, derivedRubric, _ := gradingagentsvc.DeriveLegacyFields(&spec.Graph)
			includeContent = derivedContent
			includeRubric = derivedRubric
		}
		var insertedID uuid.UUID
		if err := q.QueryRow(ctx, `
INSERT INTO assessment.grading_agent_templates (
	course_id, name, prompt, include_assignment_content, include_rubric, workflow_graph, created_by, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
RETURNING id
`, courseID, spec.Name, prompt, includeContent, includeRubric, nullableJSON(raw), createdBy).Scan(&insertedID); err != nil {
			return err
		}
	}
	return nil
}

// SeedDefaultTemplatesPool is a convenience wrapper for callers outside a transaction.
func SeedDefaultTemplatesPool(ctx context.Context, pool *pgxpool.Pool, courseID, createdBy uuid.UUID) error {
	return SeedDefaultTemplates(ctx, pool, courseID, createdBy)
}
