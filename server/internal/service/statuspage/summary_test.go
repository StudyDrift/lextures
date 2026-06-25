package statuspage

import "testing"

func TestNormalizeSummary_ActiveIncident(t *testing.T) {
	raw := upstreamSummary{}
	raw.Page.URL = "https://status.example.com"
	raw.Incidents = []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
		Impact string `json:"impact"`
	}{
		{ID: "inc-1", Name: "API latency elevated", Status: "investigating", Impact: "minor"},
		{ID: "inc-2", Name: "Resolved outage", Status: "resolved", Impact: "major"},
	}
	out := normalizeSummary("https://status.lextures.io", raw)
	if out.PageURL != "https://status.example.com" {
		t.Fatalf("pageUrl=%q", out.PageURL)
	}
	if len(out.Incidents) != 1 {
		t.Fatalf("incidents=%d want 1", len(out.Incidents))
	}
	if out.Status != "minor" {
		t.Fatalf("status=%q want minor", out.Status)
	}
}

func TestComponentKeyFromAlert(t *testing.T) {
	alert := AlertmanagerAlert{
		Status: "firing",
		Labels: map[string]string{
			"alertname":            "HighErrorRate",
			"statuspage_component": "api",
		},
	}
	if got := componentKeyFromAlert(alert); got != "api" {
		t.Fatalf("key=%q want api", got)
	}
}

func TestComponentStatusForAlert(t *testing.T) {
	if got := ComponentStatusForAlert("resolved", "critical"); got != "operational" {
		t.Fatalf("resolved=%q", got)
	}
	if got := ComponentStatusForAlert("firing", "critical"); got != "major_outage" {
		t.Fatalf("critical=%q", got)
	}
	if got := ComponentStatusForAlert("firing", "warning"); got != "degraded_performance" {
		t.Fatalf("warning=%q", got)
	}
}