package coursestructure

import (
	"context"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MaxChecklistBodyScanBytes caps markdown scanned for embedded file IDs (CC.4 FR-11).
const MaxChecklistBodyScanBytes = 64 * 1024

// ChecklistItemMeta is lightweight per-item detail for the course checklist snapshot (CC.1 / CC.4).
type ChecklistItemMeta struct {
	ItemID               uuid.UUID
	Kind                 string
	HasBody              bool
	PointsWorth          *int
	ExternalURL          string
	QuestionCount        int
	LateSubmissionPolicy string // assignment/quiz only; "" otherwise
	Attribution          string
	BodyMarkdown         string
	EmbeddedFileIDs      []uuid.UUID
}

var courseFileRefRE = regexp.MustCompile(`(?i)/course-files/([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})`)

// ListChecklistItemMeta loads content/assignment/quiz/survey/external-link/library/textbook
// metadata for all structure items in a course in a single query (CC.1 / CC.4 query budget).
func ListChecklistItemMeta(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (map[uuid.UUID]ChecklistItemMeta, error) {
	rows, err := pool.Query(ctx, `
SELECT c.id, c.kind,
       CASE c.kind
         WHEN 'content_page' THEN COALESCE(LENGTH(TRIM(p.markdown)) > 0, false)
         WHEN 'assignment' THEN COALESCE(LENGTH(TRIM(a.markdown)) > 0, false)
         WHEN 'quiz' THEN COALESCE(LENGTH(TRIM(q.markdown)) > 0, false)
         WHEN 'survey' THEN COALESCE(LENGTH(TRIM(s.description)) > 0, false)
         WHEN 'external_link' THEN COALESCE(LENGTH(TRIM(e.url)) > 0, false)
         WHEN 'library_resource' THEN COALESCE(LENGTH(TRIM(lr.metadata::text)) > 2, false)
         WHEN 'textbook_resource' THEN COALESCE(LENGTH(TRIM(tr.metadata::text)) > 2, false)
         ELSE false
       END AS has_body,
       a.points_worth,
       COALESCE(e.url, '') AS external_url,
       COALESCE(jsonb_array_length(q.questions_json), 0) AS question_count,
       CASE c.kind
         WHEN 'assignment' THEN COALESCE(a.late_submission_policy, '')
         WHEN 'quiz' THEN COALESCE(q.late_submission_policy, '')
         ELSE ''
       END AS late_submission_policy,
       CASE c.kind
         WHEN 'external_link' THEN COALESCE(NULLIF(TRIM(e.attribution_text), ''), '')
         WHEN 'library_resource' THEN COALESCE(
           NULLIF(TRIM(lr.metadata->>'attribution'), ''),
           NULLIF(TRIM(lr.metadata->>'source'), ''),
           NULLIF(TRIM(lr.metadata->>'author'), ''),
           ''
         )
         WHEN 'textbook_resource' THEN COALESCE(
           NULLIF(TRIM(tr.metadata->>'attribution'), ''),
           NULLIF(TRIM(tr.metadata->>'source'), ''),
           NULLIF(TRIM(tr.metadata->>'publisher'), ''),
           NULLIF(TRIM(tr.metadata->>'isbn'), ''),
           ''
         )
         ELSE ''
       END AS attribution,
       CASE c.kind
         WHEN 'content_page' THEN LEFT(COALESCE(p.markdown, ''), $2)
         WHEN 'assignment' THEN LEFT(COALESCE(a.markdown, ''), $2)
         WHEN 'quiz' THEN LEFT(COALESCE(q.markdown, ''), $2)
         ELSE ''
       END AS body_markdown
FROM course.course_structure_items c
LEFT JOIN course.module_content_pages p ON p.structure_item_id = c.id AND c.kind = 'content_page'
LEFT JOIN course.module_assignments a ON a.structure_item_id = c.id AND c.kind = 'assignment'
LEFT JOIN course.module_quizzes q ON q.structure_item_id = c.id AND c.kind = 'quiz'
LEFT JOIN course.module_surveys s ON s.structure_item_id = c.id AND c.kind = 'survey'
LEFT JOIN course.module_external_links e ON e.structure_item_id = c.id AND c.kind = 'external_link'
LEFT JOIN course.module_library_resources lr ON lr.structure_item_id = c.id AND c.kind = 'library_resource'
LEFT JOIN course.module_textbook_resources tr ON tr.structure_item_id = c.id AND c.kind = 'textbook_resource'
WHERE c.course_id = $1
  AND NOT c.archived
  AND c.kind IN (
    'content_page', 'assignment', 'quiz', 'survey', 'external_link',
    'library_resource', 'textbook_resource'
  )
`, courseID, MaxChecklistBodyScanBytes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[uuid.UUID]ChecklistItemMeta)
	for rows.Next() {
		var m ChecklistItemMeta
		if err := rows.Scan(
			&m.ItemID, &m.Kind, &m.HasBody, &m.PointsWorth, &m.ExternalURL,
			&m.QuestionCount, &m.LateSubmissionPolicy, &m.Attribution, &m.BodyMarkdown,
		); err != nil {
			return nil, err
		}
		m.EmbeddedFileIDs = extractEmbeddedFileIDs(m.BodyMarkdown)
		out[m.ItemID] = m
	}
	return out, rows.Err()
}

func extractEmbeddedFileIDs(markdown string) []uuid.UUID {
	if strings.TrimSpace(markdown) == "" {
		return nil
	}
	matches := courseFileRefRE.FindAllStringSubmatch(markdown, 64)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[uuid.UUID]struct{}, len(matches))
	var out []uuid.UUID
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		id, err := uuid.Parse(m[1])
		if err != nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
