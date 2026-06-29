package config

import "testing"

func TestObservabilityFromEnv_Defaults(t *testing.T) {
	// Clear anything the surrounding env might set.
	for _, k := range []string{
		"METRICS_ENABLED", "METRICS_ADDR", "OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_ENDPOINT",
		"OTEL_EXPORTER_OTLP_INSECURE", "OTEL_TRACES_SAMPLE_RATIO", "SENTRY_DSN",
		"SENTRY_TRACES_SAMPLE_RATE", "OTEL_SERVICE_NAME", "OBSERVABILITY_SERVICE_NAME",
	} {
		t.Setenv(k, "")
	}

	o := observabilityFromEnv()
	if o.ServiceName != "lextures-api" {
		t.Errorf("ServiceName = %q", o.ServiceName)
	}
	if !o.MetricsEnabled {
		t.Error("metrics should default enabled")
	}
	if o.MetricsAddr != ":9090" {
		t.Errorf("MetricsAddr = %q", o.MetricsAddr)
	}
	if o.OTelEndpoint != "" {
		t.Error("OTel endpoint should default empty (tracing off)")
	}
	if !o.OTelInsecure {
		t.Error("OTel insecure should default true (in-VPC collector)")
	}
	if o.OTelSampleRatio != 0.1 {
		t.Errorf("OTelSampleRatio = %v, want 0.1", o.OTelSampleRatio)
	}
	if o.SentryDSN != "" {
		t.Error("Sentry DSN should default empty (disabled)")
	}
	if o.SentryTracesSampleRate != 0.1 {
		t.Errorf("SentryTracesSampleRate = %v, want 0.1", o.SentryTracesSampleRate)
	}
}

func TestObservabilityFromEnv_Overrides(t *testing.T) {
	t.Setenv("METRICS_ENABLED", "false")
	t.Setenv("METRICS_ADDR", "127.0.0.1:7000")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "collector:4318")
	t.Setenv("OTEL_TRACES_SAMPLE_RATIO", "1.0")
	t.Setenv("SENTRY_DSN", "https://abc@sentry.example/1")
	t.Setenv("OTEL_SERVICE_NAME", "lextures-staging")

	o := observabilityFromEnv()
	if o.MetricsEnabled {
		t.Error("METRICS_ENABLED=false should disable metrics")
	}
	if o.MetricsAddr != "127.0.0.1:7000" {
		t.Errorf("MetricsAddr = %q", o.MetricsAddr)
	}
	if o.OTelEndpoint != "collector:4318" {
		t.Errorf("OTelEndpoint = %q", o.OTelEndpoint)
	}
	if o.OTelSampleRatio != 1.0 {
		t.Errorf("OTelSampleRatio = %v", o.OTelSampleRatio)
	}
	if o.SentryDSN == "" {
		t.Error("Sentry DSN should be set")
	}
	if o.ServiceName != "lextures-staging" {
		t.Errorf("ServiceName = %q", o.ServiceName)
	}
}

func TestFloatEnvDefault(t *testing.T) {
	t.Setenv("X_RATIO", "")
	if floatEnvDefault("X_RATIO", 0.5) != 0.5 {
		t.Error("empty should return default")
	}
	t.Setenv("X_RATIO", "bogus")
	if floatEnvDefault("X_RATIO", 0.5) != 0.5 {
		t.Error("invalid should return default")
	}
	t.Setenv("X_RATIO", "-1")
	if floatEnvDefault("X_RATIO", 0.5) != 0.5 {
		t.Error("negative should return default")
	}
	t.Setenv("X_RATIO", "0.25")
	if floatEnvDefault("X_RATIO", 0.5) != 0.25 {
		t.Error("valid value should parse")
	}
}

func TestBoolEnvDefault(t *testing.T) {
	t.Setenv("X_FLAG", "")
	if !boolEnvDefault("X_FLAG", true) {
		t.Error("empty should return default true")
	}
	t.Setenv("X_FLAG", "off")
	if boolEnvDefault("X_FLAG", true) {
		t.Error("off should be false")
	}
	t.Setenv("X_FLAG", "yes")
	if !boolEnvDefault("X_FLAG", false) {
		t.Error("yes should be true")
	}
}
