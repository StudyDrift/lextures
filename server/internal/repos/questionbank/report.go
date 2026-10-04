package questionbank

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MisconceptionReportRow is one misconception × question aggregate for a course.
type MisconceptionReportRow struct {
	MisconceptionID   uuid.UUID
	MisconceptionName string
	QuestionID        uuid.UUID
	QuestionStem      string
	TriggerCount      int
	AffectedStudents  int
	FirstSeen         time.Time
	LastSeen          time.Time
}

// ListMisconceptionReport aggregates recorded misconception events for a course.
// An empty slice means the course has no events yet.
func ListMisconceptionReport(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]MisconceptionReportRow, error) {
	rows, err := pool.Query(ctx, `
		SELECT m.id,
		       m.name,
		       q.id,
		       q.stem,
		       COUNT(*)::int,
		       COUNT(DISTINCT e.user_id)::int,
		       MIN(e.created_at),
		       MAX(e.created_at)
		FROM course.misconception_events e
		INNER JOIN course.misconceptions m ON m.id = e.misconception_id
		INNER JOIN course.questions q ON q.id = e.question_id
		WHERE e.course_id = $1
		GROUP BY m.id, m.name, q.id, q.stem
		ORDER BY COUNT(*) DESC, m.name ASC, q.id ASC
	`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []MisconceptionReportRow{}
	for rows.Next() {
		var row MisconceptionReportRow
		if err := rows.Scan(
			&row.MisconceptionID,
			&row.MisconceptionName,
			&row.QuestionID,
			&row.QuestionStem,
			&row.TriggerCount,
			&row.AffectedStudents,
			&row.FirstSeen,
			&row.LastSeen,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
