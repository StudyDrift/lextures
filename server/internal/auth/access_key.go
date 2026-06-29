package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/api"
	"github.com/lextures/lextures/server/internal/apierr"
	"github.com/lextures/lextures/server/internal/repos/apitokens"
	impersonationrepo "github.com/lextures/lextures/server/internal/repos/impersonation"
)

// APITokenAuth carries scope grants from a personal or service access key.
type APITokenAuth struct {
	TokenID            uuid.UUID
	Scopes             []string
	CourseIDs          []uuid.UUID
	OrgID              *uuid.UUID
	ServiceAccountName *string
	// RateLimitPerMin is the per-token quota override (plan 17.6 FR-6); nil uses the deployment default.
	RateLimitPerMin *int
}

type apiTokenAuthKey struct{}

// APITokenFromContext returns access-key auth metadata when the request used an API token.
func APITokenFromContext(ctx context.Context) (*APITokenAuth, bool) {
	v, ok := ctx.Value(apiTokenAuthKey{}).(*APITokenAuth)
	return v, ok && v != nil
}

// IsServiceTokenAuth reports whether the request authenticated with an org service token.
func IsServiceTokenAuth(ctx context.Context) bool {
	tok, ok := APITokenFromContext(ctx)
	return ok && tok != nil && tok.OrgID != nil && tok.ServiceAccountName != nil
}

// RequireAccessKeyScope writes 403 when an access key lacks the required scope; JWT sessions pass through.
func RequireAccessKeyScope(w http.ResponseWriter, ctx context.Context, required string) bool {
	tok, ok := APITokenFromContext(ctx)
	if !ok {
		return true
	}
	if api.HasScope(tok.Scopes, required) {
		return true
	}
	apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden,
		"Access key missing required scope: "+required+".")
	return false
}

// UserFromRequestOrAccessKey authenticates via login JWT, impersonation JWT, or access key (ltk_…).
func UserFromRequestOrAccessKey(r *http.Request, signer *JWTSigner, pool *pgxpool.Pool, ipHashKey string, tokensEnabled bool) (AuthUser, context.Context, error) {
	token, ok := BearerToken(r.Header)
	if !ok {
		return AuthUser{}, r.Context(), ErrInvalidToken
	}
	if strings.HasPrefix(token, "ltk_") {
		if !tokensEnabled || pool == nil {
			return AuthUser{}, r.Context(), ErrInvalidToken
		}
		rt, err := apitokens.ResolveBearer(r.Context(), pool, token, timeNow())
		if err != nil {
			return AuthUser{}, r.Context(), ErrInvalidToken
		}
		ipHash := apitokens.HashClientIP(ipHashKey, apitokens.ClientIPFromRequest(r))
		apitokens.RecordUsage(rt.ID, ipHash)

		meta := &APITokenAuth{
			TokenID:         rt.ID,
			Scopes:          rt.Scopes,
			CourseIDs:       rt.CourseIDs,
			OrgID:           rt.OrgID,
			RateLimitPerMin: rt.RateLimitPerMin,
		}
		if rt.ServiceAccountName != nil {
			meta.ServiceAccountName = rt.ServiceAccountName
		}

		if rt.OrgID != nil && rt.OwnerUserID == nil {
			email := "service-token"
			if rt.ServiceAccountName != nil && strings.TrimSpace(*rt.ServiceAccountName) != "" {
				email = strings.TrimSpace(*rt.ServiceAccountName)
			}
			ctx := context.WithValue(r.Context(), apiTokenAuthKey{}, meta)
			return AuthUser{
				UserID: uuid.Nil.String(),
				Email:  email,
				OrgID:  rt.OrgID.String(),
			}, ctx, nil
		}
		if rt.OwnerUserID == nil {
			return AuthUser{}, r.Context(), ErrInvalidToken
		}
		row, err := lookupAccessKeyOwner(r.Context(), pool, *rt.OwnerUserID)
		if err != nil {
			return AuthUser{}, r.Context(), ErrInvalidToken
		}
		ctx := context.WithValue(r.Context(), apiTokenAuthKey{}, meta)
		return AuthUser{UserID: row.ID.String(), Email: row.Email}, ctx, nil
	}
	if signer != nil && JWTType(token) == "impersonation" {
		imp, err := signer.VerifyImpersonation(token)
		if err != nil {
			return AuthUser{}, r.Context(), err
		}
		active, err := impersonationrepo.IsActive(r.Context(), pool, imp.JTI, timeNow())
		if err != nil {
			return AuthUser{}, r.Context(), ErrInvalidToken
		}
		if !active {
			return AuthUser{}, r.Context(), ErrInvalidToken
		}
		ctx := WithImpersonation(r.Context(), ImpersonationSession{
			AdminID:      imp.AdminID,
			TargetUserID: imp.TargetUserID,
			JTI:          imp.JTI,
		})
		return AuthUser{
			UserID:  imp.TargetUserID,
			Email:   imp.TargetEmail,
			OrgID:   imp.OrgID,
			OrgSlug: imp.OrgSlug,
		}, ctx, nil
	}
	u, err := signer.Verify(r.Context(), token)
	return u, r.Context(), err
}

type accessKeyOwner struct {
	ID    uuid.UUID
	Email string
}

func lookupAccessKeyOwner(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (*accessKeyOwner, error) {
	var row accessKeyOwner
	err := pool.QueryRow(ctx, `SELECT id, email FROM "user".users WHERE id = $1`, userID).Scan(&row.ID, &row.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvalidToken
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func timeNow() time.Time {
	return timeNowFn()
}

var timeNowFn = func() time.Time { return time.Now() }

// AccessKeyAllowsCourse is true when auth is not an access key, the key has no course
// restriction, or the given course id is in the key allowlist.
func AccessKeyAllowsCourse(ctx context.Context, courseID uuid.UUID) bool {
	tok, ok := APITokenFromContext(ctx)
	if !ok || len(tok.CourseIDs) == 0 {
		return true
	}
	for _, id := range tok.CourseIDs {
		if id == courseID {
			return true
		}
	}
	return false
}
