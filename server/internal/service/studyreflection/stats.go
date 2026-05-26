package studyreflection

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	repo "github.com/lextures/lextures/server/internal/repos/studyreflection"
)

// TimeAllocationRow is study minutes per module over a window.
type TimeAllocationRow struct {
	ModuleID    string  `json:"moduleId"`
	ModuleTitle string  `json:"moduleTitle"`
	Minutes     float64 `json:"minutes"`
}

// Stats is the GET /me/study-stats payload.
type Stats struct {
	OptedIn              bool                `json:"optedIn"`
	LoginStreakDays      int                 `json:"loginStreakDays"`
	TimeOnTaskSecondsWeek int                `json:"timeOnTaskSecondsThisWeek"`
	WeeklyGoalHours      *float32            `json:"weeklyGoalHours,omitempty"`
	GoalProgressHours    float32             `json:"goalProgressHours"`
	GoalRemainingHours   *float32            `json:"goalRemainingHours,omitempty"`
	StudyEfficiency      *float64            `json:"studyEfficiency,omitempty"`
	LowStudyEfficiency   bool                `json:"lowStudyEfficiency"`
	TimeAllocation       []TimeAllocationRow `json:"timeAllocation"`
	WeekStart            string              `json:"weekStart"`
	WeekEnd              string              `json:"weekEnd"`
}

// LoadStats aggregates engagement and goal data for a user.
func LoadStats(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, now time.Time) (Stats, error) {
	out := Stats{OptedIn: false}
	goal, err := repo.GetGoal(ctx, pool, userID)
	if err != nil {
		return out, err
	}
	if goal != nil {
		out.OptedIn = goal.OptedIn
		if goal.WeeklyHours > 0 {
			h := goal.WeeklyHours
			out.WeeklyGoalHours = &h
		}
	}

	weekStart, weekEnd := WeekBounds(now)
	out.WeekStart = weekStart.Format(time.RFC3339)
	out.WeekEnd = weekEnd.Format(time.RFC3339)

	activeDays, err := loadActiveDays(ctx, pool, userID, now.AddDate(0, 0, -90))
	if err != nil {
		return out, err
	}
	endDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	out.LoginStreakDays = LoginStreak(activeDays, endDay)

	var heartbeats int64
	_ = pool.QueryRow(ctx, `
SELECT COUNT(*) FROM analytics.engagement_events
WHERE user_id = $1 AND event_type = 'heartbeat'
  AND occurred_at >= $2 AND occurred_at <= $3
`, userID, weekStart, weekEnd).Scan(&heartbeats)
	out.TimeOnTaskSecondsWeek = int(heartbeats * 30)
	out.GoalProgressHours = float32(out.TimeOnTaskSecondsWeek) / 3600.0
	if out.WeeklyGoalHours != nil && *out.WeeklyGoalHours > out.GoalProgressHours {
		rem := *out.WeeklyGoalHours - out.GoalProgressHours
		out.GoalRemainingHours = &rem
	}

	allocation, err := loadTimeAllocation(ctx, pool, userID, now.AddDate(0, 0, -14), now)
	if err != nil {
		return out, err
	}
	out.TimeAllocation = allocation

	scoreStart, scoreEnd, timeAll, err := loadQuizEfficiencyInputs(ctx, pool, userID, weekStart, weekEnd)
	if err != nil {
		return out, err
	}
	if ratio, low, ok := StudyEfficiency(timeAll, scoreStart, scoreEnd); ok {
		out.StudyEfficiency = &ratio
		out.LowStudyEfficiency = low
	}

	return out, nil
}

func loadActiveDays(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, since time.Time) (map[string]struct{}, error) {
	days := make(map[string]struct{})
	rows, err := pool.Query(ctx, `
SELECT DISTINCT date_trunc('day', occurred_at AT TIME ZONE 'UTC')::date
FROM "user".user_audit
WHERE user_id = $1 AND occurred_at >= $2
UNION
SELECT DISTINCT date_trunc('day', occurred_at AT TIME ZONE 'UTC')::date
FROM analytics.engagement_events
WHERE user_id = $1 AND occurred_at >= $2 AND event_type = 'heartbeat'
`, userID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var d time.Time
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		days[d.Format("2006-01-02")] = struct{}{}
	}
	return days, rows.Err()
}

func loadTimeAllocation(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, from, to time.Time) ([]TimeAllocationRow, error) {
	rows, err := pool.Query(ctx, `
SELECT
    COALESCE(mod.id, si.id)::text,
    COALESCE(mod.title, si.title, 'Module'),
    COALESCE(COUNT(*) FILTER (WHERE e.event_type = 'heartbeat'), 0) * 30.0 / 60.0 AS minutes
FROM analytics.engagement_events e
JOIN course.course_structure_items si ON si.id = e.item_id
LEFT JOIN course.course_structure_items mod ON mod.id = si.parent_id AND mod.kind = 'module'
WHERE e.user_id = $1
  AND e.occurred_at >= $2 AND e.occurred_at <= $3
  AND e.item_id IS NOT NULL
GROUP BY COALESCE(mod.id, si.id), COALESCE(mod.title, si.title, 'Module')
HAVING COUNT(*) FILTER (WHERE e.event_type = 'heartbeat') > 0
ORDER BY minutes DESC
LIMIT 20
`, userID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TimeAllocationRow
	for rows.Next() {
		var r TimeAllocationRow
		if err := rows.Scan(&r.ModuleID, &r.ModuleTitle, &r.Minutes); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func loadQuizEfficiencyInputs(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, weekStart, weekEnd time.Time) (start, end *float64, timeSec int, err error) {
	_ = pool.QueryRow(ctx, `
SELECT COUNT(*) * 30 FROM analytics.engagement_events
WHERE user_id = $1 AND event_type = 'heartbeat'
  AND occurred_at >= $2 AND occurred_at <= $3
`, userID, weekStart, weekEnd).Scan(&timeSec)

	var early, late *float64
	_ = pool.QueryRow(ctx, `
SELECT AVG(score_pct) FROM (
    SELECT qa.score_percent::float AS score_pct
    FROM course.quiz_attempts qa
    WHERE qa.student_user_id = $1
      AND qa.status = 'submitted'
      AND qa.submitted_at >= $2 AND qa.submitted_at <= $3
      AND qa.score_percent IS NOT NULL
    ORDER BY qa.submitted_at ASC
    LIMIT 3
) early_scores
`, userID, weekStart, weekEnd).Scan(&early)

	_ = pool.QueryRow(ctx, `
SELECT AVG(score_pct) FROM (
    SELECT qa.score_percent::float AS score_pct
    FROM course.quiz_attempts qa
    WHERE qa.student_user_id = $1
      AND qa.status = 'submitted'
      AND qa.submitted_at >= $2 AND qa.submitted_at <= $3
      AND qa.score_percent IS NOT NULL
    ORDER BY qa.submitted_at DESC
    LIMIT 3
) late_scores
`, userID, weekStart, weekEnd).Scan(&late)

	if early != nil && late != nil {
		return early, late, timeSec, nil
	}
	return nil, nil, timeSec, nil
}
