// Package drm provides database access for DRM settings and license request tracking (plan 8.10).
package drm

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DRMType mirrors the storage.drm_type PostgreSQL enum.
type DRMType string

const (
	DRMTypeNone         DRMType = "none"
	DRMTypeWatermark    DRMType = "watermark_only"
	DRMTypeWidevine     DRMType = "widevine"
	DRMTypeFairPlay     DRMType = "fairplay"
)

// ObjectDRM holds the DRM configuration for a storage object.
type ObjectDRM struct {
	ID          uuid.UUID
	ObjectKey   string
	MimeType    string
	DRMType     DRMType
	DRMKeyID    *string
	DRMProvider *string
}

// LicenseRequest is a row from storage.drm_license_requests.
type LicenseRequest struct {
	ID           uuid.UUID
	ObjectID     uuid.UUID
	UserID       uuid.UUID
	IPAddress    string
	Granted      bool
	DenialReason *string
	RequestedAt  time.Time
}

// Anomaly is a (user, object) pair that has exceeded the redistribution threshold.
type Anomaly struct {
	UserID      uuid.UUID `json:"userId"`
	ObjectID    uuid.UUID `json:"objectId"`
	ObjectKey   string    `json:"objectKey"`
	DownloadCount int64   `json:"downloadCount"`
	WindowStart time.Time `json:"windowStart"`
}

// GetObjectDRM returns the DRM configuration for the given object ID.
// Returns nil, nil when not found.
func GetObjectDRM(ctx context.Context, pool *pgxpool.Pool, objectID uuid.UUID) (*ObjectDRM, error) {
	var o ObjectDRM
	err := pool.QueryRow(ctx, `
		SELECT id, object_key, mime_type, drm_type, drm_key_id, drm_provider
		FROM storage.objects
		WHERE id = $1 AND deleted_at IS NULL
	`, objectID).Scan(&o.ID, &o.ObjectKey, &o.MimeType, &o.DRMType, &o.DRMKeyID, &o.DRMProvider)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// SetObjectDRM updates the DRM fields on a storage.objects row.
func SetObjectDRM(ctx context.Context, pool *pgxpool.Pool, objectID uuid.UUID, drmType DRMType, keyID, provider *string) error {
	_, err := pool.Exec(ctx, `
		UPDATE storage.objects
		SET drm_type = $2, drm_key_id = $3, drm_provider = $4
		WHERE id = $1 AND deleted_at IS NULL
	`, objectID, string(drmType), keyID, provider)
	return err
}

// InsertLicenseRequest logs a DRM license request to the audit table.
func InsertLicenseRequest(ctx context.Context, pool *pgxpool.Pool, objectID, userID uuid.UUID, ipAddress string, granted bool, denialReason *string) error {
	var ip interface{}
	if ipAddress != "" {
		ip = ipAddress
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO storage.drm_license_requests
		  (object_id, user_id, ip_address, granted, denial_reason)
		VALUES ($1, $2, $3, $4, $5)
	`, objectID, userID, ip, granted, denialReason)
	return err
}

// DownloadCountLastHour returns how many times userID accessed objectID in the last hour.
func DownloadCountLastHour(ctx context.Context, pool *pgxpool.Pool, objectID, userID uuid.UUID) (int64, error) {
	var count int64
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM storage.drm_license_requests
		WHERE object_id = $1
		  AND user_id = $2
		  AND granted = true
		  AND requested_at >= NOW() - INTERVAL '1 hour'
	`, objectID, userID).Scan(&count)
	return count, err
}

// ListAnomalies returns (user, object) pairs where granted downloads in the last hour exceed threshold.
func ListAnomalies(ctx context.Context, pool *pgxpool.Pool, threshold int64) ([]Anomaly, error) {
	rows, err := pool.Query(ctx, `
		SELECT r.user_id, r.object_id, o.object_key, COUNT(*) AS download_count,
		       MIN(r.requested_at) AS window_start
		FROM storage.drm_license_requests r
		JOIN storage.objects o ON o.id = r.object_id
		WHERE r.granted = true
		  AND r.requested_at >= NOW() - INTERVAL '1 hour'
		GROUP BY r.user_id, r.object_id, o.object_key
		HAVING COUNT(*) >= $1
		ORDER BY download_count DESC
	`, threshold)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var anomalies []Anomaly
	for rows.Next() {
		var a Anomaly
		if err := rows.Scan(&a.UserID, &a.ObjectID, &a.ObjectKey, &a.DownloadCount, &a.WindowStart); err != nil {
			return nil, err
		}
		anomalies = append(anomalies, a)
	}
	return anomalies, rows.Err()
}
