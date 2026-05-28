// Package backup implements backup/restore ops: status, alerts, and restore drills (plan 10.15).
package backup

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	repobackup "github.com/lextures/lextures/server/internal/repos/backup"
	"github.com/lextures/lextures/server/internal/repos/rbac"
)

const (
	AdminPermission = "compliance:backup:admin:*"

	postgresRPOTargetMinutes       = 60
	postgresRTOTargetMinutes       = 240
	objectStorageRPOTargetHours    = 24
	dailyBackupStaleThresholdHours = 25
	walLagAlertSeconds             = 900 // 15 minutes per plan risk mitigation
)

// CheckAdmin returns true when the user holds backup ops admin permission.
func CheckAdmin(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (bool, error) {
	return rbac.UserHasPermission(ctx, pool, userID, AdminPermission)
}

// Targets documents contractual RPO/RTO targets (FR-1, FR-2).
type Targets struct {
	PostgresRPOMinutes        int `json:"postgresRpoMinutes"`
	PostgresRTOMinutes        int `json:"postgresRtoMinutes"`
	ObjectStorageRPOHours     int `json:"objectStorageRpoHours"`
	ObjectStorageRTOHours     int `json:"objectStorageRtoHours"`
}

// TierStatusJSON is backup health for one tier.
type TierStatusJSON struct {
	Tier                string  `json:"tier"`
	LastSuccessAt       *string `json:"lastSuccessAt,omitempty"`
	LastDurationSeconds *int    `json:"lastDurationSeconds,omitempty"`
	WALLagSeconds       *int    `json:"walLagSeconds,omitempty"`
	NextScheduledAt     *string `json:"nextScheduledAt,omitempty"`
	LastError           *string `json:"lastError,omitempty"`
	Healthy             bool    `json:"healthy"`
}

// Alert describes a backup observability alert (FR-8, AC-4).
type Alert struct {
	Tier   string `json:"tier"`
	Reason string `json:"reason"`
}

// RestoreDrillJSON is API shape for a restore drill row.
type RestoreDrillJSON struct {
	ID                 string  `json:"id"`
	DrillDate          string  `json:"drillDate"`
	BackupTimestamp    string  `json:"backupTimestamp"`
	RestoreStart       string  `json:"restoreStart"`
	RestoreEnd         *string `json:"restoreEnd,omitempty"`
	RPOAchievedMinutes *int    `json:"rpoAchievedMinutes,omitempty"`
	RTOAchievedMinutes *int    `json:"rtoAchievedMinutes,omitempty"`
	Pass               *bool   `json:"pass,omitempty"`
	SmokeTestOutput    *string `json:"smokeTestOutput,omitempty"`
	ConductedBy        *string `json:"conductedBy,omitempty"`
	Notes              *string `json:"notes,omitempty"`
}

// BackupStatus is GET /api/v1/internal/ops/backup-status response.
type BackupStatus struct {
	Targets       Targets            `json:"targets"`
	Tiers         []TierStatusJSON   `json:"tiers"`
	Alerts        []Alert            `json:"alerts"`
	RestoreDrills []RestoreDrillJSON `json:"restoreDrills"`
}

// RecordRestoreDrillInput is POST /api/v1/internal/ops/restore-drill body.
type RecordRestoreDrillInput struct {
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
}

// GetBackupStatus aggregates tier status, alerts, and drill history.
func GetBackupStatus(ctx context.Context, pool *pgxpool.Pool) (BackupStatus, error) {
	rows, err := repobackup.ListTierStatus(ctx, pool)
	if err != nil {
		return BackupStatus{}, fmt.Errorf("backup: list tier status: %w", err)
	}
	tiers := mergeEnvTierStatus(rows)
	alerts := computeAlerts(tiers)

	drills, err := repobackup.ListRestoreDrills(ctx, pool, 50)
	if err != nil {
		return BackupStatus{}, fmt.Errorf("backup: list restore drills: %w", err)
	}

	return BackupStatus{
		Targets: Targets{
			PostgresRPOMinutes:    postgresRPOTargetMinutes,
			PostgresRTOMinutes:    postgresRTOTargetMinutes,
			ObjectStorageRPOHours: objectStorageRPOTargetHours,
			ObjectStorageRTOHours: postgresRTOTargetMinutes / 60,
		},
		Tiers:         tiers,
		Alerts:        alerts,
		RestoreDrills: drillsToJSON(drills),
	}, nil
}

// RecordRestoreDrill persists a quarterly restore drill (FR-7).
func RecordRestoreDrill(ctx context.Context, pool *pgxpool.Pool, in RecordRestoreDrillInput) (uuid.UUID, error) {
	id, err := repobackup.InsertRestoreDrill(ctx, pool, repobackup.RestoreDrill{
		DrillDate:          in.DrillDate,
		BackupTimestamp:    in.BackupTimestamp,
		RestoreStart:       in.RestoreStart,
		RestoreEnd:         in.RestoreEnd,
		RPOAchievedMinutes: in.RPOAchievedMinutes,
		RTOAchievedMinutes: in.RTOAchievedMinutes,
		Pass:               in.Pass,
		SmokeTestOutput:    in.SmokeTestOutput,
		ConductedBy:        in.ConductedBy,
		Notes:              in.Notes,
	})
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("backup: record restore drill: %w", err)
	}
	return id, nil
}

// ReportTierHeartbeat updates tier status (used by backup-report CLI / cron).
func ReportTierHeartbeat(ctx context.Context, pool *pgxpool.Pool, tier repobackup.Tier, lastSuccess *time.Time, durationSec, walLagSec *int, nextScheduled *time.Time, lastErr *string) error {
	if err := repobackup.UpsertTierStatus(ctx, pool, tier, lastSuccess, durationSec, walLagSec, nextScheduled, lastErr); err != nil {
		return fmt.Errorf("backup: report heartbeat: %w", err)
	}
	return nil
}

func mergeEnvTierStatus(rows []repobackup.TierStatus) []TierStatusJSON {
	byTier := map[repobackup.Tier]repobackup.TierStatus{
		repobackup.TierPostgres:      {},
		repobackup.TierObjectStorage: {},
	}
	for _, r := range rows {
		byTier[r.Tier] = r
	}
	pg := byTier[repobackup.TierPostgres]
	obj := byTier[repobackup.TierObjectStorage]
	applyEnvOverlay(&pg, "POSTGRES")
	applyEnvOverlay(&obj, "OBJECT_STORAGE")
	byTier[repobackup.TierPostgres] = pg
	byTier[repobackup.TierObjectStorage] = obj

	order := []repobackup.Tier{repobackup.TierPostgres, repobackup.TierObjectStorage}
	out := make([]TierStatusJSON, 0, len(order))
	for _, tier := range order {
		out = append(out, tierToJSON(byTier[tier]))
	}
	return out
}

func applyEnvOverlay(s *repobackup.TierStatus, prefix string) {
	if t := envTime(prefix + "_LAST_SUCCESS"); t != nil {
		s.LastSuccessAt = t
	}
	if v := envInt(prefix + "_WAL_LAG_SECONDS"); v != nil {
		s.WALLagSeconds = v
	}
	if v := envInt(prefix + "_DURATION_SECONDS"); v != nil {
		s.LastDurationSeconds = v
	}
	if t := envTime(prefix + "_NEXT_SCHEDULED"); t != nil {
		s.NextScheduledAt = t
	}
	if e := strings.TrimSpace(os.Getenv("BACKUP_" + prefix + "_LAST_ERROR")); e != "" {
		s.LastError = &e
	}
}

func envTime(key string) *time.Time {
	s := strings.TrimSpace(os.Getenv("BACKUP_" + key))
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	u := t.UTC()
	return &u
}

func envInt(key string) *int {
	s := strings.TrimSpace(os.Getenv("BACKUP_" + key))
	if s == "" {
		return nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &v
}

func tierToJSON(s repobackup.TierStatus) TierStatusJSON {
	j := TierStatusJSON{
		Tier:                string(s.Tier),
		LastDurationSeconds: s.LastDurationSeconds,
		WALLagSeconds:       s.WALLagSeconds,
		LastError:           s.LastError,
		Healthy:             true,
	}
	if s.LastSuccessAt != nil {
		iso := s.LastSuccessAt.UTC().Format(time.RFC3339)
		j.LastSuccessAt = &iso
	}
	if s.NextScheduledAt != nil {
		iso := s.NextScheduledAt.UTC().Format(time.RFC3339)
		j.NextScheduledAt = &iso
	}
	if s.LastError != nil && strings.TrimSpace(*s.LastError) != "" {
		j.Healthy = false
	}
	return j
}

func computeAlerts(tiers []TierStatusJSON) []Alert {
	var alerts []Alert
	now := time.Now().UTC()
	stale := time.Duration(dailyBackupStaleThresholdHours) * time.Hour

	for _, t := range tiers {
		if t.LastError != nil && strings.TrimSpace(*t.LastError) != "" {
			alerts = append(alerts, Alert{Tier: t.Tier, Reason: "last job failed: " + strings.TrimSpace(*t.LastError)})
		}
		if t.LastSuccessAt == nil {
			alerts = append(alerts, Alert{Tier: t.Tier, Reason: "no successful backup recorded"})
			continue
		}
		last, err := time.Parse(time.RFC3339, *t.LastSuccessAt)
		if err != nil {
			continue
		}
		if now.Sub(last) > stale {
			alerts = append(alerts, Alert{
				Tier:   t.Tier,
				Reason: fmt.Sprintf("last success older than %d hours", dailyBackupStaleThresholdHours),
			})
		}
		if t.Tier == string(repobackup.TierPostgres) && t.WALLagSeconds != nil && *t.WALLagSeconds > walLagAlertSeconds {
			alerts = append(alerts, Alert{
				Tier:   t.Tier,
				Reason: fmt.Sprintf("WAL lag %d seconds exceeds %d second threshold", *t.WALLagSeconds, walLagAlertSeconds),
			})
		}
	}
	return alerts
}

func drillsToJSON(drills []repobackup.RestoreDrill) []RestoreDrillJSON {
	out := make([]RestoreDrillJSON, 0, len(drills))
	for _, d := range drills {
		j := RestoreDrillJSON{
			ID:              d.ID.String(),
			DrillDate:         d.DrillDate.Format("2006-01-02"),
			BackupTimestamp:   d.BackupTimestamp.UTC().Format(time.RFC3339),
			RestoreStart:      d.RestoreStart.UTC().Format(time.RFC3339),
			RPOAchievedMinutes: d.RPOAchievedMinutes,
			RTOAchievedMinutes: d.RTOAchievedMinutes,
			Pass:              d.Pass,
			SmokeTestOutput:   d.SmokeTestOutput,
			Notes:             d.Notes,
		}
		if d.RestoreEnd != nil {
			iso := d.RestoreEnd.UTC().Format(time.RFC3339)
			j.RestoreEnd = &iso
		}
		if d.ConductedBy != nil {
			s := d.ConductedBy.String()
			j.ConductedBy = &s
		}
		out = append(out, j)
	}
	return out
}
