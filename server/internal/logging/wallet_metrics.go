package logging

import "sync/atomic"

// WalletMetrics tracks wallet_view_total / share / export counters (T09).
type WalletMetrics struct {
	views         atomic.Uint64
	sharesCreated atomic.Uint64
	sharesRevoked atomic.Uint64
	exports       atomic.Uint64
}

// GlobalWalletMetrics is incremented by wallet handlers and jobs.
var GlobalWalletMetrics = &WalletMetrics{}

func (m *WalletMetrics) IncView()         { m.views.Add(1) }
func (m *WalletMetrics) IncShareCreated() { m.sharesCreated.Add(1) }
func (m *WalletMetrics) IncShareRevoked() { m.sharesRevoked.Add(1) }
func (m *WalletMetrics) IncExport()       { m.exports.Add(1) }

func (m *WalletMetrics) Snapshot() (views, sharesCreated, sharesRevoked, exports uint64) {
	return m.views.Load(), m.sharesCreated.Load(), m.sharesRevoked.Load(), m.exports.Load()
}
