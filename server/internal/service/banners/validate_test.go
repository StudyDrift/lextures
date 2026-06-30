package banners

import (
	"testing"
)

func TestValidateMessage(t *testing.T) {
	if err := ValidateMessage("Scheduled maintenance Sunday 2am UTC"); err != nil {
		t.Fatalf("valid message: %v", err)
	}
	if err := ValidateMessage(""); err == nil {
		t.Fatal("expected empty error")
	}
	if err := ValidateMessage("Contact admin@example.com for help"); err == nil {
		t.Fatal("expected PII error")
	}
}

func TestIncidentSeverity(t *testing.T) {
	if got := IncidentSeverity("critical", "investigating"); got != "error" {
		t.Fatalf("got %q want error", got)
	}
	if got := IncidentSeverity("minor", "monitoring"); got != "warning" {
		t.Fatalf("got %q want warning", got)
	}
	if got := IncidentSeverity("major", "resolved"); got != "" {
		t.Fatalf("resolved should return empty severity, got %q", got)
	}
}

func TestParseStatuspageWebhook(t *testing.T) {
	_, err := ParseStatuspageWebhook([]byte(`{"incident":{"id":"abc","name":"Outage"}}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	_, err = ParseStatuspageWebhook([]byte(`{}`))
	if err == nil {
		t.Fatal("expected error for missing incident")
	}
}
