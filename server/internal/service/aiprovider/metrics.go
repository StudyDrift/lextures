package aiprovider

import (
	"expvar"
	"strings"
	"sync/atomic"

	"github.com/lextures/lextures/server/internal/telemetry"
)

type latencySample struct {
	Provider  string
	Model     string
	Operation string
	Seconds   float64
}

var metricsState = struct {
	latencyCount  atomic.Uint64
	errorCount    atomic.Uint64
	estimatedCost atomic.Uint64 // milli-cents as uint64 for atomic add
	lastLatency   atomic.Value  // latencySample
	lastError     atomic.Value  // string
}{}

// credentialsConfigured tracks ai_credentials_configured{scope,provider} via expvar map.
var credentialsConfigured = expvar.NewMap("ai_credentials_configured")

// catalogFetch tracks ai_model_catalog_fetch{provider,result} via expvar map (AP.3).
var catalogFetch = expvar.NewMap("ai_model_catalog_fetch")

// errorByType tracks ai_provider_errors_total{provider,error_type} via expvar map (AP.8).
var errorByType = expvar.NewMap("ai_provider_errors_by_type")

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

// RecordCredentialConfigured sets the ai_credentials_configured gauge for scope+provider (0 or 1).
func RecordCredentialConfigured(scope, provider string, configured bool) {
	key := scope + "," + provider
	v := new(expvar.Int)
	if configured {
		v.Set(1)
	}
	credentialsConfigured.Set(key, v)
}

// RecordCatalogFetch increments ai_model_catalog_fetch for provider+result
// (curated|live|cached|live_fallback|error).
func RecordCatalogFetch(provider, result string) {
	key := provider + "," + result
	v := catalogFetch.Get(key)
	if v == nil {
		n := new(expvar.Int)
		n.Set(1)
		catalogFetch.Set(key, n)
		return
	}
	if n, ok := v.(*expvar.Int); ok {
		n.Add(1)
	}
}

func recordLatency(provider ProviderName, model, operation string, seconds float64) {
	metricsState.latencyCount.Add(1)
	metricsState.lastLatency.Store(latencySample{
		Provider:  string(provider),
		Model:     model,
		Operation: operation,
		Seconds:   seconds,
	})
}

func recordError(provider ProviderName, operation string) {
	recordErrorTyped(provider, operation, ErrorTypeOther)
}

func recordErrorTyped(provider ProviderName, operation string, errType ErrorType) {
	if errType == "" {
		errType = ErrorTypeOther
	}
	metricsState.errorCount.Add(1)
	metricsState.lastError.Store(string(provider) + ":" + operation + ":" + string(errType))
	key := string(provider) + "," + string(errType)
	v := errorByType.Get(key)
	if v == nil {
		n := new(expvar.Int)
		n.Set(1)
		errorByType.Set(key, n)
		return
	}
	if n, ok := v.(*expvar.Int); ok {
		n.Add(1)
	}
}

func recordCostUSD(provider ProviderName, cost float64) {
	if cost <= 0 {
		return
	}
	millicents := uint64(cost * 100_000)
	metricsState.estimatedCost.Add(millicents)
	_ = provider
}

// recordTelemetry mirrors latency/cost into Prometheus with real provider labels (AP.6 FR-6).
// Model label uses alias when present to bound cardinality.
func recordTelemetry(provider ProviderName, model, outcome string, seconds, costDollars float64) {
	label := strings.TrimSpace(model)
	if label == "" {
		label = "unknown"
	}
	p := strings.TrimSpace(string(provider))
	if p == "" {
		p = "unknown"
	}
	telemetry.RecordAIProvider(p, label, outcome, seconds, costDollars)
}
