// Package scormextract extracts uploaded SCORM archives into object storage (plan 2.14).
package scormextract

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/repos/scormpackages"
	"github.com/lextures/lextures/server/internal/repos/scormscos"
	"github.com/lextures/lextures/server/internal/repos/storageobjects"
	"github.com/lextures/lextures/server/internal/scorm"
	"github.com/lextures/lextures/server/internal/service/filestorage"
)

// Worker processes pending SCORM package extractions.
type Worker struct {
	Pool    *pgxpool.Pool
	Storage filestorage.Driver
}

// New creates a Worker.
func New(pool *pgxpool.Pool, storage filestorage.Driver) *Worker {
	return &Worker{Pool: pool, Storage: storage}
}

// ProcessNext extracts one pending package. Returns false when queue is empty.
func (w *Worker) ProcessNext(ctx context.Context) (bool, error) {
	if w.Pool == nil || w.Storage == nil {
		return false, fmt.Errorf("scormextract: missing pool or storage")
	}
	id, err := scormpackages.ClaimPendingForExtract(ctx, w.Pool)
	if err != nil {
		return false, err
	}
	if id == nil {
		return false, nil
	}
	pkg, err := scormpackages.LoadByIDGlobal(ctx, w.Pool, *id)
	if err != nil || pkg == nil {
		return true, err
	}
	obj, err := storageobjects.LoadByID(ctx, w.Pool, pkg.StorageObjectID)
	if err != nil || obj == nil {
		_ = scormpackages.MarkExtractFailed(ctx, w.Pool, *id, "storage object not found")
		return true, err
	}
	if obj.ScanStatus == storageobjects.ScanQuarantined {
		_ = scormpackages.MarkExtractFailed(ctx, w.Pool, *id, "package quarantined by antivirus scan")
		return true, nil
	}
	if obj.ScanStatus == storageobjects.ScanPending {
		return false, nil
	}
	tmp, err := scorm.DownloadToTemp(ctx, w.Storage, obj.ObjectKey)
	if err != nil {
		_ = scormpackages.MarkExtractFailed(ctx, w.Pool, *id, err.Error())
		return true, err
	}
	defer func() { _ = os.Remove(tmp) }()
	if err := scorm.ExtractZipToStorage(ctx, w.Storage, tmp, pkg.AssetsPrefix); err != nil {
		_ = scormpackages.MarkExtractFailed(ctx, w.Pool, *id, err.Error())
		slog.Warn("scorm extract failed", "package_id", *id, "err", err)
		return true, err
	}
	if err := scormpackages.MarkExtractReady(ctx, w.Pool, *id); err != nil {
		return true, err
	}
	slog.Info("scorm extract ready", "package_id", *id, "package_type", pkg.PackageType)
	return true, nil
}

// ExtractSync extracts immediately after upload when AV scanning is disabled.
func ExtractSync(ctx context.Context, pool *pgxpool.Pool, storage filestorage.Driver, packageID uuid.UUID, zipPath string) error {
	pkg, err := scormpackages.LoadByIDGlobal(ctx, pool, packageID)
	if err != nil || pkg == nil {
		return fmt.Errorf("scormextract: package not found")
	}
	if err := scorm.ExtractZipToStorage(ctx, storage, zipPath, pkg.AssetsPrefix); err != nil {
		_ = scormpackages.MarkExtractFailed(ctx, pool, packageID, err.Error())
		return err
	}
	return scormpackages.MarkExtractReady(ctx, pool, packageID)
}

// InsertScosFromManifest creates SCO rows after manifest parse at upload time.
func InsertScosFromManifest(ctx context.Context, pool *pgxpool.Pool, packageID uuid.UUID, manifest scorm.Manifest) error {
	for _, sco := range manifest.Scos {
		id := uuid.New()
		if err := scormscos.Insert(ctx, pool, id, packageID, sco.Identifier, sco.Title, sco.LaunchHref, sco.Mastery); err != nil {
			return err
		}
	}
	return nil
}
