// Package calendartokens stores short-lived capability tokens for calendar feed URLs (plan 16.5).
package calendartokens

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
	tokenPrefixLabel = "lcf_"
	tokenLifetime    = 30 * 24 * time.Hour
)

// Row is a stored calendar capability token (secret is never persisted).
type Row struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ExpiresAt time.Time
	CreatedAt time.Time
}

// GenerateSecret returns a new lcf_ secret and its SHA-256 hex hash.
func GenerateSecret() (secret, hashHex string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	secret = tokenPrefixLabel + base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(secret))
	hashHex = hex.EncodeToString(sum[:])
	return secret, hashHex, nil
}

func hashSecret(raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return hex.EncodeToString(sum[:])
}

// RotateForUser deletes existing tokens for the user and inserts a fresh one.
func RotateForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, now time.Time) (Row, string, error) {
	secret, hashHex, err := GenerateSecret()
	if err != nil {
		return Row{}, "", err
	}
	expiresAt := now.Add(tokenLifetime)
	tx, err := pool.Begin(ctx)
	if err != nil {
		return Row{}, "", err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM auth.calendar_tokens WHERE user_id = $1`, userID); err != nil {
		return Row{}, "", err
	}

	var row Row
	err = tx.QueryRow(ctx, `
INSERT INTO auth.calendar_tokens (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING id, user_id, expires_at, created_at
`, userID, hashHex, expiresAt).Scan(&row.ID, &row.UserID, &row.ExpiresAt, &row.CreatedAt)
	if err != nil {
		return Row{}, "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return Row{}, "", err
	}
	return row, secret, nil
}

// GetActiveForUser returns the current non-expired token row for a user, if any.
func GetActiveForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, now time.Time) (*Row, error) {
	var row Row
	err := pool.QueryRow(ctx, `
SELECT id, user_id, expires_at, created_at
FROM auth.calendar_tokens
WHERE user_id = $1 AND expires_at > $2
ORDER BY created_at DESC
LIMIT 1
`, userID, now).Scan(&row.ID, &row.UserID, &row.ExpiresAt, &row.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// ResolveQueryParam validates a ?token= value and returns the owning user id.
func ResolveQueryParam(ctx context.Context, pool *pgxpool.Pool, rawToken string, now time.Time) (uuid.UUID, error) {
	raw := strings.TrimSpace(rawToken)
	if raw == "" || !strings.HasPrefix(raw, tokenPrefixLabel) {
		return uuid.Nil, errors.New("invalid calendar token")
	}
	h := hashSecret(raw)
	var userID uuid.UUID
	var expiresAt time.Time
	err := pool.QueryRow(ctx, `
SELECT user_id, expires_at
FROM auth.calendar_tokens
WHERE token_hash = $1
`, h).Scan(&userID, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, errors.New("invalid calendar token")
	}
	if err != nil {
		return uuid.Nil, err
	}
	if !expiresAt.After(now) {
		return uuid.Nil, errors.New("expired calendar token")
	}
	return userID, nil
}
