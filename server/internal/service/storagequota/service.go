// Package storagequota enforces per-tenant / per-course / per-user storage limits (plan 8.5).
package storagequota

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// QuotaViolation describes which quota level was exceeded.
type QuotaViolation struct {
	QuotaType  string `json:"quota_type"`  // "tenant", "course", or "user"
	UsedBytes  int64  `json:"used_bytes"`
	LimitBytes int64  `json:"limit_bytes"`
}

// UsageInfo is returned by GET /courses/:code/storage-usage.
type UsageInfo struct {
	UsedBytes   int64    `json:"used_bytes"`
	LimitBytes  *int64   `json:"limit_bytes"` // nil = unlimited
	PercentUsed float64  `json:"percent_used"`
}

// QuotaEntry is one row in the admin quota list response.
type QuotaEntry struct {
	Scope       string  `json:"scope"`
	ScopeID     string  `json:"scope_id"`
	LimitBytes  *int64  `json:"limit_bytes"`
	UsedBytes   int64   `json:"used_bytes"`
	PercentUsed float64 `json:"percent_used"`
}

// Service manages storage quota settings and usage counters.
type Service struct {
	Pool *pgxpool.Pool
}

type scopeRef struct {
	name string
	id   uuid.UUID
}

func buildScopes(tenantID uuid.UUID, courseID *uuid.UUID, userID uuid.UUID) []scopeRef {
	ss := []scopeRef{{"tenant", tenantID}, {"user", userID}}
	if courseID != nil {
		ss = append(ss, scopeRef{"course", *courseID})
	}
	return ss
}

// CheckAndReserve atomically checks all applicable quota levels and, if none
// are exceeded, increments the usage counters. Returns a non-nil QuotaViolation
// if any limit would be breached — counters are NOT modified in that case.
func (s *Service) CheckAndReserve(ctx context.Context, tenantID uuid.UUID, courseID *uuid.UUID, userID uuid.UUID, bytes int64) (*QuotaViolation, error) {
	scopes := buildScopes(tenantID, courseID, userID)

	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Ensure a counter row exists for every scope so SELECT FOR UPDATE can lock it.
	for _, sc := range scopes {
		if _, err = tx.Exec(ctx, `
			INSERT INTO storage.usage_counters (scope, scope_id, used_bytes)
			VALUES ($1, $2, 0)
			ON CONFLICT (scope, scope_id) DO NOTHING`,
			sc.name, sc.id); err != nil {
			return nil, err
		}
	}

	// Lock counter rows and check limits.
	for _, sc := range scopes {
		var usedBytes int64
		var limitBytes *int64
		err = tx.QueryRow(ctx, `
			SELECT uc.used_bytes, qs.limit_bytes
			FROM storage.usage_counters uc
			LEFT JOIN storage.quota_settings qs
			  ON qs.scope = uc.scope AND qs.scope_id = uc.scope_id
			WHERE uc.scope = $1 AND uc.scope_id = $2
			FOR UPDATE OF uc`,
			sc.name, sc.id).Scan(&usedBytes, &limitBytes)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		if limitBytes != nil && usedBytes+bytes > *limitBytes {
			return &QuotaViolation{
				QuotaType:  sc.name,
				UsedBytes:  usedBytes,
				LimitBytes: *limitBytes,
			}, nil
		}
	}

	// All levels cleared — increment counters.
	for _, sc := range scopes {
		if _, err = tx.Exec(ctx, `
			UPDATE storage.usage_counters
			SET used_bytes = used_bytes + $1, updated_at = now()
			WHERE scope = $2 AND scope_id = $3`,
			bytes, sc.name, sc.id); err != nil {
			return nil, err
		}
	}

	return nil, tx.Commit(ctx)
}

// Release decrements usage counters (called on upload cancellation or file deletion).
// Counters are clamped to zero so drift from failed deletions never goes negative.
func (s *Service) Release(ctx context.Context, tenantID uuid.UUID, courseID *uuid.UUID, userID uuid.UUID, bytes int64) error {
	for _, sc := range buildScopes(tenantID, courseID, userID) {
		if _, err := s.Pool.Exec(ctx, `
			UPDATE storage.usage_counters
			SET used_bytes = GREATEST(0, used_bytes - $1), updated_at = now()
			WHERE scope = $2 AND scope_id = $3`,
			bytes, sc.name, sc.id); err != nil {
			return err
		}
	}
	return nil
}

// GetCourseUsage returns current usage and limit for a course.
func (s *Service) GetCourseUsage(ctx context.Context, courseID uuid.UUID) (*UsageInfo, error) {
	var usedBytes int64
	var limitBytes *int64
	err := s.Pool.QueryRow(ctx, `
		SELECT COALESCE(uc.used_bytes, 0), qs.limit_bytes
		FROM (SELECT $1::UUID AS cid) AS sub
		LEFT JOIN storage.usage_counters uc
		  ON uc.scope = 'course' AND uc.scope_id = sub.cid
		LEFT JOIN storage.quota_settings qs
		  ON qs.scope = 'course' AND qs.scope_id = sub.cid`,
		courseID).Scan(&usedBytes, &limitBytes)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	info := &UsageInfo{UsedBytes: usedBytes, LimitBytes: limitBytes}
	if limitBytes != nil && *limitBytes > 0 {
		info.PercentUsed = float64(usedBytes) / float64(*limitBytes) * 100
	}
	return info, nil
}

// SetQuota upserts the byte limit for a given scope + scope_id.
// Pass limitBytes = nil to remove an explicit limit (restores unlimited).
func (s *Service) SetQuota(ctx context.Context, scopeName string, scopeID uuid.UUID, limitBytes *int64) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO storage.quota_settings (scope, scope_id, limit_bytes, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (scope, scope_id) DO UPDATE
		  SET limit_bytes = EXCLUDED.limit_bytes, updated_at = now()`,
		scopeName, scopeID, limitBytes)
	return err
}

// ListQuotas returns all quota settings alongside their current usage.
func (s *Service) ListQuotas(ctx context.Context) ([]QuotaEntry, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT qs.scope, qs.scope_id::text, qs.limit_bytes,
		       COALESCE(uc.used_bytes, 0) AS used_bytes
		FROM storage.quota_settings qs
		LEFT JOIN storage.usage_counters uc USING (scope, scope_id)
		ORDER BY qs.scope, qs.scope_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []QuotaEntry
	for rows.Next() {
		var e QuotaEntry
		if err = rows.Scan(&e.Scope, &e.ScopeID, &e.LimitBytes, &e.UsedBytes); err != nil {
			return nil, err
		}
		if e.LimitBytes != nil && *e.LimitBytes > 0 {
			e.PercentUsed = float64(e.UsedBytes) / float64(*e.LimitBytes) * 100
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// Reconcile recomputes all usage_counters from the current state of storage.objects.
// It is safe to run at any time; it overwrites counter values with the ground truth.
func (s *Service) Reconcile(ctx context.Context) error {
	_, err := s.Pool.Exec(ctx, `
		WITH actual AS (
			SELECT tenant_id AS scope_id, 'tenant' AS scope,
			       COALESCE(SUM(size_bytes), 0) AS used_bytes
			FROM storage.objects WHERE deleted_at IS NULL
			GROUP BY tenant_id
			UNION ALL
			SELECT course_id AS scope_id, 'course' AS scope,
			       COALESCE(SUM(size_bytes), 0) AS used_bytes
			FROM storage.objects WHERE deleted_at IS NULL AND course_id IS NOT NULL
			GROUP BY course_id
			UNION ALL
			SELECT uploaded_by AS scope_id, 'user' AS scope,
			       COALESCE(SUM(size_bytes), 0) AS used_bytes
			FROM storage.objects WHERE deleted_at IS NULL AND uploaded_by IS NOT NULL
			GROUP BY uploaded_by
		)
		INSERT INTO storage.usage_counters (scope, scope_id, used_bytes, updated_at)
		SELECT scope, scope_id, used_bytes, now() FROM actual
		ON CONFLICT (scope, scope_id) DO UPDATE
		  SET used_bytes = EXCLUDED.used_bytes, updated_at = now()`)
	return err
}
