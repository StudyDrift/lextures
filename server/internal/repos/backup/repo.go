// Package backup persists backup tier status and restore drill records (plan 10.15).
package backup

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Tier identifies a backup data tier.
type Tier string

const (
	TierPostgres      Tier = "postgres"
	TierObjectStorage Tier = "object_storage"
)

// TierStatus is one row from compliance.backup_tier_status.
type TierStatus struct {
	Tier                Tier
	LastSuccessAt       *time.Time
	LastDurationSeconds *int
	WALLagSeconds       *int
	NextScheduledAt     *time.Time
	LastError           *string
	UpdatedAt           time.Time
}

// RestoreDrill is one row from compliance.restore_drills.
type RestoreDrill struct {
	ID                 uuid.UUID
	DrillDate          time.Time
	BackupTimestamp    time.Time
	RestoreStart       time.Time
	RestoreEnd         *time.Time
	RPOAchievedMinutes *int
	RTOAchievedMinutes *int
	Pass               *bool
	SmokeTestOutput    *string
	ConductedBy        *uuid.UUID
	Notes              *string
	CreatedAt          time.Time
}

// ListTierStatus returns all tier status rows.
func ListTierStatus(ctx context.Context, pool *pgxpool.Pool) ([]TierStatus, error) {
	rows, err := pool.Query(ctx, `
SELECT tier, last_success_at, last_duration_seconds, wal_lag_seconds, next_scheduled_at, last_error, updated_at
  FROM compliance.backup_tier_status
 ORDER BY tier
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TierStatus
	for rows.Next() {
		var s TierStatus
		var tier string
		if err := rows.Scan(&tier, &s.LastSuccessAt, &s.LastDurationSeconds, &s.WALLagSeconds, &s.NextScheduledAt, &s.LastError, &s.UpdatedAt); err != nil {
			return nil, err
		}
		s.Tier = Tier(tier)
		out = append(out, s)
	}
	return out, rows.Err()
}

// UpsertTierStatus records a backup heartbeat for a tier.
func UpsertTierStatus(ctx context.Context, pool *pgxpool.Pool, tier Tier, lastSuccess *time.Time, durationSec, walLagSec *int, nextScheduled *time.Time, lastErr *string) error {
	_, err := pool.Exec(ctx, `
INSERT INTO compliance.backup_tier_status (tier, last_success_at, last_duration_seconds, wal_lag_seconds, next_scheduled_at, last_error, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, NOW())
ON CONFLICT (tier) DO UPDATE SET
  last_success_at       = COALESCE(EXCLUDED.last_success_at, compliance.backup_tier_status.last_success_at),
  last_duration_seconds = COALESCE(EXCLUDED.last_duration_seconds, compliance.backup_tier_status.last_duration_seconds),
  wal_lag_seconds       = COALESCE(EXCLUDED.wal_lag_seconds, compliance.backup_tier_status.wal_lag_seconds),
  next_scheduled_at     = COALESCE(EXCLUDED.next_scheduled_at, compliance.backup_tier_status.next_scheduled_at),
  last_error            = EXCLUDED.last_error,
  updated_at            = NOW()
`, string(tier), lastSuccess, durationSec, walLagSec, nextScheduled, lastErr)
	return err
}

// InsertRestoreDrill creates a restore drill record.
func InsertRestoreDrill(ctx context.Context, pool *pgxpool.Pool, drill RestoreDrill) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO compliance.restore_drills (
  drill_date, backup_timestamp, restore_start, restore_end,
  rpo_achieved_minutes, rto_achieved_minutes, pass, smoke_test_output, conducted_by, notes
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id
`, drill.DrillDate, drill.BackupTimestamp, drill.RestoreStart, drill.RestoreEnd,
		drill.RPOAchievedMinutes, drill.RTOAchievedMinutes, drill.Pass, drill.SmokeTestOutput, drill.ConductedBy, drill.Notes,
	).Scan(&id)
	return id, err
}

// ListRestoreDrills returns recent drills ordered by drill_date DESC.
func ListRestoreDrills(ctx context.Context, pool *pgxpool.Pool, limit int) ([]RestoreDrill, error) {
	rows, err := pool.Query(ctx, `
SELECT id, drill_date, backup_timestamp, restore_start, restore_end,
       rpo_achieved_minutes, rto_achieved_minutes, pass, smoke_test_output, conducted_by, notes, created_at
  FROM compliance.restore_drills
 ORDER BY drill_date DESC, created_at DESC
 LIMIT $1
`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RestoreDrill
	for rows.Next() {
		var d RestoreDrill
		if err := rows.Scan(
			&d.ID, &d.DrillDate, &d.BackupTimestamp, &d.RestoreStart, &d.RestoreEnd,
			&d.RPOAchievedMinutes, &d.RTOAchievedMinutes, &d.Pass, &d.SmokeTestOutput, &d.ConductedBy, &d.Notes, &d.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
