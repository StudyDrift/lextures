package contenttools

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserContentToolsExport is the DSAR export slice for Content Tools artifacts (CT.1/CT.4/CT.8).
type UserContentToolsExport struct {
	States      []map[string]any `json:"states"`
	Events      []map[string]any `json:"events"`
	Resets      []map[string]any `json:"resets"`
	AIConsents  []map[string]any `json:"aiConsents"`
	Moderation  []map[string]any `json:"moderation"`
	FilterFlags []map[string]any `json:"filterFlags"`
}

// ExportUserContent collects a user's content tool states and related events.
func ExportUserContent(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (UserContentToolsExport, error) {
	out := UserContentToolsExport{
		States:      []map[string]any{},
		Events:      []map[string]any{},
		Resets:      []map[string]any{},
		AIConsents:  []map[string]any{},
		Moderation:  []map[string]any{},
		FilterFlags: []map[string]any{},
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
	if err := erows.Err(); err != nil {
		return out, err
	}

	rrows, err := pool.Query(ctx, `
SELECT r.id, r.instance_id, r.course_id, c.course_code, r.enrollment_id, r.tool_id,
       r.scope, r.reason, r.prior_status, r.prior_revision, r.reset_at, r.purge_after,
       r.restored_at, r.prior_state_json
FROM course.content_tool_state_resets r
JOIN course.courses c ON c.id = r.course_id
JOIN course.course_enrollments ce ON ce.id = r.enrollment_id
WHERE ce.user_id = $1
ORDER BY r.reset_at ASC
`, userID)
	if err != nil {
		return out, err
	}
	defer rrows.Close()
	for rrows.Next() {
		var id, instanceID, courseID, enrollmentID uuid.UUID
		var courseCode, toolID, scope, priorStatus string
		var reason *string
		var priorRevision int64
		var resetAt, purgeAfter time.Time
		var restoredAt *time.Time
		var prior []byte
		if err := rrows.Scan(&id, &instanceID, &courseID, &courseCode, &enrollmentID, &toolID,
			&scope, &reason, &priorStatus, &priorRevision, &resetAt, &purgeAfter, &restoredAt, &prior); err != nil {
			return out, err
		}
		row := map[string]any{
			"id":            id.String(),
			"instanceId":    instanceID.String(),
			"courseId":      courseID.String(),
			"courseCode":    courseCode,
			"enrollmentId":  enrollmentID.String(),
			"toolId":        toolID,
			"scope":         scope,
			"priorStatus":   priorStatus,
			"priorRevision": priorRevision,
			"resetAt":       resetAt.UTC().Format(time.RFC3339),
			"purgeAfter":    purgeAfter.UTC().Format(time.RFC3339),
		}
		if reason != nil {
			row["reason"] = *reason
		}
		if restoredAt != nil {
			row["restoredAt"] = restoredAt.UTC().Format(time.RFC3339)
		}
		if len(prior) > 0 {
			var parsed any
			if json.Unmarshal(prior, &parsed) == nil {
				row["priorState"] = parsed
			}
		}
		out.Resets = append(out.Resets, row)
	}
	if err := rrows.Err(); err != nil {
		return out, err
	}

	crows, err := pool.Query(ctx, `
SELECT id, course_id, tool_id, decision, decided_at
FROM course.content_tool_ai_consents
WHERE user_id = $1
ORDER BY decided_at ASC
`, userID)
	if err != nil {
		return out, err
	}
	defer crows.Close()
	for crows.Next() {
		var id uuid.UUID
		var courseID *uuid.UUID
		var toolID *string
		var decision string
		var decidedAt time.Time
		if err := crows.Scan(&id, &courseID, &toolID, &decision, &decidedAt); err != nil {
			return out, err
		}
		row := map[string]any{
			"id":        id.String(),
			"decision":  decision,
			"decidedAt": decidedAt.UTC().Format(time.RFC3339),
		}
		if courseID != nil {
			row["courseId"] = courseID.String()
		}
		if toolID != nil {
			row["toolId"] = *toolID
		}
		out.AIConsents = append(out.AIConsents, row)
	}
	if err := crows.Err(); err != nil {
		return out, err
	}

	mrows, err := pool.Query(ctx, `
SELECT id, instance_id, action, category, reason, created_at
FROM course.content_tool_moderation
WHERE actor_user_id = $1 OR subject_user_id = $1
ORDER BY created_at ASC
`, userID)
	if err != nil {
		return out, err
	}
	defer mrows.Close()
	for mrows.Next() {
		var id, instanceID uuid.UUID
		var action string
		var category, reason *string
		var createdAt time.Time
		if err := mrows.Scan(&id, &instanceID, &action, &category, &reason, &createdAt); err != nil {
			return out, err
		}
		row := map[string]any{
			"id":         id.String(),
			"instanceId": instanceID.String(),
			"action":     action,
			"createdAt":  createdAt.UTC().Format(time.RFC3339),
		}
		if category != nil {
			row["category"] = *category
		}
		if reason != nil {
			row["reason"] = *reason
		}
		out.Moderation = append(out.Moderation, row)
	}
	if err := mrows.Err(); err != nil {
		return out, err
	}

	frows, err := pool.Query(ctx, `
SELECT id, instance_id, course_id, category, action, created_at
FROM course.content_tool_filter_flags
WHERE user_id = $1
ORDER BY created_at ASC
`, userID)
	if err != nil {
		return out, err
	}
	defer frows.Close()
	for frows.Next() {
		var id, instanceID, courseID uuid.UUID
		var category, action string
		var createdAt time.Time
		if err := frows.Scan(&id, &instanceID, &courseID, &category, &action, &createdAt); err != nil {
			return out, err
		}
		out.FilterFlags = append(out.FilterFlags, map[string]any{
			"id":         id.String(),
			"instanceId": instanceID.String(),
			"courseId":   courseID.String(),
			"category":   category,
			"action":     action,
			"createdAt":  createdAt.UTC().Format(time.RFC3339),
		})
	}
	return out, frows.Err()
}

// EraseUserContent deletes content tool states and nulls event actor for a user.
func EraseUserContent(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) error {
	if pool == nil {
		return nil
	}
	if _, err := pool.Exec(ctx, `
DELETE FROM course.content_tool_state_resets r
USING course.course_enrollments ce
WHERE r.enrollment_id = ce.id AND ce.user_id = $1
`, userID); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `DELETE FROM course.content_tool_filter_flags WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `DELETE FROM course.content_tool_ai_consents WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `
UPDATE course.content_tool_moderation SET actor_user_id = NULL WHERE actor_user_id = $1
`, userID); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `
UPDATE course.content_tool_moderation SET subject_user_id = NULL WHERE subject_user_id = $1
`, userID); err != nil {
		return err
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
	if _, err := pool.Exec(ctx, `
UPDATE course.content_tool_state_resets SET reset_by = NULL WHERE reset_by = $1
`, userID); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `
UPDATE course.content_tool_state_resets SET restored_by = NULL WHERE restored_by = $1
`, userID); err != nil {
		return err
	}
	return nil
}

// CountUserContentRows returns remaining state/consent/filter rows for a user (erasure verification).
func CountUserContentRows(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `
SELECT
  (SELECT COUNT(*) FROM course.content_tool_states WHERE user_id = $1) +
  (SELECT COUNT(*) FROM course.content_tool_ai_consents WHERE user_id = $1) +
  (SELECT COUNT(*) FROM course.content_tool_filter_flags WHERE user_id = $1)
`, userID).Scan(&n)
	return n, err
}
