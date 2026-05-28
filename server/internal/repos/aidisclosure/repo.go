// Package aidisclosure persists AI governance settings, inference logs, and feature acknowledgements (plan 10.17).
package aidisclosure

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// InferenceLogEntry is one row from compliance.ai_inference_log.
type InferenceLogEntry struct {
	ID              uuid.UUID
	OrgID           *uuid.UUID
	UserIDHash      string
	FeatureName     string
	ModelID         string
	Provider        string
	ContentHash     string
	OptInConfirmed  bool
	Blocked         bool
	Timestamp       time.Time
}

// TenantConfig is compliance.tenant_ai_config.
type TenantConfig struct {
	OrgID           uuid.UUID
	FeaturesEnabled map[string]bool
	AllowedModels   []string
	UpdatedAt       time.Time
	UpdatedBy       *uuid.UUID
}

// InsertLog appends one inference log row (blocked or allowed).
func InsertLog(ctx context.Context, pool *pgxpool.Pool, e InferenceLogEntry) error {
	_, err := pool.Exec(ctx, `
INSERT INTO compliance.ai_inference_log
  (org_id, user_id_hash, feature_name, model_id, provider, content_hash, opt_in_confirmed, blocked)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`, e.OrgID, e.UserIDHash, e.FeatureName, e.ModelID, e.Provider, e.ContentHash, e.OptInConfirmed, e.Blocked)
	return err
}

// ListLogsByUserHash returns recent log rows for a user hash (DSAR / FERPA disclosure).
func ListLogsByUserHash(ctx context.Context, pool *pgxpool.Pool, userHash string, limit int) ([]InferenceLogEntry, error) {
	if limit <= 0 || limit > 5000 {
		limit = 500
	}
	rows, err := pool.Query(ctx, `
SELECT id, org_id, user_id_hash, feature_name, model_id, provider,
       content_hash, opt_in_confirmed, blocked, "timestamp"
  FROM compliance.ai_inference_log
 WHERE user_id_hash = $1
 ORDER BY "timestamp" DESC
 LIMIT $2
`, userHash, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLogRows(rows)
}

// QueryLogs lists log rows with optional org filter (compliance admin).
func QueryLogs(ctx context.Context, pool *pgxpool.Pool, orgID *uuid.UUID, userHash string, limit int) ([]InferenceLogEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var rows pgx.Rows
	var err error
	switch {
	case orgID != nil && userHash != "":
		rows, err = pool.Query(ctx, `
SELECT id, org_id, user_id_hash, feature_name, model_id, provider,
       content_hash, opt_in_confirmed, blocked, "timestamp"
  FROM compliance.ai_inference_log
 WHERE org_id = $1 AND user_id_hash = $2
 ORDER BY "timestamp" DESC
 LIMIT $3
`, *orgID, userHash, limit)
	case orgID != nil:
		rows, err = pool.Query(ctx, `
SELECT id, org_id, user_id_hash, feature_name, model_id, provider,
       content_hash, opt_in_confirmed, blocked, "timestamp"
  FROM compliance.ai_inference_log
 WHERE org_id = $1
 ORDER BY "timestamp" DESC
 LIMIT $2
`, *orgID, limit)
	case userHash != "":
		rows, err = pool.Query(ctx, `
SELECT id, org_id, user_id_hash, feature_name, model_id, provider,
       content_hash, opt_in_confirmed, blocked, "timestamp"
  FROM compliance.ai_inference_log
 WHERE user_id_hash = $1
 ORDER BY "timestamp" DESC
 LIMIT $2
`, userHash, limit)
	default:
		rows, err = pool.Query(ctx, `
SELECT id, org_id, user_id_hash, feature_name, model_id, provider,
       content_hash, opt_in_confirmed, blocked, "timestamp"
  FROM compliance.ai_inference_log
 ORDER BY "timestamp" DESC
 LIMIT $1
`, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLogRows(rows)
}

func scanLogRows(rows pgx.Rows) ([]InferenceLogEntry, error) {
	var out []InferenceLogEntry
	for rows.Next() {
		var e InferenceLogEntry
		if err := rows.Scan(
			&e.ID, &e.OrgID, &e.UserIDHash, &e.FeatureName, &e.ModelID, &e.Provider,
			&e.ContentHash, &e.OptInConfirmed, &e.Blocked, &e.Timestamp,
		); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// GetOptOut returns the user's ai_processing_opt_out flag.
func GetOptOut(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (bool, error) {
	var optedOut bool
	err := pool.QueryRow(ctx, `
SELECT ai_processing_opt_out FROM "user".users WHERE id = $1
`, userID).Scan(&optedOut)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return optedOut, err
}

// SetOptOut updates ai_processing_opt_out for the user.
func SetOptOut(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, optedOut bool) error {
	tag, err := pool.Exec(ctx, `
UPDATE "user".users SET ai_processing_opt_out = $2 WHERE id = $1
`, userID, optedOut)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// GetTenantConfig loads tenant AI governance config; nil config means defaults (all features on).
func GetTenantConfig(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) (*TenantConfig, error) {
	var raw []byte
	var allowed []string
	var updatedBy *uuid.UUID
	var updatedAt time.Time
	err := pool.QueryRow(ctx, `
SELECT features_enabled, allowed_models, updated_at, updated_by
  FROM compliance.tenant_ai_config
 WHERE org_id = $1
`, orgID).Scan(&raw, &allowed, &updatedAt, &updatedBy)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	fe := map[string]bool{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &fe)
	}
	return &TenantConfig{
		OrgID:           orgID,
		FeaturesEnabled: fe,
		AllowedModels:   allowed,
		UpdatedAt:       updatedAt,
		UpdatedBy:       updatedBy,
	}, nil
}

// UpsertTenantConfig writes tenant AI governance settings.
func UpsertTenantConfig(ctx context.Context, pool *pgxpool.Pool, orgID, updatedBy uuid.UUID, features map[string]bool, allowed []string) error {
	raw, err := json.Marshal(features)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
INSERT INTO compliance.tenant_ai_config (org_id, features_enabled, allowed_models, updated_by)
VALUES ($1, $2, $3, $4)
ON CONFLICT (org_id) DO UPDATE SET
  features_enabled = EXCLUDED.features_enabled,
  allowed_models   = EXCLUDED.allowed_models,
  updated_at       = NOW(),
  updated_by       = EXCLUDED.updated_by
`, orgID, raw, allowed, updatedBy)
	return err
}

// HasFeatureAck returns true when the user acknowledged first-use disclosure for a feature.
func HasFeatureAck(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, featureKey string) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM settings.user_ai_feature_acknowledgements
   WHERE user_id = $1 AND feature_key = $2
)
`, userID, featureKey).Scan(&exists)
	return exists, err
}

// ListFeatureAcks returns all acknowledged feature keys for a user.
func ListFeatureAcks(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]string, error) {
	rows, err := pool.Query(ctx, `
SELECT feature_key FROM settings.user_ai_feature_acknowledgements
 WHERE user_id = $1 ORDER BY feature_key
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// AcknowledgeFeature records first-use disclosure confirmation.
func AcknowledgeFeature(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, featureKey string) error {
	_, err := pool.Exec(ctx, `
INSERT INTO settings.user_ai_feature_acknowledgements (user_id, feature_key)
VALUES ($1, $2)
ON CONFLICT (user_id, feature_key) DO NOTHING
`, userID, featureKey)
	return err
}
