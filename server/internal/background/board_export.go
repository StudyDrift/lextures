package background

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/repos/board"
	"github.com/lextures/lextures/server/internal/repos/jobqueue"
	"github.com/lextures/lextures/server/internal/service/boardexport"
	"github.com/lextures/lextures/server/internal/service/filestorage"
)

// JobTypeBoardExport is the durable queue type for board exports (VC.9).
const JobTypeBoardExport = "board.export"

// BoardExportPayload is the job payload for board.export.
type BoardExportPayload struct {
	JobID             string `json:"jobId"`
	CourseCode        string `json:"courseCode"`
	BoardID           string `json:"boardId"`
	RequestedBy       string `json:"requestedBy"`
	Format            string `json:"format"`
	IncludeModeration bool   `json:"includeModeration"`
	CanManage         bool   `json:"canManage"`
}

type boardExportHandler struct {
	pool    *pgxpool.Pool
	storage filestorage.Driver
}

// RegisterBoardExportJob registers the board.export handler.
func RegisterBoardExportJob(r *Registry, pool *pgxpool.Pool, storage filestorage.Driver) {
	if r == nil || pool == nil {
		return
	}
	r.Register(JobTypeBoardExport, boardExportHandler{pool: pool, storage: storage})
}

// EnqueueBoardExport queues a board export job.
func EnqueueBoardExport(ctx context.Context, pool *pgxpool.Pool, p BoardExportPayload) (uuid.UUID, error) {
	return jobqueue.Enqueue(ctx, pool, jobqueue.EnqueueParams{
		JobType:   JobTypeBoardExport,
		Payload:   p,
		Priority:  5,
		UniqueKey: "board-export:" + p.JobID,
	})
}

func (h boardExportHandler) Execute(ctx context.Context, payload json.RawMessage) error {
	var p BoardExportPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("board.export: invalid payload: %w", err)
	}
	if p.JobID == "" {
		return fmt.Errorf("board.export: jobId required")
	}
	viewerID, err := uuid.Parse(p.RequestedBy)
	if err != nil {
		return fmt.Errorf("board.export: invalid requestedBy")
	}
	_ = board.UpdateExportJobStatus(ctx, h.pool, p.JobID, board.ExportStatusRunning, nil, "")

	caps := board.Capabilities{CanView: true, CanManage: p.CanManage}
	res, err := boardexport.Build(ctx, h.pool, boardexport.BuildOpts{
		CourseCode:        p.CourseCode,
		BoardID:           p.BoardID,
		ViewerID:          viewerID,
		Format:            p.Format,
		IncludeModeration: p.IncludeModeration,
		Caps:              caps,
	})
	if err != nil {
		_ = board.UpdateExportJobStatus(ctx, h.pool, p.JobID, board.ExportStatusFailed, nil, err.Error())
		return err
	}

	key := fmt.Sprintf("boards/%s/%s/exports/%s.%s", p.CourseCode, p.BoardID, p.JobID, res.Extension)
	storage := h.storage
	if storage == nil {
		storage = &filestorage.LocalDriver{Root: "data/course-files"}
	}
	if err := storage.PutObject(ctx, key, bytes.NewReader(res.Bytes), int64(len(res.Bytes)), res.ContentType); err != nil {
		_ = board.UpdateExportJobStatus(ctx, h.pool, p.JobID, board.ExportStatusFailed, nil, err.Error())
		return err
	}
	if err := board.UpdateExportJobStatus(ctx, h.pool, p.JobID, board.ExportStatusDone, &key, ""); err != nil {
		slog.Warn("board.export: update job done", "err", err)
		return err
	}
	return nil
}

// LocalBoardExportStorageFromRoot builds a local driver for export registration.
func LocalBoardExportStorageFromRoot(root string) filestorage.Driver {
	root = strings.TrimSpace(root)
	if root == "" {
		root = "data/course-files"
	}
	return &filestorage.LocalDriver{Root: root}
}
