// Package stateprivacy persists state-specific student data privacy records
// (CA SOPIPA, NY Ed Law 2-d, IL SOPPA) — plan 10.6.
package stateprivacy

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DisclosureEvent is one row from compliance.state_disclosure_events.
type DisclosureEvent struct {
	ID           uuid.UUID
	OrgID        uuid.UUID
	StudentID    uuid.UUID
	Accessor     string
	Purpose      string
	DataElements []string
	OccurredAt   time.Time
}

// DeletionRequest is one row from compliance.state_deletion_requests.
type DeletionRequest struct {
	ID             uuid.UUID
	OrgID          uuid.UUID
	StudentID      uuid.UUID
	RequesterID    *uuid.UUID
	RequesterEmail string
	Status         string
	ResponseNotes  *string
	SubmittedAt    time.Time
	DueAt          time.Time
	CompletedAt    *time.Time
	ActionedBy     *uuid.UUID
}

// AnnualNoticeJob is one row from compliance.annual_notice_jobs.
type AnnualNoticeJob struct {
	ID           uuid.UUID
	OrgID        uuid.UUID
	Jurisdiction string
	Year         int
	SentAt       *time.Time
}

// OrgJurisdiction returns the state_privacy_jurisdiction for an org, or "" if unset.
func OrgJurisdiction(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) (string, error) {
	var j *string
	err := pool.QueryRow(ctx, `
SELECT state_privacy_jurisdiction FROM tenant.organizations WHERE id = $1
`, orgID).Scan(&j)
	if err != nil {
		return "", err
	}
	if j == nil {
		return "", nil
	}
	return *j, nil
}

// ListDisclosureEvents returns events for a student within a school year window.
// schoolYearStart filters to events on or after that timestamp.
func ListDisclosureEvents(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID, schoolYearStart time.Time) ([]DisclosureEvent, error) {
	rows, err := pool.Query(ctx, `
SELECT id, org_id, student_id, accessor, purpose, data_elements, occurred_at
  FROM compliance.state_disclosure_events
 WHERE student_id = $1
   AND occurred_at >= $2
 ORDER BY occurred_at DESC
`, studentID, schoolYearStart)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEvents(rows)
}

func scanEvents(rows pgx.Rows) ([]DisclosureEvent, error) {
	var out []DisclosureEvent
	for rows.Next() {
		var e DisclosureEvent
		if err := rows.Scan(&e.ID, &e.OrgID, &e.StudentID, &e.Accessor, &e.Purpose, &e.DataElements, &e.OccurredAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// InsertDeletionRequest creates an IL SOPPA deletion request row.
func InsertDeletionRequest(ctx context.Context, pool *pgxpool.Pool, orgID, studentID uuid.UUID, requesterID *uuid.UUID, requesterEmail string) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO compliance.state_deletion_requests (org_id, student_id, requester_id, requester_email)
VALUES ($1, $2, $3, $4)
RETURNING id
`, orgID, studentID, requesterID, requesterEmail).Scan(&id)
	return id, err
}

// GetDeletionRequest returns a deletion request by ID.
func GetDeletionRequest(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*DeletionRequest, error) {
	r, err := scanDeletionRequest(pool.QueryRow(ctx, `
SELECT id, org_id, student_id, requester_id, requester_email, status, response_notes,
       submitted_at, due_at, completed_at, actioned_by
  FROM compliance.state_deletion_requests
 WHERE id = $1
`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return r, err
}

// ListDeletionRequestsForStudent returns all deletion requests for a student.
func ListDeletionRequestsForStudent(ctx context.Context, pool *pgxpool.Pool, studentID uuid.UUID) ([]DeletionRequest, error) {
	return queryDeletionRequests(ctx, pool, `
SELECT id, org_id, student_id, requester_id, requester_email, status, response_notes,
       submitted_at, due_at, completed_at, actioned_by
  FROM compliance.state_deletion_requests
 WHERE student_id = $1
 ORDER BY submitted_at DESC
`, studentID)
}

// UpdateDeletionRequestStatus transitions a request to a new status.
func UpdateDeletionRequestStatus(ctx context.Context, pool *pgxpool.Pool, id, actionedBy uuid.UUID, status string, notes *string) error {
	var completedAt *time.Time
	if status == "completed" || status == "denied" {
		t := time.Now().UTC()
		completedAt = &t
	}
	_, err := pool.Exec(ctx, `
UPDATE compliance.state_deletion_requests
   SET status         = $2,
       response_notes = COALESCE($3, response_notes),
       completed_at   = COALESCE($4, completed_at),
       actioned_by    = $5
 WHERE id = $1
`, id, status, notes, completedAt, actionedBy)
	return err
}

// CountOverdueDeletionRequests returns requests past their 30-day deadline.
func CountOverdueDeletionRequests(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM compliance.state_deletion_requests
 WHERE status IN ('pending','in_progress')
   AND due_at < NOW()
`).Scan(&n)
	return n, err
}

// GetAnnualNoticeJob returns the job record for an org/jurisdiction/year, or nil.
func GetAnnualNoticeJob(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, jurisdiction string, year int) (*AnnualNoticeJob, error) {
	var j AnnualNoticeJob
	err := pool.QueryRow(ctx, `
SELECT id, org_id, jurisdiction, year, sent_at
  FROM compliance.annual_notice_jobs
 WHERE org_id = $1 AND jurisdiction = $2 AND year = $3
`, orgID, jurisdiction, year).Scan(&j.ID, &j.OrgID, &j.Jurisdiction, &j.Year, &j.SentAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &j, nil
}

func scanDeletionRequest(row pgx.Row) (*DeletionRequest, error) {
	var r DeletionRequest
	err := row.Scan(
		&r.ID, &r.OrgID, &r.StudentID, &r.RequesterID, &r.RequesterEmail,
		&r.Status, &r.ResponseNotes,
		&r.SubmittedAt, &r.DueAt, &r.CompletedAt, &r.ActionedBy,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func queryDeletionRequests(ctx context.Context, pool *pgxpool.Pool, query string, args ...any) ([]DeletionRequest, error) {
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DeletionRequest
	for rows.Next() {
		var r DeletionRequest
		if err := rows.Scan(
			&r.ID, &r.OrgID, &r.StudentID, &r.RequesterID, &r.RequesterEmail,
			&r.Status, &r.ResponseNotes,
			&r.SubmittedAt, &r.DueAt, &r.CompletedAt, &r.ActionedBy,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
