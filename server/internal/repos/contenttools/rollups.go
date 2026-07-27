package contenttools

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DailyRollupRow is analytics.content_tool_daily_rollups.
type DailyRollupRow struct {
	Day          time.Time
	OrgID        *uuid.UUID
	ToolID       string
	Instances    int
	Learners     int
	Engagements  int
	Completions  int
	MeanScorePct *float64
	AITokens     int64
	AICostUSD    float64
	RenderErrors int
}

// ListDailyRollups returns platform telemetry rows in [from,to] inclusive.
func ListDailyRollups(ctx context.Context, pool *pgxpool.Pool, from, to time.Time, orgID *uuid.UUID) ([]DailyRollupRow, error) {
	rows, err := pool.Query(ctx, `
SELECT day, org_id, tool_id, instances, learners, engagements, completions,
       mean_score_pct, ai_tokens, ai_cost_usd, render_errors
FROM analytics.content_tool_daily_rollups
WHERE day >= $1::date AND day <= $2::date
  AND ($3::uuid IS NULL OR org_id = $3)
ORDER BY tool_id ASC, day ASC
`, from, to, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]DailyRollupRow, 0)
	for rows.Next() {
		var r DailyRollupRow
		if err := rows.Scan(
			&r.Day, &r.OrgID, &r.ToolID, &r.Instances, &r.Learners, &r.Engagements, &r.Completions,
			&r.MeanScorePct, &r.AITokens, &r.AICostUSD, &r.RenderErrors,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ComputeDailyRollupsFromSummaries materialises one day of cross-course telemetry (no free-text).
func ComputeDailyRollupsFromSummaries(ctx context.Context, pool *pgxpool.Pool, day time.Time) error {
	_, err := pool.Exec(ctx, `
INSERT INTO analytics.content_tool_daily_rollups (
  day, org_id, tool_id, instances, learners, engagements, completions, mean_score_pct, ai_tokens, ai_cost_usd, render_errors
)
SELECT
  $1::date,
  c.org_id,
  s.tool_id,
  COUNT(DISTINCT s.instance_id)::int,
  COUNT(DISTINCT s.enrollment_id) FILTER (WHERE s.role = 'student')::int,
  COUNT(*) FILTER (WHERE s.engaged AND s.role = 'student')::int,
  COUNT(*) FILTER (WHERE s.completed AND s.role = 'student')::int,
  ROUND(AVG(s.score_pct) FILTER (WHERE s.role = 'student' AND s.score_pct IS NOT NULL)::numeric, 2),
  0, 0, 0
FROM analytics.content_tool_state_summaries s
INNER JOIN course.courses c ON c.id = s.course_id
WHERE s.updated_at::date = $1::date
GROUP BY c.org_id, s.tool_id
ON CONFLICT (day, org_id, tool_id) DO UPDATE SET
  instances = EXCLUDED.instances,
  learners = EXCLUDED.learners,
  engagements = EXCLUDED.engagements,
  completions = EXCLUDED.completions,
  mean_score_pct = EXCLUDED.mean_score_pct
`, day)
	return err
}
