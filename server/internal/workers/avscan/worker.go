// Package avscan implements the antivirus scanning worker (plan 8.6).
package avscan

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/repos/avscanjobs"
	"github.com/lextures/lextures/server/internal/repos/board"
	"github.com/lextures/lextures/server/internal/repos/emailjobs"
	"github.com/lextures/lextures/server/internal/repos/notificationsinbox"
	"github.com/lextures/lextures/server/internal/repos/storageobjects"
	"github.com/lextures/lextures/server/internal/repos/useraudit"
	"github.com/lextures/lextures/server/internal/service/clamav"
	"github.com/lextures/lextures/server/internal/service/filestorage"
)

const maxAttemptsDefault = 3

// Worker processes queued AV scan jobs.
type Worker struct {
	Pool         *pgxpool.Pool
	Storage      filestorage.Driver
	LocalRoot    string
	ClamAV       *clamav.Client
	MaxAttempts  int
	EmailEnabled bool
}

// New creates a Worker with defaults.
func New(pool *pgxpool.Pool, storage filestorage.Driver, clam *clamav.Client) *Worker {
	return &Worker{
		Pool:        pool,
		Storage:     storage,
		ClamAV:      clam,
		MaxAttempts: maxAttemptsDefault,
	}
}

// ProcessNext claims and processes one queued job.
func (w *Worker) ProcessNext(ctx context.Context) (bool, error) {
	if w.Pool == nil || w.Storage == nil || w.ClamAV == nil {
		return false, fmt.Errorf("avscan: worker not configured")
	}
	job, err := avscanjobs.ClaimNext(ctx, w.Pool)
	if err != nil {
		return false, fmt.Errorf("avscan: claim: %w", err)
	}
	if job == nil {
		return false, nil
	}

	start := time.Now()
	obj, err := storageobjects.LoadByID(ctx, w.Pool, job.StorageObjectID)
	if err != nil || obj == nil {
		_ = avscanjobs.MarkFailed(ctx, w.Pool, job.ID, "object not found", w.MaxAttempts)
		return true, err
	}

	slog.Info("avscan: start", "job_id", job.ID, "object_id", obj.ID, "key", obj.ObjectKey)

	infected, procErr := w.scanObject(ctx, obj)
	if procErr != nil {
		slog.Error("avscan: failed", "job_id", job.ID, "err", procErr)
		attempts, _ := storageobjects.BumpScanAttempts(ctx, w.Pool, obj.ID)
		if int(attempts) >= w.MaxAttempts {
			_ = storageobjects.MarkScanError(ctx, w.Pool, obj.ID, attempts)
			_ = avscanjobs.MarkFailed(ctx, w.Pool, job.ID, procErr.Error(), w.MaxAttempts)
			w.notifyScanFailure(ctx, obj)
		} else {
			_ = avscanjobs.MarkFailed(ctx, w.Pool, job.ID, procErr.Error(), w.MaxAttempts)
		}
		return true, procErr
	}

	_ = avscanjobs.MarkDone(ctx, w.Pool, job.ID)
	if infected {
		slog.Warn("avscan: quarantined",
			"job_id", job.ID,
			"object_id", obj.ID,
			"scan_duration_ms", time.Since(start).Milliseconds(),
		)
	} else {
		slog.Info("avscan: clean",
			"job_id", job.ID,
			"object_id", obj.ID,
			"scan_duration_ms", time.Since(start).Milliseconds(),
		)
	}
	return true, nil
}

func (w *Worker) scanObject(ctx context.Context, obj *storageobjects.Object) (infected bool, err error) {
	rc, err := w.Storage.GetObject(ctx, obj.ObjectKey)
	if err != nil {
		return false, fmt.Errorf("get object: %w", err)
	}
	defer func() { _ = rc.Close() }()

	result, err := w.ClamAV.ScanStream(ctx, rc)
	if err != nil {
		return false, fmt.Errorf("clamav scan: %w", err)
	}
	if result.Clean {
		if err := storageobjects.MarkClean(ctx, w.Pool, obj.ID); err != nil {
			return false, err
		}
		_, _ = board.SyncAttachmentScanByStorageKey(ctx, w.Pool, obj.ObjectKey, board.ScanClean)
		return false, nil
	}
	if err := w.quarantine(ctx, obj, result.VirusName); err != nil {
		return true, err
	}
	w.syncBoardAttachmentBlocked(ctx, obj.ObjectKey)
	return true, nil
}

func (w *Worker) syncBoardAttachmentBlocked(ctx context.Context, storageKey string) {
	refs, err := board.SyncAttachmentScanByStorageKey(ctx, w.Pool, storageKey, board.ScanBlocked)
	if err != nil || len(refs) == 0 {
		return
	}
	for _, ref := range refs {
		pid := ref.PostID
		_, _ = board.CreateReport(ctx, w.Pool, ref.CourseCode, ref.BoardID, nil, &pid, nil,
			"Attachment blocked by antivirus scan", board.ReportKindAVBlocked)
		tid, _ := uuid.Parse(ref.PostID)
		_ = board.InsertModerationLog(ctx, w.Pool, ref.BoardID, nil, board.ModActionAVBlocked, board.TargetPost, &tid, "scan_status=blocked")
		actionURL := "/courses/" + ref.CourseCode + "/boards/" + ref.BoardID + "?moderation=1"
		rows, qerr := w.Pool.Query(ctx, `
			SELECT ce.user_id
			FROM course.course_enrollments ce
			INNER JOIN course.courses c ON c.id = ce.course_id
			INNER JOIN course.enrollment_roles er ON er.role_key = ce.role AND er.is_staff = true
			WHERE c.course_code = $1 AND ce.status = 'active'
		`, ref.CourseCode)
		if qerr != nil {
			continue
		}
		for rows.Next() {
			var staffID uuid.UUID
			if rows.Scan(&staffID) == nil {
				_, _ = notificationsinbox.Insert(ctx, w.Pool, staffID, "board_moderation_av",
					"Board attachment blocked",
					"An attachment was blocked by scanning and needs review.",
					actionURL)
			}
		}
		rows.Close()
	}
}

func (w *Worker) quarantine(ctx context.Context, obj *storageobjects.Object, virusName string) error {
	qKey := clamav.QuarantineKey(obj.ObjectKey)
	if err := w.moveStorage(ctx, obj.ObjectKey, qKey); err != nil {
		return err
	}
	if err := storageobjects.MarkQuarantined(ctx, w.Pool, obj.ID, virusName, qKey); err != nil {
		return err
	}
	if obj.CourseID != nil && obj.UploadedBy != nil {
		_ = useraudit.Insert(ctx, w.Pool, *obj.UploadedBy, *obj.CourseID, nil, "file_quarantined")
	}
	w.notifyQuarantine(ctx, obj, virusName)
	return nil
}

func (w *Worker) moveStorage(ctx context.Context, srcKey, destKey string) error {
	if s3d, ok := w.Storage.(*filestorage.S3Driver); ok {
		return filestorage.CopyObjectS3(ctx, s3d, srcKey, destKey)
	}
	if w.LocalRoot != "" {
		return filestorage.MoveObjectLocal(w.LocalRoot, srcKey, destKey)
	}
	return filestorage.MoveObject(ctx, w.Storage, srcKey, destKey)
}

func (w *Worker) notifyQuarantine(ctx context.Context, obj *storageobjects.Object, virusName string) {
	if !w.EmailEnabled || obj.UploadedBy == nil {
		return
	}
	vars := map[string]string{
		"virus_name": virusName,
		"object_key": obj.ObjectKey,
	}
	_, _ = emailjobs.Enqueue(ctx, w.Pool, *obj.UploadedBy, "file_quarantined",
		"File quarantined — security threat detected",
		"file_quarantined", vars)
	// Notify course instructor when course-linked.
	if obj.CourseID != nil {
		w.enqueueInstructorAlert(ctx, *obj.CourseID, virusName)
	}
}

func (w *Worker) enqueueInstructorAlert(ctx context.Context, courseID uuid.UUID, virusName string) {
	rows, err := w.Pool.Query(ctx, `
		SELECT user_id FROM course.course_enrollments
		WHERE course_id = $1 AND role IN ('teacher', 'ta')`, courseID)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var uid uuid.UUID
		if err := rows.Scan(&uid); err != nil {
			continue
		}
		vars := map[string]string{"virus_name": virusName}
		_, _ = emailjobs.Enqueue(ctx, w.Pool, uid, "file_quarantined_instructor",
			"Student file quarantined — security threat",
			"file_quarantined_instructor", vars)
	}
}

func (w *Worker) notifyScanFailure(ctx context.Context, obj *storageobjects.Object) {
	slog.Warn("avscan: permanent scan failure — admin attention", "object_id", obj.ID, "key", obj.ObjectKey)
}

// EnqueueForObject registers an AV scan job for a storage object.
func EnqueueForObject(ctx context.Context, pool *pgxpool.Pool, objectID uuid.UUID) (uuid.UUID, error) {
	return avscanjobs.Enqueue(ctx, pool, objectID)
}

// RegisterAndEnqueue upserts a storage object after upload and queues a scan when AV is enabled.
func RegisterAndEnqueue(
	ctx context.Context,
	pool *pgxpool.Pool,
	tenantID uuid.UUID,
	courseID *uuid.UUID,
	objectKey, bucket, mime string,
	size int64,
	uploadedBy *uuid.UUID,
	avEnabled bool,
) (uuid.UUID, error) {
	id, err := storageobjects.Upsert(ctx, pool, tenantID, courseID, objectKey, bucket, mime, size, uploadedBy, avEnabled)
	if err != nil {
		return uuid.Nil, err
	}
	if !avEnabled {
		return id, nil
	}
	_, err = avscanjobs.Enqueue(ctx, pool, id)
	return id, err
}

// IsBlockedDownload reports whether serving the object key should be denied for AV.
func IsBlockedDownload(ctx context.Context, pool *pgxpool.Pool, objectKey string, avEnabled bool) (bool, string, error) {
	if !avEnabled || pool == nil {
		return false, "", nil
	}
	obj, err := storageobjects.LoadByObjectKey(ctx, pool, objectKey)
	if err != nil {
		return false, "", err
	}
	if obj == nil {
		return false, "", nil
	}
	if strings.HasPrefix(obj.ObjectKey, "quarantine/") {
		return true, "quarantined", nil
	}
	if !obj.IsAccessible(avEnabled) {
		switch obj.ScanStatus {
		case storageobjects.ScanQuarantined:
			return true, "quarantined", nil
		case storageobjects.ScanPending:
			return true, "scan_pending", nil
		default:
			return true, "unavailable", nil
		}
	}
	return false, "", nil
}
