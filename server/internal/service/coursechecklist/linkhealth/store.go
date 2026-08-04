package linkhealth

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Row is a persisted link-health cache entry.
type Row struct {
	CourseID   uuid.UUID
	URLHash    []byte
	URL        string
	StatusCode *int
	Result     ResultCode
	Reason     string
	CheckedAt  time.Time
}

// ListForCourse returns cached rows for a course.
func ListForCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]Row, error) {
	rows, err := pool.Query(ctx, `
SELECT course_id, url_hash, url, status_code, result, reason, checked_at
FROM course.course_checklist_link_health
WHERE course_id = $1
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Row
	for rows.Next() {
		var r Row
		var code *int
		var result string
		if err := rows.Scan(&r.CourseID, &r.URLHash, &r.URL, &code, &result, &r.Reason, &r.CheckedAt); err != nil {
			return nil, err
		}
		r.StatusCode = code
		r.Result = ResultCode(result)
		out = append(out, r)
	}
	return out, rows.Err()
}

// UpsertResults writes check outcomes.
func UpsertResults(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, results []CheckResult) error {
	for _, res := range results {
		var code any
		if res.StatusCode > 0 {
			code = res.StatusCode
		}
		checked := res.CheckedAt
		if checked.IsZero() {
			checked = time.Now().UTC()
		}
		_, err := pool.Exec(ctx, `
INSERT INTO course.course_checklist_link_health (
    course_id, url_hash, url, status_code, result, reason, checked_at
) VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (course_id, url_hash) DO UPDATE SET
    url = EXCLUDED.url,
    status_code = EXCLUDED.status_code,
    result = EXCLUDED.result,
    reason = EXCLUDED.reason,
    checked_at = EXCLUDED.checked_at
`, courseID, res.URLHash, res.URL, code, string(res.Result), res.Reason, checked)
		if err != nil {
			return err
		}
	}
	return nil
}

// CacheFresh reports whether the newest row is within CacheTTL and covers the URL set size.
func CacheFresh(rows []Row, now time.Time) bool {
	if len(rows) == 0 {
		return false
	}
	newest := rows[0].CheckedAt
	for _, r := range rows[1:] {
		if r.CheckedAt.After(newest) {
			newest = r.CheckedAt
		}
	}
	return now.Sub(newest) <= CacheTTL
}

// SweepOlderThan deletes rows older than cutoff and returns the delete count.
func SweepOlderThan(ctx context.Context, pool *pgxpool.Pool, cutoff time.Time) (int64, error) {
	tag, err := pool.Exec(ctx, `
DELETE FROM course.course_checklist_link_health WHERE checked_at < $1
`, cutoff)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// LastEnqueuedKey builds a unique_key for the 24h enqueue floor.
func LastEnqueuedKey(courseID uuid.UUID) string {
	return "checklist-linkcheck:" + courseID.String()
}
