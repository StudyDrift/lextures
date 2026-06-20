package calendar

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/repos/assignmentoverrides"
)

// Event is one assignment or quiz deadline surfaced in an iCal feed.
type Event struct {
	ItemID       uuid.UUID
	CourseID     uuid.UUID
	CourseCode   string
	CourseTitle  string
	Kind         string
	Title        string
	Description  string
	Start        time.Time
	End          time.Time
	AllDay       bool
	IsQuizWindow bool
}

// LoadEventsForCourses returns visible assignment/quiz calendar events for the given courses,
// resolved to userID's own assign-to targeting (plan 2.15): items not assigned to this user are
// omitted, and due/availability reflect their effective (most-specific-wins) dates.
func LoadEventsForCourses(ctx context.Context, pool *pgxpool.Pool, courseIDs []uuid.UUID, userID uuid.UUID, rangeStart, rangeEnd *time.Time) ([]Event, error) {
	if len(courseIDs) == 0 {
		return nil, nil
	}
	enrollmentByCourse, err := enrollmentIDsByCourse(ctx, pool, courseIDs, userID)
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
SELECT
	c.id,
	c.course_code,
	c.title,
	si.id,
	si.kind,
	si.title,
	COALESCE(ma.markdown, mq.markdown, '') AS description,
	si.due_at,
	ma.available_from,
	ma.available_until,
	mq.available_from,
	mq.available_until
FROM course.course_structure_items si
INNER JOIN course.courses c ON c.id = si.course_id
LEFT JOIN course.course_structure_items mod ON mod.id = si.parent_id AND mod.kind = 'module'
LEFT JOIN course.module_assignments ma ON ma.structure_item_id = si.id AND si.kind = 'assignment'
LEFT JOIN course.module_quizzes mq ON mq.structure_item_id = si.id AND si.kind = 'quiz'
WHERE si.course_id = ANY($1)
  AND si.kind IN ('assignment', 'quiz')
  AND si.published = true
  AND si.archived = false
  AND c.archived = false
  AND (mod.id IS NULL OR (mod.published = true AND mod.archived = false))
`, courseIDs)
	if err != nil {
		return nil, err
	}
	type rawRow struct {
		courseID                 uuid.UUID
		courseCode, courseTitle  string
		itemID                   uuid.UUID
		kind, title, description string
		dueAt                    *time.Time
		assignFrom, assignUntil  *time.Time
		quizFrom, quizUntil      *time.Time
	}
	var raw []rawRow
	itemsByCourse := map[uuid.UUID][]uuid.UUID{}
	basesByItem := map[uuid.UUID]assignmentoverrides.BaseDates{}
	for rows.Next() {
		var r rawRow
		if err := rows.Scan(
			&r.courseID, &r.courseCode, &r.courseTitle, &r.itemID, &r.kind, &r.title, &r.description,
			&r.dueAt, &r.assignFrom, &r.assignUntil, &r.quizFrom, &r.quizUntil,
		); err != nil {
			rows.Close()
			return nil, err
		}
		raw = append(raw, r)
		itemsByCourse[r.courseID] = append(itemsByCourse[r.courseID], r.itemID)
		if r.kind == "quiz" {
			basesByItem[r.itemID] = assignmentoverrides.BaseDates{DueAt: r.dueAt, AvailableFrom: r.quizFrom, AvailableUntil: r.quizUntil}
		} else {
			basesByItem[r.itemID] = assignmentoverrides.BaseDates{DueAt: r.dueAt, AvailableFrom: r.assignFrom, AvailableUntil: r.assignUntil}
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	effByItem := map[uuid.UUID]assignmentoverrides.Effective{}
	for courseID, itemIDs := range itemsByCourse {
		eid, ok := enrollmentByCourse[courseID]
		if !ok {
			// Not a student in this course (e.g. staff calendar): show base dates unfiltered.
			for _, id := range itemIDs {
				effByItem[id] = assignmentoverrides.Effective{Visible: true, DueAt: basesByItem[id].DueAt, AvailableFrom: basesByItem[id].AvailableFrom, AvailableUntil: basesByItem[id].AvailableUntil}
			}
			continue
		}
		eff, err := assignmentoverrides.EffectiveForStudentBatch(ctx, pool, eid, itemIDs, basesByItem)
		if err != nil {
			return nil, err
		}
		for id, e := range eff {
			effByItem[id] = e
		}
	}

	var out []Event
	for _, r := range raw {
		eff := effByItem[r.itemID]
		if !eff.Visible {
			continue
		}
		dueAt, assignFrom, assignUntil := eff.DueAt, eff.AvailableFrom, eff.AvailableUntil
		quizFrom, quizUntil := eff.AvailableFrom, eff.AvailableUntil

		if r.kind == "quiz" && (quizFrom != nil || quizUntil != nil) {
			start, end, allDay := quizWindowTimes(quizFrom, quizUntil, dueAt)
			if start.IsZero() {
				continue
			}
			ev := Event{
				ItemID: r.itemID, CourseID: r.courseID, CourseCode: r.courseCode, CourseTitle: r.courseTitle,
				Kind: r.kind, Title: r.title, Description: "Quiz window.", Start: start, End: end, AllDay: allDay,
				IsQuizWindow: true,
			}
			if inRange(ev.Start, rangeStart, rangeEnd) {
				out = append(out, ev)
			}
			continue
		}

		start, end, allDay := assignmentTimes(dueAt, assignFrom, assignUntil)
		if start.IsZero() {
			continue
		}
		desc := truncateDescription(strings.TrimSpace(r.description), 500)
		ev := Event{
			ItemID: r.itemID, CourseID: r.courseID, CourseCode: r.courseCode, CourseTitle: r.courseTitle,
			Kind: r.kind, Title: r.title, Description: desc, Start: start, End: end, AllDay: allDay,
		}
		if inRange(ev.Start, rangeStart, rangeEnd) {
			out = append(out, ev)
		}
	}
	return out, nil
}

// enrollmentIDsByCourse returns userID's active student-equivalent enrollment id per course,
// for the subset of courseIDs where one exists.
func enrollmentIDsByCourse(ctx context.Context, pool *pgxpool.Pool, courseIDs []uuid.UUID, userID uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	rows, err := pool.Query(ctx, `
SELECT ce.course_id, ce.id
FROM course.course_enrollments ce
INNER JOIN course.enrollment_roles er ON er.role_key = ce.role AND er.is_student_equivalent = true
WHERE ce.course_id = ANY($1) AND ce.user_id = $2 AND ce.active
`, courseIDs, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[uuid.UUID]uuid.UUID{}
	for rows.Next() {
		var courseID, enrollmentID uuid.UUID
		if err := rows.Scan(&courseID, &enrollmentID); err != nil {
			return nil, err
		}
		out[courseID] = enrollmentID
	}
	return out, rows.Err()
}

func assignmentTimes(dueAt, availableFrom, availableUntil *time.Time) (start, end time.Time, allDay bool) {
	if availableFrom != nil && availableUntil != nil {
		return *availableFrom, *availableUntil, isMidnightUTC(*availableFrom) && isMidnightUTC(*availableUntil)
	}
	if dueAt == nil {
		return time.Time{}, time.Time{}, false
	}
	d := dueAt.UTC()
	if isMidnightUTC(d) {
		endDay := d.AddDate(0, 0, 1)
		return d, endDay, true
	}
	return d, d.Add(time.Hour), false
}

func quizWindowTimes(availableFrom, availableUntil, dueAt *time.Time) (start, end time.Time, allDay bool) {
	if availableFrom != nil {
		start = *availableFrom
	}
	if availableUntil != nil {
		end = *availableUntil
	} else if dueAt != nil {
		end = *dueAt
	}
	if start.IsZero() && !end.IsZero() {
		start = end.Add(-time.Hour)
	}
	if !start.IsZero() && end.IsZero() {
		end = start.Add(time.Hour)
	}
	allDay = !start.IsZero() && !end.IsZero() && isMidnightUTC(start) && isMidnightUTC(end)
	return start, end, allDay
}

func isMidnightUTC(t time.Time) bool {
	u := t.UTC()
	return u.Hour() == 0 && u.Minute() == 0 && u.Second() == 0
}

func inRange(start time.Time, rangeStart, rangeEnd *time.Time) bool {
	if rangeStart != nil && start.Before(*rangeStart) {
		return false
	}
	if rangeEnd != nil && start.After(*rangeEnd) {
		return false
	}
	return true
}

func truncateDescription(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
