// Package licensesvc enforces org seat limits and utilization alerts (plan 18.8).
package licensesvc

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	licenserepo "github.com/lextures/lextures/server/internal/repos/license"
	"github.com/lextures/lextures/server/internal/repos/orgrolegrant"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/service/notifications"
)

// ErrSeatLimitReached is returned when an org has no remaining learner seats.
var ErrSeatLimitReached = errors.New("seat_limit_reached")

// Service manages seat license enforcement.
type Service struct {
	Pool   *pgxpool.Pool
	Config config.Config
}

func New(pool *pgxpool.Pool, cfg config.Config) *Service {
	return &Service{Pool: pool, Config: cfg}
}

func (s *Service) enabled() bool {
	return s.Config.SeatManagementEnabled
}

// UserExemptFromSeatLimit is true for org_admin and Global Admin users.
func (s *Service) UserExemptFromSeatLimit(ctx context.Context, userID, orgID uuid.UUID) (bool, error) {
	isGA, err := rbac.UserHasPermission(ctx, s.Pool, userID, "global:app:rbac:manage")
	if err != nil {
		return false, err
	}
	if isGA {
		return true, nil
	}
	return orgrolegrant.HasActiveOrgAdmin(ctx, s.Pool, userID, orgID)
}

// CheckCanActivate verifies a learner seat is available before activating/creating a user.
func (s *Service) CheckCanActivate(ctx context.Context, userID, orgID uuid.UUID) error {
	if !s.enabled() {
		return nil
	}
	exempt, err := s.UserExemptFromSeatLimit(ctx, userID, orgID)
	if err != nil {
		return err
	}
	if exempt {
		return nil
	}
	lic, err := licenserepo.Effective(ctx, s.Pool, orgID)
	if err != nil {
		return err
	}
	if lic.MaxSeats < 0 {
		return nil
	}
	if lic.UsedSeats >= lic.MaxSeats {
		return ErrSeatLimitReached
	}
	return nil
}

// CheckCanActivateTx is the transactional variant; locks the license row when present.
func (s *Service) CheckCanActivateTx(ctx context.Context, tx pgx.Tx, userID, orgID uuid.UUID) error {
	if !s.enabled() {
		return nil
	}
	exempt, err := s.UserExemptFromSeatLimit(ctx, userID, orgID)
	if err != nil {
		return err
	}
	if exempt {
		return nil
	}
	var maxSeats, usedSeats int
	err = tx.QueryRow(ctx, `
SELECT COALESCE(l.max_seats, -1), COALESCE(l.used_seats, tenant.count_learner_seats($1))
FROM (SELECT $1::uuid AS org_id) sub
LEFT JOIN tenant.licenses l ON l.org_id = sub.org_id
FOR UPDATE OF l
`, orgID).Scan(&maxSeats, &usedSeats)
	if err != nil {
		return err
	}
	if maxSeats < 0 {
		return nil
	}
	if usedSeats >= maxSeats {
		return ErrSeatLimitReached
	}
	return nil
}

// UtilizationPercent returns used/max * 100, or 0 when unlimited.
func UtilizationPercent(used, max int) float64 {
	if max <= 0 {
		return 0
	}
	return float64(used) / float64(max) * 100
}

// ContractExpiringSoon is true when contract_end is within days.
func ContractExpiringSoon(lic licenserepo.Row, withinDays int) bool {
	if lic.ContractEnd == nil {
		return false
	}
	deadline := lic.ContractEnd.AddDate(0, 0, -withinDays)
	now := lic.UpdatedAt
	if now.IsZero() {
		now = lic.CreatedAt
	}
	return !lic.ContractEnd.Before(now) && !deadline.After(now)
}

// MaybeSendUtilizationAlerts emails org admins when utilization crosses 80% or 95%.
func (s *Service) MaybeSendUtilizationAlerts(ctx context.Context, orgID uuid.UUID) error {
	if !s.enabled() || !s.Config.EmailNotificationsEnabled {
		return nil
	}
	lic, err := licenserepo.Effective(ctx, s.Pool, orgID)
	if err != nil {
		return err
	}
	if lic.MaxSeats <= 0 {
		return nil
	}
	pct := UtilizationPercent(lic.UsedSeats, lic.MaxSeats)
	thresholds := []int{}
	if pct >= 95 {
		thresholds = append(thresholds, 95)
	}
	if pct >= 80 {
		thresholds = append(thresholds, 80)
	}
	if len(thresholds) == 0 {
		return nil
	}

	var orgName string
	if err := s.Pool.QueryRow(ctx, `SELECT name FROM tenant.organizations WHERE id = $1`, orgID).Scan(&orgName); err != nil {
		return err
	}

	notif := &notifications.Service{Pool: s.Pool, Config: s.Config}
	for _, th := range thresholds {
		var exists bool
		err := s.Pool.QueryRow(ctx, `
SELECT EXISTS(
  SELECT 1 FROM tenant.license_utilization_alerts
  WHERE org_id = $1 AND threshold_pct = $2
)`, orgID, th).Scan(&exists)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		adminIDs, err := listOrgAdminIDs(ctx, s.Pool, orgID)
		if err != nil {
			return err
		}
		vars := map[string]string{
			"orgName":      orgName,
			"usedSeats":    strconv.Itoa(lic.UsedSeats),
			"maxSeats":     strconv.Itoa(lic.MaxSeats),
			"percentUsed":  fmt.Sprintf("%.0f", pct),
			"thresholdPct": strconv.Itoa(th),
			"subject":      fmt.Sprintf("Seat license alert: %d%% utilization for %s", th, orgName),
		}
		for _, adminID := range adminIDs {
			if err := notif.EnqueueEmail(ctx, adminID, notifications.EventSeatUtilizationAlert, "seat_utilization_alert", vars, nil); err != nil {
				return err
			}
		}
		if _, err := s.Pool.Exec(ctx, `
INSERT INTO tenant.license_utilization_alerts (org_id, threshold_pct)
VALUES ($1, $2)
ON CONFLICT DO NOTHING`, orgID, th); err != nil {
			return err
		}
	}
	return nil
}

func listOrgAdminIDs(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := pool.Query(ctx, `
SELECT g.user_id
FROM "user".org_role_grants g
WHERE g.org_id = $1
  AND g.role = $2
  AND g.org_unit_id IS NULL
  AND (g.expires_at IS NULL OR g.expires_at > NOW())
`, orgID, orgrolegrant.RoleOrgAdmin)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// SweepUtilizationAlerts checks all limited orgs and sends threshold emails.
func (s *Service) SweepUtilizationAlerts(ctx context.Context) error {
	if !s.enabled() {
		return nil
	}
	rows, err := s.Pool.Query(ctx, `
SELECT org_id FROM tenant.licenses WHERE max_seats > 0`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var orgID uuid.UUID
		if err := rows.Scan(&orgID); err != nil {
			return err
		}
		if err := s.MaybeSendUtilizationAlerts(ctx, orgID); err != nil {
			return err
		}
	}
	return rows.Err()
}

// Reconcile refreshes all license counters.
func (s *Service) Reconcile(ctx context.Context) (int, error) {
	return licenserepo.ReconcileAll(ctx, s.Pool)
}

// AfterSeatCountChange should be called after user activation changes when triggers are disabled in tests.
func (s *Service) AfterSeatCountChange(ctx context.Context, orgID uuid.UUID) error {
	if err := licenserepo.RefreshUsedSeats(ctx, s.Pool, orgID); err != nil {
		return err
	}
	return s.MaybeSendUtilizationAlerts(ctx, orgID)
}
