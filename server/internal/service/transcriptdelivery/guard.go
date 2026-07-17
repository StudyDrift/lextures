package transcriptdelivery

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	transcriptsrepo "github.com/lextures/lextures/server/internal/repos/transcripts"
)

// GuardResult is the outcome of the pre-send release guard.
type GuardResult struct {
	OK     bool
	Reason string
	OnHold bool
}

// ReleaseGuard re-checks holds, consent, and payment immediately before send (T06 FR-5).
func ReleaseGuard(ctx context.Context, pool *pgxpool.Pool, order *transcriptsrepo.Order) (GuardResult, error) {
	if order == nil {
		return GuardResult{OK: false, Reason: "order missing"}, nil
	}
	cfg, err := transcriptsrepo.GetConfig(ctx, pool)
	if err != nil {
		return GuardResult{}, err
	}
	blocked, err := transcriptsrepo.HasBlockingHold(ctx, pool, order.UserID, order.OrgID)
	if err != nil {
		return GuardResult{}, err
	}
	if blocked {
		return GuardResult{OK: false, Reason: "active hold blocks delivery", OnHold: true}, nil
	}
	consentOK, err := transcriptsrepo.ConsentSatisfiedForOrder(ctx, pool, cfg, order)
	if err != nil {
		return GuardResult{}, err
	}
	if !consentOK {
		return GuardResult{OK: false, Reason: "consent not satisfied"}, nil
	}
	paymentOK, err := transcriptsrepo.PaymentSatisfiedForOrder(ctx, pool, cfg, order)
	if err != nil {
		return GuardResult{}, err
	}
	if !paymentOK {
		return GuardResult{OK: false, Reason: "payment not satisfied"}, nil
	}
	return GuardResult{OK: true}, nil
}

// ErrTransient marks an adapter failure that should be retried by the job queue.
var ErrTransient = errors.New("transient delivery failure")

func wrapTransient(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %v", ErrTransient, err)
}
