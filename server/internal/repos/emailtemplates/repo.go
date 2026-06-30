// Package emailtemplates stores transactional email template slots and org overrides (plan 18.5).
package emailtemplates

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Slot is a system email template slot definition.
type Slot struct {
	ID          string
	Description string
	MergeFields map[string]string
	DefaultHTML string
	DefaultText string
}

// OrgVersion is one org template version row.
type OrgVersion struct {
	ID         uuid.UUID
	OrgID      uuid.UUID
	SlotID     string
	HTMLBody   string
	TextBody   *string
	ReplyTo    *string
	SenderName *string
	CreatedBy  *uuid.UUID
	CreatedAt  time.Time
	IsActive   bool
}

// SlotStatus summarizes customization state for list APIs.
type SlotStatus struct {
	Slot
	HasCustom   bool
	ActiveID    *uuid.UUID
	UpdatedAt   *time.Time
	ReplyTo     *string
	SenderName  *string
	UnknownWarn []string
}

// ListSlots returns all system slots.
func ListSlots(ctx context.Context, pool *pgxpool.Pool) ([]Slot, error) {
	if pool == nil {
		return nil, errors.New("db pool is nil")
	}
	rows, err := pool.Query(ctx, `
SELECT id, description, merge_fields, default_html, default_text
FROM settings.email_template_slots
ORDER BY id ASC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Slot
	for rows.Next() {
		var s Slot
		var mergeRaw []byte
		if err := rows.Scan(&s.ID, &s.Description, &mergeRaw, &s.DefaultHTML, &s.DefaultText); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(mergeRaw, &s.MergeFields)
		if s.MergeFields == nil {
			s.MergeFields = map[string]string{}
		}
		out = append(out, s)
	}
	if out == nil {
		out = []Slot{}
	}
	return out, rows.Err()
}

// GetSlot returns one slot by id.
func GetSlot(ctx context.Context, pool *pgxpool.Pool, slotID string) (*Slot, error) {
	if pool == nil {
		return nil, errors.New("db pool is nil")
	}
	var s Slot
	var mergeRaw []byte
	err := pool.QueryRow(ctx, `
SELECT id, description, merge_fields, default_html, default_text
FROM settings.email_template_slots
WHERE id = $1
`, slotID).Scan(&s.ID, &s.Description, &mergeRaw, &s.DefaultHTML, &s.DefaultText)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	_ = json.Unmarshal(mergeRaw, &s.MergeFields)
	if s.MergeFields == nil {
		s.MergeFields = map[string]string{}
	}
	return &s, nil
}

// GetActive returns the active org template for a slot, or nil if using system default.
func GetActive(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, slotID string) (*OrgVersion, error) {
	if pool == nil {
		return nil, errors.New("db pool is nil")
	}
	var v OrgVersion
	var textBody, replyTo, senderName *string
	var createdBy *uuid.UUID
	err := pool.QueryRow(ctx, `
SELECT id, org_id, slot_id, html_body, text_body, reply_to, sender_name, created_by, created_at, is_active
FROM settings.org_email_templates
WHERE org_id = $1 AND slot_id = $2 AND is_active = true
`, orgID, slotID).Scan(
		&v.ID, &v.OrgID, &v.SlotID, &v.HTMLBody, &textBody, &replyTo, &senderName, &createdBy, &v.CreatedAt, &v.IsActive,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	v.TextBody = textBody
	v.ReplyTo = replyTo
	v.SenderName = senderName
	v.CreatedBy = createdBy
	return &v, nil
}

// ListHistory returns all versions for an org slot, newest first.
func ListHistory(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, slotID string) ([]OrgVersion, error) {
	if pool == nil {
		return nil, errors.New("db pool is nil")
	}
	rows, err := pool.Query(ctx, `
SELECT id, org_id, slot_id, html_body, text_body, reply_to, sender_name, created_by, created_at, is_active
FROM settings.org_email_templates
WHERE org_id = $1 AND slot_id = $2
ORDER BY created_at DESC
`, orgID, slotID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OrgVersion
	for rows.Next() {
		var v OrgVersion
		var textBody, replyTo, senderName *string
		var createdBy *uuid.UUID
		if err := rows.Scan(
			&v.ID, &v.OrgID, &v.SlotID, &v.HTMLBody, &textBody, &replyTo, &senderName, &createdBy, &v.CreatedAt, &v.IsActive,
		); err != nil {
			return nil, err
		}
		v.TextBody = textBody
		v.ReplyTo = replyTo
		v.SenderName = senderName
		v.CreatedBy = createdBy
		out = append(out, v)
	}
	if out == nil {
		out = []OrgVersion{}
	}
	return out, rows.Err()
}

// SaveInput is input for creating a new active template version.
type SaveInput struct {
	OrgID      uuid.UUID
	SlotID     string
	HTMLBody   string
	TextBody   *string
	ReplyTo    *string
	SenderName *string
	CreatedBy  uuid.UUID
}

// Save deactivates the prior active version and inserts a new active row.
func Save(ctx context.Context, pool *pgxpool.Pool, in SaveInput) (*OrgVersion, error) {
	if pool == nil {
		return nil, errors.New("db pool is nil")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx, `
UPDATE settings.org_email_templates
SET is_active = false
WHERE org_id = $1 AND slot_id = $2 AND is_active = true
`, in.OrgID, in.SlotID)
	if err != nil {
		return nil, err
	}

	var v OrgVersion
	var textBody, replyTo, senderName *string
	var createdBy *uuid.UUID
	err = tx.QueryRow(ctx, `
INSERT INTO settings.org_email_templates (org_id, slot_id, html_body, text_body, reply_to, sender_name, created_by, is_active)
VALUES ($1, $2, $3, $4, $5, $6, $7, true)
RETURNING id, org_id, slot_id, html_body, text_body, reply_to, sender_name, created_by, created_at, is_active
`, in.OrgID, in.SlotID, in.HTMLBody, in.TextBody, in.ReplyTo, in.SenderName, in.CreatedBy).Scan(
		&v.ID, &v.OrgID, &v.SlotID, &v.HTMLBody, &textBody, &replyTo, &senderName, &createdBy, &v.CreatedAt, &v.IsActive,
	)
	if err != nil {
		return nil, err
	}
	v.TextBody = textBody
	v.ReplyTo = replyTo
	v.SenderName = senderName
	v.CreatedBy = createdBy
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &v, nil
}

// Restore activates a prior version by id.
func Restore(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, slotID string, versionID uuid.UUID) (*OrgVersion, error) {
	if pool == nil {
		return nil, errors.New("db pool is nil")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var src OrgVersion
	var textBody, replyTo, senderName *string
	var createdBy *uuid.UUID
	err = tx.QueryRow(ctx, `
SELECT id, org_id, slot_id, html_body, text_body, reply_to, sender_name, created_by, created_at, is_active
FROM settings.org_email_templates
WHERE id = $1 AND org_id = $2 AND slot_id = $3
`, versionID, orgID, slotID).Scan(
		&src.ID, &src.OrgID, &src.SlotID, &src.HTMLBody, &textBody, &replyTo, &senderName, &createdBy, &src.CreatedAt, &src.IsActive,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}
	src.TextBody = textBody
	src.ReplyTo = replyTo
	src.SenderName = senderName
	src.CreatedBy = createdBy

	_, err = tx.Exec(ctx, `
UPDATE settings.org_email_templates
SET is_active = false
WHERE org_id = $1 AND slot_id = $2 AND is_active = true
`, orgID, slotID)
	if err != nil {
		return nil, err
	}

	var v OrgVersion
	err = tx.QueryRow(ctx, `
INSERT INTO settings.org_email_templates (org_id, slot_id, html_body, text_body, reply_to, sender_name, created_by, is_active)
VALUES ($1, $2, $3, $4, $5, $6, $7, true)
RETURNING id, org_id, slot_id, html_body, text_body, reply_to, sender_name, created_by, created_at, is_active
`, orgID, slotID, src.HTMLBody, src.TextBody, src.ReplyTo, src.SenderName, createdBy).Scan(
		&v.ID, &v.OrgID, &v.SlotID, &v.HTMLBody, &textBody, &replyTo, &senderName, &createdBy, &v.CreatedAt, &v.IsActive,
	)
	if err != nil {
		return nil, err
	}
	v.TextBody = textBody
	v.ReplyTo = replyTo
	v.SenderName = senderName
	v.CreatedBy = createdBy
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &v, nil
}

// Reset deactivates all custom versions so the system default is used.
func Reset(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, slotID string) error {
	if pool == nil {
		return errors.New("db pool is nil")
	}
	_, err := pool.Exec(ctx, `
UPDATE settings.org_email_templates
SET is_active = false
WHERE org_id = $1 AND slot_id = $2 AND is_active = true
`, orgID, slotID)
	return err
}
