package conditionalrelease

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/models/conditionalrelease"
)

// ListRequirementsReport returns students × item requirements for instructor reporting.
func ListRequirementsReport(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]conditionalrelease.ReportRow, error) {
	rows, err := pool.Query(ctx, `
SELECT
    ce.id AS enrollment_id,
    u.id AS user_id,
    COALESCE(u.display_name, '') AS display_name,
    u.email,
    i.id AS item_id,
    i.title AS item_title,
    m.title AS module_title,
    icr.rule_type,
    COALESCE(sip.status, 'incomplete') AS status,
    sip.met_at
FROM course.course_enrollments ce
INNER JOIN auth.users u ON u.id = ce.user_id
INNER JOIN course.course_structure_items m ON m.course_id = ce.course_id AND m.kind = 'module' AND NOT m.archived
INNER JOIN course.course_structure_items i ON i.parent_id = m.id AND i.kind NOT IN ('heading') AND i.published AND NOT i.archived
LEFT JOIN course.item_completion_rules icr ON icr.item_id = i.id
LEFT JOIN course.student_item_progress sip ON sip.enrollment_id = ce.id AND sip.item_id = i.id
WHERE ce.course_id = $1 AND ce.role = 'student' AND ce.active
  AND (icr.id IS NOT NULL OR EXISTS (SELECT 1 FROM course.module_requirements mr WHERE mr.module_id = m.id))
ORDER BY u.display_name, m.sort_order, i.sort_order
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []conditionalrelease.ReportRow
	for rows.Next() {
		var row conditionalrelease.ReportRow
		var ruleType *string
		var metAt *string
		if err := rows.Scan(
			&row.EnrollmentID, &row.UserID, &row.DisplayName, &row.Email,
			&row.ItemID, &row.ItemTitle, &row.ModuleTitle,
			&ruleType, &row.Status, &metAt,
		); err != nil {
			return nil, err
		}
		if ruleType != nil {
			row.RuleType = *ruleType
		}
		if metAt != nil {
			row.MetAt = *metAt
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// ItemHasSubmission reports whether the student submitted an assignment or quiz for the item.
func ItemHasSubmission(ctx context.Context, pool *pgxpool.Pool, courseID, userID, itemID uuid.UUID) (bool, error) {
	var found bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1 FROM course.module_assignment_submissions mas
    INNER JOIN course.course_structure_items i ON i.id = mas.module_item_id
    WHERE i.id = $3 AND i.course_id = $1 AND mas.submitted_by = $2
    UNION ALL
    SELECT 1 FROM course.quiz_attempts qa
    WHERE qa.structure_item_id = $3 AND qa.course_id = $1 AND qa.student_user_id = $2
      AND qa.status = 'submitted'
)
`, courseID, userID, itemID).Scan(&found)
	return found, err
}

// ItemScorePercent returns the gradebook kept score percent for an item, or nil when ungraded.
func ItemScorePercent(ctx context.Context, pool *pgxpool.Pool, courseID, userID, itemID uuid.UUID) (*float64, error) {
	var earned *float64
	var worth *int32
	err := pool.QueryRow(ctx, `
SELECT cg.points_earned, COALESCE(ma.points_worth, mq.points_worth)
FROM course.course_grades cg
INNER JOIN course.course_structure_items csi ON csi.id = cg.module_item_id
LEFT JOIN course.module_assignments ma ON ma.structure_item_id = csi.id
LEFT JOIN course.module_quizzes mq ON mq.structure_item_id = csi.id
WHERE cg.course_id = $1 AND cg.student_user_id = $2 AND cg.module_item_id = $3
`, courseID, userID, itemID).Scan(&earned, &worth)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if earned == nil || worth == nil || *worth <= 0 {
		return nil, nil
	}
	pct := (*earned / float64(*worth)) * 100.0
	if pct < 0 {
		pct = 0
	} else if pct > 100 {
		pct = 100
	}
	return &pct, nil
}

// ItemHasDiscussionContribution reports whether the user posted in a discussion linked to the item.
func ItemHasDiscussionContribution(ctx context.Context, pool *pgxpool.Pool, courseID, userID, itemID uuid.UUID) (bool, error) {
	var found bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1 FROM course.discussion_posts dp
    INNER JOIN course.discussion_threads dt ON dt.id = dp.thread_id
    INNER JOIN course.course_structure_items i ON i.id = dt.structure_item_id
    WHERE i.id = $3 AND i.course_id = $1 AND dp.author_id = $2
)
`, courseID, userID, itemID).Scan(&found)
	return found, err
}

// ItemWasViewed reports whether the learner opened the item (self-paced progress or student progress).
func ItemWasViewed(ctx context.Context, pool *pgxpool.Pool, enrollmentID, itemID uuid.UUID) (bool, error) {
	var found bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1 FROM course.learner_item_progress
    WHERE enrollment_id = $1 AND item_id = $2 AND last_visited_at IS NOT NULL
    UNION ALL
    SELECT 1 FROM course.student_item_progress
    WHERE enrollment_id = $1 AND item_id = $2 AND status = 'complete'
)
`, enrollmentID, itemID).Scan(&found)
	return found, err
}

// ItemWasMarkedDone reports whether the learner explicitly completed the item.
func ItemWasMarkedDone(ctx context.Context, pool *pgxpool.Pool, enrollmentID, itemID uuid.UUID) (bool, error) {
	var found bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1 FROM course.learner_item_progress
    WHERE enrollment_id = $1 AND item_id = $2 AND status = 'completed'
    UNION ALL
    SELECT 1 FROM course.student_item_progress
    WHERE enrollment_id = $1 AND item_id = $2 AND status = 'complete'
)
`, enrollmentID, itemID).Scan(&found)
	return found, err
}
