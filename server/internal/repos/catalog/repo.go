// Package catalog provides data access for HE course catalog sections and registrations (plan 14.2).
package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Section status values.
const (
	StatusActive    = "active"
	StatusCancelled = "cancelled"
	StatusPending   = "pending"
)

// Registration status values.
const (
	RegRegistered = "registered"
	RegWaitlisted = "waitlisted"
	RegAuditing   = "auditing"
	RegWithdrawn  = "withdrawn"
)

// SyncStatus values.
const (
	SyncStatusRunning = "running"
	SyncStatusSuccess = "success"
	SyncStatusPartial = "partial"
	SyncStatusFailed  = "failed"
)

// MeetingPattern holds days/time/instructor for a section.
type MeetingPattern struct {
	Days       string `json:"days,omitempty"`
	StartTime  string `json:"startTime,omitempty"`
	EndTime    string `json:"endTime,omitempty"`
	Instructor string `json:"instructor,omitempty"`
}

// Prerequisite describes one prerequisite course.
type Prerequisite struct {
	Code  string `json:"code"`
	Title string `json:"title,omitempty"`
}

// PrereqStatus is per-student prerequisite completion.
type PrereqStatus struct {
	Code   string `json:"code"`
	Status string `json:"status"` // met, not_met, waived
}

// Section is a catalog section row.
type Section struct {
	ID             uuid.UUID
	OrgID          uuid.UUID
	TermID         uuid.UUID
	SISCourseID    string
	SISSectionID   string
	CRN            *string
	Subject        string
	CourseNumber   string
	SectionNumber  *string
	Title          string
	Credits        *float64
	MeetingPattern *MeetingPattern
	Room           *string
	Department     *string
	Prerequisites  []Prerequisite
	InstructorName *string
	Status         string
	LMSCourseID    *uuid.UUID
	SyncedAt       *time.Time
}

// Registration is a student's registration for a catalog section.
type Registration struct {
	ID               uuid.UUID
	OrgID            uuid.UUID
	UserID           uuid.UUID
	CatalogSectionID uuid.UUID
	Status           string
	PrereqStatus     []PrereqStatus
	SyncedAt         *time.Time
}

// ListFilter filters catalog browse queries.
type ListFilter struct {
	TermID     *uuid.UUID
	Department *string
	Days       *string
	MinCredits *float64
	MaxCredits *float64
	Query      *string
	Cursor     *uuid.UUID
	Limit      int
}

// SyncLog is a catalog sync audit record.
type SyncLog struct {
	ID             uuid.UUID
	OrgID          uuid.UUID
	ConnectionID   *uuid.UUID
	StartedAt      time.Time
	FinishedAt     *time.Time
	Status         string
	SectionsSynced int
	ShellsCreated  int
	ShellsUpdated  int
	Errors         []SyncError
}

// SyncError is one error in a catalog sync log.
type SyncError struct {
	RecordID string `json:"record_id"`
	Message  string `json:"message"`
}

// UpsertSectionInput is used by the sync worker.
type UpsertSectionInput struct {
	TermID         uuid.UUID
	SISCourseID    string
	SISSectionID   string
	CRN            *string
	Subject        string
	CourseNumber   string
	SectionNumber  *string
	Title          string
	Credits        *float64
	MeetingPattern *MeetingPattern
	Room           *string
	Department     *string
	Prerequisites  []Prerequisite
	InstructorName *string
	Status         string
}

// ScheduleEntry combines section + registration for dashboard display.
type ScheduleEntry struct {
	Section      Section
	Registration Registration
	CourseCode   *string
	CourseTitle  *string
}

func scanSection(row pgx.Row) (*Section, error) {
	var s Section
	var crn, sectionNum, room, dept, instructor *string
	var credits *float64
	var meetingRaw, prereqRaw []byte
	var lmsCourseID *uuid.UUID
	err := row.Scan(
		&s.ID, &s.OrgID, &s.TermID, &s.SISCourseID, &s.SISSectionID,
		&crn, &s.Subject, &s.CourseNumber, &sectionNum, &s.Title, &credits,
		&meetingRaw, &room, &dept, &prereqRaw, &instructor, &s.Status,
		&lmsCourseID, &s.SyncedAt,
	)
	if err != nil {
		return nil, err
	}
	s.CRN = crn
	s.SectionNumber = sectionNum
	s.Credits = credits
	s.Room = room
	s.Department = dept
	s.InstructorName = instructor
	s.LMSCourseID = lmsCourseID
	if len(meetingRaw) > 0 {
		var mp MeetingPattern
		if json.Unmarshal(meetingRaw, &mp) == nil {
			s.MeetingPattern = &mp
		}
	}
	if len(prereqRaw) > 0 {
		_ = json.Unmarshal(prereqRaw, &s.Prerequisites)
	}
	return &s, nil
}

const sectionCols = `
	id, org_id, term_id, sis_course_id, sis_section_id,
	crn, subject, course_number, section_number, title, credits,
	meeting_pattern, room, department, prerequisites, instructor_name,
	status, lms_course_id, synced_at
`

// ListSections returns catalog sections for browse UI.
func ListSections(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, f ListFilter) ([]Section, error) {
	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	args := []any{orgID}
	where := []string{"org_id = $1", "status = 'active'"}
	n := 2
	if f.TermID != nil {
		where = append(where, fmt.Sprintf("term_id = $%d", n))
		args = append(args, *f.TermID)
		n++
	}
	if f.Department != nil && strings.TrimSpace(*f.Department) != "" {
		where = append(where, fmt.Sprintf("department ILIKE $%d", n))
		args = append(args, strings.TrimSpace(*f.Department))
		n++
	}
	if f.Days != nil && strings.TrimSpace(*f.Days) != "" {
		where = append(where, fmt.Sprintf("meeting_pattern->>'days' = $%d", n))
		args = append(args, strings.ToUpper(strings.TrimSpace(*f.Days)))
		n++
	}
	if f.MinCredits != nil {
		where = append(where, fmt.Sprintf("credits >= $%d", n))
		args = append(args, *f.MinCredits)
		n++
	}
	if f.MaxCredits != nil {
		where = append(where, fmt.Sprintf("credits <= $%d", n))
		args = append(args, *f.MaxCredits)
		n++
	}
	if f.Query != nil && strings.TrimSpace(*f.Query) != "" {
		where = append(where, fmt.Sprintf(
			"(title ILIKE $%d OR subject ILIKE $%d OR course_number ILIKE $%d OR crn ILIKE $%d)",
			n, n, n, n))
		q := "%" + strings.TrimSpace(*f.Query) + "%"
		args = append(args, q)
		n++
	}
	if f.Cursor != nil {
		where = append(where, fmt.Sprintf("id > $%d", n))
		args = append(args, *f.Cursor)
		n++
	}
	args = append(args, limit)
	q := fmt.Sprintf(`
SELECT %s FROM catalog.catalog_sections
WHERE %s
ORDER BY id ASC
LIMIT $%d
`, sectionCols, strings.Join(where, " AND "), n)

	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Section
	for rows.Next() {
		s, err := scanSection(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

// GetSection returns one section by id scoped to org.
func GetSection(ctx context.Context, pool *pgxpool.Pool, orgID, id uuid.UUID) (*Section, error) {
	row := pool.QueryRow(ctx, `
SELECT `+sectionCols+`
FROM catalog.catalog_sections
WHERE id = $1 AND org_id = $2
`, id, orgID)
	s, err := scanSection(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

// GetSectionByLMSCourseID returns catalog metadata for an LMS course.
func GetSectionByLMSCourseID(ctx context.Context, pool *pgxpool.Pool, orgID, courseID uuid.UUID) (*Section, error) {
	row := pool.QueryRow(ctx, `
SELECT `+sectionCols+`
FROM catalog.catalog_sections
WHERE lms_course_id = $1 AND org_id = $2
`, courseID, orgID)
	s, err := scanSection(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

// UpsertSection inserts or updates a catalog section from SIS sync.
func UpsertSection(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, in UpsertSectionInput) (*Section, error) {
	status := strings.TrimSpace(in.Status)
	if status == "" {
		status = StatusActive
	}
	meetingJSON, _ := json.Marshal(in.MeetingPattern)
	prereqJSON, _ := json.Marshal(in.Prerequisites)
	now := time.Now().UTC()
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO catalog.catalog_sections (
	org_id, term_id, sis_course_id, sis_section_id, crn, subject, course_number,
	section_number, title, credits, meeting_pattern, room, department,
	prerequisites, instructor_name, status, synced_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$17)
ON CONFLICT (org_id, sis_section_id) DO UPDATE SET
	term_id = EXCLUDED.term_id,
	crn = EXCLUDED.crn,
	subject = EXCLUDED.subject,
	course_number = EXCLUDED.course_number,
	section_number = EXCLUDED.section_number,
	title = EXCLUDED.title,
	credits = EXCLUDED.credits,
	meeting_pattern = EXCLUDED.meeting_pattern,
	room = EXCLUDED.room,
	department = EXCLUDED.department,
	prerequisites = EXCLUDED.prerequisites,
	instructor_name = EXCLUDED.instructor_name,
	status = EXCLUDED.status,
	synced_at = EXCLUDED.synced_at,
	updated_at = EXCLUDED.updated_at
RETURNING id
`, orgID, in.TermID, in.SISCourseID, in.SISSectionID, in.CRN, in.Subject, in.CourseNumber,
		in.SectionNumber, in.Title, in.Credits, meetingJSON, in.Room, in.Department,
		prereqJSON, in.InstructorName, status, now).Scan(&id)
	if err != nil {
		return nil, err
	}
	return GetSection(ctx, pool, orgID, id)
}

// LinkLMSShell sets lms_course_id on a catalog section.
func LinkLMSShell(ctx context.Context, pool *pgxpool.Pool, sectionID, courseID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
UPDATE catalog.catalog_sections SET lms_course_id = $2, updated_at = NOW() WHERE id = $1
`, sectionID, courseID)
	return err
}

// ListScheduleForUser returns enrolled catalog sections with registration status for dashboard.
func ListScheduleForUser(ctx context.Context, pool *pgxpool.Pool, orgID, userID uuid.UUID) ([]ScheduleEntry, error) {
	rows, err := pool.Query(ctx, `
SELECT
	cs.id, cs.org_id, cs.term_id, cs.sis_course_id, cs.sis_section_id,
	cs.crn, cs.subject, cs.course_number, cs.section_number, cs.title, cs.credits,
	cs.meeting_pattern, cs.room, cs.department, cs.prerequisites, cs.instructor_name,
	cs.status, cs.lms_course_id, cs.synced_at,
	sr.id, sr.user_id, sr.catalog_section_id, sr.status, sr.prereq_status, sr.synced_at,
	c.course_code, c.title
FROM catalog.student_registrations sr
JOIN catalog.catalog_sections cs ON cs.id = sr.catalog_section_id
LEFT JOIN course.courses c ON c.id = cs.lms_course_id
WHERE sr.user_id = $1 AND sr.org_id = $2 AND sr.status != 'withdrawn'
ORDER BY cs.subject, cs.course_number
`, userID, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ScheduleEntry
	for rows.Next() {
		var s Section
		var reg Registration
		var crn, sectionNum, room, dept, instructor *string
		var credits *float64
		var meetingRaw, prereqRaw, prereqStatusRaw []byte
		var lmsCourseID *uuid.UUID
		var courseCode, courseTitle *string
		if err := rows.Scan(
			&s.ID, &s.OrgID, &s.TermID, &s.SISCourseID, &s.SISSectionID,
			&crn, &s.Subject, &s.CourseNumber, &sectionNum, &s.Title, &credits,
			&meetingRaw, &room, &dept, &prereqRaw, &instructor, &s.Status,
			&lmsCourseID, &s.SyncedAt,
			&reg.ID, &reg.UserID, &reg.CatalogSectionID, &reg.Status, &prereqStatusRaw, &reg.SyncedAt,
			&courseCode, &courseTitle,
		); err != nil {
			return nil, err
		}
		s.CRN = crn
		s.SectionNumber = sectionNum
		s.Credits = credits
		s.Room = room
		s.Department = dept
		s.InstructorName = instructor
		s.LMSCourseID = lmsCourseID
		if len(meetingRaw) > 0 {
			var mp MeetingPattern
			if json.Unmarshal(meetingRaw, &mp) == nil {
				s.MeetingPattern = &mp
			}
		}
		if len(prereqRaw) > 0 {
			_ = json.Unmarshal(prereqRaw, &s.Prerequisites)
		}
		if len(prereqStatusRaw) > 0 {
			_ = json.Unmarshal(prereqStatusRaw, &reg.PrereqStatus)
		}
		reg.OrgID = orgID
		entry := ScheduleEntry{Section: s, Registration: reg}
		entry.CourseCode = courseCode
		entry.CourseTitle = courseTitle
		out = append(out, entry)
	}
	return out, rows.Err()
}

// GetPrereqStatusForUser returns prerequisite completion for a section.
func GetPrereqStatusForUser(ctx context.Context, pool *pgxpool.Pool, userID, sectionID uuid.UUID) ([]PrereqStatus, error) {
	var raw []byte
	err := pool.QueryRow(ctx, `
SELECT prereq_status FROM catalog.student_registrations
WHERE user_id = $1 AND catalog_section_id = $2
`, userID, sectionID).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []PrereqStatus
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	return out, nil
}

// CreateSyncLog starts a catalog sync audit record.
func CreateSyncLog(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, connectionID *uuid.UUID) (*SyncLog, error) {
	var id uuid.UUID
	var startedAt time.Time
	err := pool.QueryRow(ctx, `
INSERT INTO catalog.catalog_sync_logs (org_id, connection_id, status)
VALUES ($1, $2, 'running')
RETURNING id, started_at
`, orgID, connectionID).Scan(&id, &startedAt)
	if err != nil {
		return nil, err
	}
	return &SyncLog{ID: id, OrgID: orgID, ConnectionID: connectionID, StartedAt: startedAt, Status: SyncStatusRunning}, nil
}

// FinishSyncLog closes a catalog sync log.
func FinishSyncLog(ctx context.Context, pool *pgxpool.Pool, logID uuid.UUID, status string, sectionsSynced, shellsCreated, shellsUpdated int, errs []SyncError) error {
	errJSON, _ := json.Marshal(errs)
	_, err := pool.Exec(ctx, `
UPDATE catalog.catalog_sync_logs SET
	finished_at = NOW(),
	status = $2,
	sections_synced = $3,
	shells_created = $4,
	shells_updated = $5,
	errors = $6
WHERE id = $1
`, logID, status, sectionsSynced, shellsCreated, shellsUpdated, errJSON)
	return err
}

// GetLastSyncStatus returns the most recent catalog sync for an org.
func GetLastSyncStatus(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) (*SyncLog, error) {
	var log SyncLog
	var connID *uuid.UUID
	var finishedAt *time.Time
	var errRaw []byte
	err := pool.QueryRow(ctx, `
SELECT id, org_id, connection_id, started_at, finished_at, status,
       sections_synced, shells_created, shells_updated, errors
FROM catalog.catalog_sync_logs
WHERE org_id = $1
ORDER BY started_at DESC
LIMIT 1
`, orgID).Scan(
		&log.ID, &log.OrgID, &connID, &log.StartedAt, &finishedAt, &log.Status,
		&log.SectionsSynced, &log.ShellsCreated, &log.ShellsUpdated, &errRaw,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	log.ConnectionID = connID
	log.FinishedAt = finishedAt
	if len(errRaw) > 0 {
		_ = json.Unmarshal(errRaw, &log.Errors)
	}
	return &log, nil
}

// FindOrgBootstrapUser returns the first org_admin user for shell creation.
func FindOrgBootstrapUser(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) (*uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
SELECT g.user_id FROM "user".org_role_grants g
WHERE g.org_id = $1 AND g.role = 'org_admin'
  AND (g.expires_at IS NULL OR g.expires_at > NOW())
ORDER BY g.granted_at ASC
LIMIT 1
`, orgID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}
