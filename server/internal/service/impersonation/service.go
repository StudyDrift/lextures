// Package impersonation implements admin "view as student" sessions (plan 18.3).
package impersonation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/auth"
	"github.com/lextures/lextures/server/internal/repos/organization"
	impersonationrepo "github.com/lextures/lextures/server/internal/repos/impersonation"
	"github.com/lextures/lextures/server/internal/repos/orgroles"
	"github.com/lextures/lextures/server/internal/repos/rbac"
	"github.com/lextures/lextures/server/internal/repos/user"
	auditservice "github.com/lextures/lextures/server/internal/service/adminaudit"
	"github.com/lextures/lextures/server/internal/telemetry"
)

const globalRBACManage = "global:app:rbac:manage"

var (
	ErrForbidden       = errors.New("impersonation: forbidden")
	ErrTargetNotFound  = errors.New("impersonation: target not found")
	ErrPrivilegedTarget = errors.New("impersonation: cannot impersonate privileged user")
	ErrStoreDown       = errors.New("impersonation: token store unavailable")
)

// StartResult is returned when an impersonation session begins.
type StartResult struct {
	Token     string
	ExpiresAt time.Time
	Target    TargetSummary
}

// TargetSummary describes the impersonated user for clients.
type TargetSummary struct {
	ID          string  `json:"id"`
	Email       string  `json:"email"`
	DisplayName *string `json:"displayName,omitempty"`
}

// StartParams is input for beginning an impersonation session.
type StartParams struct {
	ActorID      uuid.UUID
	TargetUserID uuid.UUID
	TargetOrgID  uuid.UUID
	ActorIP      *string
	UserAgent    *string
}

// Start validates scope and issues an impersonation JWT.
func Start(
	ctx context.Context,
	pool *pgxpool.Pool,
	signer *auth.JWTSigner,
	auditEnabled bool,
	p StartParams,
) (StartResult, error) {
	if pool == nil || signer == nil {
		return StartResult{}, ErrStoreDown
	}
	row, err := user.FindByID(ctx, pool, p.TargetUserID)
	if err != nil || row == nil {
		return StartResult{}, ErrTargetNotFound
	}
	targetOrg, err := organization.OrgIDForUser(ctx, pool, p.TargetUserID)
	if err != nil || targetOrg != p.TargetOrgID {
		return StartResult{}, ErrForbidden
	}
	if privileged, err := isPrivilegedUser(ctx, pool, p.TargetUserID, targetOrg); err != nil {
		return StartResult{}, err
	} else if privileged {
		return StartResult{}, ErrPrivilegedTarget
	}
	orgSlug, err := organization.OrgSlugForUser(ctx, pool, p.TargetUserID)
	if err != nil {
		return StartResult{}, fmt.Errorf("impersonation: org slug: %w", err)
	}
	tok, imp, err := signer.SignImpersonation(
		p.ActorID.String(), p.TargetUserID.String(), row.Email, targetOrg.String(), orgSlug,
	)
	if err != nil {
		return StartResult{}, fmt.Errorf("impersonation: sign: %w", err)
	}
	if err := impersonationrepo.Insert(ctx, pool, imp.JTI, p.ActorID, p.TargetUserID, imp.ExpiresAt); err != nil {
		return StartResult{}, ErrStoreDown
	}
	if auditEnabled {
		after, _ := json.Marshal(map[string]string{"action": "start"})
		targetType := "user"
		_, _ = auditservice.Record(ctx, pool, auditservice.RecordParams{
			OrgID:       &targetOrg,
			EventType:   auditservice.EventUserImpersonation,
			ActorID:     p.ActorID,
			ActorIP:     p.ActorIP,
			UserAgent:   p.UserAgent,
			TargetType:  &targetType,
			TargetID:    &p.TargetUserID,
			AfterValue:  after,
		})
	}
	telemetry.RecordBusinessEvent("impersonation_sessions_started")
	return StartResult{
		Token:     tok,
		ExpiresAt: imp.ExpiresAt,
		Target: TargetSummary{
			ID:          row.ID,
			Email:       row.Email,
			DisplayName: row.DisplayName,
		},
	}, nil
}

// EndParams is input for ending an impersonation session.
type EndParams struct {
	JTI          string
	AdminID      uuid.UUID
	TargetUserID uuid.UUID
	TargetOrgID  uuid.UUID
	ActorIP      *string
	UserAgent    *string
}

// End revokes an impersonation token and records the audit event.
func End(ctx context.Context, pool *pgxpool.Pool, auditEnabled bool, p EndParams) error {
	if pool == nil {
		return ErrStoreDown
	}
	now := time.Now().UTC()
	if err := impersonationrepo.Revoke(ctx, pool, p.JTI, now); err != nil {
		return ErrStoreDown
	}
	if auditEnabled {
		after, _ := json.Marshal(map[string]string{"action": "end"})
		targetType := "user"
		_, _ = auditservice.Record(ctx, pool, auditservice.RecordParams{
			OrgID:       &p.TargetOrgID,
			EventType:   auditservice.EventUserImpersonation,
			ActorID:     p.AdminID,
			ActorIP:     p.ActorIP,
			UserAgent:   p.UserAgent,
			TargetType:  &targetType,
			TargetID:    &p.TargetUserID,
			AfterValue:  after,
		})
	}
	telemetry.RecordBusinessEvent("impersonation_sessions_ended")
	return nil
}

func isPrivilegedUser(ctx context.Context, pool *pgxpool.Pool, userID, orgID uuid.UUID) (bool, error) {
	ga, err := rbac.UserHasPermission(ctx, pool, userID, globalRBACManage)
	if err != nil {
		return false, err
	}
	if ga {
		return true, nil
	}
	admin, err := orgroles.UserHasRole(ctx, pool, userID, orgID, orgroles.RoleOrgAdmin)
	if err != nil {
		return false, err
	}
	if admin {
		return true, nil
	}
	viewer, err := orgroles.UserHasRole(ctx, pool, userID, orgID, orgroles.RoleOrgViewer)
	if err != nil {
		return false, err
	}
	return viewer, nil
}

// LookupTargetOrg returns the org id for a user or ErrTargetNotFound.
func LookupTargetOrg(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (uuid.UUID, error) {
	var orgID uuid.UUID
	err := pool.QueryRow(ctx, `SELECT org_id FROM "user".users WHERE id = $1`, userID).Scan(&orgID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.UUID{}, ErrTargetNotFound
	}
	if err != nil {
		return uuid.UUID{}, err
	}
	return orgID, nil
}

// ClientIP extracts the client IP from an HTTP request.
func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := len(xff); i > 0 {
			for j := 0; j < len(xff); j++ {
				if xff[j] == ',' {
					return xff[:j]
				}
			}
			return xff
		}
	}
	return r.RemoteAddr
}
