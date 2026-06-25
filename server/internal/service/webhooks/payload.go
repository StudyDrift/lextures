package webhooksvc

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/webhooks"
)

// SubscriptionSettings holds connector metadata stored on webhook subscriptions (plan 16.10).
type SubscriptionSettings struct {
	Source     string `json:"source,omitempty"`
	IncludePII bool   `json:"includePII,omitempty"`
}

// ParseSubscriptionSettings decodes settings JSON from a subscription row.
func ParseSubscriptionSettings(raw json.RawMessage) SubscriptionSettings {
	if len(raw) == 0 || string(raw) == "null" {
		return SubscriptionSettings{}
	}
	var s SubscriptionSettings
	_ = json.Unmarshal(raw, &s)
	return s
}

// ConnectorSource returns zapier, make, or empty for a subscription.
func ConnectorSource(raw json.RawMessage) string {
	return strings.TrimSpace(ParseSubscriptionSettings(raw).Source)
}

// AdaptPayloadForSubscription adjusts a webhook envelope for the target subscription.
// When includePII is set, student email is added to data; otherwise email fields are stripped.
func AdaptPayloadForSubscription(ctx context.Context, pool *pgxpool.Pool, payload []byte, settings json.RawMessage) ([]byte, error) {
	cfg := ParseSubscriptionSettings(settings)
	var env webhooks.Envelope
	if err := json.Unmarshal(payload, &env); err != nil {
		return payload, err
	}
	var data map[string]any
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return payload, err
	}
	if cfg.IncludePII {
		if err := enrichStudentEmail(ctx, pool, data); err != nil {
			return payload, err
		}
	} else {
		stripPIIFields(data)
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return payload, err
	}
	env.Data = raw
	out, err := json.Marshal(env)
	if err != nil {
		return payload, err
	}
	return out, nil
}

func enrichStudentEmail(ctx context.Context, pool *pgxpool.Pool, data map[string]any) error {
	if pool == nil {
		return nil
	}
	uid := userIDFromData(data)
	if uid == uuid.Nil {
		return nil
	}
	var email string
	err := pool.QueryRow(ctx, `SELECT email FROM "user".users WHERE id = $1`, uid).Scan(&email)
	if err != nil || strings.TrimSpace(email) == "" {
		return err
	}
	data["studentEmail"] = email
	return nil
}

func userIDFromData(data map[string]any) uuid.UUID {
	for _, key := range []string{"studentUserId", "submittedBy"} {
		if v, ok := data[key]; ok {
			if s, ok := v.(string); ok {
				if id, err := uuid.Parse(strings.TrimSpace(s)); err == nil {
					return id
				}
			}
		}
	}
	return uuid.Nil
}

func stripPIIFields(data map[string]any) {
	for _, key := range []string{"studentEmail", "email", "submittedByEmail"} {
		delete(data, key)
	}
}
