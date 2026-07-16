package transcripts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRecipientNotFound      = errors.New("recipient not found")
	ErrRecipientDuplicateKey  = errors.New("recipient canonical key already exists")
	ErrInvalidDeliveryMethod  = errors.New("delivery method not supported by recipient")
	ErrDeliveryNotOrgEnabled  = errors.New("delivery method not enabled for organization")
)

// Recipient is a directory entry for transcript delivery.
type Recipient struct {
	ID           uuid.UUID
	OrgID        *uuid.UUID
	Type         RecipientType
	Name         string
	CanonicalKey *string
	Capabilities []string
	Email        *string
	Address      json.RawMessage
	PeerConfig   json.RawMessage
	Verified     bool
	Active       bool
	CreatedAt    time.Time
}

// UpsertRecipientInput creates or updates a directory recipient.
type UpsertRecipientInput struct {
	OrgID        *uuid.UUID
	Type         RecipientType
	Name         string
	CanonicalKey *string
	Capabilities []string
	Email        *string
	Address      json.RawMessage
	PeerConfig   json.RawMessage
	Verified     *bool
	Active       *bool
}

// AdHocRecipientInput is an inline recipient when not found in the directory.
type AdHocRecipientInput struct {
	Type         RecipientType
	Name         string
	CanonicalKey *string
	Capabilities []string
	Email        *string
	Address      json.RawMessage
}

const recipientSelectColumns = `
id, org_id, type, name, canonical_key, capabilities, email, address, peer_config,
verified, active, created_at`

func scanRecipient(row pgx.Row, r *Recipient) error {
	var typ string
	var caps []string
	err := row.Scan(
		&r.ID, &r.OrgID, &typ, &r.Name, &r.CanonicalKey, &caps, &r.Email, &r.Address, &r.PeerConfig,
		&r.Verified, &r.Active, &r.CreatedAt,
	)
	if err != nil {
		return err
	}
	r.Type = RecipientType(typ)
	if caps == nil {
		caps = []string{}
	}
	r.Capabilities = caps
	return nil
}

// SearchRecipients typeahead over global + org-scoped active recipients.
func SearchRecipients(
	ctx context.Context,
	pool *pgxpool.Pool,
	orgID *uuid.UUID,
	q string,
	typ *RecipientType,
	limit int,
) ([]Recipient, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	q = strings.TrimSpace(q)
	args := []any{}
	argN := 1
	where := `active = TRUE AND (org_id IS NULL`
	if orgID != nil {
		where += fmt.Sprintf(` OR org_id = $%d`, argN)
		args = append(args, *orgID)
		argN++
	}
	where += `)`
	if typ != nil {
		where += fmt.Sprintf(` AND type = $%d`, argN)
		args = append(args, string(*typ))
		argN++
	}
	if q != "" {
		where += fmt.Sprintf(` AND (name ILIKE $%d OR COALESCE(canonical_key, '') ILIKE $%d OR COALESCE(email, '') ILIKE $%d)`, argN, argN, argN)
		args = append(args, "%"+q+"%")
		argN++
	}
	args = append(args, limit)
	rows, err := pool.Query(ctx, `
SELECT `+recipientSelectColumns+`
FROM transcripts.recipients
WHERE `+where+`
ORDER BY
  CASE WHEN type = 'self' THEN 0 ELSE 1 END,
  verified DESC,
  name ASC
LIMIT $`+fmt.Sprintf("%d", argN)+`
`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Recipient
	for rows.Next() {
		var r Recipient
		if err := scanRecipient(rows, &r); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetRecipient returns a recipient by id.
func GetRecipient(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Recipient, error) {
	var r Recipient
	err := scanRecipient(pool.QueryRow(ctx, `
SELECT `+recipientSelectColumns+`
FROM transcripts.recipients WHERE id = $1
`, id), &r)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRecipientNotFound
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// FindRecipientByCanonicalKey looks up by org scope + canonical key.
func FindRecipientByCanonicalKey(
	ctx context.Context,
	pool *pgxpool.Pool,
	orgID *uuid.UUID,
	canonicalKey string,
) (*Recipient, error) {
	key := NormalizeCanonicalKey(canonicalKey)
	if key == "" {
		return nil, ErrRecipientNotFound
	}
	var r Recipient
	err := scanRecipient(pool.QueryRow(ctx, `
SELECT `+recipientSelectColumns+`
FROM transcripts.recipients
WHERE canonical_key = $1
  AND COALESCE(org_id, '00000000-0000-0000-0000-000000000000'::uuid)
      = COALESCE($2::uuid, '00000000-0000-0000-0000-000000000000'::uuid)
LIMIT 1
`, key, orgID), &r)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRecipientNotFound
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// InsertRecipient creates a directory row; on unique canonical key conflict returns existing (dedupe).
func InsertRecipient(ctx context.Context, pool *pgxpool.Pool, in UpsertRecipientInput) (*Recipient, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, errors.New("recipient name is required")
	}
	caps := NormalizeCapabilities(in.Capabilities)
	var key *string
	if in.CanonicalKey != nil {
		k := NormalizeCanonicalKey(*in.CanonicalKey)
		if k != "" {
			key = &k
		}
	}
	if key == nil {
		k := CanonicalKeyFromName(name)
		if k != "" {
			key = &k
		}
	}
	if key != nil {
		if existing, err := FindRecipientByCanonicalKey(ctx, pool, in.OrgID, *key); err == nil {
			return existing, nil
		} else if !errors.Is(err, ErrRecipientNotFound) {
			return nil, err
		}
	}
	verified := false
	if in.Verified != nil {
		verified = *in.Verified
	}
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	var r Recipient
	err := scanRecipient(pool.QueryRow(ctx, `
INSERT INTO transcripts.recipients (
    org_id, type, name, canonical_key, capabilities, email, address, peer_config, verified, active
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING `+recipientSelectColumns+`
`, in.OrgID, string(in.Type), name, key, caps, in.Email, nullableJSON(in.Address), nullableJSON(in.PeerConfig), verified, active), &r)
	if err != nil {
		if isUniqueViolation(err) && key != nil {
			existing, findErr := FindRecipientByCanonicalKey(ctx, pool, in.OrgID, *key)
			if findErr == nil {
				return existing, nil
			}
			return nil, ErrRecipientDuplicateKey
		}
		return nil, err
	}
	return &r, nil
}

// ResolveOrCreateAdHoc links to an existing canonical key or inserts an ad-hoc recipient.
func ResolveOrCreateAdHoc(
	ctx context.Context,
	pool *pgxpool.Pool,
	orgID *uuid.UUID,
	in AdHocRecipientInput,
) (*Recipient, error) {
	typ := in.Type
	if typ == "" {
		typ = RecipientOther
	}
	if _, ok := ParseRecipientType(string(typ)); !ok {
		return nil, errors.New("invalid recipient type")
	}
	caps := in.Capabilities
	if len(caps) == 0 {
		caps = []string{string(DeliverySecureLink), string(DeliveryPostalMail), string(DeliveryElectronicPDF)}
	}
	return InsertRecipient(ctx, pool, UpsertRecipientInput{
		OrgID:        orgID,
		Type:         typ,
		Name:         in.Name,
		CanonicalKey: in.CanonicalKey,
		Capabilities: caps,
		Email:        in.Email,
		Address:      in.Address,
	})
}

// UpdateRecipient patches an existing recipient (admin).
func UpdateRecipient(
	ctx context.Context,
	pool *pgxpool.Pool,
	id uuid.UUID,
	in UpsertRecipientInput,
) (*Recipient, error) {
	existing, err := GetRecipient(ctx, pool, id)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = existing.Name
	}
	typ := in.Type
	if typ == "" {
		typ = existing.Type
	}
	caps := existing.Capabilities
	if in.Capabilities != nil {
		caps = NormalizeCapabilities(in.Capabilities)
	}
	key := existing.CanonicalKey
	if in.CanonicalKey != nil {
		k := NormalizeCanonicalKey(*in.CanonicalKey)
		if k == "" {
			key = nil
		} else {
			key = &k
		}
	}
	email := existing.Email
	if in.Email != nil {
		email = in.Email
	}
	addr := existing.Address
	if in.Address != nil {
		addr = in.Address
	}
	peer := existing.PeerConfig
	if in.PeerConfig != nil {
		peer = in.PeerConfig
	}
	verified := existing.Verified
	if in.Verified != nil {
		verified = *in.Verified
	}
	active := existing.Active
	if in.Active != nil {
		active = *in.Active
	}
	var r Recipient
	err = scanRecipient(pool.QueryRow(ctx, `
UPDATE transcripts.recipients
SET type = $2, name = $3, canonical_key = $4, capabilities = $5,
    email = $6, address = $7, peer_config = $8, verified = $9, active = $10
WHERE id = $1
RETURNING `+recipientSelectColumns+`
`, id, string(typ), name, key, caps, email, nullableJSON(addr), nullableJSON(peer), verified, active), &r)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrRecipientDuplicateKey
		}
		return nil, err
	}
	return &r, nil
}

// ListAdminRecipients returns org + global recipients for registrar management.
func ListAdminRecipients(
	ctx context.Context,
	pool *pgxpool.Pool,
	orgID uuid.UUID,
	includeInactive bool,
) ([]Recipient, error) {
	q := `
SELECT ` + recipientSelectColumns + `
FROM transcripts.recipients
WHERE org_id IS NULL OR org_id = $1`
	if !includeInactive {
		q += ` AND active = TRUE`
	}
	q += ` ORDER BY type ASC, name ASC LIMIT 500`
	rows, err := pool.Query(ctx, q, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Recipient
	for rows.Next() {
		var r Recipient
		if err := scanRecipient(rows, &r); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func nullableJSON(raw json.RawMessage) any {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	return raw
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	// pgx wraps pgconn errors; match common unique violation text/code.
	msg := err.Error()
	return strings.Contains(msg, "ux_recipients_canonical") ||
		strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "SQLSTATE 23505")
}
