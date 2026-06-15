// Package apitokens stores personal API access keys.
package apitokens

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/repos/enrollment"
)

const (
	tokenPrefixLabel = "ltk_"
	maxPersonalTokens = 20
)

// Row is a stored access key (secret is never persisted).
type Row struct {
	ID          uuid.UUID
	OwnerUserID uuid.UUID
	Label       string
	TokenPrefix string
	Scopes      []string
	CourseIDs   []uuid.UUID
	ExpiresAt   *time.Time
	LastUsedAt  *time.Time
	RevokedAt   *time.Time
	CreatedAt   time.Time
}

// GenerateSecret returns a new ltk_ secret and its SHA-256 hex hash and 8-char prefix.
func GenerateSecret() (secret, hashHex, prefix string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", "", err
	}
	secret = tokenPrefixLabel + base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(secret))
	hashHex = hex.EncodeToString(sum[:])
	if len(secret) < 8 {
		return "", "", "", errors.New("generated token too short")
	}
	prefix = secret[:8]
	return secret, hashHex, prefix, nil
}

func hashSecret(raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return hex.EncodeToString(sum[:])
}

// CountActiveForUser returns non-revoked tokens owned by the user.
func CountActiveForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
SELECT COUNT(*)::int FROM auth.api_tokens
WHERE owner_user_id = $1 AND revoked_at IS NULL
`, userID).Scan(&n)
	return n, err
}

// Insert creates a token row; rawSecret is returned once to the caller.
// courseIDs empty means all courses the owner can access.
func Insert(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, label string, scopes []string, courseIDs []uuid.UUID, expiresAt *time.Time) (Row, string, error) {
	label = strings.TrimSpace(label)
	if label == "" {
		return Row{}, "", errors.New("label required")
	}
	if len(scopes) == 0 {
		return Row{}, "", errors.New("at least one scope required")
	}
	n, err := CountActiveForUser(ctx, pool, userID)
	if err != nil {
		return Row{}, "", err
	}
	if n >= maxPersonalTokens {
		return Row{}, "", errors.New("maximum number of access keys reached")
	}
	secret, hashHex, prefix, err := GenerateSecret()
	if err != nil {
		return Row{}, "", err
	}
	if len(courseIDs) == 0 {
		courseIDs = []uuid.UUID{}
	}
	var row Row
	err = pool.QueryRow(ctx, `
INSERT INTO auth.api_tokens (owner_user_id, label, token_hash, token_prefix, scopes, course_ids, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, owner_user_id, label, token_prefix, scopes, course_ids, expires_at, last_used_at, revoked_at, created_at
`, userID, label, hashHex, prefix, scopes, courseIDs, expiresAt).Scan(
		&row.ID, &row.OwnerUserID, &row.Label, &row.TokenPrefix, &row.Scopes, &row.CourseIDs, &row.ExpiresAt, &row.LastUsedAt, &row.RevokedAt, &row.CreatedAt,
	)
	if err != nil {
		return Row{}, "", err
	}
	return row, secret, nil
}

// ListByUser returns active and revoked tokens for the owner (newest first).
func ListByUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]Row, error) {
	rows, err := pool.Query(ctx, `
SELECT id, owner_user_id, label, token_prefix, scopes, course_ids, expires_at, last_used_at, revoked_at, created_at
FROM auth.api_tokens
WHERE owner_user_id = $1
ORDER BY created_at DESC
LIMIT 50
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Row
	for rows.Next() {
		var r Row
		if err := rows.Scan(&r.ID, &r.OwnerUserID, &r.Label, &r.TokenPrefix, &r.Scopes, &r.CourseIDs, &r.ExpiresAt, &r.LastUsedAt, &r.RevokedAt, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// RevokeForUser marks a token revoked when owned by userID.
func RevokeForUser(ctx context.Context, pool *pgxpool.Pool, userID, tokenID uuid.UUID) (bool, error) {
	tag, err := pool.Exec(ctx, `
UPDATE auth.api_tokens SET revoked_at = NOW()
WHERE id = $1 AND owner_user_id = $2 AND revoked_at IS NULL
`, tokenID, userID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// ResolvedToken is an active access key matched from a bearer secret.
type ResolvedToken struct {
	ID          uuid.UUID
	OwnerUserID uuid.UUID
	Scopes      []string
	CourseIDs   []uuid.UUID
}

// ResolveBearer looks up a bearer secret and returns the token when valid.
func ResolveBearer(ctx context.Context, pool *pgxpool.Pool, rawToken string, now time.Time) (*ResolvedToken, error) {
	raw := strings.TrimSpace(rawToken)
	if raw == "" || !strings.HasPrefix(raw, tokenPrefixLabel) {
		return nil, errors.New("not an access key")
	}
	h := hashSecret(raw)
	var rt ResolvedToken
	var expiresAt *time.Time
	err := pool.QueryRow(ctx, `
SELECT id, owner_user_id, scopes, course_ids, expires_at
FROM auth.api_tokens
WHERE token_hash = $1 AND revoked_at IS NULL
`, h).Scan(&rt.ID, &rt.OwnerUserID, &rt.Scopes, &rt.CourseIDs, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("invalid access key")
	}
	if err != nil {
		return nil, err
	}
	if expiresAt != nil && !expiresAt.After(now) {
		return nil, errors.New("expired access key")
	}
	// Best-effort last-used timestamp; ignore errors.
	_, _ = pool.Exec(ctx, `UPDATE auth.api_tokens SET last_used_at = NOW() WHERE id = $1`, rt.ID)
	return &rt, nil
}

// MaskedDisplay returns a safe display string like ltk_abcd…wxyz.
func MaskedDisplay(prefix string) string {
	p := strings.TrimSpace(prefix)
	if len(p) >= 8 {
		return p + "…"
	}
	return tokenPrefixLabel + "…"
}

// NormalizeCourseIDs deduplicates course UUID strings.
func NormalizeCourseIDs(raw []string) ([]uuid.UUID, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	seen := make(map[uuid.UUID]struct{}, len(raw))
	out := make([]uuid.UUID, 0, len(raw))
	for _, s := range raw {
		id, err := uuid.Parse(strings.TrimSpace(s))
		if err != nil {
			return nil, errors.New("invalid course id")
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, nil
}

// ValidateCourseIDsForUser ensures each course id is accessible to the user.
func ValidateCourseIDsForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, courseIDs []uuid.UUID) error {
	if len(courseIDs) == 0 {
		return nil
	}
	for _, cid := range courseIDs {
		ok, err := enrollment.UserHasAccessByCourseID(ctx, pool, cid, userID)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("course not accessible")
		}
	}
	return nil
}
