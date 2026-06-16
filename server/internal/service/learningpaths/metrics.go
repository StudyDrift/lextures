package learningpaths

import (
	"expvar"
	"sync/atomic"
)

var (
	enrollmentTotal  atomic.Uint64
	completionTotal  atomic.Uint64
)

func init() {
	expvar.Publish("path_enrollments_total", expvar.Func(func() any {
		return enrollmentTotal.Load()
	}))
	expvar.Publish("path_completions_total", expvar.Func(func() any {
		return completionTotal.Load()
	}))
}

// RecordEnrollment increments path enrollment counter (plan 15.4 observability).
func RecordEnrollment() {
	enrollmentTotal.Add(1)
}

// RecordCompletion increments path completion counter (plan 15.4 observability).
func RecordCompletion() {
	completionTotal.Add(1)
}
