package accessibility

import (
	"expvar"
	"sync"
)

// accommodations_applied_total{type} (NFR Observability).
var (
	appliedMu    sync.Mutex
	appliedCount = map[string]uint64{}
)

func init() {
	expvar.Publish("accommodations_applied_total", expvar.Func(func() any {
		appliedMu.Lock()
		defer appliedMu.Unlock()
		out := make(map[string]uint64, len(appliedCount))
		for k, v := range appliedCount {
			out[k] = v
		}
		return out
	}))
}

// RecordApplied increments accommodations_applied_total for an accommodation type.
func RecordApplied(accommodationType string) {
	appliedMu.Lock()
	appliedCount[accommodationType]++
	appliedMu.Unlock()
}
