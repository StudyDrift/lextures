// Package conferences provides DB access for parent-teacher conference scheduling (plan 13.12).
package conferences

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Availability is a row in conference.conference_availability.
type Availability struct {
	ID           string    `json:"id"`
	TeacherID    string    `json:"teacherId"`
	SchoolID     string    `json:"schoolId"`
	Date         string    `json:"date"`
	SlotDuration int       `json:"slotDuration"`
	GapDuration  int       `json:"gapDuration"`
	WindowStart  string    `json:"windowStart"`
	WindowEnd    string    `json:"windowEnd"`
	Location     *string   `json:"location,omitempty"`
	VideoLink    *string   `json:"videoLink,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

// Slot is a row in conference.conference_slots.
type Slot struct {
	ID              string     `json:"id"`
	AvailabilityID  string     `json:"availabilityId"`
	StartAt         time.Time  `json:"startAt"`
	EndAt           time.Time  `json:"endAt"`
	Status          string     `json:"status"`
	BookedByParent  *string    `json:"bookedByParent,omitempty"`
	BookedForChild  *string    `json:"bookedForChild,omitempty"`
	BookedAt        *time.Time `json:"bookedAt,omitempty"`
	ReminderSentAt  *time.Time `json:"reminderSentAt,omitempty"`
}

// TeacherSummary is a teacher linked to a student via section enrollment.
type TeacherSummary struct {
	TeacherID   string  `json:"teacherId"`
	DisplayName *string `json:"displayName,omitempty"`
}

// ScheduleEntry combines slot + teacher metadata for admin grid views.
type ScheduleEntry struct {
	Slot
	TeacherID          string  `json:"teacherId"`
	TeacherDisplayName *string `json:"teacherDisplayName,omitempty"`
	Location           *string `json:"location,omitempty"`
	VideoLink          *string `json:"videoLink,omitempty"`
	ChildDisplayName   *string `json:"childDisplayName,omitempty"`
}

var (
	ErrAlreadyBooked      = errors.New("slot already booked")
	ErrNotBookedByParent  = errors.New("slot not booked by this parent")
	ErrTeacherNotLinked   = errors.New("teacher not linked to student")
)

// AllowedSlotDurations are valid slot lengths in minutes (FR-1).
var AllowedSlotDurations = map[int]bool{5: true, 10: true, 15: true, 20: true, 30: true}

// GenerateSlotTimes computes bookable start times from a window (exported for unit tests).
func GenerateSlotTimes(windowStart, windowEnd time.Time, slotDuration, gapDuration int) []time.Time {
	if slotDuration <= 0 || !windowEnd.After(windowStart) {
		return nil
	}
	slotLen := time.Duration(slotDuration) * time.Minute
	gapLen := time.Duration(gapDuration) * time.Minute
	step := slotLen + gapLen

	var out []time.Time
	for cursor := windowStart; cursor.Add(slotLen).Compare(windowEnd) <= 0; cursor = cursor.Add(step) {
		out = append(out, cursor)
	}
	return out
}

func parseLocalTime(date string, clock string) (time.Time, error) {
	t, err := time.Parse("15:04:05", clock)
	if err != nil {
		t, err = time.Parse("15:04", clock)
		if err != nil {
			return time.Time{}, err
		}
	}
	d, err := time.Parse("2006-01-02", date)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(d.Year(), d.Month(), d.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.UTC), nil
}

// CreateAvailability inserts availability and generates slots for the given date.
func CreateAvailability(
	ctx context.Context, pool *pgxpool.Pool,
	teacherID, schoolID uuid.UUID,
	date, windowStart, windowEnd string,
	slotDuration, gapDuration int,
	location, videoLink *string,
) (*Availability, []*Slot, error) {
	if !AllowedSlotDurations[slotDuration] {
		return nil, nil, errors.New("invalid slot duration")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	av := &Availability{}
	err = tx.QueryRow(ctx, `
		INSERT INTO conference.conference_availability
		  (teacher_id, school_id, date, slot_duration, gap_duration, window_start, window_end, location, video_link)
		VALUES ($1, $2, $3::date, $4, $5, $6::time, $7::time, $8, $9)
		RETURNING id, teacher_id, school_id, date::text, slot_duration, gap_duration,
		          window_start::text, window_end::text, location, video_link, created_at
	`, teacherID, schoolID, date, slotDuration, gapDuration, windowStart, windowEnd, location, videoLink,
	).Scan(
		&av.ID, &av.TeacherID, &av.SchoolID, &av.Date,
		&av.SlotDuration, &av.GapDuration,
		&av.WindowStart, &av.WindowEnd,
		&av.Location, &av.VideoLink, &av.CreatedAt,
	)
	if err != nil {
		return nil, nil, err
	}

	ws, err := parseLocalTime(date, windowStart)
	if err != nil {
		return nil, nil, err
	}
	we, err := parseLocalTime(date, windowEnd)
	if err != nil {
		return nil, nil, err
	}

	avUUID, _ := uuid.Parse(av.ID)
	slots, err := insertSlots(ctx, tx, avUUID, ws, we, slotDuration, gapDuration)
	if err != nil {
		return nil, nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}
	return av, slots, nil
}

func insertSlots(ctx context.Context, tx pgx.Tx, availabilityID uuid.UUID, ws, we time.Time, slotDuration, gapDuration int) ([]*Slot, error) {
	starts := GenerateSlotTimes(ws, we, slotDuration, gapDuration)
	slotLen := time.Duration(slotDuration) * time.Minute
	var out []*Slot
	for _, start := range starts {
		end := start.Add(slotLen)
		s := &Slot{}
		err := tx.QueryRow(ctx, `
			INSERT INTO conference.conference_slots (availability_id, start_at, end_at)
			VALUES ($1, $2, $3)
			ON CONFLICT (availability_id, start_at) DO NOTHING
			RETURNING id, availability_id, start_at, end_at, status,
			          booked_by_parent, booked_for_child, booked_at, reminder_sent_at
		`, availabilityID, start, end).Scan(
			&s.ID, &s.AvailabilityID, &s.StartAt, &s.EndAt, &s.Status,
			&s.BookedByParent, &s.BookedForChild, &s.BookedAt, &s.ReminderSentAt,
		)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		if err == nil {
			out = append(out, s)
		}
	}
	return out, nil
}

// ListSlotsByTeacherDate returns slots for a teacher on a given date.
func ListSlotsByTeacherDate(ctx context.Context, pool *pgxpool.Pool, teacherID uuid.UUID, date string) ([]*Slot, *Availability, error) {
	rows, err := pool.Query(ctx, `
		SELECT s.id, s.availability_id, s.start_at, s.end_at, s.status,
		       s.booked_by_parent, s.booked_for_child, s.booked_at, s.reminder_sent_at,
		       a.id, a.teacher_id, a.school_id, a.date::text, a.slot_duration, a.gap_duration,
		       a.window_start::text, a.window_end::text, a.location, a.video_link, a.created_at
		FROM conference.conference_slots s
		JOIN conference.conference_availability a ON a.id = s.availability_id
		WHERE a.teacher_id = $1 AND a.date = $2::date
		ORDER BY s.start_at ASC
	`, teacherID, date)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var slots []*Slot
	var av *Availability
	for rows.Next() {
		s := &Slot{}
		a := &Availability{}
		if err := rows.Scan(
			&s.ID, &s.AvailabilityID, &s.StartAt, &s.EndAt, &s.Status,
			&s.BookedByParent, &s.BookedForChild, &s.BookedAt, &s.ReminderSentAt,
			&a.ID, &a.TeacherID, &a.SchoolID, &a.Date, &a.SlotDuration, &a.GapDuration,
			&a.WindowStart, &a.WindowEnd, &a.Location, &a.VideoLink, &a.CreatedAt,
		); err != nil {
			return nil, nil, err
		}
		slots = append(slots, s)
		if av == nil {
			av = a
		}
	}
	return slots, av, rows.Err()
}

// ListTeachersForStudent returns distinct section instructors for a student.
func ListTeachersForStudent(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID) ([]TeacherSummary, error) {
	rows, err := pool.Query(ctx, `
		SELECT DISTINCT u.id, u.display_name
		FROM course.course_enrollments ce
		JOIN course.course_sections cs
		  ON (ce.section_id = cs.id OR (ce.section_id IS NULL AND ce.course_id = cs.course_id))
		JOIN "user".users u ON u.id = cs.instructor_user_id
		WHERE ce.user_id = $1 AND ce.active AND cs.instructor_user_id IS NOT NULL
		ORDER BY u.display_name NULLS LAST, u.id
	`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TeacherSummary
	for rows.Next() {
		var t TeacherSummary
		if err := rows.Scan(&t.TeacherID, &t.DisplayName); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// TeacherTeachesStudent verifies a teacher instructs a student via section enrollment.
func TeacherTeachesStudent(ctx context.Context, pool *pgxpool.Pool, teacherID, studentID uuid.UUID) (bool, error) {
	var ok bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM course.course_enrollments ce
			JOIN course.course_sections cs
			  ON (ce.section_id = cs.id OR (ce.section_id IS NULL AND ce.course_id = cs.course_id))
			WHERE ce.user_id = $1 AND ce.active AND cs.instructor_user_id = $2
		)
	`, studentID, teacherID).Scan(&ok)
	return ok, err
}

// GetSlotByID returns a slot with its availability metadata.
func GetSlotByID(ctx context.Context, pool *pgxpool.Pool, slotID uuid.UUID) (*Slot, *Availability, error) {
	s := &Slot{}
	a := &Availability{}
	err := pool.QueryRow(ctx, `
		SELECT s.id, s.availability_id, s.start_at, s.end_at, s.status,
		       s.booked_by_parent, s.booked_for_child, s.booked_at, s.reminder_sent_at,
		       a.id, a.teacher_id, a.school_id, a.date::text, a.slot_duration, a.gap_duration,
		       a.window_start::text, a.window_end::text, a.location, a.video_link, a.created_at
		FROM conference.conference_slots s
		JOIN conference.conference_availability a ON a.id = s.availability_id
		WHERE s.id = $1
	`, slotID).Scan(
		&s.ID, &s.AvailabilityID, &s.StartAt, &s.EndAt, &s.Status,
		&s.BookedByParent, &s.BookedForChild, &s.BookedAt, &s.ReminderSentAt,
		&a.ID, &a.TeacherID, &a.SchoolID, &a.Date, &a.SlotDuration, &a.GapDuration,
		&a.WindowStart, &a.WindowEnd, &a.Location, &a.VideoLink, &a.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil
	}
	return s, a, err
}

// BookSlot atomically books a slot for a parent and child.
func BookSlot(ctx context.Context, pool *pgxpool.Pool, slotID, parentID, childID uuid.UUID) (*Slot, *Availability, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var status string
	var teacherID uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT s.status, a.teacher_id
		FROM conference.conference_slots s
		JOIN conference.conference_availability a ON a.id = s.availability_id
		WHERE s.id = $1
		FOR UPDATE OF s
	`, slotID).Scan(&status, &teacherID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	if status != "open" {
		return nil, nil, ErrAlreadyBooked
	}

	linked, err := TeacherTeachesStudent(ctx, pool, teacherID, childID)
	if err != nil {
		return nil, nil, err
	}
	if !linked {
		return nil, nil, ErrTeacherNotLinked
	}

	s := &Slot{}
	a := &Availability{}
	err = tx.QueryRow(ctx, `
		UPDATE conference.conference_slots s
		SET status = 'booked',
		    booked_by_parent = $2,
		    booked_for_child = $3,
		    booked_at = now()
		FROM conference.conference_availability a
		WHERE s.id = $1 AND a.id = s.availability_id
		RETURNING s.id, s.availability_id, s.start_at, s.end_at, s.status,
		          s.booked_by_parent, s.booked_for_child, s.booked_at, s.reminder_sent_at,
		          a.id, a.teacher_id, a.school_id, a.date::text, a.slot_duration, a.gap_duration,
		          a.window_start::text, a.window_end::text, a.location, a.video_link, a.created_at
	`, slotID, parentID, childID).Scan(
		&s.ID, &s.AvailabilityID, &s.StartAt, &s.EndAt, &s.Status,
		&s.BookedByParent, &s.BookedForChild, &s.BookedAt, &s.ReminderSentAt,
		&a.ID, &a.TeacherID, &a.SchoolID, &a.Date, &a.SlotDuration, &a.GapDuration,
		&a.WindowStart, &a.WindowEnd, &a.Location, &a.VideoLink, &a.CreatedAt,
	)
	if err != nil {
		return nil, nil, err
	}
	return s, a, tx.Commit(ctx)
}

// CancelBooking frees a slot booked by the given parent.
func CancelBooking(ctx context.Context, pool *pgxpool.Pool, slotID, parentID uuid.UUID) (*Slot, *Availability, error) {
	s := &Slot{}
	a := &Availability{}
	err := pool.QueryRow(ctx, `
		UPDATE conference.conference_slots s
		SET status = 'open',
		    booked_by_parent = NULL,
		    booked_for_child = NULL,
		    booked_at = NULL,
		    reminder_sent_at = NULL
		FROM conference.conference_availability a
		WHERE s.id = $1 AND s.booked_by_parent = $2 AND s.status = 'booked'
		  AND a.id = s.availability_id
		RETURNING s.id, s.availability_id, s.start_at, s.end_at, s.status,
		          s.booked_by_parent, s.booked_for_child, s.booked_at, s.reminder_sent_at,
		          a.id, a.teacher_id, a.school_id, a.date::text, a.slot_duration, a.gap_duration,
		          a.window_start::text, a.window_end::text, a.location, a.video_link, a.created_at
	`, slotID, parentID).Scan(
		&s.ID, &s.AvailabilityID, &s.StartAt, &s.EndAt, &s.Status,
		&s.BookedByParent, &s.BookedForChild, &s.BookedAt, &s.ReminderSentAt,
		&a.ID, &a.TeacherID, &a.SchoolID, &a.Date, &a.SlotDuration, &a.GapDuration,
		&a.WindowStart, &a.WindowEnd, &a.Location, &a.VideoLink, &a.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrNotBookedByParent
	}
	return s, a, err
}

// ListSchoolSchedule returns all slots for a school on a date (admin grid).
func ListSchoolSchedule(ctx context.Context, pool *pgxpool.Pool, schoolID uuid.UUID, date string) ([]ScheduleEntry, error) {
	rows, err := pool.Query(ctx, `
		SELECT s.id, s.availability_id, s.start_at, s.end_at, s.status,
		       s.booked_by_parent, s.booked_for_child, s.booked_at, s.reminder_sent_at,
		       a.teacher_id, tu.display_name, a.location, a.video_link, cu.display_name
		FROM conference.conference_slots s
		JOIN conference.conference_availability a ON a.id = s.availability_id
		JOIN "user".users tu ON tu.id = a.teacher_id
		LEFT JOIN "user".users cu ON cu.id = s.booked_for_child
		WHERE a.school_id = $1 AND a.date = $2::date
		ORDER BY tu.display_name NULLS LAST, s.start_at ASC
	`, schoolID, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ScheduleEntry
	for rows.Next() {
		var e ScheduleEntry
		if err := rows.Scan(
			&e.ID, &e.AvailabilityID, &e.StartAt, &e.EndAt, &e.Status,
			&e.BookedByParent, &e.BookedForChild, &e.BookedAt, &e.ReminderSentAt,
			&e.TeacherID, &e.TeacherDisplayName, &e.Location, &e.VideoLink, &e.ChildDisplayName,
		); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ReminderCandidate is a booked slot needing a 24h reminder.
type ReminderCandidate struct {
	SlotID    uuid.UUID
	ParentID  uuid.UUID
	TeacherID uuid.UUID
	ChildName string
	StartAt   time.Time
	EndAt     time.Time
	Location  string
	VideoLink string
	Teacher   string
}

// ListDueReminders returns booked slots starting in ~24 hours without reminder_sent_at.
func ListDueReminders(ctx context.Context, pool *pgxpool.Pool, now time.Time) ([]ReminderCandidate, error) {
	from := now.Add(23 * time.Hour)
	to := now.Add(25 * time.Hour)
	rows, err := pool.Query(ctx, `
		SELECT s.id, s.booked_by_parent, a.teacher_id,
		       COALESCE(cu.display_name, 'Student'), s.start_at, s.end_at,
		       COALESCE(a.location, ''), COALESCE(a.video_link, ''),
		       COALESCE(tu.display_name, 'Teacher')
		FROM conference.conference_slots s
		JOIN conference.conference_availability a ON a.id = s.availability_id
		JOIN "user".users tu ON tu.id = a.teacher_id
		LEFT JOIN "user".users cu ON cu.id = s.booked_for_child
		WHERE s.status = 'booked'
		  AND s.reminder_sent_at IS NULL
		  AND s.start_at >= $1 AND s.start_at <= $2
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ReminderCandidate
	for rows.Next() {
		var c ReminderCandidate
		if err := rows.Scan(
			&c.SlotID, &c.ParentID, &c.TeacherID,
			&c.ChildName, &c.StartAt, &c.EndAt,
			&c.Location, &c.VideoLink, &c.Teacher,
		); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// MarkReminderSent records that a reminder email was sent for a slot.
func MarkReminderSent(ctx context.Context, pool *pgxpool.Pool, slotID uuid.UUID, sentAt time.Time) error {
	_, err := pool.Exec(ctx, `
		UPDATE conference.conference_slots SET reminder_sent_at = $2 WHERE id = $1
	`, slotID, sentAt)
	return err
}
