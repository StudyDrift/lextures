package aiprovider

import (
	"expvar"
	"sync/atomic"
)

type latencySample struct {
	Provider string
	Model    string
	Seconds  float64
}

var metricsState = struct {
	latencyCount   atomic.Uint64
	errorCount     atomic.Uint64
	estimatedCost  atomic.Uint64 // milli-cents as uint64 for atomic add
	lastLatency    atomic.Value  // latencySample
	lastError      atomic.Value  // string
}{}

func init() {
	expvar.Publish("ai_provider.latency_samples_total", expvar.Func(func() any {
		return metricsState.latencyCount.Load()
	}))
	expvar.Publish("ai_provider.errors_total", expvar.Func(func() any {
		return metricsState.errorCount.Load()
	}))
	expvar.Publish("ai_provider.estimated_cost_millicents_total", expvar.Func(func() any {
		return metricsState.estimatedCost.Load()
	}))
	expvar.Publish("ai_provider.last_latency", expvar.Func(func() any {
		v := metricsState.lastLatency.Load()
		if v == nil {
			return nil
		}
		return v
	}))
}

func recordLatency(provider ProviderName, model string, seconds float64) {
	metricsState.latencyCount.Add(1)
	metricsState.lastLatency.Store(latencySample{
		Provider: string(provider),
		Model:    model,
		Seconds:  seconds,
	})
}

func recordError(provider ProviderName, errType string) {
	metricsState.errorCount.Add(1)
	metricsState.lastError.Store(string(provider) + ":" + errType)
}

func recordCostUSD(provider ProviderName, cost float64) {
	if cost <= 0 {
		return
	}
	millicents := uint64(cost * 100_000)
	metricsState.estimatedCost.Add(millicents)
	_ = provider
}