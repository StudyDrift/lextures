// Package atriskscoring computes nightly at-risk scores and manages alerts (plan 9.2).
package atriskscoring

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/repos/atrisk"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/emailjobs"
)

const enrollmentChunkSize = 500

// Service runs at-risk scoring jobs.
type Service struct {
	Pool   *pgxpool.Pool
	Config config.Config
}

// RunForCourse scores all active student enrollments in one course for computedDate (UTC date).
func (s *Service) RunForCourse(ctx context.Context, courseID uuid.UUID, computedDate time.Time) (int, error) {
	if s.Pool == nil {
		return 0, fmt.Errorf("atriskscoring: pool required")
	}
	orgID, err := orgIDForCourse(ctx, s.Pool, courseID)
	if err != nil {
		return 0, err
	}
	cfg, err := atrisk.LoadEffective(ctx, s.Pool, orgID)
	if err != nil {
		return 0, err
	}
	courseCodePtr, err := course.GetCourseCodeByID(ctx, s.Pool, courseID)
	if err != nil || courseCodePtr == nil {
		return 0, fmt.Errorf("atriskscoring: course code not found")
	}
	courseCode := *courseCodePtr
	day := dateOnly(computedDate)
	enrollmentIDs, err := listStudentEnrollmentIDs(ctx, s.Pool, courseID)
	if err != nil {
		return 0, err
	}
	processed := 0
	for i := 0; i < len(enrollmentIDs); i += enrollmentChunkSize {
		end := i + enrollmentChunkSize
		if end > len(enrollmentIDs) {
			end = len(enrollmentIDs)
		}
		chunk := enrollmentIDs[i:end]
		signals, err := loadSignals(ctx, s.Pool, courseID, chunk, day)
		if err != nil {
			return processed, err
		}
		for _, eid := range chunk {
			in, ok := signals[eid]
			if !ok {
				in = SignalInputs{}
			}
			score, comp := ComputeWeightedScore(in, cfg)
			if err := atrisk.UpsertScore(ctx, s.Pool, atrisk.ScoreRow{
				EnrollmentID: eid,
				ComputedDate: day,
				Score:        score,
				MissingPct:   &in.MissingPct,
				QuizAvg:      in.QuizAvg,
				DaysInactive: in.DaysInactive,
				GradeTrend:   &in.GradeTrend,
				TopFactor:    comp.TopFactor,
			}); err != nil {
				return processed, err
			}
			slog.Info("at_risk.score",
				"enrollment_id", eid,
				"course_id", courseID,
				"score", score,
				"top_factor", comp.TopFactor,
			)
			if err := s.syncAlert(ctx, courseID, courseCode, eid, day, score, comp.TopFactor, cfg); err != nil {
				return processed, err
			}
			processed++
		}
	}
	return processed, nil
}

// RunAllCourses scores every non-archived course (nightly sweep).
func (s *Service) RunAllCourses(ctx context.Context, computedDate time.Time) (int, error) {
	ids, err := listActiveCourseIDs(ctx, s.Pool)
	if err != nil {
		return 0, err
	}
	total := 0
	for _, cid := range ids {
		n, err := s.RunForCourse(ctx, cid, computedDate)
		if err != nil {
			slog.Warn("at_risk.course_failed", "course_id", cid, "err", err)
			continue
		}
		total += n
	}
	return total, nil
}

func (s *Service) syncAlert(
	ctx context.Context,
	courseID uuid.UUID,
	courseCode string,
	enrollmentID uuid.UUID,
	day time.Time,
	score float32,
	topFactor string,
	cfg atrisk.Config,
) error {
	now := time.Now().UTC()
	if score < cfg.Threshold {
		if _, err := atrisk.ResolveActiveAlerts(ctx, s.Pool, enrollmentID, now); err != nil {
			return err
		}
		return atrisk.ResolveDismissedEpisode(ctx, s.Pool, enrollmentID, now)
	}

	blocking, err := atrisk.HasBlockingAlert(ctx, s.Pool, enrollmentID, day)
	if err != nil {
		return err
	}
	if blocking {
		return nil
	}
	if dismissed, err := atrisk.LatestDismissedWithoutResolve(ctx, s.Pool, enrollmentID); err != nil {
		return err
	} else if dismissed != nil {
		return nil
	}

	id, created, err := atrisk.CreateAlert(ctx, s.Pool, enrollmentID, day, score, topFactor)
	if err != nil {
		return err
	}
	if !created {
		return nil
	}
	if s.Config.EmailNotificationsEnabled {
		if err := s.queueAlertEmails(ctx, courseID, courseCode, enrollmentID, score, topFactor); err != nil {
			slog.Warn("at_risk.email_enqueue", "alert_id", id, "err", err)
		}
	}
	return nil
}

func (s *Service) queueAlertEmails(ctx context.Context, courseID uuid.UUID, courseCode string, enrollmentID uuid.UUID, score float32, topFactor string) error {
	studentName, err := studentDisplayName(ctx, s.Pool, enrollmentID)
	if err != nil {
		return err
	}
	courseTitle, err := courseTitleByID(ctx, s.Pool, courseID)
	if err != nil {
		return err
	}
	instructors, err := atrisk.ListInstructorUserIDs(ctx, s.Pool, courseID)
	if err != nil {
		return err
	}
	link := fmt.Sprintf("%s/courses/%s/at-risk", s.Config.PublicWebOrigin, courseCode)
	factorLabel := factorLabelForEmail(topFactor)
	for _, uid := range instructors {
		vars := map[string]string{
			"courseName":    courseTitle,
			"studentName":   studentName,
			"score":         fmt.Sprintf("%.0f", score),
			"topFactor":     factorLabel,
			"topFactor2":    "",
			"topFactor3":    "",
			"link":          link,
			"progressLink":  fmt.Sprintf("%s/courses/%s/enrollments/%s", s.Config.PublicWebOrigin, courseCode, enrollmentID),
		}
		subject := fmt.Sprintf("At-risk alert: %s in %s", studentName, courseTitle)
		if _, err := emailjobs.Enqueue(ctx, s.Pool, uid, "at_risk_alert", subject, "at_risk_alert", vars); err != nil {
			return err
		}
	}
	return nil
}

func dateOnly(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func listStudentEnrollmentIDs(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := pool.Query(ctx, `
SELECT ce.id
FROM course.course_enrollments ce
INNER JOIN course.enrollment_roles er ON er.role_key = ce.role AND er.is_student_equivalent = true
WHERE ce.course_id = $1 AND ce.active
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func listActiveCourseIDs(ctx context.Context, pool *pgxpool.Pool) ([]uuid.UUID, error) {
	rows, err := pool.Query(ctx, `
SELECT id FROM course.courses WHERE NOT archived
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func loadSignals(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, enrollmentIDs []uuid.UUID, day time.Time) (map[uuid.UUID]SignalInputs, error) {
	out := make(map[uuid.UUID]SignalInputs, len(enrollmentIDs))
	if len(enrollmentIDs) == 0 {
		return out, nil
	}
	rows, err := pool.Query(ctx, `
WITH enrollments AS (
    SELECT ce.id AS enrollment_id, ce.user_id
    FROM course.course_enrollments ce
    WHERE ce.id = ANY($1)
),
assignments AS (
    SELECT
        e.enrollment_id,
        COUNT(*) FILTER (
            WHERE si.kind = 'assignment'
              AND si.published AND NOT si.archived
              AND ma.available_until IS NOT NULL
              AND ma.available_until < $3
              AND sub.id IS NULL
        )::float / NULLIF(COUNT(*) FILTER (
            WHERE si.kind = 'assignment'
              AND si.published AND NOT si.archived
              AND ma.available_until IS NOT NULL
              AND ma.available_until < $3
        ), 0) * 100 AS missing_pct
    FROM enrollments e
    CROSS JOIN course.course_structure_items si
    LEFT JOIN course.module_assignments ma ON ma.structure_item_id = si.id
    LEFT JOIN course.module_assignment_submissions sub
        ON sub.module_item_id = si.id AND sub.submitted_by = e.user_id
    WHERE si.course_id = $2 AND si.kind = 'assignment'
    GROUP BY e.enrollment_id
),
quizzes AS (
    SELECT
        e.enrollment_id,
        AVG(latest.score_percent)::real AS quiz_avg
    FROM enrollments e
    LEFT JOIN LATERAL (
        SELECT qa.score_percent
        FROM course.quiz_attempts qa
        INNER JOIN course.course_structure_items si ON si.id = qa.structure_item_id
        WHERE qa.student_user_id = e.user_id
          AND qa.course_id = $2
          AND qa.status = 'submitted'
          AND si.kind = 'quiz'
          AND si.published AND NOT si.archived
        ORDER BY qa.submitted_at DESC
        LIMIT 20
    ) latest ON true
    GROUP BY e.enrollment_id
),
activity AS (
    SELECT
        e.enrollment_id,
        GREATEST(0, EXTRACT(DAY FROM ($3 - COALESCE(
            (SELECT MAX(ua.occurred_at) FROM "user".user_audit ua WHERE ua.user_id = e.user_id AND ua.course_id = $2),
            (SELECT MAX(sub.submitted_at) FROM course.module_assignment_submissions sub
             WHERE sub.course_id = $2 AND sub.submitted_by = e.user_id),
            (SELECT MAX(qa.submitted_at) FROM course.quiz_attempts qa
             WHERE qa.course_id = $2 AND qa.student_user_id = e.user_id AND qa.status = 'submitted'),
            '1970-01-01'::timestamptz
        )))::int) AS days_inactive
    FROM enrollments e
),
trends AS (
    SELECT
        e.enrollment_id,
        COALESCE(
            GREATEST(0,
                (SELECT AVG(cg.points_earned) FROM course.course_grades cg
                 WHERE cg.course_id = $2 AND cg.student_user_id = e.user_id
                   AND cg.updated_at >= $3 - interval '14 days') -
                (SELECT AVG(cg.points_earned) FROM course.course_grades cg
                 WHERE cg.course_id = $2 AND cg.student_user_id = e.user_id
                   AND cg.updated_at >= $3 - interval '28 days'
                   AND cg.updated_at < $3 - interval '14 days')
            ) * -1,
            0
        )::real AS grade_trend
    FROM enrollments e
)
SELECT
    e.enrollment_id,
    COALESCE(a.missing_pct, 0),
    q.quiz_avg,
    COALESCE(act.days_inactive, 0),
    COALESCE(t.grade_trend, 0)
FROM enrollments e
LEFT JOIN assignments a ON a.enrollment_id = e.enrollment_id
LEFT JOIN quizzes q ON q.enrollment_id = e.enrollment_id
LEFT JOIN activity act ON act.enrollment_id = e.enrollment_id
LEFT JOIN trends t ON t.enrollment_id = e.enrollment_id
`, enrollmentIDs, courseID, day)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var eid uuid.UUID
		var missing float32
		var quizAvg *float32
		var inactive int
		var trend float32
		if err := rows.Scan(&eid, &missing, &quizAvg, &inactive, &trend); err != nil {
			return nil, err
		}
		out[eid] = SignalInputs{
			MissingPct:   missing,
			QuizAvg:      quizAvg,
			DaysInactive: inactive,
			GradeTrend:   trend,
		}
	}
	return out, rows.Err()
}

func studentDisplayName(ctx context.Context, pool *pgxpool.Pool, enrollmentID uuid.UUID) (string, error) {
	var name, email string
	err := pool.QueryRow(ctx, `
SELECT COALESCE(NULLIF(TRIM(u.display_name), ''), u.email), u.email
FROM course.course_enrollments ce
INNER JOIN "user".users u ON u.id = ce.user_id
WHERE ce.id = $1
`, enrollmentID).Scan(&name, &email)
	if err != nil {
		return "", err
	}
	if name == "" {
		return email, nil
	}
	return name, nil
}

func courseTitleByID(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (string, error) {
	var title string
	err := pool.QueryRow(ctx, `SELECT title FROM course.courses WHERE id = $1`, courseID).Scan(&title)
	return title, err
}

func orgIDForCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (uuid.UUID, error) {
	var orgID uuid.UUID
	err := pool.QueryRow(ctx, `SELECT org_id FROM course.courses WHERE id = $1`, courseID).Scan(&orgID)
	return orgID, err
}

func factorLabelForEmail(key string) string {
	switch key {
	case "quiz":
		return "Below quiz average"
	case "inactive":
		return "Inactive 7+ days"
	case "trend":
		return "Declining grades"
	default:
		return "Missing assignments"
	}
}
