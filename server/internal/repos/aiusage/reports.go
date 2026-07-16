package aiusage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	models "github.com/lextures/lextures/server/internal/models/aiusage"
)

// Filters scopes report queries.
type Filters struct {
	From       time.Time
	To         time.Time
	Feature    string
	Provider   string
	UserID     *uuid.UUID
	CourseID   *uuid.UUID
	UserQuery  string
	CourseCode string
}

func (f Filters) whereSQL(alias string) (string, []any) {
	col := func(name string) string {
		if alias == "" {
			return name
		}
		return alias + "." + name
	}
	logRef := "analytics.ai_usage_log"
	if alias != "" {
		logRef = alias
	}

	clauses := []string{
		col("created_at") + " >= $1",
		col("created_at") + " < $2",
		col("succeeded") + " = TRUE",
	}
	args := []any{f.From, f.To}
	n := 3
	if s := strings.TrimSpace(f.Feature); s != "" {
		clauses = append(clauses, fmt.Sprintf("%s = $%d", col("feature"), n))
		args = append(args, s)
		n++
	}
	if s := strings.TrimSpace(f.Provider); s != "" {
		clauses = append(clauses, fmt.Sprintf("%s = $%d", col("provider"), n))
		args = append(args, s)
		n++
	}
	if f.UserID != nil {
		clauses = append(clauses, fmt.Sprintf("%s = $%d", col("user_id"), n))
		args = append(args, *f.UserID)
		n++
	}
	if f.CourseID != nil {
		clauses = append(clauses, fmt.Sprintf("%s = $%d", col("course_id"), n))
		args = append(args, *f.CourseID)
		n++
	}
	if q := strings.TrimSpace(f.UserQuery); q != "" {
		pat := "%" + q + "%"
		clauses = append(clauses, fmt.Sprintf(`EXISTS (
  SELECT 1 FROM "user".users u
   WHERE u.id = %s.user_id
     AND (
       u.email ILIKE $%d
       OR COALESCE(u.display_name, '') ILIKE $%d
       OR COALESCE(u.first_name, '') ILIKE $%d
       OR COALESCE(u.last_name, '') ILIKE $%d
     )
)`, logRef, n, n, n, n))
		args = append(args, pat)
		n++
	}
	if code := strings.TrimSpace(f.CourseCode); code != "" {
		pat := "%" + code + "%"
		clauses = append(clauses, fmt.Sprintf(`EXISTS (
  SELECT 1 FROM course.courses c
   WHERE c.id = %s.course_id
     AND (c.course_code ILIKE $%d OR c.title ILIKE $%d)
)`, logRef, n, n))
		args = append(args, pat)
	}
	return strings.Join(clauses, " AND "), args
}

// CostSummary aggregates spend over the window.
func CostSummary(ctx context.Context, pool *pgxpool.Pool, f Filters) (models.CostSummary, error) {
	where, args := f.whereSQL("")
	var s models.CostSummary
	err := pool.QueryRow(ctx, `
SELECT
  COALESCE(SUM(cost_usd), 0)::float8,
  COUNT(*)::bigint,
  COALESCE(SUM(total_tokens), 0)::bigint
FROM analytics.ai_usage_log
WHERE `+where, args...).Scan(&s.TotalCostUSD, &s.TotalCalls, &s.TotalTokens)
	return s, err
}

// CostByDay returns UTC day buckets.
func CostByDay(ctx context.Context, pool *pgxpool.Pool, f Filters) ([]models.DayCostBucket, error) {
	where, args := f.whereSQL("")
	rows, err := pool.Query(ctx, `
SELECT
  (date_trunc('day', created_at AT TIME ZONE 'UTC'))::date,
  COALESCE(SUM(cost_usd), 0)::float8,
  COUNT(*)::bigint,
  COALESCE(SUM(total_tokens), 0)::bigint
FROM analytics.ai_usage_log
WHERE `+where+`
GROUP BY 1
ORDER BY 1 ASC
`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDayCost(rows)
}

func scanDayCost(rows pgx.Rows) ([]models.DayCostBucket, error) {
	out := make([]models.DayCostBucket, 0)
	for rows.Next() {
		var d pgtype.Date
		var b models.DayCostBucket
		if err := rows.Scan(&d, &b.CostUSD, &b.Calls, &b.Tokens); err != nil {
			return nil, err
		}
		if d.Valid {
			t := d.Time.UTC()
			b.Day = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).Format("2006-01-02")
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// CostByFeature returns spend grouped by feature.
func CostByFeature(ctx context.Context, pool *pgxpool.Pool, f Filters) ([]models.FeatureCostRow, error) {
	where, args := f.whereSQL("")
	rows, err := pool.Query(ctx, `
SELECT
  feature,
  COALESCE(SUM(cost_usd), 0)::float8,
  COUNT(*)::bigint,
  COALESCE(SUM(total_tokens), 0)::bigint
FROM analytics.ai_usage_log
WHERE `+where+`
GROUP BY feature
ORDER BY SUM(cost_usd) DESC, COUNT(*) DESC
`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.FeatureCostRow, 0)
	for rows.Next() {
		var r models.FeatureCostRow
		if err := rows.Scan(&r.Feature, &r.CostUSD, &r.Calls, &r.Tokens); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// CostByProvider returns spend grouped by provider (AP.6 FR-3/FR-8).
func CostByProvider(ctx context.Context, pool *pgxpool.Pool, f Filters) ([]models.ProviderCostRow, error) {
	where, args := f.whereSQL("")
	rows, err := pool.Query(ctx, `
SELECT
  provider,
  COALESCE(SUM(cost_usd), 0)::float8,
  COUNT(*)::bigint,
  COALESCE(SUM(total_tokens), 0)::bigint
FROM analytics.ai_usage_log
WHERE `+where+`
GROUP BY provider
ORDER BY SUM(cost_usd) DESC, COUNT(*) DESC
`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.ProviderCostRow, 0)
	for rows.Next() {
		var r models.ProviderCostRow
		if err := rows.Scan(&r.Provider, &r.CostUSD, &r.Calls, &r.Tokens); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// DistinctProviders returns provider labels seen in the window (for UI filters).
func DistinctProviders(ctx context.Context, pool *pgxpool.Pool, f Filters) ([]string, error) {
	// Ignore provider filter so the dropdown still lists peers in the window.
	f2 := f
	f2.Provider = ""
	where, args := f2.whereSQL("")
	rows, err := pool.Query(ctx, `
SELECT DISTINCT provider
FROM analytics.ai_usage_log
WHERE `+where+`
ORDER BY provider ASC
`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0)
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// UsageByUser returns top users by cost in the window.
func UsageByUser(ctx context.Context, pool *pgxpool.Pool, f Filters, limit int) ([]models.UserUsageRow, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	where, args := f.whereSQL("l")
	args = append(args, limit)
	lim := len(args)
	rows, err := pool.Query(ctx, `
SELECT
  u.id,
  u.email,
  COALESCE(NULLIF(TRIM(CONCAT_WS(' ', u.first_name, u.last_name)), ''), u.display_name, u.email),
  COUNT(*)::bigint,
  COALESCE(SUM(l.prompt_tokens), 0)::bigint,
  COALESCE(SUM(l.completion_tokens), 0)::bigint,
  COALESCE(SUM(l.total_tokens), 0)::bigint,
  COALESCE(SUM(l.cost_usd), 0)::float8
FROM analytics.ai_usage_log l
INNER JOIN "user".users u ON u.id = l.user_id
WHERE `+where+`
GROUP BY u.id, u.email, u.first_name, u.last_name, u.display_name
ORDER BY SUM(l.cost_usd) DESC, SUM(l.total_tokens) DESC
LIMIT $`+fmt.Sprint(lim), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.UserUsageRow, 0)
	for rows.Next() {
		var id uuid.UUID
		var r models.UserUsageRow
		if err := rows.Scan(&id, &r.Email, &r.DisplayName, &r.Calls, &r.PromptTokens, &r.CompletionTokens, &r.TotalTokens, &r.CostUSD); err != nil {
			return nil, err
		}
		r.UserID = id.String()
		out = append(out, r)
	}
	return out, rows.Err()
}

// UsageByCourse returns top courses by cost in the window.
func UsageByCourse(ctx context.Context, pool *pgxpool.Pool, f Filters, limit int) ([]models.CourseUsageRow, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	where, args := f.whereSQL("l")
	args = append(args, limit)
	lim := len(args)
	rows, err := pool.Query(ctx, `
SELECT
  c.id,
  c.course_code,
  c.title,
  COUNT(*)::bigint,
  COALESCE(SUM(l.total_tokens), 0)::bigint,
  COALESCE(SUM(l.cost_usd), 0)::float8
FROM analytics.ai_usage_log l
INNER JOIN course.courses c ON c.id = l.course_id
WHERE `+where+`
GROUP BY c.id, c.course_code, c.title
ORDER BY SUM(l.cost_usd) DESC, SUM(l.total_tokens) DESC
LIMIT $`+fmt.Sprint(lim), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.CourseUsageRow, 0)
	for rows.Next() {
		var id uuid.UUID
		var r models.CourseUsageRow
		if err := rows.Scan(&id, &r.CourseCode, &r.Title, &r.Calls, &r.TotalTokens, &r.CostUSD); err != nil {
			return nil, err
		}
		r.CourseID = id.String()
		out = append(out, r)
	}
	return out, rows.Err()
}