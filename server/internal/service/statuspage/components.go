package statuspage

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ComponentMap maps logical service keys to Statuspage component IDs.
type ComponentMap map[string]string

func ParseComponentMap(raw string) (ComponentMap, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ComponentMap{}, nil
	}
	var m ComponentMap
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil, fmt.Errorf("parse statuspage component map: %w", err)
	}
	return m, nil
}

func (m ComponentMap) IDForKey(key string) (string, bool) {
	if m == nil {
		return "", false
	}
	id := strings.TrimSpace(m[strings.ToLower(strings.TrimSpace(key))])
	if id == "" {
		return "", false
	}
	return id, true
}

// ComponentStatusForAlert maps an Alertmanager alert to a Statuspage component status.
func ComponentStatusForAlert(alertStatus, severity string) string {
	if strings.EqualFold(strings.TrimSpace(alertStatus), "resolved") {
		return "operational"
	}
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical", "page":
		return "major_outage"
	case "warning", "warn":
		return "degraded_performance"
	default:
		return "degraded_performance"
	}
}