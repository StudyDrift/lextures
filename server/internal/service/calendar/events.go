package calendar

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Event is one assignment or quiz deadline surfaced in an iCal feed.
type Event struct {
	ItemID      uuid.UUID
	CourseID    uuid.UUID
	CourseCode  string
	CourseTitle string
	Kind        string
	Title       string
	Description string
	Start       time.Time
	End         time.Time
	AllDay      bool
	IsQuizWindow bool
}

// LoadEventsForCourses returns visible assignment/quiz calendar events for the given courses.
func LoadEventsForCourses(ctx context.Context, pool *pgxpool.Pool, courseIDs []uuid.UUID, rangeStart, rangeEnd *time.Time) ([]Event, error) {
	if len(courseIDs) == 0 {
		return nil, nil
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
	defer rows.Close()

	var out []Event
	for rows.Next() {
		var (
			courseID                      uuid.UUID
			courseCode, courseTitle       string
			itemID                        uuid.UUID
			kind, title, description      string
			dueAt                         *time.Time
			assignFrom, assignUntil       *time.Time
			quizFrom, quizUntil           *time.Time
		)
		if err := rows.Scan(
			&courseID, &courseCode, &courseTitle, &itemID, &kind, &title, &description,
			&dueAt, &assignFrom, &assignUntil, &quizFrom, &quizUntil,
		); err != nil {
			return nil, err
		}

		if kind == "quiz" && (quizFrom != nil || quizUntil != nil) {
			start, end, allDay := quizWindowTimes(quizFrom, quizUntil, dueAt)
			if start.IsZero() {
				continue
			}
			ev := Event{
				ItemID: itemID, CourseID: courseID, CourseCode: courseCode, CourseTitle: courseTitle,
				Kind: kind, Title: title, Description: "Quiz window.", Start: start, End: end, AllDay: allDay,
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
		desc := truncateDescription(strings.TrimSpace(description), 500)
		ev := Event{
			ItemID: itemID, CourseID: courseID, CourseCode: courseCode, CourseTitle: courseTitle,
			Kind: kind, Title: title, Description: desc, Start: start, End: end, AllDay: allDay,
		}
		if inRange(ev.Start, rangeStart, rangeEnd) {
			out = append(out, ev)
		}
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
