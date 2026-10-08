package coursestructure

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/relativeschedule"
)

// LTILinkVisibleToStudent checks whether an lti_link module item is accessible to a student
// (item and parent module published and not archived, and the module's visible-from date reached).
func LTILinkVisibleToStudent(
	ctx context.Context, pool *pgxpool.Pool, courseID, itemID, userID uuid.UUID, now time.Time,
) (bool, error) {
	var (
		cPub, cArch, mPub, mArch bool
		mVF                      *time.Time
		scheduleMode             string
		crsAnchor                *time.Time
		enrollCreatedAt          *time.Time
	)
	err := pool.QueryRow(ctx, `
		SELECT
			el.published,
			el.archived,
			m.published,
			m.archived,
			m.visible_from,
			crs.schedule_mode,
			crs.relative_schedule_anchor_at,
			stu.created_at
		FROM course.course_structure_items el
		INNER JOIN course.course_structure_items m
			ON m.id = el.parent_id AND m.course_id = el.course_id AND m.kind = 'module'
		INNER JOIN course.courses crs ON crs.id = el.course_id
		LEFT JOIN course.course_enrollments stu
			ON stu.course_id = crs.id AND stu.user_id = $3 AND stu.active AND stu.role IN ('student', 'test_student')
		WHERE el.id = $1 AND el.course_id = $2 AND el.kind = 'lti_link'
	`, itemID, courseID, userID).Scan(
		&cPub, &cArch, &mPub, &mArch, &mVF, &scheduleMode, &crsAnchor, &enrollCreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	utc := now.UTC()
	effVF := mVF
	if scheduleMode == "relative" && crsAnchor != nil && enrollCreatedAt != nil {
		shift := &relativeschedule.Context{Anchor: *crsAnchor, EnrollmentStart: *enrollCreatedAt}
		effVF = shift.ShiftOpt(mVF)
	}
	return cPub && !cArch && mPub && !mArch && moduleVisibleFromOK(effVF, utc), nil
}
