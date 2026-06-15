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

	"github.com/lextures/lextures/server/internal/repos/apitokens"
)

// APITokenAuth carries scope grants from a personal access key.
type APITokenAuth struct {
	TokenID   uuid.UUID
	Scopes    []string
	CourseIDs []uuid.UUID
}

type apiTokenAuthKey struct{}

// APITokenFromContext returns access-key auth metadata when the request used an API token.
func APITokenFromContext(ctx context.Context) (*APITokenAuth, bool) {
	v, ok := ctx.Value(apiTokenAuthKey{}).(*APITokenAuth)
	return v, ok && v != nil
}

// UserFromRequestOrAccessKey authenticates via login JWT or personal access key (ltk_…).
func UserFromRequestOrAccessKey(r *http.Request, signer *JWTSigner, pool *pgxpool.Pool) (AuthUser, context.Context, error) {
	token, ok := BearerToken(r.Header)
	if !ok {
		return AuthUser{}, r.Context(), ErrInvalidToken
	}
	if strings.HasPrefix(token, "ltk_") {
		if pool == nil {
			return AuthUser{}, r.Context(), ErrInvalidToken
		}
		rt, err := apitokens.ResolveBearer(r.Context(), pool, token, timeNow())
		if err != nil {
			return AuthUser{}, r.Context(), ErrInvalidToken
		}
		row, err := lookupAccessKeyOwner(r.Context(), pool, rt.OwnerUserID)
		if err != nil {
			return AuthUser{}, r.Context(), ErrInvalidToken
		}
		ctx := context.WithValue(r.Context(), apiTokenAuthKey{}, &APITokenAuth{
			TokenID:   rt.ID,
			Scopes:    rt.Scopes,
			CourseIDs: rt.CourseIDs,
		})
		return AuthUser{UserID: row.ID.String(), Email: row.Email}, ctx, nil
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
