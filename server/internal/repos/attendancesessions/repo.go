// Package attendancesessions persists course-level attendance sessions (roll call / self report).
package attendancesessions

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	CollectionRollCall   = "roll_call"
	CollectionSelfReport = "self_report"
	StatusOpen           = "open"
	StatusClosed         = "closed"
)

var (
	ErrSessionNotFound      = errors.New("attendance session not found")
	ErrSessionClosed        = errors.New("attendance session is closed")
	ErrSelfReportClosed     = errors.New("self-report window is closed")
	ErrAlreadySubmitted     = errors.New("attendance already submitted")
	ErrInvalidStatus        = errors.New("invalid attendance status")
	ErrInvalidCollection    = errors.New("invalid collection method")
	ErrSectionNotInCourse   = errors.New("section not in course")
)

// Session is one attendance event for a course.
type Session struct {
	ID               uuid.UUID
	CourseID         uuid.UUID
	SectionID        *uuid.UUID
	StructureItemID  *uuid.UUID
	Title            string
	CollectionMethod string
	SessionDate      time.Time
	OpensAt          *time.Time
	ClosesAt         *time.Time
	Status           string
	GradebookEnabled bool
	PointsPossible   *int
	TardyPointsRatio float64
	CreatedBy        uuid.UUID
	CreatedAt        time.Time
	ClosedAt         *time.Time
	UpdatedAt        time.Time
}

// RecordRow is one student's attendance for a session.
type RecordRow struct {
	StudentUserID uuid.UUID
	DisplayName   string
	Status        string
	Source        string
	RecordedBy    *uuid.UUID
	RecordedAt    time.Time
}

// CreateInput is input for CreateSession.
type CreateInput struct {
	CourseID         uuid.UUID
	SectionID        *uuid.UUID
	Title            string
	CollectionMethod string
	SessionDate      time.Time
	OpensAt          *time.Time
	ClosesAt         *time.Time
	GradebookEnabled bool
	PointsPossible   *int
	CreatedBy        uuid.UUID
}

// RecordUpsert is one row for BatchUpsertRecords.
type RecordUpsert struct {
	StudentUserID uuid.UUID
	Status        string
	Source        string
	RecordedBy    uuid.UUID
}

func validStatus(s string) bool {
	switch s {
	case "present", "absent", "tardy", "excused", "not_recorded":
		return true
	default:
		return false
	}
}

func validSelfReportStatus(s string) bool {
	return s == "present" || s == "tardy"
}

// IsSelfReportWindowOpen reports whether students may self-report now.
func IsSelfReportWindowOpen(sess Session, now time.Time) bool {
	if sess.CollectionMethod != CollectionSelfReport || sess.Status != StatusOpen {
		return false
	}
	if sess.OpensAt != nil && now.Before(*sess.OpensAt) {
		return false
	}
	if sess.ClosesAt != nil && now.After(*sess.ClosesAt) {
		return false
	}
	return true
}

// CreateSession inserts a new attendance session.
func CreateSession(ctx context.Context, pool *pgxpool.Pool, in CreateInput) (*Session, error) {
	if in.CollectionMethod != CollectionRollCall && in.CollectionMethod != CollectionSelfReport {
		return nil, ErrInvalidCollection
	}
	points := in.PointsPossible
	if in.GradebookEnabled && (points == nil || *points <= 0) {
		one := 1
		points = &one
	}
	if !in.GradebookEnabled {
		points = nil
	}
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO course.attendance_sessions (
    course_id, section_id, title, collection_method, session_date,
    opens_at, closes_at, gradebook_enabled, points_possible, created_by
)
VALUES ($1, $2, $3, $4, $5::date, $6, $7, $8, $9, $10)
RETURNING id
`, in.CourseID, in.SectionID, in.Title, in.CollectionMethod,
		in.SessionDate.Format("2006-01-02"), in.OpensAt, in.ClosesAt,
		in.GradebookEnabled, points, in.CreatedBy).Scan(&id)
	if err != nil {
		return nil, err
	}
	return GetSession(ctx, pool, in.CourseID, id)
}

// ListSessions returns sessions for a course ordered by date desc.
func ListSessions(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, limit int) ([]Session, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := pool.Query(ctx, `
SELECT id, course_id, section_id, structure_item_id, title, collection_method, session_date,
       opens_at, closes_at, status, gradebook_enabled, points_possible, tardy_points_ratio,
       created_by, created_at, closed_at, updated_at
FROM course.attendance_sessions
WHERE course_id = $1
ORDER BY session_date DESC, created_at DESC
LIMIT $2
`, courseID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSessions(rows)
}

// GetSession loads a session by course and id.
func GetSession(ctx context.Context, pool *pgxpool.Pool, courseID, sessionID uuid.UUID) (*Session, error) {
	row := pool.QueryRow(ctx, `
SELECT id, course_id, section_id, structure_item_id, title, collection_method, session_date,
       opens_at, closes_at, status, gradebook_enabled, points_possible, tardy_points_ratio,
       created_by, created_at, closed_at, updated_at
FROM course.attendance_sessions
WHERE course_id = $1 AND id = $2
`, courseID, sessionID)
	s, err := scanSession(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSessionNotFound
	}
	return s, err
}

// ListRecordsForSession returns stored records keyed by student.
func ListRecordsForSession(ctx context.Context, pool *pgxpool.Pool, sessionID uuid.UUID) (map[uuid.UUID]RecordRow, error) {
	rows, err := pool.Query(ctx, `
SELECT r.student_user_id,
       COALESCE(NULLIF(TRIM(u.display_name), ''), u.email) AS display_label,
       r.status, r.source, r.recorded_by, r.recorded_at
FROM course.attendance_session_records r
JOIN "user".users u ON u.id = r.student_user_id
WHERE r.session_id = $1
`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[uuid.UUID]RecordRow)
	for rows.Next() {
		var rec RecordRow
		var recordedBy sql.NullString
		if err := rows.Scan(&rec.StudentUserID, &rec.DisplayName, &rec.Status, &rec.Source, &recordedBy, &rec.RecordedAt); err != nil {
			return nil, err
		}
		if recordedBy.Valid && recordedBy.String != "" {
			u, err := uuid.Parse(recordedBy.String)
			if err == nil {
				rec.RecordedBy = &u
			}
		}
		out[rec.StudentUserID] = rec
	}
	return out, rows.Err()
}

// BatchUpsertRecords upserts attendance rows for a session.
func BatchUpsertRecords(ctx context.Context, pool *pgxpool.Pool, sessionID uuid.UUID, rows []RecordUpsert, allowClosed bool) error {
	sess, err := getSessionByID(ctx, pool, sessionID)
	if err != nil {
		return err
	}
	if sess.Status == StatusClosed && !allowClosed {
		return ErrSessionClosed
	}
	if len(rows) == 0 {
		return nil
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, r := range rows {
		if !validStatus(r.Status) {
			return ErrInvalidStatus
		}
		_, err := tx.Exec(ctx, `
INSERT INTO course.attendance_session_records
    (session_id, student_user_id, status, source, recorded_by, recorded_at, updated_at)
VALUES ($1, $2, $3, $4, $5, now(), now())
ON CONFLICT (session_id, student_user_id) DO UPDATE SET
    status = EXCLUDED.status,
    source = EXCLUDED.source,
    recorded_by = EXCLUDED.recorded_by,
    updated_at = now()
`, sessionID, r.StudentUserID, r.Status, r.Source, r.RecordedBy)
		if err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	if sess.GradebookEnabled && sess.Status == StatusClosed {
		return SyncGradebook(ctx, pool, sess.ID)
	}
	return nil
}

// SelfReport lets a student check in during an open self-report window.
func SelfReport(ctx context.Context, pool *pgxpool.Pool, sessionID, studentID uuid.UUID, status string) error {
	if !validSelfReportStatus(status) {
		return ErrInvalidStatus
	}
	sess, err := getSessionByID(ctx, pool, sessionID)
	if err != nil {
		return err
	}
	if !IsSelfReportWindowOpen(*sess, time.Now().UTC()) {
		return ErrSelfReportClosed
	}
	var existing string
	err = pool.QueryRow(ctx, `
SELECT status FROM course.attendance_session_records
WHERE session_id = $1 AND student_user_id = $2
`, sessionID, studentID).Scan(&existing)
	if err == nil && existing != "not_recorded" {
		return ErrAlreadySubmitted
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	tag, err := pool.Exec(ctx, `
INSERT INTO course.attendance_session_records
    (session_id, student_user_id, status, source, recorded_by, recorded_at, updated_at)
VALUES ($1, $2, $3, 'self', $2, now(), now())
ON CONFLICT (session_id, student_user_id) DO UPDATE SET
    status = EXCLUDED.status,
    source = EXCLUDED.source,
    recorded_by = EXCLUDED.recorded_by,
    updated_at = now()
WHERE course.attendance_session_records.status = 'not_recorded'
`, sessionID, studentID, status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAlreadySubmitted
	}
	return nil
}

// CloseSession marks a session closed and optionally syncs gradebook.
func CloseSession(ctx context.Context, pool *pgxpool.Pool, courseID, sessionID uuid.UUID, finalizeMissingAsAbsent bool) (*Session, error) {
	sess, err := GetSession(ctx, pool, courseID, sessionID)
	if err != nil {
		return nil, err
	}
	if sess.Status == StatusClosed {
		return sess, nil
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if finalizeMissingAsAbsent {
		if err := markMissingAsAbsent(ctx, tx, sess); err != nil {
			return nil, err
		}
	}

	if sess.GradebookEnabled {
		itemID, err := ensureStructureItem(ctx, tx, sess)
		if err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx, `
UPDATE course.attendance_sessions
SET structure_item_id = $2, updated_at = now()
WHERE id = $1 AND structure_item_id IS NULL
`, sessionID, itemID); err != nil {
			return nil, err
		}
		sess.StructureItemID = &itemID
	}

	if _, err := tx.Exec(ctx, `
UPDATE course.attendance_sessions
SET status = 'closed', closed_at = now(), updated_at = now()
WHERE id = $1
`, sessionID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	if sess.GradebookEnabled {
		if err := SyncGradebook(ctx, pool, sessionID); err != nil {
			return nil, err
		}
	}
	return GetSession(ctx, pool, courseID, sessionID)
}

// SyncGradebook writes course_grades from session records.
func SyncGradebook(ctx context.Context, pool *pgxpool.Pool, sessionID uuid.UUID) error {
	sess, err := getSessionByID(ctx, pool, sessionID)
	if err != nil {
		return err
	}
	if !sess.GradebookEnabled || sess.PointsPossible == nil || sess.StructureItemID == nil {
		return nil
	}
	records, err := ListRecordsForSession(ctx, pool, sessionID)
	if err != nil {
		return err
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, rec := range records {
		pts := PointsForStatus(rec.Status, *sess.PointsPossible, sess.TardyPointsRatio)
		_, err := tx.Exec(ctx, `
INSERT INTO course.course_grades (course_id, student_user_id, module_item_id, points_earned, updated_at, posted_at)
VALUES ($1, $2, $3, $4, NOW(), NOW())
ON CONFLICT (student_user_id, module_item_id) DO UPDATE SET
    course_id = EXCLUDED.course_id,
    points_earned = EXCLUDED.points_earned,
    updated_at = NOW(),
    posted_at = COALESCE(course.course_grades.posted_at, NOW())
`, sess.CourseID, rec.StudentUserID, *sess.StructureItemID, pts)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// LoadPointsByStructureItemIDs returns max points for attendance structure items.
func LoadPointsByStructureItemIDs(ctx context.Context, pool *pgxpool.Pool, itemIDs []uuid.UUID) (map[uuid.UUID]int, error) {
	if len(itemIDs) == 0 {
		return map[uuid.UUID]int{}, nil
	}
	rows, err := pool.Query(ctx, `
SELECT structure_item_id, points_possible
FROM course.attendance_sessions
WHERE structure_item_id = ANY($1::uuid[]) AND points_possible IS NOT NULL
`, itemIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[uuid.UUID]int)
	for rows.Next() {
		var id uuid.UUID
		var pts int
		if err := rows.Scan(&id, &pts); err != nil {
			return nil, err
		}
		out[id] = pts
	}
	return out, rows.Err()
}

func ensureStructureItem(ctx context.Context, tx pgx.Tx, sess *Session) (uuid.UUID, error) {
	if sess.StructureItemID != nil {
		return *sess.StructureItemID, nil
	}
	itemID := uuid.New()
	err := tx.QueryRow(ctx, `
WITH mx AS (
    SELECT COALESCE(MAX(sort_order), -1) AS max_ord
    FROM course.course_structure_items
    WHERE course_id = $1 AND parent_id IS NULL
)
INSERT INTO course.course_structure_items (
    id, course_id, sort_order, kind, title, parent_id, published, archived
)
SELECT $2, $1, max_ord + 1, 'attendance', $3, NULL, true, false
FROM mx
RETURNING id
`, sess.CourseID, itemID, sess.Title).Scan(&itemID)
	return itemID, err
}

func markMissingAsAbsent(ctx context.Context, tx pgx.Tx, sess *Session) error {
	students, err := listStudentIDsForSession(ctx, tx, sess)
	if err != nil {
		return err
	}
	for _, sid := range students {
		_, err := tx.Exec(ctx, `
INSERT INTO course.attendance_session_records
    (session_id, student_user_id, status, source, recorded_at, updated_at)
VALUES ($1, $2, 'absent', 'instructor', now(), now())
ON CONFLICT (session_id, student_user_id) DO UPDATE SET
    status = CASE
        WHEN course.attendance_session_records.status = 'not_recorded' THEN 'absent'
        ELSE course.attendance_session_records.status
    END,
    updated_at = now()
`, sess.ID, sid)
		if err != nil {
			return err
		}
	}
	return nil
}

func listStudentIDsForSession(ctx context.Context, tx pgx.Tx, sess *Session) ([]uuid.UUID, error) {
	var rows pgx.Rows
	var err error
	if sess.SectionID != nil {
		rows, err = tx.Query(ctx, `
SELECT DISTINCT ce.user_id
FROM course.course_enrollments ce
INNER JOIN course.enrollment_roles er ON er.role_key = ce.role AND er.is_student_equivalent = true
WHERE ce.course_id = $1 AND ce.active AND ce.section_id = $2
`, sess.CourseID, *sess.SectionID)
	} else {
		rows, err = tx.Query(ctx, `
SELECT DISTINCT ce.user_id
FROM course.course_enrollments ce
INNER JOIN course.enrollment_roles er ON er.role_key = ce.role AND er.is_student_equivalent = true
WHERE ce.course_id = $1 AND ce.active
`, sess.CourseID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func getSessionByID(ctx context.Context, pool *pgxpool.Pool, sessionID uuid.UUID) (*Session, error) {
	row := pool.QueryRow(ctx, `
SELECT id, course_id, section_id, structure_item_id, title, collection_method, session_date,
       opens_at, closes_at, status, gradebook_enabled, points_possible, tardy_points_ratio,
       created_by, created_at, closed_at, updated_at
FROM course.attendance_sessions WHERE id = $1
`, sessionID)
	s, err := scanSession(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSessionNotFound
	}
	return s, err
}

func scanSessions(rows pgx.Rows) ([]Session, error) {
	var out []Session
	for rows.Next() {
		s, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func scanSession(row pgx.Row) (*Session, error) {
	var s Session
	var sectionID, structureID sql.NullString
	var opens, closes, closed sql.NullTime
	var points sql.NullInt32
	var tardyRatio float64
	if err := row.Scan(
		&s.ID, &s.CourseID, &sectionID, &structureID, &s.Title, &s.CollectionMethod, &s.SessionDate,
		&opens, &closes, &s.Status, &s.GradebookEnabled, &points, &tardyRatio,
		&s.CreatedBy, &s.CreatedAt, &closed, &s.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if sectionID.Valid && sectionID.String != "" {
		u, err := uuid.Parse(sectionID.String)
		if err == nil {
			s.SectionID = &u
		}
	}
	if structureID.Valid && structureID.String != "" {
		u, err := uuid.Parse(structureID.String)
		if err == nil {
			s.StructureItemID = &u
		}
	}
	if opens.Valid {
		t := opens.Time
		s.OpensAt = &t
	}
	if closes.Valid {
		t := closes.Time
		s.ClosesAt = &t
	}
	if closed.Valid {
		t := closed.Time
		s.ClosedAt = &t
	}
	if points.Valid {
		v := int(points.Int32)
		s.PointsPossible = &v
	}
	s.TardyPointsRatio = tardyRatio
	return &s, nil
}

// MergeRosterWithRecords builds full roster rows with defaults for missing records.
func MergeRosterWithRecords(
	roster []struct {
		UserID      uuid.UUID
		DisplayName string
	},
	records map[uuid.UUID]RecordRow,
) []RecordRow {
	out := make([]RecordRow, 0, len(roster))
	for _, s := range roster {
		if rec, ok := records[s.UserID]; ok {
			if rec.DisplayName == "" {
				rec.DisplayName = s.DisplayName
			}
			out = append(out, rec)
			continue
		}
		out = append(out, RecordRow{
			StudentUserID: s.UserID,
			DisplayName:   s.DisplayName,
			Status:        "not_recorded",
			Source:        "instructor",
		})
	}
	return out
}

// ValidateSectionInCourse ensures section belongs to course.
func ValidateSectionInCourse(ctx context.Context, pool *pgxpool.Pool, courseID, sectionID uuid.UUID) (bool, error) {
	var ok bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS(SELECT 1 FROM course.course_sections WHERE id = $1 AND course_id = $2)
`, sectionID, courseID).Scan(&ok)
	return ok, err
}

// PatchSession updates title or window times on an open session.
func PatchSession(ctx context.Context, pool *pgxpool.Pool, courseID, sessionID uuid.UUID, title *string, opensAt, closesAt *time.Time) (*Session, error) {
	sess, err := GetSession(ctx, pool, courseID, sessionID)
	if err != nil {
		return nil, err
	}
	if sess.Status != StatusOpen {
		return nil, ErrSessionClosed
	}
	newTitle := sess.Title
	if title != nil && strings.TrimSpace(*title) != "" {
		newTitle = strings.TrimSpace(*title)
	}
	newOpens := sess.OpensAt
	if opensAt != nil {
		newOpens = opensAt
	}
	newCloses := sess.ClosesAt
	if closesAt != nil {
		newCloses = closesAt
	}
	_, err = pool.Exec(ctx, `
UPDATE course.attendance_sessions
SET title = $3, opens_at = $4, closes_at = $5, updated_at = now()
WHERE course_id = $1 AND id = $2
`, courseID, sessionID, newTitle, newOpens, newCloses)
	if err != nil {
		return nil, err
	}
	return GetSession(ctx, pool, courseID, sessionID)
}

// AttendanceEnabledForCourseCode returns whether attendance is enabled by course code.
func AttendanceEnabledForCourseCode(ctx context.Context, pool *pgxpool.Pool, courseCode string) (bool, error) {
	var ok bool
	err := pool.QueryRow(ctx, `
SELECT attendance_enabled FROM course.courses WHERE course_code = $1
`, courseCode).Scan(&ok)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return ok, err
}

// DefaultSelfReportWindow returns opens=now, closes=now+15min.
func DefaultSelfReportWindow(now time.Time) (opens, closes time.Time) {
	opens = now.UTC()
	closes = opens.Add(15 * time.Minute)
	return opens, closes
}

// SessionSummary counts present/absent for list view.
type SessionSummary struct {
	Present int
	Absent  int
	Tardy   int
	Total   int
}

// SummarizeRecords counts statuses.
func SummarizeRecords(records []RecordRow) SessionSummary {
	var s SessionSummary
	s.Total = len(records)
	for _, r := range records {
		switch r.Status {
		case "present", "excused":
			s.Present++
		case "absent", "not_recorded":
			s.Absent++
		case "tardy":
			s.Tardy++
		}
	}
	return s
}

// FormatSessionDate returns YYYY-MM-DD.
func FormatSessionDate(t time.Time) string {
	return t.UTC().Format("2006-01-02")
}
