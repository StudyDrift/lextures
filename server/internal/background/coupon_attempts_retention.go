package background

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	repoBilling "github.com/lextures/lextures/server/internal/repos/billing"
	"github.com/lextures/lextures/server/internal/scheduler"
)

// RegisterCouponAttemptsRetentionJobs registers the MKTC.7 attempt-log retention job.
func RegisterCouponAttemptsRetentionJobs(r *Registry, pool *pgxpool.Pool) {
	if r == nil || pool == nil {
		return
	}
	r.Register(scheduler.JobTypeCouponAttemptsRetention, HandlerFunc(func(ctx context.Context, _ json.RawMessage) error {
		n, err := repoBilling.DeleteExpiredCouponAttempts(ctx, pool, time.Now().UTC())
		if err != nil {
			return err
		}
		if n > 0 {
			slog.Info("scheduled.coupon_attempts_retention", "deleted", n)
		}
		return nil
	}))
}
