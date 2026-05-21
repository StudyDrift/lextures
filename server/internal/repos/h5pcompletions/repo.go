// Package h5pcompletions provides DB access for content.h5p_completions (plan 8.12).
package h5pcompletions

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Row is a per-user completion record.
type Row struct {
	UserID     uuid.UUID
	Status     string
	ScoreRaw   *float64
	ScoreMax   *float64
	UpdatedAt  time.Time
}

// UpsertFromStatement updates completion from an xAPI-derived status.
func UpsertFromStatement(
	ctx context.Context,
	pool *pgxpool.Pool,
	packageID, userID uuid.UUID,
	status string,
	scoreRaw, scoreMax *float64,
	stmtJSON json.RawMessage,
) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO content.h5p_completions (package_id, user_id, status, score_raw, score_max, xapi_statement, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, now())
		ON CONFLICT (package_id, user_id) DO UPDATE SET
		  status = CASE
		    WHEN EXCLUDED.status IN ('completed', 'passed', 'failed') THEN EXCLUDED.status
		    WHEN content.h5p_completions.status IN ('completed', 'passed', 'failed') THEN content.h5p_completions.status
		    ELSE EXCLUDED.status
		  END,
		  score_raw = COALESCE(EXCLUDED.score_raw, content.h5p_completions.score_raw),
		  score_max = COALESCE(EXCLUDED.score_max, content.h5p_completions.score_max),
		  xapi_statement = EXCLUDED.xapi_statement,
		  updated_at = now()`,
		packageID, userID, status, scoreRaw, scoreMax, stmtJSON,
	)
	return err
}

// ListForPackage returns all completion rows for a package (gradebook).
func ListForPackage(ctx context.Context, pool *pgxpool.Pool, packageID uuid.UUID) ([]Row, error) {
	rows, err := pool.Query(ctx, `
		SELECT c.user_id, c.status, c.score_raw, c.score_max, c.updated_at,
		       COALESCE(u.display_name, u.email, c.user_id::text)
		FROM content.h5p_completions c
		INNER JOIN "user".users u ON u.id = c.user_id
		WHERE c.package_id = $1
		ORDER BY u.display_name`, packageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Row
	for rows.Next() {
		var r Row
		var displayName string
		if err := rows.Scan(&r.UserID, &r.Status, &r.ScoreRaw, &r.ScoreMax, &r.UpdatedAt, &displayName); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// DisplayGradesForCourse returns map[userID][itemID]display label for all h5p items in a course.
func DisplayGradesForCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (map[string]map[string]string, error) {
	rows, err := pool.Query(ctx, `
		SELECT p.structure_item_id, c.user_id, c.status
		FROM content.h5p_completions c
		INNER JOIN content.h5p_packages p ON p.id = c.package_id
		WHERE p.course_id = $1 AND p.structure_item_id IS NOT NULL`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]map[string]string)
	for rows.Next() {
		var itemID, userID uuid.UUID
		var status string
		if err := rows.Scan(&itemID, &userID, &status); err != nil {
			return nil, err
		}
		uid := userID.String()
		iid := itemID.String()
		if out[uid] == nil {
			out[uid] = make(map[string]string)
		}
		out[uid][iid] = labelForStatus(status)
	}
	return out, rows.Err()
}

func labelForStatus(status string) string {
	switch status {
	case "completed":
		return "Completed"
	case "in_progress":
		return "In progress"
	case "passed":
		return "Passed"
	case "failed":
		return "Failed"
	default:
		return ""
	}
}

// DeleteByUser removes all H5P completion/xAPI rows for a user (GDPR erasure).
func DeleteByUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (int64, error) {
	tag, err := pool.Exec(ctx, `DELETE FROM content.h5p_completions WHERE user_id = $1`, userID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// CountRecentStatements counts statements in the last minute (rate limit).
func CountRecentStatements(ctx context.Context, pool *pgxpool.Pool, packageID, userID uuid.UUID) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*)::int FROM content.h5p_completions
		WHERE package_id = $1 AND user_id = $2 AND updated_at > now() - interval '1 minute'`,
		packageID, userID).Scan(&n)
	return n, err
}
