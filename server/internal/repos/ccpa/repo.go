// Package ccpa persists CCPA/CPRA opt-out flags and rights request records (plan 10.4).
package ccpa

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OptOutState holds the current CCPA opt-out flags for a user.
type OptOutState struct {
	DoNotSell       bool
	LimitSensitivePI bool
}

// GetOptOut returns the current opt-out flags for a user.
func GetOptOut(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (OptOutState, error) {
	var s OptOutState
	err := pool.QueryRow(ctx, `
SELECT ccpa_do_not_sell, ccpa_limit_sensitive_pi
  FROM "user".users
 WHERE id = $1
`, userID).Scan(&s.DoNotSell, &s.LimitSensitivePI)
	return s, err
}

// SetDoNotSell updates the do_not_sell flag for a user.
func SetDoNotSell(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, value bool) error {
	_, err := pool.Exec(ctx, `
UPDATE "user".users SET ccpa_do_not_sell = $2 WHERE id = $1
`, userID, value)
	return err
}

// SetLimitSensitivePI updates the limit_sensitive_pi flag for a user.
func SetLimitSensitivePI(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, value bool) error {
	_, err := pool.Exec(ctx, `
UPDATE "user".users SET ccpa_limit_sensitive_pi = $2 WHERE id = $1
`, userID, value)
	return err
}

// CCPARequest is one row from compliance.ccpa_requests.
type CCPARequest struct {
	ID                    uuid.UUID
	UserID                *uuid.UUID
	RequesterEmail        string
	RequestType           string
	Status                string
	VerificationTokenHash *string
	ResponsePayload       *string
	RequestedAt           time.Time
	DueAt                 time.Time
	CompletedAt           *time.Time
	Extended              bool
	ActionedBy            *uuid.UUID
}

// InsertRequest creates a new CCPA rights request row.
func InsertRequest(ctx context.Context, pool *pgxpool.Pool, userID *uuid.UUID, requesterEmail, requestType string) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO compliance.ccpa_requests (user_id, requester_email, request_type)
VALUES ($1, $2, $3)
RETURNING id
`, userID, requesterEmail, requestType).Scan(&id)
	return id, err
}

// GetRequest returns a single request by ID.
func GetRequest(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*CCPARequest, error) {
	r, err := scanRequest(pool.QueryRow(ctx, `
SELECT id, user_id, requester_email, request_type, status,
       verification_token_hash, response_payload,
       requested_at, due_at, completed_at, extended, actioned_by
  FROM compliance.ccpa_requests
 WHERE id = $1
`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return r, err
}

// ListRequestsForUser returns all requests for a registered user.
func ListRequestsForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]CCPARequest, error) {
	return queryRequests(ctx, pool, `
SELECT id, user_id, requester_email, request_type, status,
       verification_token_hash, response_payload,
       requested_at, due_at, completed_at, extended, actioned_by
  FROM compliance.ccpa_requests
 WHERE user_id = $1
 ORDER BY requested_at DESC
`, userID)
}

// ListPendingRequests returns all pending/in-progress requests for admin review.
func ListPendingRequests(ctx context.Context, pool *pgxpool.Pool, limit int) ([]CCPARequest, error) {
	return queryRequests(ctx, pool, `
SELECT id, user_id, requester_email, request_type, status,
       verification_token_hash, response_payload,
       requested_at, due_at, completed_at, extended, actioned_by
  FROM compliance.ccpa_requests
 WHERE status IN ('pending','verified','in_progress')
 ORDER BY due_at ASC
 LIMIT $1
`, limit)
}

// UpdateRequestStatus transitions a request to a new status.
func UpdateRequestStatus(ctx context.Context, pool *pgxpool.Pool, id, actionedBy uuid.UUID, status string, responsePayload *string) error {
	var completedAt *time.Time
	if status == "completed" || status == "denied" {
		t := time.Now().UTC()
		completedAt = &t
	}
	_, err := pool.Exec(ctx, `
UPDATE compliance.ccpa_requests
   SET status           = $2,
       response_payload = COALESCE($3, response_payload),
       completed_at     = COALESCE($4, completed_at),
       actioned_by      = $5
 WHERE id = $1
`, id, status, responsePayload, completedAt, actionedBy)
	return err
}

// CountOverdueRequests returns the number of pending/in-progress requests past their 45-day deadline.
func CountOverdueRequests(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM compliance.ccpa_requests
 WHERE status IN ('pending','verified','in_progress')
   AND due_at < NOW()
`).Scan(&n)
	return n, err
}

// ListRequestsDueSoon returns requests whose due_at is within the given horizon.
func ListRequestsDueSoon(ctx context.Context, pool *pgxpool.Pool, horizon time.Duration) ([]CCPARequest, error) {
	cutoff := time.Now().UTC().Add(horizon)
	return queryRequests(ctx, pool, `
SELECT id, user_id, requester_email, request_type, status,
       verification_token_hash, response_payload,
       requested_at, due_at, completed_at, extended, actioned_by
  FROM compliance.ccpa_requests
 WHERE status IN ('pending','verified','in_progress')
   AND due_at <= $1
   AND due_at > NOW()
 ORDER BY due_at ASC
`, cutoff)
}

func scanRequest(row pgx.Row) (*CCPARequest, error) {
	var r CCPARequest
	err := row.Scan(
		&r.ID, &r.UserID, &r.RequesterEmail, &r.RequestType, &r.Status,
		&r.VerificationTokenHash, &r.ResponsePayload,
		&r.RequestedAt, &r.DueAt, &r.CompletedAt, &r.Extended, &r.ActionedBy,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func queryRequests(ctx context.Context, pool *pgxpool.Pool, query string, args ...any) ([]CCPARequest, error) {
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CCPARequest
	for rows.Next() {
		var r CCPARequest
		if err := rows.Scan(
			&r.ID, &r.UserID, &r.RequesterEmail, &r.RequestType, &r.Status,
			&r.VerificationTokenHash, &r.ResponsePayload,
			&r.RequestedAt, &r.DueAt, &r.CompletedAt, &r.Extended, &r.ActionedBy,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
