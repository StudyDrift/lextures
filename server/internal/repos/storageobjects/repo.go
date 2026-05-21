// Package storageobjects provides DB access for storage.objects (plan 8.1 / 8.6).
package storageobjects

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ScanStatus mirrors storage.scan_status.
type ScanStatus string

const (
	ScanPending     ScanStatus = "pending"
	ScanClean       ScanStatus = "clean"
	ScanQuarantined ScanStatus = "quarantined"
	ScanError       ScanStatus = "scan_error"
)

// Object is a row from storage.objects.
type Object struct {
	ID              uuid.UUID
	TenantID        uuid.UUID
	CourseID        *uuid.UUID
	ObjectKey       string
	Bucket          string
	MimeType        string
	SizeBytes       int64
	UploadedBy      *uuid.UUID
	ScanStatus      ScanStatus
	ScanCompletedAt *time.Time
	VirusName       *string
	ScanAttempts    int16
	CreatedAt       time.Time
}

// Upsert registers or updates a storage object after upload. When avEnabled, scan_status is pending.
func Upsert(
	ctx context.Context,
	pool *pgxpool.Pool,
	tenantID uuid.UUID,
	courseID *uuid.UUID,
	objectKey, bucket, mimeType string,
	sizeBytes int64,
	uploadedBy *uuid.UUID,
	avEnabled bool,
) (uuid.UUID, error) {
	status := ScanClean
	if avEnabled {
		status = ScanPending
	}
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
		INSERT INTO storage.objects
		  (tenant_id, course_id, object_key, bucket, mime_type, size_bytes, uploaded_by, scan_status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (object_key) DO UPDATE SET
		  mime_type = EXCLUDED.mime_type,
		  size_bytes = EXCLUDED.size_bytes,
		  deleted_at = NULL,
		  scan_status = CASE
		    WHEN storage.objects.scan_status = 'quarantined' THEN storage.objects.scan_status
		    WHEN $9::bool THEN 'pending'::storage.scan_status
		    ELSE storage.objects.scan_status
		  END
		RETURNING id`,
		tenantID, courseID, objectKey, bucket, mimeType, sizeBytes, uploadedBy, status, avEnabled,
	).Scan(&id)
	return id, err
}

// LoadByID fetches an object by primary key.
func LoadByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Object, error) {
	return scanObject(pool.QueryRow(ctx, `
		SELECT id, tenant_id, course_id, object_key, bucket, mime_type, size_bytes, uploaded_by,
		       scan_status, scan_completed_at, virus_name, scan_attempts, created_at
		FROM storage.objects
		WHERE id = $1 AND deleted_at IS NULL`, id))
}

// LoadByObjectKey fetches an object by storage key.
func LoadByObjectKey(ctx context.Context, pool *pgxpool.Pool, objectKey string) (*Object, error) {
	return scanObject(pool.QueryRow(ctx, `
		SELECT id, tenant_id, course_id, object_key, bucket, mime_type, size_bytes, uploaded_by,
		       scan_status, scan_completed_at, virus_name, scan_attempts, created_at
		FROM storage.objects
		WHERE object_key = $1 AND deleted_at IS NULL`, objectKey))
}

// MarkClean sets scan_status to clean after a successful AV scan.
func MarkClean(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) error {
	_, err := pool.Exec(ctx, `
		UPDATE storage.objects
		SET scan_status = 'clean', scan_completed_at = now(), virus_name = NULL, scan_attempts = 0
		WHERE id = $1`, id)
	return err
}

// MarkQuarantined moves metadata to quarantined and records the virus name.
func MarkQuarantined(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, virusName, newObjectKey string) error {
	_, err := pool.Exec(ctx, `
		UPDATE storage.objects
		SET scan_status = 'quarantined',
		    scan_completed_at = now(),
		    virus_name = $2,
		    object_key = $3
		WHERE id = $1`, id, virusName, newObjectKey)
	return err
}

// MarkScanError records a permanent scan failure.
func MarkScanError(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, attempts int16) error {
	_, err := pool.Exec(ctx, `
		UPDATE storage.objects
		SET scan_status = 'scan_error', scan_attempts = $2, scan_completed_at = now()
		WHERE id = $1`, id, attempts)
	return err
}

// BumpScanAttempts increments scan_attempts while keeping status pending.
func BumpScanAttempts(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (int16, error) {
	var attempts int16
	err := pool.QueryRow(ctx, `
		UPDATE storage.objects
		SET scan_attempts = scan_attempts + 1
		WHERE id = $1
		RETURNING scan_attempts`, id).Scan(&attempts)
	return attempts, err
}

// ListPendingLegacy returns object IDs with pending scan (for bulk admin scan).
func ListPendingLegacy(ctx context.Context, pool *pgxpool.Pool, limit int) ([]uuid.UUID, error) {
	if limit <= 0 {
		limit = 500
	}
	rows, err := pool.Query(ctx, `
		SELECT id FROM storage.objects
		WHERE deleted_at IS NULL AND scan_status = 'pending'
		ORDER BY created_at
		LIMIT $1`, limit)
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

// QuarantineRow is a quarantined file for the admin list.
type QuarantineRow struct {
	ObjectID     uuid.UUID
	ObjectKey    string
	VirusName    *string
	UploadedBy   *uuid.UUID
	UploaderName *string
	UploaderEmail *string
	CourseID     *uuid.UUID
	CourseCode   *string
	CourseTitle  *string
	CreatedAt    time.Time
}

// ListQuarantined returns quarantined objects for admin review.
func ListQuarantined(ctx context.Context, pool *pgxpool.Pool, limit int) ([]QuarantineRow, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := pool.Query(ctx, `
		SELECT o.id, o.object_key, o.virus_name, o.uploaded_by,
		       u.display_name, u.email,
		       o.course_id, c.course_code, c.title, o.created_at
		FROM storage.objects o
		LEFT JOIN "user".users u ON u.id = o.uploaded_by
		LEFT JOIN course.courses c ON c.id = o.course_id
		WHERE o.deleted_at IS NULL AND o.scan_status = 'quarantined'
		ORDER BY o.created_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []QuarantineRow
	for rows.Next() {
		var r QuarantineRow
		if err := rows.Scan(
			&r.ObjectID, &r.ObjectKey, &r.VirusName, &r.UploadedBy,
			&r.UploaderName, &r.UploaderEmail,
			&r.CourseID, &r.CourseCode, &r.CourseTitle, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// SoftDelete marks an object deleted (admin permanent delete).
func SoftDelete(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) error {
	_, err := pool.Exec(ctx, `
		UPDATE storage.objects SET deleted_at = now() WHERE id = $1`, id)
	return err
}

// ReleaseFromQuarantine moves a false-positive back to clean with a new object key.
func ReleaseFromQuarantine(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, newObjectKey string) error {
	_, err := pool.Exec(ctx, `
		UPDATE storage.objects
		SET scan_status = 'clean',
		    virus_name = NULL,
		    object_key = $2,
		    scan_completed_at = now()
		WHERE id = $1 AND scan_status = 'quarantined'`, id, newObjectKey)
	return err
}

// IsAccessible returns true when the object may be served to non-uploader users.
func (o *Object) IsAccessible(avEnabled bool) bool {
	if o == nil {
		return false
	}
	if !avEnabled {
		return true
	}
	switch o.ScanStatus {
	case ScanClean:
		return true
	case ScanPending, ScanError:
		return false
	case ScanQuarantined:
		return false
	default:
		return false
	}
}

func scanObject(row pgx.Row) (*Object, error) {
	var o Object
	var status string
	err := row.Scan(
		&o.ID, &o.TenantID, &o.CourseID, &o.ObjectKey, &o.Bucket, &o.MimeType, &o.SizeBytes, &o.UploadedBy,
		&status, &o.ScanCompletedAt, &o.VirusName, &o.ScanAttempts, &o.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	o.ScanStatus = ScanStatus(status)
	return &o, nil
}
