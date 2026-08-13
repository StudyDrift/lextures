package kernel

import "sync"

// RouteInfo is one toolkit-registered handler, used by the FR-11 ratchet.
type RouteInfo struct {
	Name      string
	Method    string
	Guard     string
	Unguarded bool
}

var (
	regMu    sync.Mutex
	registry []RouteInfo
)

func registerRoute(info RouteInfo) {
	regMu.Lock()
	defer regMu.Unlock()
	registry = append(registry, info)
}

// RegisteredRoutes returns a snapshot of toolkit handlers constructed so far.
func RegisteredRoutes() []RouteInfo {
	regMu.Lock()
	defer regMu.Unlock()
	out := make([]RouteInfo, len(registry))
	copy(out, registry)
	return out
}

// UnguardedCount is the number of toolkit routes with no real guard (nil
// substituted to Authenticated, or Public()).
func UnguardedCount() int {
	n := 0
	for _, r := range RegisteredRoutes() {
		if r.Unguarded {
			n++
		}
	}
	return n
}

// ResetRegistryForTest clears the in-process registry. Tests only.
func ResetRegistryForTest() {
	regMu.Lock()
	defer regMu.Unlock()
	registry = nil
}
