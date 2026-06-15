// Package advising stores advising notes, degree audit cache, and config (plan 14.14).
package advising

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	ProviderNone        = "none"
	ProviderDegreeWorks = "degreeworks"
	ProviderStellic     = "stellic"
)

// Note is one advising_notes row.
type Note struct {
	ID               uuid.UUID `json:"id"`
	StudentID        uuid.UUID `json:"studentId"`
	AdvisorID        uuid.UUID `json:"advisorId"`
	Content          string    `json:"content"`
	VisibleToStudent bool      `json:"visibleToStudent"`
	CreatedAt        time.Time `json:"createdAt"`
	AdvisorEmail     string    `json:"advisorEmail,omitempty"`
	AdvisorDisplay   *string   `json:"advisorDisplayName,omitempty"`
}

// DegreeAuditCache is cached degree audit JSON for a student.
type DegreeAuditCache struct {
	UserID    uuid.UUID
	Data      json.RawMessage
	FetchedAt time.Time
	Source    string
}

// Config is the singleton advising_config row.
type Config struct {
	AppointmentURL        *string
	DegreeAuditProvider string
	DegreeAuditBaseURL    *string
	APICredentialsRef   *string
	AtRiskBannerEnabled   bool
	UpdatedAt             time.Time
}

// GetConfig returns the singleton advising config.
func GetConfig(ctx context.Context, pool *pgxpool.Pool) (*Config, error) {
	var c Config
	var appt, base, creds *string
	err := pool.QueryRow(ctx, `
SELECT appointment_url, degree_audit_provider, degree_audit_base_url, api_credentials_ref,
       at_risk_banner_enabled, updated_at
FROM settings.advising_config
WHERE id = 1
`).Scan(&appt, &c.DegreeAuditProvider, &base, &creds, &c.AtRiskBannerEnabled, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return &Config{DegreeAuditProvider: ProviderNone}, nil
	}
	if err != nil {
		return nil, err
	}
	c.AppointmentURL = appt
	c.DegreeAuditBaseURL = base
	c.APICredentialsRef = creds
	if c.DegreeAuditProvider == "" {
		c.DegreeAuditProvider = ProviderNone
	}
	return &c, nil
}

// UpsertConfig saves advising configuration.
func UpsertConfig(
	ctx context.Context,
	pool *pgxpool.Pool,
	appointmentURL *string,
	provider string,
	baseURL *string,
	credentialsRef *string,
	atRiskBanner *bool,
) (*Config, error) {
	if provider == "" {
		provider = ProviderNone
	}
	var c Config
	var appt, base, creds *string
	err := pool.QueryRow(ctx, `
UPDATE settings.advising_config
SET
    appointment_url = COALESCE($1, appointment_url),
    degree_audit_provider = $2,
    degree_audit_base_url = COALESCE($3, degree_audit_base_url),
    api_credentials_ref = COALESCE($4, api_credentials_ref),
    at_risk_banner_enabled = COALESCE($5, at_risk_banner_enabled),
    updated_at = NOW()
WHERE id = 1
RETURNING appointment_url, degree_audit_provider, degree_audit_base_url, api_credentials_ref,
          at_risk_banner_enabled, updated_at
`, appointmentURL, provider, baseURL, credentialsRef, atRiskBanner).Scan(
		&appt, &c.DegreeAuditProvider, &base, &creds, &c.AtRiskBannerEnabled, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	c.AppointmentURL = appt
	c.DegreeAuditBaseURL = base
	c.APICredentialsRef = creds
	return &c, nil
}

// InsertNote creates an advising note.
func InsertNote(ctx context.Context, pool *pgxpool.Pool, studentID, advisorID uuid.UUID, content string) (*Note, error) {
	var n Note
	var advisorDisplay *string
	err := pool.QueryRow(ctx, `
WITH ins AS (
    INSERT INTO advising.advising_notes (student_id, advisor_id, content)
    VALUES ($1, $2, $3)
    RETURNING id, student_id, advisor_id, content, visible_to_student, created_at
)
SELECT ins.id, ins.student_id, ins.advisor_id, ins.content, ins.visible_to_student, ins.created_at,
       u.email, u.display_name
FROM ins
INNER JOIN "user".users u ON u.id = ins.advisor_id
`, studentID, advisorID, content).Scan(
		&n.ID, &n.StudentID, &n.AdvisorID, &n.Content, &n.VisibleToStudent, &n.CreatedAt,
		&n.AdvisorEmail, &advisorDisplay,
	)
	if err != nil {
		return nil, err
	}
	n.AdvisorDisplay = advisorDisplay
	return &n, nil
}

// ListNotesForStudent returns notes visible to the student, newest first.
func ListNotesForStudent(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID) ([]Note, error) {
	rows, err := pool.Query(ctx, `
SELECT n.id, n.student_id, n.advisor_id, n.content, n.visible_to_student, n.created_at,
       u.email, u.display_name
FROM advising.advising_notes n
INNER JOIN "user".users u ON u.id = n.advisor_id
WHERE n.student_id = $1 AND n.visible_to_student = TRUE
ORDER BY n.created_at DESC
LIMIT 200
`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNotes(rows)
}

// ListNotesForAdvisor returns all notes for a student (advisor view).
func ListNotesForAdvisor(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID) ([]Note, error) {
	rows, err := pool.Query(ctx, `
SELECT n.id, n.student_id, n.advisor_id, n.content, n.visible_to_student, n.created_at,
       u.email, u.display_name
FROM advising.advising_notes n
INNER JOIN "user".users u ON u.id = n.advisor_id
WHERE n.student_id = $1
ORDER BY n.created_at DESC
LIMIT 200
`, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNotes(rows)
}

func scanNotes(rows pgx.Rows) ([]Note, error) {
	var out []Note
	for rows.Next() {
		var n Note
		var advisorDisplay *string
		if err := rows.Scan(
			&n.ID, &n.StudentID, &n.AdvisorID, &n.Content, &n.VisibleToStudent, &n.CreatedAt,
			&n.AdvisorEmail, &advisorDisplay,
		); err != nil {
			return nil, err
		}
		n.AdvisorDisplay = advisorDisplay
		out = append(out, n)
	}
	return out, rows.Err()
}

// CountUnreadNotesForStudent counts notes created in the last 30 days (badge heuristic).
func CountRecentNotesForStudent(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID, since time.Time) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
SELECT COUNT(*)::int
FROM advising.advising_notes
WHERE student_id = $1 AND visible_to_student = TRUE AND created_at >= $2
`, studentID, since).Scan(&n)
	return n, err
}

// GetDegreeAuditCache returns cached audit data for a user.
func GetDegreeAuditCache(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (*DegreeAuditCache, error) {
	var c DegreeAuditCache
	err := pool.QueryRow(ctx, `
SELECT user_id, data, fetched_at, source
FROM advising.degree_audit_cache
WHERE user_id = $1
`, userID).Scan(&c.UserID, &c.Data, &c.FetchedAt, &c.Source)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// UpsertDegreeAuditCache stores fetched degree audit data.
func UpsertDegreeAuditCache(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, data json.RawMessage, source string, fetchedAt time.Time) error {
	_, err := pool.Exec(ctx, `
INSERT INTO advising.degree_audit_cache (user_id, data, fetched_at, source)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id) DO UPDATE SET
    data = EXCLUDED.data,
    fetched_at = EXCLUDED.fetched_at,
    source = EXCLUDED.source
`, userID, data, fetchedAt, source)
	return err
}

// ActiveAdvisorLinkBetween returns true when advisor has an active link to student in org.
func ActiveAdvisorLinkBetween(ctx context.Context, pool *pgxpool.Pool, orgID, advisorID, studentID uuid.UUID) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS(
    SELECT 1 FROM "user".advisor_student_links
    WHERE org_id = $1 AND advisor_user_id = $2 AND student_user_id = $3 AND status = 'active'
)
`, orgID, advisorID, studentID).Scan(&exists)
	return exists, err
}

// UpsertAdvisorLink creates or reactivates an advisor-student link.
func UpsertAdvisorLink(ctx context.Context, pool *pgxpool.Pool, orgID, advisorID, studentID uuid.UUID, linkedBy *uuid.UUID) error {
	_, err := pool.Exec(ctx, `
INSERT INTO "user".advisor_student_links (org_id, advisor_user_id, student_user_id, status, linked_by)
VALUES ($1, $2, $3, 'active', $4)
ON CONFLICT (advisor_user_id, student_user_id) DO UPDATE SET
    org_id = EXCLUDED.org_id,
    status = 'active',
    linked_by = EXCLUDED.linked_by,
    linked_at = NOW()
`, orgID, advisorID, studentID, linkedBy)
	return err
}
