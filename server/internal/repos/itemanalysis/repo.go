// Package itemanalysis persists and retrieves CTT item and test statistics.
package itemanalysis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ItemStatRow is a row from analytics.item_stats.
type ItemStatRow struct {
	ID              uuid.UUID
	QuizID          uuid.UUID
	QuestionIndex   int
	QuestionText    string
	NResponses      int
	PValue          *float64
	RPB             *float64
	DistractorFreqs map[string]float64
	Flag            *string
	ComputedAt      time.Time
}

// TestStatRow is a row from analytics.test_stats.
type TestStatRow struct {
	ID            uuid.UUID
	QuizID        uuid.UUID
	NResponses    int
	KR20          *float64
	CronbachAlpha *float64
	MeanScore     *float64
	StdDev        *float64
	ComputedAt    time.Time
}

// AttemptResponseRow holds one student-question observation for CTT computation.
type AttemptResponseRow struct {
	AttemptID     uuid.UUID
	ScorePercent  *float64 // nil when not yet scored
	QuestionIndex int
	QuestionType  string
	PromptText    *string
	IsCorrect     *bool
	ChoiceIndex   *int
	PointsAwarded float64
	MaxPoints     float64
}

// FetchAttemptResponses returns all submitted attempt+response rows for a quiz
// (identified by structure_item_id). Each row is one student-question pair.
func FetchAttemptResponses(ctx context.Context, pool *pgxpool.Pool, structureItemID uuid.UUID) ([]AttemptResponseRow, error) {
	rows, err := pool.Query(ctx, `
SELECT
    a.id,
    a.score_percent,
    qr.question_index,
    qr.question_type,
    qr.prompt_snapshot,
    qr.is_correct,
    (qr.response_json->>'selectedChoiceIndex')::int,
    COALESCE(qr.points_awarded, 0.0),
    qr.max_points
FROM course.quiz_attempts a
JOIN course.quiz_responses qr ON qr.attempt_id = a.id
WHERE a.structure_item_id = $1
  AND a.status = 'submitted'
ORDER BY a.id, qr.question_index
`, structureItemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []AttemptResponseRow
	for rows.Next() {
		var r AttemptResponseRow
		if err := rows.Scan(
			&r.AttemptID,
			&r.ScorePercent,
			&r.QuestionIndex,
			&r.QuestionType,
			&r.PromptText,
			&r.IsCorrect,
			&r.ChoiceIndex,
			&r.PointsAwarded,
			&r.MaxPoints,
		); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// CountSubmittedAttempts returns the count of submitted attempts for a quiz.
func CountSubmittedAttempts(ctx context.Context, pool *pgxpool.Pool, structureItemID uuid.UUID) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
SELECT COUNT(*)::int FROM course.quiz_attempts
WHERE structure_item_id = $1 AND status = 'submitted'
`, structureItemID).Scan(&n)
	return n, err
}

// UpsertTestStats inserts or replaces the test-level stats for a quiz.
func UpsertTestStats(ctx context.Context, pool *pgxpool.Pool, row TestStatRow) error {
	_, err := pool.Exec(ctx, `
INSERT INTO analytics.test_stats (quiz_id, n_responses, kr20, cronbach_alpha, mean_score, std_dev, computed_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (quiz_id) DO UPDATE SET
    n_responses    = EXCLUDED.n_responses,
    kr20           = EXCLUDED.kr20,
    cronbach_alpha = EXCLUDED.cronbach_alpha,
    mean_score     = EXCLUDED.mean_score,
    std_dev        = EXCLUDED.std_dev,
    computed_at    = EXCLUDED.computed_at
`, row.QuizID, row.NResponses, row.KR20, row.CronbachAlpha, row.MeanScore, row.StdDev, row.ComputedAt)
	return err
}

// InsertItemStats inserts a batch of per-item statistics for a quiz at a given computed_at timestamp.
// Duplicate (quiz_id, question_index, computed_at) rows are silently ignored.
func InsertItemStats(ctx context.Context, pool *pgxpool.Pool, items []ItemStatRow) error {
	for _, item := range items {
		var freqJSON []byte
		if len(item.DistractorFreqs) > 0 {
			var err error
			freqJSON, err = json.Marshal(item.DistractorFreqs)
			if err != nil {
				return err
			}
		}
		_, err := pool.Exec(ctx, `
INSERT INTO analytics.item_stats
    (quiz_id, question_index, question_text, n_responses, p_value, r_pb, distractor_freqs, flag, computed_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (quiz_id, question_index, computed_at) DO NOTHING
`,
			item.QuizID,
			item.QuestionIndex,
			item.QuestionText,
			item.NResponses,
			item.PValue,
			item.RPB,
			freqJSON,
			item.Flag,
			item.ComputedAt,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetTestStats returns the most recent test-level stats for a quiz, or nil if none.
func GetTestStats(ctx context.Context, pool *pgxpool.Pool, quizID uuid.UUID) (*TestStatRow, error) {
	var r TestStatRow
	err := pool.QueryRow(ctx, `
SELECT id, quiz_id, n_responses, kr20, cronbach_alpha, mean_score, std_dev, computed_at
FROM analytics.test_stats
WHERE quiz_id = $1
`, quizID).Scan(&r.ID, &r.QuizID, &r.NResponses, &r.KR20, &r.CronbachAlpha, &r.MeanScore, &r.StdDev, &r.ComputedAt)
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &r, nil
}

// GetItemStats returns the most recent per-item stats for a quiz, ordered by question_index.
func GetItemStats(ctx context.Context, pool *pgxpool.Pool, quizID uuid.UUID) ([]ItemStatRow, error) {
	rows, err := pool.Query(ctx, `
SELECT DISTINCT ON (question_index)
    id, quiz_id, question_index, COALESCE(question_text, ''), n_responses,
    p_value, r_pb, distractor_freqs, flag, computed_at
FROM analytics.item_stats
WHERE quiz_id = $1
ORDER BY question_index ASC, computed_at DESC
`, quizID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ItemStatRow
	for rows.Next() {
		var r ItemStatRow
		var freqJSON []byte
		if err := rows.Scan(
			&r.ID, &r.QuizID, &r.QuestionIndex, &r.QuestionText,
			&r.NResponses, &r.PValue, &r.RPB, &freqJSON, &r.Flag, &r.ComputedAt,
		); err != nil {
			return nil, err
		}
		if len(freqJSON) > 0 {
			if err := json.Unmarshal(freqJSON, &r.DistractorFreqs); err != nil {
				return nil, err
			}
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// ListStaleQuizzes returns structure_item_ids for quizzes whose due_at has passed,
// have at least minN submitted attempts, and whose test_stats are older than staleAfter
// (or do not exist). Limit caps the number of returned IDs.
func ListStaleQuizzes(ctx context.Context, pool *pgxpool.Pool, now time.Time, minN, limit int, staleAfter time.Duration) ([]uuid.UUID, error) {
	cutoff := now.Add(-staleAfter)
	rows, err := pool.Query(ctx, `
SELECT a.structure_item_id
FROM course.quiz_attempts a
JOIN course.course_structure_items si ON si.id = a.structure_item_id
JOIN course.module_quizzes mq ON mq.structure_item_id = si.id
WHERE a.status = 'submitted'
  AND (mq.due_at IS NOT NULL AND mq.due_at < $1)
GROUP BY a.structure_item_id
HAVING COUNT(*) >= $2
  AND NOT EXISTS (
      SELECT 1 FROM analytics.test_stats ts
      WHERE ts.quiz_id = a.structure_item_id
        AND ts.computed_at >= $3
  )
ORDER BY MAX(a.submitted_at) ASC
LIMIT $4
`, now, minN, cutoff, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func isNotFound(err error) bool {
	return err != nil && err.Error() == "no rows in result set"
}
