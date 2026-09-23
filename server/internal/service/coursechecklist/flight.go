package coursechecklist

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
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

type evalOut struct {
	res       Result
	at        time.Time
	truncated bool
}

// evaluateFull coalesces concurrent evaluations for one course.
// A caller that arrives after the snapshot is published reuses that computedAt
// instead of starting a second evaluation. Forced refreshes use a separate key
// so they do not adopt a cached snapshot.
func (s *Service) evaluateFull(ctx context.Context, courseID uuid.UUID, courseCode string, force bool) (Result, time.Time, bool, error) {
	key := courseID.String()
	if force {
		key += ":force"
	}
	v, err, _ := evalFlight.Do(key, func() (any, error) {
		if !force {
			out, ok, snapErr := s.reuseFreshSnapshot(ctx, courseID)
			if snapErr != nil {
				return nil, snapErr
			}
			if ok {
				return out, nil
			}
		}
		acquireEvalSlot()
		defer releaseEvalSlot()
		needs := DataNeedsForEvaluate(MustDefault(), EvaluateOptions{})
		loaded, loadErr := LoadSnapshot(ctx, s.Pool, courseCode, needs)
		if loadErr != nil {
			return nil, loadErr
		}
		opt := EvaluateOptions{LazyLoaders: s.lazyLoaders()}
		res := Evaluate(ctx, loaded, opt)
		truncated := false
		res, truncated = fitPayload(res)
		out := evalOut{res: res, at: s.now(), truncated: truncated}
		if writeErr := s.writeSnapshotBestEffort(ctx, courseID, res, out.at, truncated); writeErr != nil {
			slog.Warn("coursechecklist.snapshot_write_failed",
				"course_id", courseID.String(), "err", writeErr.Error())
		}
		return out, nil
	})
	if err != nil {
		return Result{}, time.Time{}, false, err
	}
	out := v.(evalOut)
	return out.res, out.at, out.truncated, nil
}

func (s *Service) reuseFreshSnapshot(ctx context.Context, courseID uuid.UUID) (evalOut, bool, error) {
	snap, freshness, err := s.loadSnapshotAndFreshness(ctx, courseID)
	if err != nil {
		return evalOut{}, false, err
	}
	if IsSnapshotStale(snap, EngineVersion(), CatalogVersion(), s.TTL, freshness, s.now()) {
		return evalOut{}, false, nil
	}
	res, truncated, err := decodeSnapshotPayload(snap.Payload)
	if err != nil {
		return evalOut{}, false, nil
	}
	return evalOut{res: res, at: snap.ComputedAt.UTC(), truncated: truncated}, true, nil
}
