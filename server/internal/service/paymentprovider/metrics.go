package paymentprovider

import (
	"sync/atomic"
)

var (
	transactionsTotal atomic.Uint64
)

// RecordTransaction increments payment_transactions_total (plan 16.8 observability).
func RecordTransaction(provider ProviderName, status, currency string) {
	transactionsTotal.Add(1)
	_ = provider
	_ = status
	_ = currency
}
