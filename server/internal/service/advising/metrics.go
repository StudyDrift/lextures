package advising

import (
	"expvar"
	"sync/atomic"
)

var degreeAuditAPICalls = struct {
	success atomic.Uint64
	failure atomic.Uint64
}{}

func init() {
	expvar.Publish("degree_audit_api_calls_total", expvar.Func(func() any {
		return map[string]uint64{
			"success": degreeAuditAPICalls.success.Load(),
			"failure": degreeAuditAPICalls.failure.Load(),
		}
	}))
}

// RecordDegreeAuditAPICall increments degree_audit_api_calls_total{status}.
func RecordDegreeAuditAPICall(success bool) {
	if success {
		degreeAuditAPICalls.success.Add(1)
	} else {
		degreeAuditAPICalls.failure.Add(1)
	}
}
