package xapistatements

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Row is a stored xAPI statement summary for list APIs.
type Row struct {
	StatementID     uuid.UUID
	ActorHash       string
	VerbID          string
	ObjectID        string
	ObjectType      *string
	ObjectTitle     *string
	ResultScore     *float32
	ResultSuccess   *bool
	ContextCourseID *uuid.UUID
	StoredAt        time.Time
	FullJSON        json.RawMessage
}

// Insert stores an immutable xAPI/Caliper payload.
func Insert(ctx context.Context, pool *pgxpool.Pool, row Row) error {
	_, err := pool.Exec(ctx, `
INSERT INTO analytics.xapi_statements (
  statement_id, actor_hash, verb_id, object_id, object_type, object_title,
  result_score, result_success, context_course_id, stored_at, full_json
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb)
`, row.StatementID, row.ActorHash, row.VerbID, row.ObjectID, row.ObjectType, row.ObjectTitle,
		row.ResultScore, row.ResultSuccess, row.ContextCourseID, row.StoredAt.UTC(), row.FullJSON)
	return err
}

// ListForCourse returns statements for a course in [since, until), newest first.
func ListForCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, since, until time.Time, limit int) ([]Row, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := pool.Query(ctx, `
SELECT statement_id, actor_hash, verb_id, object_id, object_type, object_title,
       result_score, result_success, context_course_id, stored_at, full_json
FROM analytics.xapi_statements
WHERE context_course_id = $1
  AND stored_at >= $2
  AND stored_at < $3
ORDER BY stored_at DESC
LIMIT $4
`, courseID, since.UTC(), until.UTC(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRows(rows)
}

// GetByID returns one statement by id (most recent stored_at if duplicated).
func GetByID(ctx context.Context, pool *pgxpool.Pool, statementID uuid.UUID) (*Row, error) {
	var r Row
	err := pool.QueryRow(ctx, `
SELECT statement_id, actor_hash, verb_id, object_id, object_type, object_title,
       result_score, result_success, context_course_id, stored_at, full_json
FROM analytics.xapi_statements
WHERE statement_id = $1
ORDER BY stored_at DESC
LIMIT 1
`, statementID).Scan(
		&r.StatementID, &r.ActorHash, &r.VerbID, &r.ObjectID, &r.ObjectType, &r.ObjectTitle,
		&r.ResultScore, &r.ResultSuccess, &r.ContextCourseID, &r.StoredAt, &r.FullJSON,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func scanRows(rows pgx.Rows) ([]Row, error) {
	var out []Row
	for rows.Next() {
		var r Row
		if err := rows.Scan(
			&r.StatementID, &r.ActorHash, &r.VerbID, &r.ObjectID, &r.ObjectType, &r.ObjectTitle,
			&r.ResultScore, &r.ResultSuccess, &r.ContextCourseID, &r.StoredAt, &r.FullJSON,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
