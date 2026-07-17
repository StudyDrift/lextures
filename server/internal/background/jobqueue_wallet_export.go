package background

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/logging"
	"github.com/lextures/lextures/server/internal/repos/jobqueue"
	"github.com/lextures/lextures/server/internal/service/credentialwallet"
)

// JobTypeWalletExport is the durable queue type for T09 portable wallet bundles.
const JobTypeWalletExport = "wallet.export"

// WalletExportPayload identifies one export row to build.
type WalletExportPayload struct {
	ExportID uuid.UUID `json:"exportId"`
}

type walletExportHandler struct {
	pool *pgxpool.Pool
	cfg  config.Config
}

func (h walletExportHandler) Execute(ctx context.Context, payload json.RawMessage) error {
	var p WalletExportPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("wallet.export: bad payload: %w", err)
	}
	if p.ExportID == uuid.Nil {
		return fmt.Errorf("wallet.export: missing exportId")
	}
	err := credentialwallet.ProcessExport(ctx, h.pool, h.cfg, p.ExportID)
	if err == nil {
		logging.GlobalWalletMetrics.IncExport()
	}
	return err
}

// RegisterWalletExportJob registers the wallet.export handler.
func RegisterWalletExportJob(r *Registry, pool *pgxpool.Pool, cfg config.Config) {
	if r == nil {
		return
	}
	r.Register(JobTypeWalletExport, walletExportHandler{pool: pool, cfg: cfg})
}

// EnqueueWalletExport queues a portable wallet export build.
func EnqueueWalletExport(ctx context.Context, pool *pgxpool.Pool, exportID uuid.UUID) (uuid.UUID, error) {
	if pool == nil || exportID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("wallet.export: missing pool or export")
	}
	return jobqueue.Enqueue(ctx, pool, jobqueue.EnqueueParams{
		JobType:     JobTypeWalletExport,
		Payload:     WalletExportPayload{ExportID: exportID},
		Priority:    5,
		MaxAttempts: 3,
		UniqueKey:   "wallet-export:" + exportID.String(),
	})
}
