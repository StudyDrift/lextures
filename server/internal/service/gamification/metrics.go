package gamification

import (
	"expvar"
	"sync/atomic"
)

var (
	xpAwardedTotal   = expvar.NewMap("xp_awarded_total")
	streakResetsTotal int64
)

func init() {
	expvar.Publish("streak_resets_total", expvar.Func(func() any {
		return atomic.LoadInt64(&streakResetsTotal)
	}))
}

// RecordXPAwarded increments observability counters for XP awards.
func RecordXPAwarded(activityType string, xp int) {
	if xp <= 0 {
		return
	}
	v := xpAwardedTotal.Get(activityType)
	if v == nil {
		xpAwardedTotal.Set(activityType, new(expvar.Int))
		v = xpAwardedTotal.Get(activityType)
	}
	if counter, ok := v.(*expvar.Int); ok {
		counter.Add(int64(xp))
	}
}

// RecordStreakReset increments the streak reset counter.
func RecordStreakReset() {
	atomic.AddInt64(&streakResetsTotal, 1)
}
