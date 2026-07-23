package questionbank

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ListFilter narrows course question-bank listing.
type ListFilter struct {
	Query     string
	Type      string
	Status    string
	ConceptID *uuid.UUID
	Limit     int
}

// ListQuestionsForCourse returns bank rows for instructor search/import UIs.
func ListQuestionsForCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, filter ListFilter) ([]QuestionEntity, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	args := []any{courseID}
	where := []string{"q.course_id = $1"}

	if q := strings.TrimSpace(filter.Query); q != "" {
		args = append(args, "%"+q+"%")
		where = append(where, fmt.Sprintf("q.stem ILIKE $%d", len(args)))
	}
	if t := strings.TrimSpace(filter.Type); t != "" && t != "all" {
		args = append(args, t)
		where = append(where, fmt.Sprintf("q.question_type::text = $%d", len(args)))
	}
	if s := strings.TrimSpace(filter.Status); s != "" && s != "all" {
		args = append(args, s)
		where = append(where, fmt.Sprintf("q.status::text = $%d", len(args)))
	}
	if filter.ConceptID != nil {
		args = append(args, *filter.ConceptID)
		where = append(where, fmt.Sprintf(`EXISTS (
    SELECT 1 FROM course.concept_question_tags t
    WHERE t.question_id = q.id AND t.concept_id = $%d
  )`, len(args)))
	}

	args = append(args, limit)
	sql := `
SELECT id, course_id, question_type::text, stem, options, correct_answer, explanation,
       points::float8, status::text, shared, source, metadata, shuffle_choices_override,
       irt_a::float8, irt_b::float8, irt_c::float8,
       irt_status::text, irt_sample_n, irt_calibrated_at,
       created_by, created_at, updated_at,
       version_number, is_published, srs_eligible
FROM course.questions q
WHERE ` + strings.Join(where, " AND ") + `
ORDER BY q.updated_at DESC, q.id DESC
LIMIT $` + fmt.Sprintf("%d", len(args))

	rows, err := pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]QuestionEntity, 0)
	for rows.Next() {
		var e QuestionEntity
		if err := rows.Scan(
			&e.ID, &e.CourseID, &e.QuestionType, &e.Stem, &e.Options, &e.CorrectAnswer, &e.Explanation,
			&e.Points, &e.Status, &e.Shared, &e.Source, &e.Metadata, &e.ShuffleChoicesOverride,
			&e.IrtA, &e.IrtB, &e.IrtC, &e.IrtStatus, &e.IrtSampleN, &e.IrtCalibratedAt,
			&e.CreatedBy, &e.CreatedAt, &e.UpdatedAt,
			&e.VersionNumber, &e.IsPublished, &e.SRSEligible,
		); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
