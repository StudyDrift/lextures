package apitokens

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/repos/adminaudit"
)

const (
	maxServiceTokensPerOrg = 50
	defaultRotateOverlap   = 24 * time.Hour
)

// HashClientIP returns an HMAC-SHA256 hex digest of the client IP (plan 16.2).
func HashClientIP(key, rawIP string) string {
	ip := strings.TrimSpace(rawIP)
	if ip == "" || strings.TrimSpace(key) == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(ip)
	if err == nil {
		ip = host
	}
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte(ip))
	return hex.EncodeToString(mac.Sum(nil))
}

// ClientIPFromRequest extracts the best-effort client IP for usage logging.
func ClientIPFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		return xri
	}
	return r.RemoteAddr
}

// AuditCreate writes api_token_created to the admin audit log.
func AuditCreate(ctx context.Context, pool *pgxpool.Pool, actorID uuid.UUID, orgID *uuid.UUID, tokenID uuid.UUID, scopes []string, r *http.Request) {
	if pool == nil {
		return
	}
	after, _ := json.Marshal(map[string]any{
		"token_id": tokenID.String(),
		"scopes":   scopes,
	})
	ip := ClientIPFromRequest(r)
	var actorIP *string
	if ip != "" {
		actorIP = &ip
	}
	ua := strings.TrimSpace(r.UserAgent())
	var userAgent *string
	if ua != "" {
		userAgent = &ua
	}
	targetType := "api_token"
	_, _, _ = adminaudit.Insert(ctx, pool, adminaudit.InsertParams{
		OrgID:       orgID,
		EventType:   "api_token_created",
		ActorID:     actorID,
		ActorIP:     actorIP,
		UserAgent:   userAgent,
		TargetType:  &targetType,
		TargetID:    &tokenID,
		AfterValue:  after,
	})
}

// AuditRotate writes api_token_rotated to the admin audit log.
func AuditRotate(ctx context.Context, pool *pgxpool.Pool, actorID uuid.UUID, orgID *uuid.UUID, oldID, newID uuid.UUID, r *http.Request) {
	if pool == nil {
		return
	}
	after, _ := json.Marshal(map[string]any{
		"old_token_id": oldID.String(),
		"new_token_id": newID.String(),
	})
	ip := ClientIPFromRequest(r)
	var actorIP *string
	if ip != "" {
		actorIP = &ip
	}
	targetType := "api_token"
	_, _, _ = adminaudit.Insert(ctx, pool, adminaudit.InsertParams{
		OrgID:      orgID,
		EventType:  "api_token_rotated",
		ActorID:    actorID,
		ActorIP:    actorIP,
		TargetType: &targetType,
		TargetID:   &newID,
		AfterValue: after,
	})
}

// AuditRevoke writes api_token_revoked to the admin audit log.
func AuditRevoke(ctx context.Context, pool *pgxpool.Pool, actorID uuid.UUID, orgID *uuid.UUID, tokenID uuid.UUID, r *http.Request) {
	if pool == nil {
		return
	}
	after, _ := json.Marshal(map[string]any{"token_id": tokenID.String()})
	ip := ClientIPFromRequest(r)
	var actorIP *string
	if ip != "" {
		actorIP = &ip
	}
	targetType := "api_token"
	_, _, _ = adminaudit.Insert(ctx, pool, adminaudit.InsertParams{
		OrgID:      orgID,
		EventType:  "api_token_revoked",
		ActorID:    actorID,
		ActorIP:    actorIP,
		TargetType: &targetType,
		TargetID:   &tokenID,
		AfterValue: after,
	})
}
