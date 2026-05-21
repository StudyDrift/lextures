// Package h5pextract extracts uploaded .h5p archives into object storage (plan 8.12).
package h5pextract

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/h5p"
	"github.com/lextures/lextures/server/internal/repos/h5ppackages"
	"github.com/lextures/lextures/server/internal/repos/storageobjects"
	"github.com/lextures/lextures/server/internal/service/filestorage"
)

// Worker processes pending H5P package extractions.
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
		return false, fmt.Errorf("h5pextract: missing pool or storage")
	}
	id, err := h5ppackages.ClaimPendingForExtract(ctx, w.Pool)
	if err != nil {
		return false, err
	}
	if id == nil {
		return false, nil
	}
	pkg, err := h5ppackages.LoadByIDGlobal(ctx, w.Pool, *id)
	if err != nil || pkg == nil {
		return true, err
	}
	obj, err := storageobjects.LoadByID(ctx, w.Pool, pkg.StorageObjectID)
	if err != nil || obj == nil {
		_ = h5ppackages.MarkExtractFailed(ctx, w.Pool, *id, "storage object not found")
		return true, err
	}
	if obj.ScanStatus == storageobjects.ScanQuarantined {
		_ = h5ppackages.MarkExtractFailed(ctx, w.Pool, *id, "package quarantined by antivirus scan")
		return true, nil
	}
	if obj.ScanStatus == storageobjects.ScanPending {
		return false, nil
	}
	tmp, err := h5p.DownloadToTemp(ctx, w.Storage, obj.ObjectKey)
	if err != nil {
		_ = h5ppackages.MarkExtractFailed(ctx, w.Pool, *id, err.Error())
		return true, err
	}
	defer func() { _ = os.Remove(tmp) }()
	if err := h5p.ExtractZipToStorage(ctx, w.Storage, tmp, pkg.AssetsPrefix); err != nil {
		_ = h5ppackages.MarkExtractFailed(ctx, w.Pool, *id, err.Error())
		slog.Warn("h5p extract failed", "package_id", *id, "err", err)
		return true, err
	}
	if err := h5ppackages.MarkExtractReady(ctx, w.Pool, *id); err != nil {
		return true, err
	}
	slog.Info("h5p extract ready", "package_id", *id, "content_type", pkg.ContentType)
	return true, nil
}

// ExtractSync extracts immediately (used right after upload for small packages).
func ExtractSync(ctx context.Context, pool *pgxpool.Pool, storage filestorage.Driver, packageID uuid.UUID, zipPath string) error {
	pkg, err := h5ppackages.LoadByIDGlobal(ctx, pool, packageID)
	if err != nil || pkg == nil {
		return fmt.Errorf("h5pextract: package not found")
	}
	if err := h5p.ExtractZipToStorage(ctx, storage, zipPath, pkg.AssetsPrefix); err != nil {
		_ = h5ppackages.MarkExtractFailed(ctx, pool, packageID, err.Error())
		return err
	}
	return h5ppackages.MarkExtractReady(ctx, pool, packageID)
}
