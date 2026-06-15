package research_consent

import (
	"expvar"
	"sync"
)

// consent_decisions_total{study_id, decision} (NFR Observability).
var (
	decisionMu    sync.Mutex
	decisionCount = map[string]uint64{}
)

func init() {
	expvar.Publish("consent_decisions_total", expvar.Func(func() any {
		decisionMu.Lock()
		defer decisionMu.Unlock()
		out := make(map[string]uint64, len(decisionCount))
		for k, v := range decisionCount {
			out[k] = v
		}
		return out
	}))
}

// RecordDecision increments consent_decisions_total for a study/decision pair.
func RecordDecision(studyID, decision string) {
	key := studyID + "/" + decision
	decisionMu.Lock()
	decisionCount[key]++
	decisionMu.Unlock()
}
