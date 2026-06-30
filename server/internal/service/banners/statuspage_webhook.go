package banners

import (
	"encoding/json"
	"errors"
	"strings"
)

// StatuspageWebhook is the subset of a Statuspage incident webhook payload we consume.
type StatuspageWebhook struct {
	Incident *StatuspageIncident `json:"incident"`
}

// StatuspageIncident holds incident fields from Statuspage.io webhooks.
type StatuspageIncident struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Impact string `json:"impact"`
}

// ParseStatuspageWebhook unmarshals a Statuspage webhook body.
func ParseStatuspageWebhook(body []byte) (StatuspageWebhook, error) {
	var payload StatuspageWebhook
	if err := json.Unmarshal(body, &payload); err != nil {
		return StatuspageWebhook{}, errors.New("invalid Statuspage payload")
	}
	if payload.Incident == nil || strings.TrimSpace(payload.Incident.ID) == "" {
		return StatuspageWebhook{}, errors.New("incident id is required")
	}
	return payload, nil
}

// IncidentSeverity maps Statuspage impact/status to banner severity.
func IncidentSeverity(impact, status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "resolved" || status == "completed" {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(impact)) {
	case "critical", "major":
		return "error"
	case "minor":
		return "warning"
	default:
		return "warning"
	}
}

// IncidentMessage builds a plain-text banner message from an incident.
func IncidentMessage(name, status string) string {
	name = strings.TrimSpace(name)
	status = strings.TrimSpace(status)
	if name == "" {
		name = "Service incident"
	}
	if status != "" {
		return name + " — status: " + status
	}
	return name
}
