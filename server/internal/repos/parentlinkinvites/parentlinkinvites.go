// Package parentlinkinvites stores one-time activate tokens for parent/guardian invites (PP.1).
package parentlinkinvites

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Invite is one parent_link_invites row.
type Invite struct {
	ID            uuid.UUID
	OrgID         uuid.UUID
	StudentUserID uuid.UUID
	ParentUserID  uuid.UUID
	LinkID        uuid.UUID
	Email         string
	InvitedBy     *uuid.UUID
	ExpiresAt     time.Time
	ConsumedAt    *time.Time
	CreatedAt     time.Time
}

// HashToken returns a hex-encoded SHA-256 of the raw token.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// ReplaceForLink upserts an invite row for a link (invalidates prior unused token).
func ReplaceForLink(
	ctx context.Context,
	pool *pgxpool.Pool,
	orgID, studentID, parentID, linkID uuid.UUID,
	email string,
	invitedBy *uuid.UUID,
	tokenHash string,
	expiresAt time.Time,
) (*Invite, error) {
	var inv Invite
	var invitedByOut *uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO "user".parent_link_invites (
  org_id, student_user_id, parent_user_id, link_id, email, invited_by, token_hash, expires_at, consumed_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULL)
ON CONFLICT (link_id) DO UPDATE SET
  org_id = EXCLUDED.org_id,
  student_user_id = EXCLUDED.student_user_id,
  parent_user_id = EXCLUDED.parent_user_id,
  email = EXCLUDED.email,
  invited_by = EXCLUDED.invited_by,
  token_hash = EXCLUDED.token_hash,
  expires_at = EXCLUDED.expires_at,
  consumed_at = NULL,
  created_at = now()
RETURNING id, org_id, student_user_id, parent_user_id, link_id, email, invited_by, expires_at, consumed_at, created_at
`, orgID, studentID, parentID, linkID, email, invitedBy, tokenHash, expiresAt).Scan(
		&inv.ID, &inv.OrgID, &inv.StudentUserID, &inv.ParentUserID, &inv.LinkID, &inv.Email,
		&invitedByOut, &inv.ExpiresAt, &inv.ConsumedAt, &inv.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	inv.InvitedBy = invitedByOut
	return &inv, nil
}

// FindByTokenHash returns an unused invite matching the hash, or nil.
func FindByTokenHash(ctx context.Context, pool *pgxpool.Pool, tokenHash string) (*Invite, error) {
	var inv Invite
	var invitedByOut *uuid.UUID
	err := pool.QueryRow(ctx, `
SELECT id, org_id, student_user_id, parent_user_id, link_id, email, invited_by, expires_at, consumed_at, created_at
FROM "user".parent_link_invites
WHERE token_hash = $1 AND consumed_at IS NULL
`, tokenHash).Scan(
		&inv.ID, &inv.OrgID, &inv.StudentUserID, &inv.ParentUserID, &inv.LinkID, &inv.Email,
		&invitedByOut, &inv.ExpiresAt, &inv.ConsumedAt, &inv.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	inv.InvitedBy = invitedByOut
	return &inv, nil
}

// MarkConsumed sets consumed_at when still unused.
func MarkConsumed(ctx context.Context, pool *pgxpool.Pool, inviteID uuid.UUID) (bool, error) {
	tag, err := pool.Exec(ctx, `
UPDATE "user".parent_link_invites
SET consumed_at = now()
WHERE id = $1 AND consumed_at IS NULL
`, inviteID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// FindActiveByLinkID returns the unused invite for a link, or nil.
func FindActiveByLinkID(ctx context.Context, pool *pgxpool.Pool, linkID uuid.UUID) (*Invite, error) {
	var inv Invite
	var invitedByOut *uuid.UUID
	err := pool.QueryRow(ctx, `
SELECT id, org_id, student_user_id, parent_user_id, link_id, email, invited_by, expires_at, consumed_at, created_at
FROM "user".parent_link_invites
WHERE link_id = $1 AND consumed_at IS NULL
ORDER BY created_at DESC
LIMIT 1
`, linkID).Scan(
		&inv.ID, &inv.OrgID, &inv.StudentUserID, &inv.ParentUserID, &inv.LinkID, &inv.Email,
		&invitedByOut, &inv.ExpiresAt, &inv.ConsumedAt, &inv.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	inv.InvitedBy = invitedByOut
	return &inv, nil
}
