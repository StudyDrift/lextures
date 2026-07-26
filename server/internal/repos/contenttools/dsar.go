package contenttools

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserContentToolsExport is the DSAR export slice for Content Tools artifacts (CT.1).
type UserContentToolsExport struct {
	States []map[string]any `json:"states"`
	Events []map[string]any `json:"events"`
}

// ExportUserContent collects a user's content tool states and related events.
func ExportUserContent(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (UserContentToolsExport, error) {
	out := UserContentToolsExport{
		States: []map[string]any{},
		Events: []map[string]any{},
	}
	if pool == nil {
		return out, nil
	}

	srows, err := pool.Query(ctx, `
SELECT s.id, s.instance_id, i.course_id, c.course_code, s.enrollment_id, i.tool_id,
       s.state_json, s.revision, s.status, s.score_raw, s.score_max, s.interaction_count,
       s.created_at, s.updated_at
FROM course.content_tool_states s
JOIN course.content_tool_instances i ON i.id = s.instance_id
JOIN course.courses c ON c.id = i.course_id
WHERE s.user_id = $1
ORDER BY s.created_at ASC
`, userID)
	if err != nil {
		return out, err
	}
	defer srows.Close()
	for srows.Next() {
		var id, instanceID, courseID, enrollmentID uuid.UUID
		var courseCode, toolID, status string
		var state []byte
		var revision int64
		var scoreRaw, scoreMax *float64
		var interactions int
		var createdAt, updatedAt time.Time
		if err := srows.Scan(&id, &instanceID, &courseID, &courseCode, &enrollmentID, &toolID,
			&state, &revision, &status, &scoreRaw, &scoreMax, &interactions, &createdAt, &updatedAt); err != nil {
			return out, err
		}
		row := map[string]any{
			"id":               id.String(),
			"instanceId":       instanceID.String(),
			"courseId":         courseID.String(),
			"courseCode":       courseCode,
			"enrollmentId":     enrollmentID.String(),
			"toolId":           toolID,
			"revision":         revision,
			"status":           status,
			"interactionCount": interactions,
			"createdAt":        createdAt.UTC().Format(time.RFC3339),
			"updatedAt":        updatedAt.UTC().Format(time.RFC3339),
		}
		if scoreRaw != nil {
			row["scoreRaw"] = *scoreRaw
		}
		if scoreMax != nil {
			row["scoreMax"] = *scoreMax
		}
		if len(state) > 0 {
			var parsed any
			if json.Unmarshal(state, &parsed) == nil {
				row["state"] = parsed
			}
		}
		out.States = append(out.States, row)
	}
	if err := srows.Err(); err != nil {
		return out, err
	}

	erows, err := pool.Query(ctx, `
SELECT e.id, e.instance_id, e.course_id, c.course_code, e.enrollment_id, e.tool_id,
       e.event_type, e.payload_json, e.created_at
FROM course.content_tool_events e
JOIN course.courses c ON c.id = e.course_id
WHERE e.actor_user_id = $1
ORDER BY e.created_at ASC
`, userID)
	if err != nil {
		return out, err
	}
	defer erows.Close()
	for erows.Next() {
		var id int64
		var instanceID, enrollmentID *uuid.UUID
		var courseID uuid.UUID
		var courseCode, toolID, eventType string
		var payload []byte
		var createdAt time.Time
		if err := erows.Scan(&id, &instanceID, &courseID, &courseCode, &enrollmentID, &toolID,
			&eventType, &payload, &createdAt); err != nil {
			return out, err
		}
		row := map[string]any{
			"id":         id,
			"courseId":   courseID.String(),
			"courseCode": courseCode,
			"toolId":     toolID,
			"eventType":  eventType,
			"createdAt":  createdAt.UTC().Format(time.RFC3339),
		}
		if instanceID != nil {
			row["instanceId"] = instanceID.String()
		}
		if enrollmentID != nil {
			row["enrollmentId"] = enrollmentID.String()
		}
		if len(payload) > 0 {
			var parsed any
			if json.Unmarshal(payload, &parsed) == nil {
				row["payload"] = parsed
			}
		}
		out.Events = append(out.Events, row)
	}
	return out, erows.Err()
}

// EraseUserContent deletes content tool states and nulls event actor for a user.
func EraseUserContent(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) error {
	if pool == nil {
		return nil
	}
	if _, err := pool.Exec(ctx, `DELETE FROM course.content_tool_states WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `
UPDATE course.content_tool_events SET actor_user_id = NULL WHERE actor_user_id = $1
`, userID); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `
UPDATE course.content_tool_states SET last_reset_by = NULL WHERE last_reset_by = $1
`, userID); err != nil {
		return err
	}
	return nil
}

// CountUserContentRows returns remaining state rows for a user (erasure verification).
func CountUserContentRows(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM course.content_tool_states WHERE user_id = $1`, userID).Scan(&n)
	return n, err
}
