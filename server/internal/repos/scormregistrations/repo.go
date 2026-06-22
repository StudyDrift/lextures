// Package scormregistrations provides DB access for content.scorm_registrations (plan 2.14).
package scormregistrations

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/scorm"
)

// Registration is a learner runtime state row.
type Registration struct {
	ID               uuid.UUID
	ScoID            uuid.UUID
	EnrollmentID     uuid.UUID
	AttemptNo        int
	CompletionStatus string
	SuccessStatus    string
	ScoreScaled      *float64
	ScoreRaw         *float64
	ScoreMax         *float64
	TotalTimeSeconds int
	SuspendData      string
	Location         string
	UpdatedAt        time.Time
}

// LoadOrCreate loads an existing registration or creates attempt 1.
func LoadOrCreate(ctx context.Context, pool *pgxpool.Pool, scoID, enrollmentID uuid.UUID) (*Registration, error) {
	reg, err := Load(ctx, pool, scoID, enrollmentID, 1)
	if err != nil {
		return nil, err
	}
	if reg != nil {
		return reg, nil
	}
	id := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO content.scorm_registrations (id, sco_id, enrollment_id, attempt_no)
		VALUES ($1, $2, $3, 1)
		ON CONFLICT (sco_id, enrollment_id, attempt_no) DO NOTHING`,
		id, scoID, enrollmentID,
	)
	if err != nil {
		return nil, err
	}
	return Load(ctx, pool, scoID, enrollmentID, 1)
}

// Load fetches a registration by sco, enrollment, attempt.
func Load(ctx context.Context, pool *pgxpool.Pool, scoID, enrollmentID uuid.UUID, attemptNo int) (*Registration, error) {
	return scanReg(pool.QueryRow(ctx, `
		SELECT id, sco_id, enrollment_id, attempt_no, completion_status, success_status,
		       score_scaled, score_raw, score_max, total_time_seconds, suspend_data, location, updated_at
		FROM content.scorm_registrations
		WHERE sco_id = $1 AND enrollment_id = $2 AND attempt_no = $3`,
		scoID, enrollmentID, attemptNo))
}

// LoadByID loads by registration id.
func LoadByID(ctx context.Context, pool *pgxpool.Pool, registrationID uuid.UUID) (*Registration, error) {
	return scanReg(pool.QueryRow(ctx, `
		SELECT id, sco_id, enrollment_id, attempt_no, completion_status, success_status,
		       score_scaled, score_raw, score_max, total_time_seconds, suspend_data, location, updated_at
		FROM content.scorm_registrations WHERE id = $1`, registrationID))
}

// UpdateState persists CMI-derived state after commit.
func UpdateState(ctx context.Context, pool *pgxpool.Pool, registrationID uuid.UUID, state scorm.RegistrationState) error {
	_, err := pool.Exec(ctx, `
		UPDATE content.scorm_registrations SET
		  completion_status = $2,
		  success_status = $3,
		  score_scaled = $4,
		  score_raw = $5,
		  score_max = $6,
		  total_time_seconds = $7,
		  suspend_data = $8,
		  location = $9,
		  updated_at = NOW()
		WHERE id = $1`,
		registrationID,
		state.CompletionStatus,
		state.SuccessStatus,
		state.ScoreScaled,
		state.ScoreRaw,
		state.ScoreMax,
		state.TotalTimeSeconds,
		state.SuspendData,
		state.Location,
	)
	return err
}

// LogEvent records an RTE audit event.
func LogEvent(ctx context.Context, pool *pgxpool.Pool, registrationID uuid.UUID, verb string, payload []byte) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO content.scorm_rte_events (registration_id, verb, payload_json)
		VALUES ($1, $2, $3)`, registrationID, verb, payload)
	return err
}

func scanReg(row pgx.Row) (*Registration, error) {
	var r Registration
	err := row.Scan(
		&r.ID, &r.ScoID, &r.EnrollmentID, &r.AttemptNo,
		&r.CompletionStatus, &r.SuccessStatus, &r.ScoreScaled, &r.ScoreRaw, &r.ScoreMax,
		&r.TotalTimeSeconds, &r.SuspendData, &r.Location, &r.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// ToState converts a DB row to scorm.RegistrationState.
func (r *Registration) ToState() scorm.RegistrationState {
	return scorm.RegistrationState{
		CompletionStatus: r.CompletionStatus,
		SuccessStatus:    r.SuccessStatus,
		ScoreScaled:      r.ScoreScaled,
		ScoreRaw:         r.ScoreRaw,
		ScoreMax:         r.ScoreMax,
		TotalTimeSeconds: r.TotalTimeSeconds,
		SuspendData:      r.SuspendData,
		Location:         r.Location,
	}
}
