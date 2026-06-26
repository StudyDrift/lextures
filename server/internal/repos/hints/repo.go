package hints

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type QuestionHintRow struct {
	ID         uuid.UUID
	QuestionID uuid.UUID
	Level      int16
	Body       string
	MediaURL   *string
	Locale     string
	PenaltyPct float64
	CreatedAt  time.Time
}

type WorkedExampleRow struct {
	ID         uuid.UUID
	QuestionID uuid.UUID
	Title      *string
	Body       *string
	Steps      json.RawMessage
	CreatedAt  time.Time
}

// HintUseCountsForAttempt returns per-question hint request counts for an attempt.
func HintUseCountsForAttempt(ctx context.Context, pool *pgxpool.Pool, attemptID uuid.UUID) (map[string]int64, error) {
	rows, err := pool.Query(ctx, `
SELECT question_id, COUNT(*)::bigint
FROM course.hint_requests
WHERE attempt_id = $1
GROUP BY question_id
`, attemptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]int64)
	for rows.Next() {
		var qid string
		var n int64
		if err := rows.Scan(&qid, &n); err != nil {
			return nil, err
		}
		out[qid] = n
	}
	return out, rows.Err()
}
