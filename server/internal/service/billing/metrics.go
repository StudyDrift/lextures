package billing

import (
	"expvar"
	"sync/atomic"
)

var (
	paymentsTotal       atomic.Uint64
	subscriptionMRRCent atomic.Int64
)

func init() {
	expvar.Publish("payments_total", expvar.Func(func() any {
		return paymentsTotal.Load()
	}))
	expvar.Publish("subscription_mrr_cents", expvar.Func(func() any {
		return subscriptionMRRCent.Load()
	}))
}

// RecordPayment increments payment counter and optional MRR gauge (plan 15.3).
func RecordPayment(amountCents int, entitlementType string) {
	paymentsTotal.Add(1)
	if entitlementType == "subscription_monthly" && amountCents > 0 {
		subscriptionMRRCent.Add(int64(amountCents))
	}
}
