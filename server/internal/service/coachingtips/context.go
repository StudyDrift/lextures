package coachingtips

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AggregateContext holds anonymized metrics sent to the LLM (no PII).
type AggregateContext struct {
	AvgDailyTimeMinutes float64
	LoginsLast7Days     int
	AvgQuizScore        *float64
	ScoreTrend          string
	TopStudyWeekdays    []string
	WeakTopicLabels     []string
}

// String renders context for prompts and audit logs.
func (a AggregateContext) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "avg_daily_time_min=%.1f login_count_7d=%d", a.AvgDailyTimeMinutes, a.LoginsLast7Days)
	if a.AvgQuizScore != nil {
		fmt.Fprintf(&b, " avg_quiz_score=%.1f", *a.AvgQuizScore)
	}
	if a.ScoreTrend != "" {
		fmt.Fprintf(&b, " score_trend=%s", a.ScoreTrend)
	}
	if len(a.TopStudyWeekdays) > 0 {
		fmt.Fprintf(&b, " top_days=%s", strings.Join(a.TopStudyWeekdays, ","))
	}
	if len(a.WeakTopicLabels) > 0 {
		fmt.Fprintf(&b, " weak_topics=%s", strings.Join(a.WeakTopicLabels, ","))
	}
	return b.String()
}

// LoadAggregateContext builds metrics for one user (no names, emails, or raw responses).
func LoadAggregateContext(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, now time.Time) (AggregateContext, error) {
	out := AggregateContext{}
	since := now.AddDate(0, 0, -7)

	var heartbeats int64
	_ = pool.QueryRow(ctx, `
SELECT COUNT(*) FROM analytics.engagement_events
WHERE user_id = $1 AND event_type = 'heartbeat' AND occurred_at >= $2
`, userID, since).Scan(&heartbeats)
	out.AvgDailyTimeMinutes = float64(heartbeats*30) / 60.0 / 7.0

	_ = pool.QueryRow(ctx, `
SELECT COUNT(DISTINCT date_trunc('day', occurred_at))
FROM "user".user_audit
WHERE user_id = $1 AND occurred_at >= $2
`, userID, since).Scan(&out.LoginsLast7Days)

	_ = pool.QueryRow(ctx, `
SELECT AVG(qa.score_percent)::float
FROM course.quiz_attempts qa
WHERE qa.student_user_id = $1 AND qa.status = 'submitted'
  AND qa.submitted_at >= $2 AND qa.score_percent IS NOT NULL
`, userID, since).Scan(&out.AvgQuizScore)

	var early, late *float64
	_ = pool.QueryRow(ctx, `
SELECT AVG(score_percent) FROM (
    SELECT score_percent FROM course.quiz_attempts
    WHERE student_user_id = $1 AND status = 'submitted' AND score_percent IS NOT NULL
    ORDER BY submitted_at ASC LIMIT 2
) s
`, userID).Scan(&early)
	_ = pool.QueryRow(ctx, `
SELECT AVG(score_percent) FROM (
    SELECT score_percent FROM course.quiz_attempts
    WHERE student_user_id = $1 AND status = 'submitted' AND score_percent IS NOT NULL
    ORDER BY submitted_at DESC LIMIT 2
) s
`, userID).Scan(&late)
	if early != nil && late != nil {
		switch {
		case *late-*early >= 3:
			out.ScoreTrend = "improving"
		case *early-*late >= 3:
			out.ScoreTrend = "declining"
		default:
			out.ScoreTrend = "stable"
		}
	}

	rows, err := pool.Query(ctx, `
SELECT trim(to_char(d, 'Dy'))
FROM (
    SELECT date_trunc('day', occurred_at) AS d, COUNT(*) AS c
    FROM analytics.engagement_events
    WHERE user_id = $1 AND event_type = 'heartbeat' AND occurred_at >= $2
    GROUP BY 1
    ORDER BY c DESC
    LIMIT 2
) sub
`, userID, since)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var day string
			if err := rows.Scan(&day); err == nil && day != "" {
				out.TopStudyWeekdays = append(out.TopStudyWeekdays, day)
			}
		}
	}

	weakRows, err := pool.Query(ctx, `
SELECT LEFT(c.name, 40)
FROM analytics.mastery_heatmap h
JOIN course.concepts c ON c.id = h.concept_id
WHERE h.user_id = $1 AND h.mastery_score < 0.5
ORDER BY h.mastery_score ASC
LIMIT 3
`, userID)
	if err == nil {
		defer weakRows.Close()
		for weakRows.Next() {
			var label string
			if err := weakRows.Scan(&label); err == nil && label != "" {
				out.WeakTopicLabels = append(out.WeakTopicLabels, label)
			}
		}
	}

	return out, nil
}
