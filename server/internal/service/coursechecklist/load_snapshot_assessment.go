package coursechecklist

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/repos/coursegrading"
	"github.com/lextures/lextures/server/internal/repos/studentaccommodations"
)

func loadGradingSlice(ctx context.Context, pool *pgxpool.Pool, courseCode string, snap *CourseSnapshot, count func(int)) error {
	settings, err := coursegrading.GetSettingsForCourseCode(ctx, pool, courseCode)
	count(2) // settings query + groups query inside helper
	if err != nil {
		return err
	}
	if settings != nil {
		snap.GradingScale = settings.GradingScale
		for _, g := range settings.AssignmentGroups {
			w := g.WeightPercent
			snap.AssignmentGroups = append(snap.AssignmentGroups, AssignmentGroupSnap{
				ID:          g.ID,
				Name:        g.Name,
				Weight:      &w,
				DropLowest:  g.DropLowest,
				DropHighest: g.DropHighest,
			})
		}
	}
	// Active grading scheme bands (CC.5 FR-10) — fold into grading need.
	if snap.GradingSchemeID != nil {
		var scale []byte
		err := pool.QueryRow(ctx, `
SELECT scale_json FROM course.grading_schemes WHERE id = $1
`, *snap.GradingSchemeID).Scan(&scale)
		count(1)
		if err != nil {
			if !isUndefinedTable(err) && !strings.Contains(strings.ToLower(err.Error()), "no rows") {
				return err
			}
		} else if len(scale) > 0 {
			snap.GradingSchemeScaleJSON = append(json.RawMessage(nil), scale...)
		}
	}
	return nil
}

func loadAccommodationsSlice(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, snap *CourseSnapshot, count func(int)) error {
	n, err := studentaccommodations.CountActiveForCourse(ctx, pool, courseID)
	count(1)
	if err != nil {
		if isUndefinedTable(err) {
			return nil
		}
		return err
	}
	snap.AccommodationCount = n
	types, latest, err := studentaccommodations.AggregateActiveTypesForCourse(ctx, pool, courseID)
	count(1)
	if err != nil {
		if isUndefinedTable(err) {
			return nil
		}
		return err
	}
	for _, t := range types {
		snap.AccommodationTypeCounts = append(snap.AccommodationTypeCounts, AccommodationTypeCount{
			Type: t.Type, Count: t.Count,
		})
	}
	snap.LatestAccommodationAt = latest
	return nil
}

// loadAssessmentCC5Slices loads CC.5 assessment/interaction DataNeeds in ≤ 5 queries.
func loadAssessmentCC5Slices(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID uuid.UUID,
	needs []DataNeed,
	snap *CourseSnapshot,
	count func(int),
) error {
	if hasDataNeed(needs, DataNeedAssessmentItems) {
		if err := loadAssessmentItems(ctx, pool, courseID, snap, count); err != nil {
			return err
		}
	}
	if hasDataNeed(needs, DataNeedPeerReview) {
		if err := loadPeerReviewConfigs(ctx, pool, courseID, snap, count); err != nil {
			return err
		}
	}
	if hasDataNeed(needs, DataNeedDiscussions) || hasDataNeed(needs, DataNeedOfficeHours) || hasDataNeed(needs, DataNeedEnrollmentGroups) {
		if err := loadInteractionSlice(ctx, pool, courseID, needs, snap, count); err != nil {
			return err
		}
	}
	if hasDataNeed(needs, DataNeedAnnouncementCadence) {
		if err := loadAnnouncementCadence(ctx, pool, courseID, snap, count); err != nil {
			return err
		}
	}
	return nil
}

func loadAssessmentItems(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, snap *CourseSnapshot, count func(int)) error {
	rows, err := pool.Query(ctx, `
SELECT
  c.id, c.kind, c.title, c.parent_id, COALESCE(parent.title, ''),
  c.sort_order, c.published, c.archived, c.due_at, c.assignment_group_id,
  COALESCE(m.available_from, q.available_from) AS available_from,
  COALESCE(m.available_until, q.available_until) AS available_until,
  COALESCE(m.points_worth, q.points_worth) AS points_worth,
	(COALESCE(LENGTH(TRIM(m.markdown)), 0) > 0 OR COALESCE(LENGTH(TRIM(q.markdown)), 0) > 0) AS has_body,
  (m.rubric_json IS NOT NULL AND m.rubric_json::text NOT IN ('', 'null', '[]', '{}')) AS has_rubric,
  COALESCE(m.late_submission_policy, q.late_submission_policy, '') AS late_policy,
  COALESCE(m.posting_policy, '') AS posting_policy,
  COALESCE(m.originality_detection, 'disabled') AS originality,
  COALESCE(q.unlimited_attempts, false) AS unlimited_attempts,
  COALESCE(q.max_attempts, 0) AS max_attempts,
  COALESCE(q.grade_attempt_policy, '') AS grade_attempt_policy,
  COALESCE(q.show_score_timing, '') AS show_score_timing,
  COALESCE(q.review_visibility, '') AS review_visibility,
  COALESCE(q.review_when, '') AS review_when,
  q.time_limit_minutes,
  COALESCE(q.shuffle_questions, false) AS shuffle_questions,
  COALESCE(q.shuffle_choices, false) AS shuffle_choices,
  COALESCE(q.lockdown_mode::text, 'standard') AS lockdown_mode
FROM course.course_structure_items c
LEFT JOIN course.course_structure_items parent ON parent.id = c.parent_id
LEFT JOIN course.module_assignments m ON m.structure_item_id = c.id AND c.kind = 'assignment'
LEFT JOIN course.module_quizzes q ON q.structure_item_id = c.id AND c.kind = 'quiz'
WHERE c.course_id = $1
  AND c.kind IN ('assignment', 'quiz')
  AND c.archived = false
ORDER BY c.sort_order ASC, c.title ASC
LIMIT 5000
`, courseID)
	count(1)
	if err != nil {
		if isUndefinedTable(err) {
			return nil
		}
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var it AssessmentItemSnap
		var points *int
		var timeLimit *int32
		if err := rows.Scan(
			&it.ID, &it.Kind, &it.Title, &it.ParentID, &it.ModuleTitle,
			&it.SortOrder, &it.Published, &it.Archived, &it.DueAt, &it.AssignmentGroupID,
			&it.AvailableFrom, &it.AvailableUntil, &points,
			&it.HasBody, &it.HasRubric, &it.LateSubmissionPolicy, &it.PostingPolicy, &it.OriginalityDetection,
			&it.UnlimitedAttempts, &it.MaxAttempts, &it.GradeAttemptPolicy,
			&it.ShowScoreTiming, &it.ReviewVisibility, &it.ReviewWhen,
			&timeLimit, &it.ShuffleQuestions, &it.ShuffleChoices, &it.LockdownMode,
		); err != nil {
			return err
		}
		it.Points = points
		if timeLimit != nil {
			v := int(*timeLimit)
			it.TimeLimitMinutes = &v
		}
		snap.AssessmentItems = append(snap.AssessmentItems, it)
	}
	return rows.Err()
}

func loadPeerReviewConfigs(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, snap *CourseSnapshot, count func(int)) error {
	rows, err := pool.Query(ctx, `
SELECT c.assignment_id, si.title, c.reviews_per_reviewer, c.opens_at, c.closes_at, si.due_at,
       (m.rubric_json IS NOT NULL AND m.rubric_json::text NOT IN ('', 'null', '[]', '{}')) AS has_rubric
FROM course.peer_review_configs c
INNER JOIN course.course_structure_items si ON si.id = c.assignment_id
LEFT JOIN course.module_assignments m ON m.structure_item_id = si.id
WHERE si.course_id = $1
ORDER BY si.title ASC
LIMIT 500
`, courseID)
	count(1)
	if err != nil {
		if isUndefinedTable(err) {
			return nil
		}
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var p PeerReviewConfigSnap
		if err := rows.Scan(
			&p.AssignmentID, &p.AssignmentTitle, &p.ReviewsPerReviewer,
			&p.OpensAt, &p.ClosesAt, &p.DueAt, &p.HasRubric,
		); err != nil {
			return err
		}
		snap.PeerReviewConfigs = append(snap.PeerReviewConfigs, p)
	}
	return rows.Err()
}

func loadInteractionSlice(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID uuid.UUID,
	needs []DataNeed,
	snap *CourseSnapshot,
	count func(int),
) error {
	// One round-trip: discussions + office hours + enrollment groups.
	row := pool.QueryRow(ctx, `
SELECT
  (SELECT COUNT(*)::int FROM course.discussion_forums WHERE course_id = $1),
  (SELECT COUNT(*)::int FROM course.discussion_threads t
     INNER JOIN course.discussion_forums f ON f.id = t.forum_id
    WHERE f.course_id = $1),
	(SELECT COUNT(*)::int FROM course.appointment_slots s
     INNER JOIN course.availability_windows w ON w.id = s.window_id
    WHERE w.course_id = $1 AND s.status <> 'cancelled' AND w.status = 'active'
      AND s.slot_start > NOW()),
  (SELECT COUNT(*)::int FROM course.enrollment_group_sets WHERE course_id = $1),
  (SELECT COUNT(*)::int FROM course.course_enrollments ce
     INNER JOIN course.enrollment_roles er ON er.role_key = ce.role AND er.is_student_equivalent
    WHERE ce.course_id = $1 AND ce.active
      AND NOT EXISTS (
        SELECT 1 FROM course.enrollment_group_memberships m
        WHERE m.enrollment_id = ce.id
      ))
`, courseID)
	count(1)
	var forums, prompts, slots, sets, unassigned int
	if err := row.Scan(&forums, &prompts, &slots, &sets, &unassigned); err != nil {
		if isUndefinedTable(err) {
			return nil
		}
		return err
	}
	if hasDataNeed(needs, DataNeedDiscussions) {
		snap.DiscussionForumCount = forums
		snap.DiscussionPromptCount = prompts
	}
	if hasDataNeed(needs, DataNeedOfficeHours) {
		snap.FutureOfficeHourSlots = slots
	}
	if hasDataNeed(needs, DataNeedEnrollmentGroups) {
		snap.EnrollmentGroupSetCount = sets
		snap.UnassignedStudentCount = unassigned
	}
	return nil
}

func loadAnnouncementCadence(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, snap *CourseSnapshot, count func(int)) error {
	rows, err := pool.Query(ctx, `
SELECT m.created_at
FROM course.feed_messages m
INNER JOIN course.feed_channels c ON c.id = m.channel_id
WHERE c.course_id = $1
  AND LOWER(c.name) = 'announcements'
	AND m.parent_message_id IS NULL
ORDER BY m.created_at ASC
LIMIT 500
`, courseID)
	count(1)
	if err != nil {
		if isUndefinedTable(err) {
			return nil
		}
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var t time.Time
		if err := rows.Scan(&t); err != nil {
			return err
		}
		snap.AnnouncementTimes = append(snap.AnnouncementTimes, t)
	}
	return rows.Err()
}
