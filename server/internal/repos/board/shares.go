package board

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/auth"
)

// Share capabilities (VC.6).
const (
	ShareCapabilityView       = "view"
	ShareCapabilityContribute = "contribute"
	shareTokenPrefix          = "bsh_"
)

// BoardShare is a share-link row (raw token never stored).
type BoardShare struct {
	ID               string     `json:"id"`
	BoardID          string     `json:"boardId"`
	Capability       string     `json:"capability"`
	HasPassword      bool       `json:"hasPassword"`
	ExpiresAt        *time.Time `json:"expiresAt,omitempty"`
	CreatedBy        *string    `json:"createdBy,omitempty"`
	RevokedAt        *time.Time `json:"revokedAt,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
	PasswordHash     string     `json:"-"`
	TokenHash        string     `json:"-"`
}

// CreateShareInput creates a new share link.
type CreateShareInput struct {
	Capability string
	Password   string
	ExpiresAt  *time.Time
}

// NormalizeShareCapability validates capability.
func NormalizeShareCapability(raw string) (string, error) {
	v := strings.ToLower(strings.TrimSpace(raw))
	switch v {
	case ShareCapabilityView, ShareCapabilityContribute:
		return v, nil
	default:
		return "", fmt.Errorf("board: invalid share capability")
	}
}

// GenerateShareToken returns a raw URL-safe token (≥128-bit) and its SHA-256 hex hash.
func GenerateShareToken() (raw, hashHex string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	raw = shareTokenPrefix + base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(raw))
	hashHex = hex.EncodeToString(sum[:])
	return raw, hashHex, nil
}

func hashShareToken(raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return hex.EncodeToString(sum[:])
}

// TokenMatches compares a raw token to a stored hash in constant time.
func TokenMatches(raw, hashHex string) bool {
	got := hashShareToken(raw)
	return subtle.ConstantTimeCompare([]byte(got), []byte(strings.TrimSpace(hashHex))) == 1
}

// CreateShare inserts a share link and returns the row plus the raw token (shown once).
func CreateShare(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseCode, boardID string,
	createdBy uuid.UUID,
	in CreateShareInput,
) (*BoardShare, string, error) {
	cap, err := NormalizeShareCapability(in.Capability)
	if err != nil {
		return nil, "", err
	}
	bid, err := uuid.Parse(boardID)
	if err != nil {
		return nil, "", nil
	}
	raw, hashHex, err := GenerateShareToken()
	if err != nil {
		return nil, "", err
	}
	var pwHash *string
	if strings.TrimSpace(in.Password) != "" {
		h, err := auth.HashPassword(strings.TrimSpace(in.Password))
		if err != nil {
			return nil, "", err
		}
		pwHash = &h
	}
	var id uuid.UUID
	var boardUUID uuid.UUID
	var createdByNull uuid.NullUUID
	var expiresAt *time.Time
	var revokedAt *time.Time
	var createdAt time.Time
	var storedHash *string
	err = pool.QueryRow(ctx, `
		INSERT INTO board.board_shares (board_id, token_hash, capability, password_hash, expires_at, created_by)
		SELECT b.id, $3, $4, $5, $6, $7
		FROM board.boards b
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE c.course_code = $1 AND b.id = $2
		RETURNING id, board_id, capability, password_hash, expires_at, created_by, revoked_at, created_at
	`, courseCode, bid, hashHex, cap, pwHash, in.ExpiresAt, createdBy).Scan(
		&id, &boardUUID, &cap, &storedHash, &expiresAt, &createdByNull, &revokedAt, &createdAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	s := &BoardShare{
		ID:          id.String(),
		BoardID:     boardUUID.String(),
		Capability:  cap,
		HasPassword: storedHash != nil && *storedHash != "",
		ExpiresAt:   expiresAt,
		RevokedAt:   revokedAt,
		CreatedAt:   createdAt,
	}
	if createdByNull.Valid {
		u := createdByNull.UUID.String()
		s.CreatedBy = &u
	}
	if storedHash != nil {
		s.PasswordHash = *storedHash
	}
	return s, raw, nil
}

// ListShares returns non-deleted share metadata for a board (includes revoked for audit UI).
func ListShares(ctx context.Context, pool *pgxpool.Pool, courseCode, boardID string) ([]BoardShare, error) {
	bid, err := uuid.Parse(boardID)
	if err != nil {
		return nil, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT s.id, s.board_id, s.capability, s.password_hash, s.expires_at, s.created_by, s.revoked_at, s.created_at
		FROM board.board_shares s
		INNER JOIN board.boards b ON b.id = s.board_id
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE c.course_code = $1 AND b.id = $2
		ORDER BY s.created_at DESC
	`, courseCode, bid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]BoardShare, 0)
	for rows.Next() {
		var id, boardUUID uuid.UUID
		var createdBy uuid.NullUUID
		var pwHash *string
		var s BoardShare
		if err := rows.Scan(&id, &boardUUID, &s.Capability, &pwHash, &s.ExpiresAt, &createdBy, &s.RevokedAt, &s.CreatedAt); err != nil {
			return nil, err
		}
		s.ID = id.String()
		s.BoardID = boardUUID.String()
		s.HasPassword = pwHash != nil && *pwHash != ""
		if createdBy.Valid {
			u := createdBy.UUID.String()
			s.CreatedBy = &u
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// RevokeShare sets revoked_at. Returns false if not found.
func RevokeShare(ctx context.Context, pool *pgxpool.Pool, courseCode, boardID, shareID string) (bool, error) {
	bid, err := uuid.Parse(boardID)
	if err != nil {
		return false, nil
	}
	sid, err := uuid.Parse(shareID)
	if err != nil {
		return false, nil
	}
	tag, err := pool.Exec(ctx, `
		UPDATE board.board_shares s
		SET revoked_at = COALESCE(s.revoked_at, NOW())
		FROM board.boards b
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE s.board_id = b.id AND c.course_code = $1 AND b.id = $2 AND s.id = $3 AND s.revoked_at IS NULL
	`, courseCode, bid, sid)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// ResolvedShare is a validated share token lookup result.
type ResolvedShare struct {
	Share BoardShare
	Board Board
	CourseCode string
	CourseID   string
}

// ResolveShareToken looks up an active (non-revoked, non-expired) share by raw token.
func ResolveShareToken(ctx context.Context, pool *pgxpool.Pool, rawToken string, now time.Time) (*ResolvedShare, error) {
	raw := strings.TrimSpace(rawToken)
	if raw == "" || !strings.HasPrefix(raw, shareTokenPrefix) {
		return nil, nil
	}
	h := hashShareToken(raw)
	var (
		id, boardUUID uuid.UUID
		createdBy     uuid.NullUUID
		pwHash        *string
		tokenHash     string
		s             BoardShare
		courseCode    string
	)
	err := pool.QueryRow(ctx, `
		SELECT
			s.id, s.board_id, s.token_hash, s.capability, s.password_hash, s.expires_at,
			s.created_by, s.revoked_at, s.created_at, c.course_code
		FROM board.board_shares s
		INNER JOIN board.boards b ON b.id = s.board_id
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE s.token_hash = $1
	`, h).Scan(
		&id, &boardUUID, &tokenHash, &s.Capability, &pwHash, &s.ExpiresAt,
		&createdBy, &s.RevokedAt, &s.CreatedAt, &courseCode,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !TokenMatches(raw, tokenHash) {
		return nil, nil
	}
	if s.RevokedAt != nil {
		return nil, nil
	}
	if s.ExpiresAt != nil && !s.ExpiresAt.After(now) {
		return nil, nil
	}
	b, err := Get(ctx, pool, courseCode, boardUUID.String())
	if err != nil || b == nil {
		return nil, err
	}
	s.ID = id.String()
	s.BoardID = boardUUID.String()
	s.TokenHash = tokenHash
	s.HasPassword = pwHash != nil && *pwHash != ""
	if pwHash != nil {
		s.PasswordHash = *pwHash
	}
	if createdBy.Valid {
		u := createdBy.UUID.String()
		s.CreatedBy = &u
	}
	return &ResolvedShare{
		Share:      s,
		Board:      *b,
		CourseCode: courseCode,
		CourseID:   b.CourseID,
	}, nil
}

// VerifySharePassword checks an optional share password (constant-time via Argon2 verify).
func VerifySharePassword(plain string, share BoardShare) (bool, error) {
	if !share.HasPassword || share.PasswordHash == "" {
		return true, nil
	}
	return auth.VerifyPassword(plain, share.PasswordHash)
}
