package adaptivecontent

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserACEExport is the DSAR export slice for Adaptive Content Engine artifacts (AC.8 FR-5).
type UserACEExport struct {
	Profiles  []map[string]any `json:"adaptationProfiles"`
	Servings  []map[string]any `json:"adaptationServings"`
	Outcomes  []map[string]any `json:"adaptationOutcomes"`
	Optouts   []map[string]any `json:"optouts"`
	Contests  []map[string]any `json:"contests"`
	Events    []map[string]any `json:"events"`
	Variants  []map[string]any `json:"servedVariants"`
}

// ExportUserContent collects a user's ACE profiles, servings, outcomes, opt-outs, contests, and events.
func ExportUserContent(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (UserACEExport, error) {
	out := UserACEExport{
		Profiles: []map[string]any{},
		Servings: []map[string]any{},
		Outcomes: []map[string]any{},
		Optouts:  []map[string]any{},
		Contests: []map[string]any{},
		Events:   []map[string]any{},
		Variants: []map[string]any{},
	}
	if pool == nil {
		return out, nil
	}

	prows, err := pool.Query(ctx, `
SELECT p.id, p.unit_id, u.course_id, c.course_code, p.enrollment_id, p.emphasis_mode,
       p.profile_signature, p.is_neutral, p.payload_json, p.created_at
FROM course.adaptation_profiles p
JOIN course.adaptive_content_units u ON u.id = p.unit_id
JOIN course.courses c ON c.id = u.course_id
WHERE p.user_id = $1
ORDER BY p.created_at ASC
`, userID)
	if err != nil {
		return out, err
	}
	defer prows.Close()
	for prows.Next() {
		var id, unitID, courseID, enrollmentID uuid.UUID
		var courseCode, signature string
		var emphasis *string
		var isNeutral bool
		var payload []byte
		var createdAt time.Time
		if err := prows.Scan(&id, &unitID, &courseID, &courseCode, &enrollmentID, &emphasis,
			&signature, &isNeutral, &payload, &createdAt); err != nil {
			return out, err
		}
		row := map[string]any{
			"id":               id.String(),
			"unitId":           unitID.String(),
			"courseId":         courseID.String(),
			"courseCode":       courseCode,
			"enrollmentId":     enrollmentID.String(),
			"profileSignature": signature,
			"isNeutral":        isNeutral,
			"createdAt":        createdAt.UTC().Format(time.RFC3339),
		}
		if emphasis != nil {
			row["emphasisMode"] = *emphasis
		}
		if len(payload) > 0 {
			var parsed any
			if json.Unmarshal(payload, &parsed) == nil {
				row["payload"] = parsed
			}
		}
		out.Profiles = append(out.Profiles, row)
	}
	if err := prows.Err(); err != nil {
		return out, err
	}

	srows, err := pool.Query(ctx, `
SELECT s.id, s.unit_id, u.course_id, c.course_code, s.profile_id, s.variant_id,
       s.was_holdout, s.was_fallback, s.content_version, s.view_count, s.served_at
FROM course.adaptation_servings s
JOIN course.adaptive_content_units u ON u.id = s.unit_id
JOIN course.courses c ON c.id = u.course_id
JOIN course.course_enrollments e ON e.id = s.enrollment_id
WHERE e.user_id = $1
ORDER BY s.served_at ASC
`, userID)
	if err != nil {
		return out, err
	}
	defer srows.Close()
	for srows.Next() {
		var id, unitID, courseID uuid.UUID
		var courseCode string
		var profileID, variantID *uuid.UUID
		var wasHoldout, wasFallback bool
		var contentVersion, viewCount int32
		var servedAt time.Time
		if err := srows.Scan(&id, &unitID, &courseID, &courseCode, &profileID, &variantID,
			&wasHoldout, &wasFallback, &contentVersion, &viewCount, &servedAt); err != nil {
			return out, err
		}
		row := map[string]any{
			"id":             id.String(),
			"unitId":         unitID.String(),
			"courseId":       courseID.String(),
			"courseCode":     courseCode,
			"wasHoldout":     wasHoldout,
			"wasFallback":    wasFallback,
			"contentVersion": contentVersion,
			"viewCount":      viewCount,
			"servedAt":       servedAt.UTC().Format(time.RFC3339),
		}
		if profileID != nil {
			row["profileId"] = profileID.String()
		}
		if variantID != nil {
			row["variantId"] = variantID.String()
		}
		out.Servings = append(out.Servings, row)
	}
	if err := srows.Err(); err != nil {
		return out, err
	}

	orows, err := pool.Query(ctx, `
SELECT o.serving_id, o.pre_score_pct, o.post_score_pct, o.mastery_before, o.mastery_after,
       o.lift, o.emphasis_mode, o.was_holdout, o.measured_at
FROM course.adaptation_outcomes o
JOIN course.adaptation_servings s ON s.id = o.serving_id
JOIN course.course_enrollments e ON e.id = s.enrollment_id
WHERE e.user_id = $1
ORDER BY o.measured_at ASC
`, userID)
	if err != nil {
		return out, err
	}
	defer orows.Close()
	for orows.Next() {
		var servingID uuid.UUID
		var pre, post, mb, ma, lift *float32
		var emphasis *string
		var wasHoldout bool
		var measuredAt time.Time
		if err := orows.Scan(&servingID, &pre, &post, &mb, &ma, &lift, &emphasis, &wasHoldout, &measuredAt); err != nil {
			return out, err
		}
		row := map[string]any{
			"servingId":  servingID.String(),
			"wasHoldout": wasHoldout,
			"measuredAt": measuredAt.UTC().Format(time.RFC3339),
		}
		if pre != nil {
			row["preScorePct"] = *pre
		}
		if post != nil {
			row["postScorePct"] = *post
		}
		if mb != nil {
			row["masteryBefore"] = *mb
		}
		if ma != nil {
			row["masteryAfter"] = *ma
		}
		if lift != nil {
			row["lift"] = *lift
		}
		if emphasis != nil {
			row["emphasisMode"] = *emphasis
		}
		out.Outcomes = append(out.Outcomes, row)
	}
	if err := orows.Err(); err != nil {
		return out, err
	}

	optRows, err := pool.Query(ctx, `
SELECT o.course_id, c.course_code, o.opted_out, o.updated_at
FROM course.adaptive_content_optouts o
JOIN course.courses c ON c.id = o.course_id
WHERE o.user_id = $1
ORDER BY o.updated_at ASC
`, userID)
	if err != nil {
		return out, err
	}
	defer optRows.Close()
	for optRows.Next() {
		var courseID uuid.UUID
		var courseCode string
		var optedOut bool
		var updatedAt time.Time
		if err := optRows.Scan(&courseID, &courseCode, &optedOut, &updatedAt); err != nil {
			return out, err
		}
		out.Optouts = append(out.Optouts, map[string]any{
			"courseId":   courseID.String(),
			"courseCode": courseCode,
			"optedOut":   optedOut,
			"updatedAt":  updatedAt.UTC().Format(time.RFC3339),
		})
	}
	if err := optRows.Err(); err != nil {
		return out, err
	}

	crows, err := pool.Query(ctx, `
SELECT ct.id, ct.course_id, c.course_code, ct.unit_id, ct.serving_id, ct.reason,
       ct.status, ct.created_at, ct.resolved_at
FROM course.adaptive_content_contests ct
JOIN course.courses c ON c.id = ct.course_id
WHERE ct.student_user_id = $1
ORDER BY ct.created_at ASC
`, userID)
	if err != nil {
		return out, err
	}
	defer crows.Close()
	for crows.Next() {
		var id, courseID, unitID uuid.UUID
		var courseCode, status string
		var servingID *uuid.UUID
		var reason *string
		var createdAt time.Time
		var resolvedAt *time.Time
		if err := crows.Scan(&id, &courseID, &courseCode, &unitID, &servingID, &reason,
			&status, &createdAt, &resolvedAt); err != nil {
			return out, err
		}
		row := map[string]any{
			"id":         id.String(),
			"courseId":   courseID.String(),
			"courseCode": courseCode,
			"unitId":     unitID.String(),
			"status":     status,
			"createdAt":  createdAt.UTC().Format(time.RFC3339),
		}
		if servingID != nil {
			row["servingId"] = servingID.String()
		}
		if reason != nil {
			row["reason"] = *reason
		}
		if resolvedAt != nil {
			row["resolvedAt"] = resolvedAt.UTC().Format(time.RFC3339)
		}
		out.Contests = append(out.Contests, row)
	}
	if err := crows.Err(); err != nil {
		return out, err
	}

	erows, err := pool.Query(ctx, `
SELECT e.id, e.course_id, c.course_code, e.unit_id, e.event_type, e.detail_json, e.created_at
FROM course.adaptive_content_events e
JOIN course.courses c ON c.id = e.course_id
WHERE e.subject_user_id = $1 OR e.actor_user_id = $1
ORDER BY e.created_at ASC
LIMIT 5000
`, userID)
	if err != nil {
		return out, err
	}
	defer erows.Close()
	for erows.Next() {
		var id, courseID uuid.UUID
		var courseCode, eventType string
		var unitID *uuid.UUID
		var detail []byte
		var createdAt time.Time
		if err := erows.Scan(&id, &courseID, &courseCode, &unitID, &eventType, &detail, &createdAt); err != nil {
			return out, err
		}
		row := map[string]any{
			"id":         id.String(),
			"courseId":   courseID.String(),
			"courseCode": courseCode,
			"eventType":  eventType,
			"createdAt":  createdAt.UTC().Format(time.RFC3339),
		}
		if unitID != nil {
			row["unitId"] = unitID.String()
		}
		if len(detail) > 0 {
			var parsed any
			if json.Unmarshal(detail, &parsed) == nil {
				row["detail"] = parsed
			}
		}
		out.Events = append(out.Events, row)
	}
	if err := erows.Err(); err != nil {
		return out, err
	}

	return out, nil
}

// EraseUserContent removes ACE artifacts for a user (DSAR erasure / AC.8 FR-5).
func EraseUserContent(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) error {
	if pool == nil {
		return nil
	}
	// Contests by student.
	if _, err := pool.Exec(ctx, `
DELETE FROM course.adaptive_content_contests WHERE student_user_id = $1
`, userID); err != nil {
		return err
	}
	// Opt-outs.
	if _, err := pool.Exec(ctx, `
DELETE FROM course.adaptive_content_optouts WHERE user_id = $1
`, userID); err != nil {
		return err
	}
	// Events referencing the user as subject/actor (best-effort anonymise detail).
	if _, err := pool.Exec(ctx, `
DELETE FROM course.adaptive_content_events
WHERE subject_user_id = $1 OR actor_user_id = $1
`, userID); err != nil {
		return err
	}
	// Outcomes → servings → profiles via enrollments.
	if _, err := pool.Exec(ctx, `
DELETE FROM course.adaptation_outcomes o
USING course.adaptation_servings s
JOIN course.course_enrollments e ON e.id = s.enrollment_id
WHERE o.serving_id = s.id AND e.user_id = $1
`, userID); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `
DELETE FROM course.adaptation_servings s
USING course.course_enrollments e
WHERE s.enrollment_id = e.id AND e.user_id = $1
`, userID); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `
DELETE FROM course.adaptation_profiles p
USING course.course_enrollments e
WHERE p.enrollment_id = e.id AND e.user_id = $1
`, userID); err != nil {
		return err
	}
	return nil
}

// CountUserContentRows returns remaining ACE rows for a user (erasure verification).
func CountUserContentRows(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `
SELECT
  (SELECT COUNT(*) FROM course.adaptive_content_contests WHERE student_user_id = $1) +
  (SELECT COUNT(*) FROM course.adaptive_content_optouts WHERE user_id = $1) +
  (SELECT COUNT(*) FROM course.adaptation_profiles p
     JOIN course.course_enrollments e ON e.id = p.enrollment_id WHERE e.user_id = $1) +
  (SELECT COUNT(*) FROM course.adaptation_servings s
     JOIN course.course_enrollments e ON e.id = s.enrollment_id WHERE e.user_id = $1)
`, userID).Scan(&n)
	return n, err
}
