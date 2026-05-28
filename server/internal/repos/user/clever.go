package user

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FindByCleverID returns a user by clever_id or nil.
func FindByCleverID(ctx context.Context, pool *pgxpool.Pool, cleverID string) (*Row, error) {
	cid := strings.TrimSpace(cleverID)
	if cid == "" {
		return nil, nil
	}
	const q = `SELECT ` + userRowColumns + `
FROM "user".users WHERE clever_id = $1`
	return scanUserRow(ctx, pool, q, cid)
}

// FindByClassLinkID returns a user by classlink_id or nil.
func FindByClassLinkID(ctx context.Context, pool *pgxpool.Pool, classlinkSub string) (*Row, error) {
	s := strings.TrimSpace(classlinkSub)
	if s == "" {
		return nil, nil
	}
	const q = `SELECT ` + userRowColumns + `
FROM "user".users WHERE classlink_id = $1`
	return scanUserRow(ctx, pool, q, s)
}

// SetCleverID sets clever_id for a user (idempotent if already set to same value).
func SetCleverID(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, cleverID string) error {
	cid := strings.TrimSpace(cleverID)
	if cid == "" {
		return errors.New("user: empty clever_id")
	}
	tag, err := pool.Exec(ctx, `
UPDATE "user".users SET clever_id = $2
WHERE id = $1::uuid AND (clever_id IS NULL OR clever_id = $2)`, userID, cid)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("user: could not set clever_id (conflict or missing user)")
	}
	return nil
}

// SetClassLinkID sets classlink_id for a user.
func SetClassLinkID(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, sub string) error {
	s := strings.TrimSpace(sub)
	if s == "" {
		return errors.New("user: empty classlink_id")
	}
	tag, err := pool.Exec(ctx, `
UPDATE "user".users SET classlink_id = $2
WHERE id = $1::uuid AND (classlink_id IS NULL OR classlink_id = $2)`, userID, s)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("user: could not set classlink_id (conflict or missing user)")
	}
	return nil
}

// UpdateCleverMinorFlag sets is_minor from Clever COPPA signal.
func UpdateCleverMinorFlag(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, isMinor bool) error {
	_, err := pool.Exec(ctx, `UPDATE "user".users SET is_minor = $2 WHERE id = $1::uuid`, userID, isMinor)
	return err
}

// InsertUserWithClever creates a user with clever_id and optional minor flag.
func InsertUserWithClever(ctx context.Context, pool *pgxpool.Pool, email, passwordHash string, displayName *string, cleverID string, isMinor bool) (*Row, error) {
	cid := strings.TrimSpace(cleverID)
	const q = `INSERT INTO "user".users (email, password_hash, display_name, clever_id, is_minor, org_id)
VALUES ($1, $2, $3, $4, $5, (SELECT id FROM tenant.organizations WHERE slug = 'default' LIMIT 1))
RETURNING ` + userRowColumns
	row := pool.QueryRow(ctx, q, email, passwordHash, displayName, cid, isMinor)
	return scanInsertedUserRow(row)
}

// InsertUserWithClassLink creates a user with classlink_id.
func InsertUserWithClassLink(ctx context.Context, pool *pgxpool.Pool, email, passwordHash string, displayName *string, classlinkSub string) (*Row, error) {
	s := strings.TrimSpace(classlinkSub)
	const q = `INSERT INTO "user".users (email, password_hash, display_name, classlink_id, org_id)
VALUES ($1, $2, $3, $4, (SELECT id FROM tenant.organizations WHERE slug = 'default' LIMIT 1))
RETURNING ` + userRowColumns
	row := pool.QueryRow(ctx, q, email, passwordHash, displayName, s)
	return scanInsertedUserRow(row)
}
