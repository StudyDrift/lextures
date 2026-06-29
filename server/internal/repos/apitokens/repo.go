// Package apitokens stores personal and institutional API access keys (plan 16.2).
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
)

const (
	tokenPrefixLabel  = "ltk_"
	maxPersonalTokens = 20
)

// Row is a stored access key (secret is never persisted).
type Row struct {
	ID                 uuid.UUID
	OwnerUserID        *uuid.UUID
	OrgID              *uuid.UUID
	ServiceAccountName *string
	Label              string
	TokenPrefix        string
	Scopes             []string
	CourseIDs          []uuid.UUID
	ExpiresAt          *time.Time
	LastUsedAt         *time.Time
	LastUsedIPHash     *string
	RevokedAt          *time.Time
	RotatedFromID      *uuid.UUID
	CreatedAt          time.Time
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

func scanRow(row pgx.Row) (Row, error) {
	var r Row
	err := row.Scan(
		&r.ID, &r.OwnerUserID, &r.OrgID, &r.ServiceAccountName, &r.Label, &r.TokenPrefix,
		&r.Scopes, &r.CourseIDs, &r.ExpiresAt, &r.LastUsedAt, &r.LastUsedIPHash,
		&r.RevokedAt, &r.RotatedFromID, &r.CreatedAt,
	)
	return r, err
}

const rowSelectCols = `
id, owner_user_id, org_id, service_account_name, label, token_prefix, scopes, course_ids,
expires_at, last_used_at, last_used_ip_hash, revoked_at, rotated_from_id, created_at
`

// CountActiveForUser returns non-revoked personal tokens owned by the user.
func CountActiveForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
SELECT COUNT(*)::int FROM auth.api_tokens
WHERE owner_user_id = $1 AND revoked_at IS NULL
`, userID).Scan(&n)
	return n, err
}

// CountActiveServiceForOrg returns non-revoked service tokens for an org.
func CountActiveServiceForOrg(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
SELECT COUNT(*)::int FROM auth.api_tokens
WHERE org_id = $1 AND owner_user_id IS NULL AND revoked_at IS NULL
`, orgID).Scan(&n)
	return n, err
}

// Insert creates a personal token row; rawSecret is returned once to the caller.
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
RETURNING `+rowSelectCols+`
`, userID, label, hashHex, prefix, scopes, courseIDs, expiresAt).Scan(
		&row.ID, &row.OwnerUserID, &row.OrgID, &row.ServiceAccountName, &row.Label, &row.TokenPrefix,
		&row.Scopes, &row.CourseIDs, &row.ExpiresAt, &row.LastUsedAt, &row.LastUsedIPHash,
		&row.RevokedAt, &row.RotatedFromID, &row.CreatedAt,
	)
	if err != nil {
		return Row{}, "", err
	}
	return row, secret, nil
}

// InsertService creates an institutional service token for an org.
func InsertService(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, serviceAccountName, label string, scopes []string, expiresAt *time.Time) (Row, string, error) {
	serviceAccountName = strings.TrimSpace(serviceAccountName)
	label = strings.TrimSpace(label)
	if serviceAccountName == "" {
		return Row{}, "", errors.New("service account name required")
	}
	if label == "" {
		label = serviceAccountName
	}
	if len(scopes) == 0 {
		return Row{}, "", errors.New("at least one scope required")
	}
	n, err := CountActiveServiceForOrg(ctx, pool, orgID)
	if err != nil {
		return Row{}, "", err
	}
	if n >= maxServiceTokensPerOrg {
		return Row{}, "", errors.New("maximum number of service tokens reached")
	}
	secret, hashHex, prefix, err := GenerateSecret()
	if err != nil {
		return Row{}, "", err
	}
	var row Row
	err = pool.QueryRow(ctx, `
INSERT INTO auth.api_tokens (org_id, service_account_name, label, token_hash, token_prefix, scopes, course_ids, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, '{}', $7)
RETURNING `+rowSelectCols+`
`, orgID, serviceAccountName, label, hashHex, prefix, scopes, expiresAt).Scan(
		&row.ID, &row.OwnerUserID, &row.OrgID, &row.ServiceAccountName, &row.Label, &row.TokenPrefix,
		&row.Scopes, &row.CourseIDs, &row.ExpiresAt, &row.LastUsedAt, &row.LastUsedIPHash,
		&row.RevokedAt, &row.RotatedFromID, &row.CreatedAt,
	)
	if err != nil {
		return Row{}, "", err
	}
	return row, secret, nil
}

// ListByUser returns active and revoked personal tokens for the owner (newest first).
func ListByUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]Row, error) {
	rows, err := pool.Query(ctx, `
SELECT `+rowSelectCols+`
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
		r, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListByOrg returns personal and service tokens belonging to an org (newest first).
func ListByOrg(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) ([]Row, error) {
	rows, err := pool.Query(ctx, `
SELECT t.id, t.owner_user_id, t.org_id, t.service_account_name, t.label, t.token_prefix, t.scopes, t.course_ids,
       t.expires_at, t.last_used_at, t.last_used_ip_hash, t.revoked_at, t.rotated_from_id, t.created_at
FROM auth.api_tokens t
LEFT JOIN "user".users u ON u.id = t.owner_user_id
WHERE t.org_id = $1 OR u.org_id = $1
ORDER BY t.created_at DESC
LIMIT 100
`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Row
	for rows.Next() {
		r, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetByID loads a token row by id.
func GetByID(ctx context.Context, pool *pgxpool.Pool, tokenID uuid.UUID) (*Row, error) {
	row, err := scanRow(pool.QueryRow(ctx, `
SELECT `+rowSelectCols+` FROM auth.api_tokens WHERE id = $1
`, tokenID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// RevokeForUser marks a personal token revoked when owned by userID.
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

// RevokeByID marks any token revoked (admin).
func RevokeByID(ctx context.Context, pool *pgxpool.Pool, tokenID uuid.UUID) (bool, error) {
	tag, err := pool.Exec(ctx, `
UPDATE auth.api_tokens SET revoked_at = NOW()
WHERE id = $1 AND revoked_at IS NULL
`, tokenID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// RotateForUser creates a replacement token and shortens the old token overlap window.
func RotateForUser(ctx context.Context, pool *pgxpool.Pool, userID, tokenID uuid.UUID, overlap time.Duration, now time.Time) (Row, string, error) {
	if overlap <= 0 {
		overlap = defaultRotateOverlap
	}
	old, err := GetByID(ctx, pool, tokenID)
	if err != nil {
		return Row{}, "", err
	}
	if old == nil || old.RevokedAt != nil || old.OwnerUserID == nil || *old.OwnerUserID != userID {
		return Row{}, "", errors.New("not found")
	}
	overlapEnd := now.Add(overlap)
	oldExpires := overlapEnd
	if old.ExpiresAt != nil && old.ExpiresAt.Before(oldExpires) {
		oldExpires = *old.ExpiresAt
	}
	_, err = pool.Exec(ctx, `
UPDATE auth.api_tokens SET expires_at = $2 WHERE id = $1 AND revoked_at IS NULL
`, tokenID, oldExpires)
	if err != nil {
		return Row{}, "", err
	}

	secret, hashHex, prefix, err := GenerateSecret()
	if err != nil {
		return Row{}, "", err
	}
	var row Row
	err = pool.QueryRow(ctx, `
INSERT INTO auth.api_tokens (owner_user_id, label, token_hash, token_prefix, scopes, course_ids, expires_at, rotated_from_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING `+rowSelectCols+`
`, userID, old.Label, hashHex, prefix, old.Scopes, old.CourseIDs, old.ExpiresAt, tokenID).Scan(
		&row.ID, &row.OwnerUserID, &row.OrgID, &row.ServiceAccountName, &row.Label, &row.TokenPrefix,
		&row.Scopes, &row.CourseIDs, &row.ExpiresAt, &row.LastUsedAt, &row.LastUsedIPHash,
		&row.RevokedAt, &row.RotatedFromID, &row.CreatedAt,
	)
	if err != nil {
		return Row{}, "", err
	}
	return row, secret, nil
}

// ResolvedToken is an active access key matched from a bearer secret.
type ResolvedToken struct {
	ID                 uuid.UUID
	OwnerUserID        *uuid.UUID
	OrgID              *uuid.UUID
	ServiceAccountName *string
	Scopes             []string
	CourseIDs          []uuid.UUID
	// RateLimitPerMin is the per-token quota override (plan 17.6 FR-6); nil uses the deployment default.
	RateLimitPerMin *int
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
SELECT id, owner_user_id, org_id, service_account_name, scopes, course_ids, expires_at, rate_limit_per_min
FROM auth.api_tokens
WHERE token_hash = $1 AND revoked_at IS NULL
`, h).Scan(&rt.ID, &rt.OwnerUserID, &rt.OrgID, &rt.ServiceAccountName, &rt.Scopes, &rt.CourseIDs, &expiresAt, &rt.RateLimitPerMin)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("invalid access key")
	}
	if err != nil {
		return nil, err
	}
	if expiresAt != nil && !expiresAt.After(now) {
		return nil, errors.New("expired access key")
	}
	return &rt, nil
}

// OrgIDForToken returns the org id associated with a token (service or personal owner org).
func OrgIDForToken(ctx context.Context, pool *pgxpool.Pool, tokenID uuid.UUID) (*uuid.UUID, error) {
	row, err := GetByID(ctx, pool, tokenID)
	if err != nil || row == nil {
		return nil, err
	}
	if row.OrgID != nil {
		return row.OrgID, nil
	}
	if row.OwnerUserID == nil {
		return nil, nil
	}
	var orgID uuid.UUID
	err = pool.QueryRow(ctx, `SELECT org_id FROM "user".users WHERE id = $1`, *row.OwnerUserID).Scan(&orgID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &orgID, err
}

// MaskedDisplay returns a safe display string like ltk_abcd…
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

// IsServiceToken reports whether the row is an org-scoped service token.
func (r Row) IsServiceToken() bool {
	return r.OwnerUserID == nil && r.OrgID != nil
}
