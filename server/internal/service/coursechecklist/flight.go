package coursechecklist

import (
	"sync"

	"golang.org/x/sync/singleflight"
)

// MaxConcurrentEvaluations caps in-flight evaluations per process (NFR).
const MaxConcurrentEvaluations = 32

var (
	evalFlight     singleflight.Group
	evalSemOnce    sync.Once
	evalSem        chan struct{}
	flightWaiters  int64
	flightWaitersM sync.Mutex
)

func evaluationSemaphore() chan struct{} {
	evalSemOnce.Do(func() {
		evalSem = make(chan struct{}, MaxConcurrentEvaluations)
	})
	return evalSem
}

func acquireEvalSlot() {
	flightWaitersM.Lock()
	flightWaiters++
	n := flightWaiters
	flightWaitersM.Unlock()
	setSingleflightWaiters(float64(n))
	evaluationSemaphore() <- struct{}{}
}

func releaseEvalSlot() {
	<-evaluationSemaphore()
	flightWaitersM.Lock()
	if flightWaiters > 0 {
		flightWaiters--
	}
	n := flightWaiters
	flightWaitersM.Unlock()
	setSingleflightWaiters(float64(n))
}
