package consortium

import (
	"expvar"
	"sync/atomic"
)

var enrollmentTotal atomic.Uint64

func init() {
	expvar.Publish("consortium_enrollments_total", expvar.Func(func() any {
		return enrollmentTotal.Load()
	}))
}

// RecordEnrollment increments the consortium enrollment counter (plan 14.18 observability).
func RecordEnrollment() {
	enrollmentTotal.Add(1)
}
