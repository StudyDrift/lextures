package coursestructure

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ChecklistItemMeta is lightweight per-item detail for the course checklist snapshot (CC.1).
type ChecklistItemMeta struct {
	ItemID               uuid.UUID
	Kind                 string
	HasBody              bool
	PointsWorth          *int
	ExternalURL          string
	QuestionCount        int
	LateSubmissionPolicy string // assignment/quiz only; "" otherwise
}

// ListChecklistItemMeta loads content/assignment/quiz/survey/external-link metadata
// for all structure items in a course in a single query (CC.1 query budget).
func ListChecklistItemMeta(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (map[uuid.UUID]ChecklistItemMeta, error) {
	rows, err := pool.Query(ctx, `
SELECT c.id, c.kind,
       CASE c.kind
         WHEN 'content_page' THEN COALESCE(LENGTH(TRIM(p.markdown)) > 0, false)
         WHEN 'assignment' THEN COALESCE(LENGTH(TRIM(a.markdown)) > 0, false)
         WHEN 'quiz' THEN COALESCE(LENGTH(TRIM(q.markdown)) > 0, false)
         WHEN 'survey' THEN COALESCE(LENGTH(TRIM(s.description)) > 0, false)
         WHEN 'external_link' THEN COALESCE(LENGTH(TRIM(e.url)) > 0, false)
         ELSE false
       END AS has_body,
       a.points_worth,
       COALESCE(e.url, '') AS external_url,
       COALESCE(jsonb_array_length(q.questions_json), 0) AS question_count,
       CASE c.kind
         WHEN 'assignment' THEN COALESCE(a.late_submission_policy, '')
         WHEN 'quiz' THEN COALESCE(q.late_submission_policy, '')
         ELSE ''
       END AS late_submission_policy
FROM course.course_structure_items c
LEFT JOIN course.module_content_pages p ON p.structure_item_id = c.id AND c.kind = 'content_page'
LEFT JOIN course.module_assignments a ON a.structure_item_id = c.id AND c.kind = 'assignment'
LEFT JOIN course.module_quizzes q ON q.structure_item_id = c.id AND c.kind = 'quiz'
LEFT JOIN course.module_surveys s ON s.structure_item_id = c.id AND c.kind = 'survey'
LEFT JOIN course.module_external_links e ON e.structure_item_id = c.id AND c.kind = 'external_link'
WHERE c.course_id = $1
  AND NOT c.archived
  AND c.kind IN ('content_page', 'assignment', 'quiz', 'survey', 'external_link')
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[uuid.UUID]ChecklistItemMeta)
	for rows.Next() {
		var m ChecklistItemMeta
		if err := rows.Scan(&m.ItemID, &m.Kind, &m.HasBody, &m.PointsWorth, &m.ExternalURL, &m.QuestionCount, &m.LateSubmissionPolicy); err != nil {
			return nil, err
		}
		out[m.ItemID] = m
	}
	return out, rows.Err()
}
