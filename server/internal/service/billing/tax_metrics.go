package billing

import (
	"expvar"
	"sync/atomic"
)

var (
	taxCalcFailures  atomic.Uint64
	reverseChargeCnt atomic.Uint64
)

func init() {
	expvar.Publish("tax_calc_failures", expvar.Func(func() any {
		return taxCalcFailures.Load()
	}))
	expvar.Publish("reverse_charge_count", expvar.Func(func() any {
		return reverseChargeCnt.Load()
	}))
}

// RecordTaxCalcFailure increments failed tax calculation counter (plan 15.13).
func RecordTaxCalcFailure() {
	taxCalcFailures.Add(1)
}

// RecordReverseCharge increments reverse-charge transaction counter (plan 15.13).
func RecordReverseCharge() {
	reverseChargeCnt.Add(1)
}

// RecordTaxCollected records tax by jurisdiction (plan 15.13).
func RecordTaxCollected(jurisdiction string, amountCents int) {
	if amountCents <= 0 || jurisdiction == "" {
		return
	}
	key := "tax_collected_" + jurisdiction
	expvar.Publish(key, expvar.Func(func() any {
		return amountCents
	}))
}