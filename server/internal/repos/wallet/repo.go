// Package wallet persists the learner credential wallet index and collections (T09).
package wallet

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Kind identifies a credential source type in the wallet.
type Kind string

const (
	KindTranscript  Kind = "transcript"
	KindCLR         Kind = "clr"
	KindBadge       Kind = "badge"
	KindCertificate Kind = "certificate"
	KindDiploma     Kind = "diploma"
	KindCERecord    Kind = "ce_record"
)

// Disclosure controls how much a shared collection reveals.
type Disclosure string

const (
	DisclosureValidity Disclosure = "validity"
	DisclosureSummary  Disclosure = "summary"
	DisclosureFull     Disclosure = "full"
)

// ExportStatus is the async export job state.
type ExportStatus string

const (
	ExportPending ExportStatus = "pending"
	ExportReady   ExportStatus = "ready"
	ExportFailed  ExportStatus = "failed"
)

// Item is one row in credentials.wallet_items.
type Item struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Kind        Kind
	SourceID    uuid.UUID
	Title       string
	Issuer      *string
	IssuedAt    *time.Time
	VerifyToken *string
	Revoked     bool
	Metadata    json.RawMessage
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Collection is one curated shareable set.
type Collection struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	Name       string
	ShareToken *string
	Disclosure Disclosure
	ExpiresAt  *time.Time
	RevokedAt  *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
	ItemIDs    []uuid.UUID
}

// AccessEvent is one public share lookup.
type AccessEvent struct {
	ID           uuid.UUID
	CollectionID uuid.UUID
	Result       string
	RequesterIP  *string
	RequesterUA  *string
	CreatedAt    time.Time
}

// Export is one async portable bundle request.
type Export struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	Status       ExportStatus
	ZipBytes     []byte
	Manifest     json.RawMessage
	ErrorMessage *string
	CreatedAt    time.Time
	CompletedAt  *time.Time
}

// SourceItem is a provider-normalized credential used to refresh the index.
type SourceItem struct {
	Kind        Kind
	SourceID    uuid.UUID
	Title       string
	Issuer      string
	IssuedAt    *time.Time
	VerifyToken string
	Revoked     bool
	Metadata    map[string]any
}

var (
	ErrCollectionNotFound = errors.New("wallet collection not found")
	ErrItemNotFound       = errors.New("wallet item not found")
	ErrExportNotFound     = errors.New("wallet export not found")
	ErrShareRevoked       = errors.New("wallet share revoked")
	ErrShareExpired       = errors.New("wallet share expired")
)

const itemCols = `
id, user_id, kind, source_id, title, issuer, issued_at, verify_token, revoked,
metadata, created_at, updated_at`

func scanItem(row pgx.Row, it *Item) error {
	var kind string
	var meta []byte
	err := row.Scan(
		&it.ID, &it.UserID, &kind, &it.SourceID, &it.Title, &it.Issuer, &it.IssuedAt,
		&it.VerifyToken, &it.Revoked, &meta, &it.CreatedAt, &it.UpdatedAt,
	)
	if err != nil {
		return err
	}
	it.Kind = Kind(kind)
	if len(meta) == 0 {
		it.Metadata = json.RawMessage(`{}`)
	} else {
		it.Metadata = meta
	}
	return nil
}

// ListItems returns wallet items for a user, newest first.
func ListItems(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]Item, error) {
	rows, err := pool.Query(ctx, `
SELECT `+itemCols+`
FROM credentials.wallet_items
WHERE user_id = $1
ORDER BY issued_at DESC NULLS LAST, created_at DESC
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Item
	for rows.Next() {
		var it Item
		if err := scanItem(rows, &it); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// GetItem returns one wallet item owned by userID.
func GetItem(ctx context.Context, pool *pgxpool.Pool, userID, itemID uuid.UUID) (*Item, error) {
	var it Item
	err := scanItem(pool.QueryRow(ctx, `
SELECT `+itemCols+`
FROM credentials.wallet_items
WHERE id = $1 AND user_id = $2
`, itemID, userID), &it)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &it, nil
}

// GetItemsByIDs returns items for the given ids owned by userID.
func GetItemsByIDs(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, ids []uuid.UUID) ([]Item, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := pool.Query(ctx, `
SELECT `+itemCols+`
FROM credentials.wallet_items
WHERE user_id = $1 AND id = ANY($2)
`, userID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Item
	for rows.Next() {
		var it Item
		if err := scanItem(rows, &it); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// UpsertSource upserts one source credential into the wallet index.
func UpsertSource(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, src SourceItem) (*Item, error) {
	meta, err := json.Marshal(src.Metadata)
	if err != nil {
		meta = []byte(`{}`)
	}
	var issuer *string
	if strings.TrimSpace(src.Issuer) != "" {
		s := strings.TrimSpace(src.Issuer)
		issuer = &s
	}
	var verify *string
	if strings.TrimSpace(src.VerifyToken) != "" {
		s := strings.TrimSpace(src.VerifyToken)
		verify = &s
	}
	var it Item
	err = scanItem(pool.QueryRow(ctx, `
INSERT INTO credentials.wallet_items (
    user_id, kind, source_id, title, issuer, issued_at, verify_token, revoked, metadata, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb, NOW())
ON CONFLICT (user_id, kind, source_id) DO UPDATE SET
    title = EXCLUDED.title,
    issuer = EXCLUDED.issuer,
    issued_at = EXCLUDED.issued_at,
    verify_token = EXCLUDED.verify_token,
    revoked = EXCLUDED.revoked,
    metadata = EXCLUDED.metadata,
    updated_at = NOW()
RETURNING `+itemCols+`
`, userID, string(src.Kind), src.SourceID, src.Title, issuer, src.IssuedAt, verify, src.Revoked, meta), &it)
	if err != nil {
		return nil, err
	}
	return &it, nil
}

// DeleteMissing removes wallet items for the user whose (kind, source_id) are not in keep.
func DeleteMissing(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, keep []SourceItem) error {
	if len(keep) == 0 {
		_, err := pool.Exec(ctx, `DELETE FROM credentials.wallet_items WHERE user_id = $1`, userID)
		return err
	}
	kinds := make([]string, len(keep))
	sources := make([]uuid.UUID, len(keep))
	for i, s := range keep {
		kinds[i] = string(s.Kind)
		sources[i] = s.SourceID
	}
	_, err := pool.Exec(ctx, `
DELETE FROM credentials.wallet_items wi
WHERE wi.user_id = $1
  AND NOT EXISTS (
    SELECT 1
    FROM unnest($2::text[], $3::uuid[]) AS k(kind, source_id)
    WHERE k.kind = wi.kind AND k.source_id = wi.source_id
  )
`, userID, kinds, sources)
	return err
}

func newShareToken() (string, error) {
	var b [24]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func scanCollection(row pgx.Row, c *Collection) error {
	var disclosure string
	err := row.Scan(
		&c.ID, &c.UserID, &c.Name, &c.ShareToken, &disclosure,
		&c.ExpiresAt, &c.RevokedAt, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return err
	}
	c.Disclosure = Disclosure(disclosure)
	return nil
}

const collectionCols = `
id, user_id, name, share_token, disclosure, expires_at, revoked_at, created_at, updated_at`

func loadCollectionItemIDs(ctx context.Context, pool *pgxpool.Pool, collectionID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := pool.Query(ctx, `
SELECT wallet_item_id
FROM credentials.collection_items
WHERE collection_id = $1
ORDER BY position ASC, wallet_item_id ASC
`, collectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ListCollections returns collections for a user.
func ListCollections(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]Collection, error) {
	rows, err := pool.Query(ctx, `
SELECT `+collectionCols+`
FROM credentials.collections
WHERE user_id = $1
ORDER BY created_at DESC
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Collection
	for rows.Next() {
		var c Collection
		if err := scanCollection(rows, &c); err != nil {
			return nil, err
		}
		ids, err := loadCollectionItemIDs(ctx, pool, c.ID)
		if err != nil {
			return nil, err
		}
		c.ItemIDs = ids
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetCollection returns a collection owned by userID.
func GetCollection(ctx context.Context, pool *pgxpool.Pool, userID, collectionID uuid.UUID) (*Collection, error) {
	var c Collection
	err := scanCollection(pool.QueryRow(ctx, `
SELECT `+collectionCols+`
FROM credentials.collections
WHERE id = $1 AND user_id = $2
`, collectionID, userID), &c)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	ids, err := loadCollectionItemIDs(ctx, pool, c.ID)
	if err != nil {
		return nil, err
	}
	c.ItemIDs = ids
	return &c, nil
}

// CreateCollectionInput creates a curated collection and optional share token.
type CreateCollectionInput struct {
	UserID     uuid.UUID
	Name       string
	Disclosure Disclosure
	ItemIDs    []uuid.UUID
	ExpiresAt  *time.Time
	Share      bool
}

// CreateCollection inserts a collection and its items.
func CreateCollection(ctx context.Context, pool *pgxpool.Pool, in CreateCollectionInput) (*Collection, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = "Shared credentials"
	}
	disclosure := in.Disclosure
	if disclosure == "" {
		disclosure = DisclosureValidity
	}
	var shareToken *string
	if in.Share {
		tok, err := newShareToken()
		if err != nil {
			return nil, err
		}
		shareToken = &tok
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var c Collection
	err = scanCollection(tx.QueryRow(ctx, `
INSERT INTO credentials.collections (user_id, name, share_token, disclosure, expires_at)
VALUES ($1,$2,$3,$4,$5)
RETURNING `+collectionCols+`
`, in.UserID, name, shareToken, string(disclosure), in.ExpiresAt), &c)
	if err != nil {
		return nil, err
	}
	for i, itemID := range in.ItemIDs {
		tag, err := tx.Exec(ctx, `
INSERT INTO credentials.collection_items (collection_id, wallet_item_id, position)
SELECT $1, $2, $3
WHERE EXISTS (
  SELECT 1 FROM credentials.wallet_items WHERE id = $2 AND user_id = $4
)
`, c.ID, itemID, i, in.UserID)
		if err != nil {
			return nil, err
		}
		if tag.RowsAffected() == 0 {
			return nil, ErrItemNotFound
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	c.ItemIDs = append([]uuid.UUID(nil), in.ItemIDs...)
	return &c, nil
}

// UpdateCollectionInput patches name, disclosure, items, and optionally regenerates share.
type UpdateCollectionInput struct {
	UserID       uuid.UUID
	CollectionID uuid.UUID
	Name         *string
	Disclosure   *Disclosure
	ItemIDs      *[]uuid.UUID
	ExpiresAt    *time.Time
	ClearExpiry  bool
	EnableShare  *bool
}

// UpdateCollection updates a collection owned by the user.
func UpdateCollection(ctx context.Context, pool *pgxpool.Pool, in UpdateCollectionInput) (*Collection, error) {
	c, err := GetCollection(ctx, pool, in.UserID, in.CollectionID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, nil
	}
	name := c.Name
	if in.Name != nil {
		name = strings.TrimSpace(*in.Name)
		if name == "" {
			name = c.Name
		}
	}
	disclosure := c.Disclosure
	if in.Disclosure != nil {
		disclosure = *in.Disclosure
	}
	expiresAt := c.ExpiresAt
	if in.ClearExpiry {
		expiresAt = nil
	} else if in.ExpiresAt != nil {
		expiresAt = in.ExpiresAt
	}
	shareToken := c.ShareToken
	if in.EnableShare != nil {
		if *in.EnableShare {
			if shareToken == nil || c.RevokedAt != nil {
				tok, err := newShareToken()
				if err != nil {
					return nil, err
				}
				shareToken = &tok
			}
		} else {
			shareToken = nil
		}
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var out Collection
	err = scanCollection(tx.QueryRow(ctx, `
UPDATE credentials.collections
SET name = $3,
    disclosure = $4,
    expires_at = $5,
    share_token = $6,
    revoked_at = CASE WHEN $7::bool THEN NULL ELSE revoked_at END,
    updated_at = NOW()
WHERE id = $1 AND user_id = $2
RETURNING `+collectionCols+`
`, in.CollectionID, in.UserID, name, string(disclosure), expiresAt, shareToken, in.EnableShare != nil && *in.EnableShare), &out)
	if err != nil {
		return nil, err
	}

	itemIDs := c.ItemIDs
	if in.ItemIDs != nil {
		itemIDs = *in.ItemIDs
		if _, err := tx.Exec(ctx, `DELETE FROM credentials.collection_items WHERE collection_id = $1`, in.CollectionID); err != nil {
			return nil, err
		}
		for i, itemID := range itemIDs {
			tag, err := tx.Exec(ctx, `
INSERT INTO credentials.collection_items (collection_id, wallet_item_id, position)
SELECT $1, $2, $3
WHERE EXISTS (
  SELECT 1 FROM credentials.wallet_items WHERE id = $2 AND user_id = $4
)
`, in.CollectionID, itemID, i, in.UserID)
			if err != nil {
				return nil, err
			}
			if tag.RowsAffected() == 0 {
				return nil, ErrItemNotFound
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	out.ItemIDs = itemIDs
	return &out, nil
}

// DeleteCollection removes a collection.
func DeleteCollection(ctx context.Context, pool *pgxpool.Pool, userID, collectionID uuid.UUID) error {
	tag, err := pool.Exec(ctx, `
DELETE FROM credentials.collections WHERE id = $1 AND user_id = $2
`, collectionID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrCollectionNotFound
	}
	return nil
}

// RevokeCollectionShare marks the share link revoked (keeps access history).
func RevokeCollectionShare(ctx context.Context, pool *pgxpool.Pool, userID, collectionID uuid.UUID) (*Collection, error) {
	var c Collection
	err := scanCollection(pool.QueryRow(ctx, `
UPDATE credentials.collections
SET revoked_at = NOW(), updated_at = NOW()
WHERE id = $1 AND user_id = $2 AND share_token IS NOT NULL
RETURNING `+collectionCols+`
`, collectionID, userID), &c)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	ids, err := loadCollectionItemIDs(ctx, pool, c.ID)
	if err != nil {
		return nil, err
	}
	c.ItemIDs = ids
	return &c, nil
}

// GetCollectionByShareToken loads a collection by public share token (any owner).
func GetCollectionByShareToken(ctx context.Context, pool *pgxpool.Pool, token string) (*Collection, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, nil
	}
	var c Collection
	err := scanCollection(pool.QueryRow(ctx, `
SELECT `+collectionCols+`
FROM credentials.collections
WHERE share_token = $1
`, token), &c)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	ids, err := loadCollectionItemIDs(ctx, pool, c.ID)
	if err != nil {
		return nil, err
	}
	c.ItemIDs = ids
	return &c, nil
}

// RecordCollectionAccess appends an access history row.
func RecordCollectionAccess(ctx context.Context, pool *pgxpool.Pool, collectionID uuid.UUID, result, ip, ua string) error {
	var uaPtr *string
	if strings.TrimSpace(ua) != "" {
		s := strings.TrimSpace(ua)
		if len(s) > 512 {
			s = s[:512]
		}
		uaPtr = &s
	}
	ip = strings.TrimSpace(ip)
	if ip == "" {
		_, err := pool.Exec(ctx, `
INSERT INTO credentials.collection_access (collection_id, result, requester_ua)
VALUES ($1, $2, $3)
`, collectionID, result, uaPtr)
		return err
	}
	_, err := pool.Exec(ctx, `
INSERT INTO credentials.collection_access (collection_id, result, requester_ip, requester_ua)
VALUES ($1, $2, $3::inet, $4)
`, collectionID, result, ip, uaPtr)
	if err != nil {
		// Fall back without IP when the address is not a valid inet literal.
		_, err2 := pool.Exec(ctx, `
INSERT INTO credentials.collection_access (collection_id, result, requester_ua)
VALUES ($1, $2, $3)
`, collectionID, result, uaPtr)
		return err2
	}
	return nil
}

// ListCollectionAccess returns recent access events for a collection owned by userID.
func ListCollectionAccess(ctx context.Context, pool *pgxpool.Pool, userID, collectionID uuid.UUID, limit int) ([]AccessEvent, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := pool.Query(ctx, `
SELECT a.id, a.collection_id, a.result, host(a.requester_ip)::text, a.requester_ua, a.created_at
FROM credentials.collection_access a
JOIN credentials.collections c ON c.id = a.collection_id
WHERE a.collection_id = $1 AND c.user_id = $2
ORDER BY a.created_at DESC
LIMIT $3
`, collectionID, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AccessEvent
	for rows.Next() {
		var e AccessEvent
		if err := rows.Scan(&e.ID, &e.CollectionID, &e.Result, &e.RequesterIP, &e.RequesterUA, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// CreateExport starts a pending export row.
func CreateExport(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (*Export, error) {
	var e Export
	var status string
	err := pool.QueryRow(ctx, `
INSERT INTO credentials.wallet_exports (user_id, status)
VALUES ($1, 'pending')
RETURNING id, user_id, status, zip_bytes, manifest, error_message, created_at, completed_at
`, userID).Scan(
		&e.ID, &e.UserID, &status, &e.ZipBytes, &e.Manifest, &e.ErrorMessage, &e.CreatedAt, &e.CompletedAt,
	)
	if err != nil {
		return nil, err
	}
	e.Status = ExportStatus(status)
	return &e, nil
}

// GetExport returns an export owned by userID.
func GetExport(ctx context.Context, pool *pgxpool.Pool, userID, exportID uuid.UUID) (*Export, error) {
	var e Export
	var status string
	err := pool.QueryRow(ctx, `
SELECT id, user_id, status, zip_bytes, manifest, error_message, created_at, completed_at
FROM credentials.wallet_exports
WHERE id = $1 AND user_id = $2
`, exportID, userID).Scan(
		&e.ID, &e.UserID, &status, &e.ZipBytes, &e.Manifest, &e.ErrorMessage, &e.CreatedAt, &e.CompletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	e.Status = ExportStatus(status)
	return &e, nil
}

// CompleteExport stores the ZIP and marks the export ready.
func CompleteExport(ctx context.Context, pool *pgxpool.Pool, exportID uuid.UUID, zipBytes []byte, manifest json.RawMessage) error {
	_, err := pool.Exec(ctx, `
UPDATE credentials.wallet_exports
SET status = 'ready', zip_bytes = $2, manifest = $3::jsonb, completed_at = NOW(), error_message = NULL
WHERE id = $1
`, exportID, zipBytes, manifest)
	return err
}

// FailExport marks an export failed.
func FailExport(ctx context.Context, pool *pgxpool.Pool, exportID uuid.UUID, message string) error {
	_, err := pool.Exec(ctx, `
UPDATE credentials.wallet_exports
SET status = 'failed', error_message = $2, completed_at = NOW()
WHERE id = $1
`, exportID, strings.TrimSpace(message))
	return err
}

// GetExportByID loads an export without ownership check (worker).
func GetExportByID(ctx context.Context, pool *pgxpool.Pool, exportID uuid.UUID) (*Export, error) {
	var e Export
	var status string
	err := pool.QueryRow(ctx, `
SELECT id, user_id, status, zip_bytes, manifest, error_message, created_at, completed_at
FROM credentials.wallet_exports
WHERE id = $1
`, exportID).Scan(
		&e.ID, &e.UserID, &status, &e.ZipBytes, &e.Manifest, &e.ErrorMessage, &e.CreatedAt, &e.CompletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	e.Status = ExportStatus(status)
	return &e, nil
}
