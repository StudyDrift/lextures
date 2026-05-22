package atrisk

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ScoreRow is one nightly computed score.
type ScoreRow struct {
	EnrollmentID uuid.UUID
	ComputedDate time.Time
	Score        float32
	MissingPct   *float32
	QuizAvg      *float32
	DaysInactive int
	GradeTrend   *float32
	TopFactor    string
}

// UpsertScore inserts or replaces a score for a date (idempotent nightly job).
func UpsertScore(ctx context.Context, pool *pgxpool.Pool, row ScoreRow) error {
	_, err := pool.Exec(ctx, `
INSERT INTO analytics.at_risk_scores (
    enrollment_id, computed_date, score, missing_pct, quiz_avg, days_inactive, grade_trend, top_factor
) VALUES ($1, $2::date, $3, $4, $5, $6, $7, $8)
ON CONFLICT (enrollment_id, computed_date) DO UPDATE SET
    score = EXCLUDED.score,
    missing_pct = EXCLUDED.missing_pct,
    quiz_avg = EXCLUDED.quiz_avg,
    days_inactive = EXCLUDED.days_inactive,
    grade_trend = EXCLUDED.grade_trend,
    top_factor = EXCLUDED.top_factor
`, row.EnrollmentID, row.ComputedDate.Format("2006-01-02"), row.Score, row.MissingPct, row.QuizAvg, row.DaysInactive, row.GradeTrend, row.TopFactor)
	return err
}

// ListHistory returns historical scores for one enrollment, newest first.
func ListHistory(ctx context.Context, pool *pgxpool.Pool, enrollmentID uuid.UUID, limit int) ([]ScoreRow, error) {
	if limit <= 0 {
		limit = 90
	}
	rows, err := pool.Query(ctx, `
SELECT enrollment_id, computed_date, score, missing_pct, quiz_avg, days_inactive, grade_trend, top_factor
FROM analytics.at_risk_scores
WHERE enrollment_id = $1
ORDER BY computed_date DESC
LIMIT $2
`, enrollmentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ScoreRow
	for rows.Next() {
		var r ScoreRow
		if err := rows.Scan(&r.EnrollmentID, &r.ComputedDate, &r.Score, &r.MissingPct, &r.QuizAvg, &r.DaysInactive, &r.GradeTrend, &r.TopFactor); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
