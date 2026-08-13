package kernel

import (
	"net/http"
	"sync/atomic"
)

// ErrorObserver is invoked after an error is mapped, labelled by apierr code
// and a low-cardinality route class (chi pattern or "unmatched").
type ErrorObserver func(code, routeClass string, status int)

var errorObserver atomic.Value // ErrorObserver

// SetErrorObserver installs the process-wide mapped-error metric hook.
// Passing nil clears it. Safe for concurrent use; no-op when unset.
func SetErrorObserver(fn ErrorObserver) {
	if fn == nil {
		errorObserver.Store(ErrorObserver(func(string, string, int) {}))
		return
	}
	errorObserver.Store(fn)
}

func observeMapped(r *http.Request, m Mapped) {
	v := errorObserver.Load()
	if v == nil {
		return
	}
	fn, ok := v.(ErrorObserver)
	if !ok || fn == nil {
		return
	}
	fn(m.Code, routeClass(r), m.Status)
}
