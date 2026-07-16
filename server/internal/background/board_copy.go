package background

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/repos/board"
	"github.com/lextures/lextures/server/internal/repos/jobqueue"
)

// JobTypeBoardCopy is the durable queue type for large full board copies (VC.8).
const JobTypeBoardCopy = "board.copy"

// BoardCopyPayload is the job payload for board.copy.
type BoardCopyPayload struct {
	JobID            string `json:"jobId"`
	SourceCourseCode string `json:"sourceCourseCode"`
	SourceBoardID    string `json:"sourceBoardId"`
	TargetCourseCode string `json:"targetCourseCode"`
	CreatedBy        string `json:"createdBy"`
	Mode             string `json:"mode"`
	Title            string `json:"title"`
	Description      string `json:"description"`
}

// BoardCopyBlobCopier copies attachment bytes for async full copies.
type BoardCopyBlobCopier interface {
	CopyBlob(ctx context.Context, srcKey, destKey string) error
}

type localBoardBlobCopier struct {
	root string
}

func (c localBoardBlobCopier) CopyBlob(_ context.Context, srcKey, destKey string) error {
	root := strings.TrimSpace(c.root)
	if root == "" {
		return fmt.Errorf("board.copy: CourseFilesRoot not configured")
	}
	srcKey = strings.TrimSpace(srcKey)
	destKey = strings.TrimSpace(destKey)
	srcPath := filepath.Join(root, filepath.FromSlash(srcKey))
	destPath := filepath.Join(root, filepath.FromSlash(destKey))
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	in, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	_, err = io.Copy(out, in)
	return err
}

type boardCopyHandler struct {
	pool   *pgxpool.Pool
	copier BoardCopyBlobCopier
}

// RegisterBoardCopyJob registers the board.copy handler.
func RegisterBoardCopyJob(r *Registry, pool *pgxpool.Pool, copier BoardCopyBlobCopier) {
	if r == nil || pool == nil {
		return
	}
	r.Register(JobTypeBoardCopy, boardCopyHandler{pool: pool, copier: copier})
}

// EnqueueBoardCopy queues a board copy job.
func EnqueueBoardCopy(ctx context.Context, pool *pgxpool.Pool, p BoardCopyPayload) (uuid.UUID, error) {
	return jobqueue.Enqueue(ctx, pool, jobqueue.EnqueueParams{
		JobType:   JobTypeBoardCopy,
		Payload:   p,
		Priority:  5,
		UniqueKey: "board-copy:" + p.JobID,
	})
}

func (h boardCopyHandler) Execute(ctx context.Context, payload json.RawMessage) error {
	var p BoardCopyPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("board.copy: invalid payload: %w", err)
	}
	if p.JobID == "" {
		return fmt.Errorf("board.copy: jobId required")
	}
	createdBy, err := uuid.Parse(p.CreatedBy)
	if err != nil {
		return fmt.Errorf("board.copy: invalid createdBy")
	}
	_ = board.UpdateCopyJobProgress(ctx, h.pool, p.JobID, "running", 1, nil, "")

	created, err := board.CopyBoard(ctx, h.pool, p.SourceCourseCode, p.SourceBoardID, p.TargetCourseCode, createdBy, board.CopyBoardOpts{
		Mode:        p.Mode,
		Title:       p.Title,
		Description: p.Description,
		AuthorID:    createdBy,
		BlobCopier:  h.copier,
		OnProgress: func(pct int) {
			_ = board.UpdateCopyJobProgress(ctx, h.pool, p.JobID, "running", pct, nil, "")
		},
	})
	if err != nil {
		_ = board.UpdateCopyJobProgress(ctx, h.pool, p.JobID, "failed", 100, nil, err.Error())
		return err
	}
	if created == nil {
		err = fmt.Errorf("board.copy: source board not found")
		_ = board.UpdateCopyJobProgress(ctx, h.pool, p.JobID, "failed", 100, nil, err.Error())
		return err
	}
	rid, err := uuid.Parse(created.ID)
	if err != nil {
		return err
	}
	if err := board.UpdateCopyJobProgress(ctx, h.pool, p.JobID, "completed", 100, &rid, ""); err != nil {
		slog.Warn("board.copy: update job completed", "err", err)
	}
	return nil
}
