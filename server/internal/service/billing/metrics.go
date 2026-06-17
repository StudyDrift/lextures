package billing

import (
	"expvar"
	"sync/atomic"
)

var (
	paymentsTotal          atomic.Uint64
	subscriptionMRRCent      atomic.Int64
	creatorPayoutsTotal    atomic.Uint64
	creatorPayoutFailures  atomic.Uint64
	platformRevenueCents   atomic.Int64
	creatorEarningsCents   atomic.Int64
	affiliateEarningsCents atomic.Int64
)

func init() {
	expvar.Publish("payments_total", expvar.Func(func() any {
		return paymentsTotal.Load()
	}))
	expvar.Publish("subscription_mrr_cents", expvar.Func(func() any {
		return subscriptionMRRCent.Load()
	}))
	expvar.Publish("creator_payouts_total", expvar.Func(func() any {
		return creatorPayoutsTotal.Load()
	}))
	expvar.Publish("creator_payout_failures", expvar.Func(func() any {
		return creatorPayoutFailures.Load()
	}))
	expvar.Publish("platform_revenue_cents_total", expvar.Func(func() any {
		return platformRevenueCents.Load()
	}))
}

// RecordPayment increments payment counter and optional MRR gauge (plan 15.3).
func RecordPayment(amountCents int, entitlementType string) {
	paymentsTotal.Add(1)
	if entitlementType == "subscription_monthly" && amountCents > 0 {
		subscriptionMRRCent.Add(int64(amountCents))
	}
}

// RecordCreatorEarnings tracks creator share (plan 15.8).
func RecordCreatorEarnings(amountCents int) {
	creatorEarningsCents.Add(int64(amountCents))
}

// RecordAffiliateEarnings tracks affiliate commission (plan 15.8).
func RecordAffiliateEarnings(amountCents int) {
	affiliateEarningsCents.Add(int64(amountCents))
}

// RecordPayoutSuccess increments successful payout counter (plan 15.8).
func RecordPayoutSuccess(amountCents int) {
	creatorPayoutsTotal.Add(1)
	platformRevenueCents.Add(int64(amountCents))
}

// RecordPayoutFailure increments failed payout counter (plan 15.8).
func RecordPayoutFailure() {
	creatorPayoutFailures.Add(1)
}
