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

// RegisterCouponReservationSweepJobs registers the MKTC.1 expired-reservation sweeper.
func RegisterCouponReservationSweepJobs(r *Registry, pool *pgxpool.Pool) {
	if r == nil || pool == nil {
		return
	}
	r.Register(scheduler.JobTypeCouponReservationSweep, HandlerFunc(func(ctx context.Context, _ json.RawMessage) error {
		n, err := repoBilling.ReleaseExpiredCouponReservations(ctx, pool, time.Now().UTC())
		if err != nil {
			return err
		}
		if n > 0 {
			slog.Info("scheduled.coupon_reservation_sweep", "released", n)
		}
		return nil
	}))
}
