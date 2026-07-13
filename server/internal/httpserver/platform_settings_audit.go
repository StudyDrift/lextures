package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/repos/organization"
	auditservice "github.com/lextures/lextures/server/internal/service/adminaudit"
)

// platformFeatureAuditChanges returns only boolean fields explicitly named by
// updateMask. This records feature toggles without ever copying secret/config
// strings from the platform-settings request into the audit log.
func platformFeatureAuditChanges(raw []byte) map[string]bool {
	var body map[string]any
	if json.Unmarshal(raw, &body) != nil {
		return nil
	}
	mask, _ := body["updateMask"].([]any)
	changes := make(map[string]bool)
	for _, item := range mask {
		key, ok := item.(string)
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value, ok := body[key].(bool)
		if key != "" && ok {
			changes[key] = value
		}
	}
	return changes
}

func (d Deps) recordPlatformFeatureAudit(
	r *http.Request,
	actorID uuid.UUID,
	raw []byte,
) {
	// Platform feature changes are security-sensitive and are always retained,
	// even when the audit-log viewer feature is disabled.
	if d.Pool == nil {
		return
	}
	changes := platformFeatureAuditChanges(raw)
	if len(changes) == 0 {
		return
	}
	orgID, _ := organization.OrgIDForUser(r.Context(), d.Pool, actorID)
	var oid *uuid.UUID
	if orgID != uuid.Nil {
		oid = &orgID
	}
	targetType := "platform_settings"
	payload, _ := json.Marshal(map[string]any{"changes": changes})
	ip := clientIP(r)
	ua := r.UserAgent()
	_, _ = auditservice.Record(r.Context(), d.Pool, auditservice.RecordParams{
		OrgID:      oid,
		EventType:  auditservice.EventPlatformSettingsChange,
		ActorID:    actorID,
		ActorIP:    &ip,
		UserAgent:  &ua,
		TargetType: &targetType,
		AfterValue: payload,
	})
}
