package statuspage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// AlertmanagerWebhook is the Alertmanager webhook payload (v4).
type AlertmanagerWebhook struct {
	Status string            `json:"status"`
	Alerts []AlertmanagerAlert `json:"alerts"`
}

// AlertmanagerAlert is a single alert in an Alertmanager webhook.
type AlertmanagerAlert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
}

func ParseAlertmanagerWebhook(body []byte) (AlertmanagerWebhook, error) {
	var payload AlertmanagerWebhook
	if err := json.Unmarshal(body, &payload); err != nil {
		return AlertmanagerWebhook{}, fmt.Errorf("invalid alertmanager payload: %w", err)
	}
	return payload, nil
}

// ApplyAlertmanagerWebhook updates Statuspage components for firing/resolved alerts.
func (c *Client) ApplyAlertmanagerWebhook(ctx context.Context, payload AlertmanagerWebhook) error {
	if c == nil || !c.Configured() {
		return fmt.Errorf("statuspage is not configured")
	}
	if len(payload.Alerts) == 0 {
		return nil
	}
	updates := make(map[string]string)
	for _, alert := range payload.Alerts {
		key := componentKeyFromAlert(alert)
		if key == "" {
			continue
		}
		componentID, ok := c.cfg.ComponentMap.IDForKey(key)
		if !ok {
			continue
		}
		severity := alert.Labels["severity"]
		if severity == "" {
			severity = alert.Annotations["severity"]
		}
		status := ComponentStatusForAlert(alert.Status, severity)
		updates[componentID] = status
	}
	for componentID, status := range updates {
		if err := c.UpdateComponentStatus(ctx, componentID, status); err != nil {
			return err
		}
	}
	return nil
}

func componentKeyFromAlert(alert AlertmanagerAlert) string {
	if alert.Labels != nil {
		if v := strings.TrimSpace(alert.Labels["statuspage_component"]); v != "" {
			return v
		}
		if v := strings.TrimSpace(alert.Labels["component"]); v != "" {
			return v
		}
	}
	if alert.Annotations != nil {
		if v := strings.TrimSpace(alert.Annotations["statuspage_component"]); v != "" {
			return v
		}
	}
	if alert.Labels == nil {
		return ""
	}
	name := strings.ToLower(strings.TrimSpace(alert.Labels["alertname"]))
	switch {
	case strings.Contains(name, "api"), strings.Contains(name, "http"), strings.Contains(name, "error_rate"):
		return "api"
	case strings.Contains(name, "web"), strings.Contains(name, "frontend"):
		return "web_app"
	case strings.Contains(name, "database"), strings.Contains(name, "postgres"), strings.Contains(name, "db"):
		return "database"
	case strings.Contains(name, "queue"), strings.Contains(name, "rabbit"), strings.Contains(name, "dead_letter"):
		return "job_queue"
	case strings.Contains(name, "ai"), strings.Contains(name, "openrouter"):
		return "ai_services"
	case strings.Contains(name, "storage"), strings.Contains(name, "s3"), strings.Contains(name, "media"):
		return "media_storage"
	default:
		return ""
	}
}