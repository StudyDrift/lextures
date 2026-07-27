package contenttools

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ListEnrollmentStateJSONForInstance returns enrollment-scoped state_json blobs for an instance.
// Used for Ask Questions theme clustering (CT.10) — never on the student ask path.
func ListEnrollmentStateJSONForInstance(ctx context.Context, pool *pgxpool.Pool, instanceID uuid.UUID) ([]json.RawMessage, error) {
	rows, err := pool.Query(ctx, `
SELECT state_json
FROM course.content_tool_states
WHERE instance_id = $1 AND scope = $2
ORDER BY updated_at DESC
`, instanceID, ScopeEnrollment)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []json.RawMessage
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		if len(raw) == 0 {
			raw = []byte(`{}`)
		}
		out = append(out, json.RawMessage(raw))
	}
	return out, rows.Err()
}
