package billing

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stripe/stripe-go/v82"
	checkoutsession "github.com/stripe/stripe-go/v82/checkout/session"

	repoPayments "github.com/lextures/lextures/server/internal/repos/payments"
	"github.com/lextures/lextures/server/internal/service/paymentprovider"
)

// StalePendingAge is how long a Stripe Checkout Session may stay pending before it is reconciled.
// Stripe expires unpaid sessions after 24 hours by default.
const StalePendingAge = 25 * time.Hour

// CheckoutSessionState is the subset of a Stripe Checkout Session needed to reconcile a stale row.
type CheckoutSessionState struct {
	Status        string // open | complete | expired
	PaymentStatus string // paid | unpaid | no_payment_required
	Missing       bool   // Stripe no longer knows the session
}

// CheckoutSessionLookup fetches the live state of a Checkout Session.
type CheckoutSessionLookup func(ctx context.Context, sessionID string) (CheckoutSessionState, error)

// StripeCheckoutSessionLookup returns a lookup backed by the Stripe API.
func StripeCheckoutSessionLookup(secretKey string) CheckoutSessionLookup {
	return func(_ context.Context, sessionID string) (CheckoutSessionState, error) {
		stripe.Key = secretKey
		sess, err := checkoutsession.Get(sessionID, nil)
		if err != nil {
			var se *stripe.Error
			if errors.As(err, &se) && se.Code == stripe.ErrorCodeResourceMissing {
				return CheckoutSessionState{Missing: true}, nil
			}
			return CheckoutSessionState{}, err
		}
		return CheckoutSessionState{Status: string(sess.Status), PaymentStatus: string(sess.PaymentStatus)}, nil
	}
}

// ReconcileDecision is the local status a stale pending row should move to ("" = leave unchanged).
func ReconcileDecision(st CheckoutSessionState) string {
	if st.Missing {
		return repoPayments.StatusCanceled
	}
	switch st.Status {
	case string(stripe.CheckoutSessionStatusExpired):
		return repoPayments.StatusCanceled
	case string(stripe.CheckoutSessionStatusComplete):
		switch st.PaymentStatus {
		case string(stripe.CheckoutSessionPaymentStatusPaid), string(stripe.CheckoutSessionPaymentStatusNoPaymentRequired):
			return repoPayments.StatusCompleted
		}
	}
	return ""
}

// ReconcileStalePendingStripe resolves Stripe transactions stuck in "pending" (abandoned checkouts whose
// checkout.session.expired webhook was never delivered, or payments whose completion webhook was lost).
// It returns the number of rows changed.
func ReconcileStalePendingStripe(ctx context.Context, pool *pgxpool.Pool, lookup CheckoutSessionLookup, now time.Time) int {
	if pool == nil || lookup == nil {
		return 0
	}
	rows, err := repoPayments.ListStalePending(ctx, pool, repoPayments.ProviderStripe, now.Add(-StalePendingAge), 50)
	if err != nil {
		slog.Warn("stale pending payments query failed", "err", err)
		return 0
	}
	changed := 0
	for _, tx := range rows {
		sid := strings.TrimSpace(tx.ProviderTxnID)
		if sid == "" {
			continue
		}
		st, err := lookup(ctx, sid)
		if err != nil {
			slog.Warn("stripe checkout session lookup failed", "session", sid, "err", err)
			continue
		}
		switch ReconcileDecision(st) {
		case repoPayments.StatusCanceled:
			if ok, err := repoPayments.CancelPendingByProviderTxn(ctx, pool, repoPayments.ProviderStripe, sid); err == nil && ok {
				changed++
			}
		case repoPayments.StatusCompleted:
			ok, err := repoPayments.MarkCompletedByProviderTxn(ctx, pool, repoPayments.ProviderStripe, sid, 0, tx.Currency, nil)
			if err == nil && ok {
				changed++
				slog.Warn("reconciled paid checkout session whose webhook was not processed; verify entitlement", "session", sid, "user_id", tx.UserID)
				paymentprovider.RecordTransaction(paymentprovider.ProviderStripe, repoPayments.StatusCompleted, tx.Currency)
			}
		}
	}
	return changed
}
