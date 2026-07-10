// Package productfeedback persists in-app product feedback submissions (plan FB0).
package productfeedback

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	pfmodel "github.com/lextures/lextures/server/internal/models/productfeedback"
)

// Submission is a feedback.submissions row.
type Submission struct {
	ID             uuid.UUID
	UserID         *uuid.UUID
	OrgID          *uuid.UUID
	Message        string
	Category       pfmodel.Category
	Source         pfmodel.Source
	AppVersion     *string
	Context        pfmodel.Context
	Status         pfmodel.Status
	AdminNote      *string
	ResolvedBy     *uuid.UUID
	ResolvedAt     *time.Time
	IdempotencyKey *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// SubmitterInfo is display context for the submitting user.
type SubmitterInfo struct {
	Name  string
	Email string
}

// ResolverInfo is display context for the admin who resolved feedback.
type ResolverInfo struct {
	Name  string
	Email string
}

// InsertInput is the data required to create a submission.
type InsertInput struct {
	UserID         uuid.UUID
	OrgID          *uuid.UUID
	Message        string
	Category       pfmodel.Category
	Source         pfmodel.Source
	AppVersion     *string
	Context        pfmodel.Context
	IdempotencyKey *string
}

// ListFilter holds admin list query parameters.
type ListFilter struct {
	Status   string
	Category string
	Source   string
	Query    string
	From     *time.Time
	To       *time.Time
	Limit    int
	Cursor   string
}

// ListItem is one admin list row with submitter display info.
type ListItem struct {
	ID             uuid.UUID
	MessagePreview string
	Category       pfmodel.Category
	Source         pfmodel.Source
	Status         pfmodel.Status
	Submitter      SubmitterInfo
	CreatedAt      time.Time
}

// Insert creates a new feedback submission. When idempotency_key collides for the
// same user, returns the existing row instead of creating a duplicate.
func Insert(ctx context.Context, pool *pgxpool.Pool, in InsertInput) (*Submission, error) {
	ctxJSON, err := json.Marshal(in.Context)
	if err != nil {
		return nil, err
	}
	if in.IdempotencyKey != nil && strings.TrimSpace(*in.IdempotencyKey) != "" {
		key := strings.TrimSpace(*in.IdempotencyKey)
		existing, err := GetByUserIdempotencyKey(ctx, pool, in.UserID, key)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return existing, nil
		}
		in.IdempotencyKey = &key
	} else {
		in.IdempotencyKey = nil
	}

	row := pool.QueryRow(ctx, `
INSERT INTO feedback.submissions (
    user_id, org_id, message, category, source, app_version, context, idempotency_key
) VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8)
RETURNING
    id, user_id, org_id, message, category, source, app_version, context,
    status, admin_note, resolved_by, resolved_at, idempotency_key, created_at, updated_at
`, in.UserID, in.OrgID, in.Message, string(in.Category), string(in.Source), in.AppVersion, ctxJSON, in.IdempotencyKey)

	sub, err := scanSubmission(row)
	if err != nil {
		if in.IdempotencyKey != nil && isUniqueViolation(err) {
			return GetByUserIdempotencyKey(ctx, pool, in.UserID, *in.IdempotencyKey)
		}
		return nil, err
	}
	return sub, nil
}

// GetByID loads one submission by primary key.
func GetByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Submission, error) {
	row := pool.QueryRow(ctx, `
SELECT
    id, user_id, org_id, message, category, source, app_version, context,
    status, admin_note, resolved_by, resolved_at, idempotency_key, created_at, updated_at
FROM feedback.submissions
WHERE id = $1
`, id)
	sub, err := scanSubmission(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return sub, err
}

// GetByUserIdempotencyKey returns an existing row for deduplication.
func GetByUserIdempotencyKey(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, key string) (*Submission, error) {
	row := pool.QueryRow(ctx, `
SELECT
    id, user_id, org_id, message, category, source, app_version, context,
    status, admin_note, resolved_by, resolved_at, idempotency_key, created_at, updated_at
FROM feedback.submissions
WHERE user_id = $1 AND idempotency_key = $2
`, userID, key)
	sub, err := scanSubmission(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return sub, err
}

// List returns paginated admin rows matching optional filters, newest first.
func List(ctx context.Context, pool *pgxpool.Pool, f ListFilter) ([]ListItem, int, string, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}
	offset, err := DecodeCursor(f.Cursor)
	if err != nil {
		return nil, 0, "", err
	}

	where, args := buildListWhere(f)
	countSQL := `SELECT count(*)::int FROM feedback.submissions s ` + where
	var total int
	if err := pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, "", err
	}

	listArgs := append(append([]any{}, args...), limit, offset)
	rows, err := pool.Query(ctx, `
SELECT
    s.id,
    s.message,
    s.category,
    s.source,
    s.status,
    s.created_at,
    COALESCE(NULLIF(TRIM(u.display_name), ''), u.email, '') AS submitter_name,
    COALESCE(u.email, '') AS submitter_email
FROM feedback.submissions s
LEFT JOIN "user".users u ON u.id = s.user_id
`+where+`
ORDER BY s.created_at DESC, s.id DESC
LIMIT $`+strconv.Itoa(len(args)+1)+` OFFSET $`+strconv.Itoa(len(args)+2), listArgs...)
	if err != nil {
		return nil, 0, "", err
	}
	defer rows.Close()

	out := make([]ListItem, 0, limit)
	for rows.Next() {
		var item ListItem
		var cat, src, st string
		var message string
		if err := rows.Scan(
			&item.ID, &message, &cat, &src, &st, &item.CreatedAt,
			&item.Submitter.Name, &item.Submitter.Email,
		); err != nil {
			return nil, 0, "", err
		}
		item.MessagePreview = pfmodel.MessagePreview(message)
		item.Category = pfmodel.Category(cat)
		item.Source = pfmodel.Source(src)
		item.Status = pfmodel.Status(st)
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, "", err
	}

	nextCursor := ""
	if offset+len(out) < total {
		nextCursor = EncodeCursor(offset + limit)
	}
	return out, total, nextCursor, nil
}

// UpdateAdmin patches status and/or admin_note. Sets resolved_by/resolved_at on terminal status.
func UpdateAdmin(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, adminID uuid.UUID, status *pfmodel.Status, adminNote *string) (*Submission, error) {
	current, err := GetByID(ctx, pool, id)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, nil
	}

	newStatus := current.Status
	if status != nil {
		newStatus = *status
	}
	note := current.AdminNote
	if adminNote != nil {
		trimmed := strings.TrimSpace(*adminNote)
		if trimmed == "" {
			note = nil
		} else {
			note = &trimmed
		}
	}

	var resolvedBy *uuid.UUID
	var resolvedAt *time.Time
	if newStatus.IsTerminal() {
		resolvedBy = &adminID
		now := time.Now().UTC()
		resolvedAt = &now
	} else {
		resolvedBy = nil
		resolvedAt = nil
	}

	row := pool.QueryRow(ctx, `
UPDATE feedback.submissions
SET status = $2,
    admin_note = $3,
    resolved_by = $4,
    resolved_at = $5,
    updated_at = now()
WHERE id = $1
RETURNING
    id, user_id, org_id, message, category, source, app_version, context,
    status, admin_note, resolved_by, resolved_at, idempotency_key, created_at, updated_at
`, id, string(newStatus), note, resolvedBy, resolvedAt)
	return scanSubmission(row)
}

// DeleteByUser removes all feedback rows for a user (DSAR erasure).
func DeleteByUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) error {
	_, err := pool.Exec(ctx, `DELETE FROM feedback.submissions WHERE user_id = $1`, userID)
	return err
}

// ListForUserExport returns submissions for DSAR export.
func ListForUserExport(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]Submission, error) {
	rows, err := pool.Query(ctx, `
SELECT
    id, user_id, org_id, message, category, source, app_version, context,
    status, admin_note, resolved_by, resolved_at, idempotency_key, created_at, updated_at
FROM feedback.submissions
WHERE user_id = $1
ORDER BY created_at ASC
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Submission
	for rows.Next() {
		sub, err := scanSubmission(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *sub)
	}
	return out, rows.Err()
}

// CountRecentByUser returns submissions in the last window (rate-limit check).
func CountRecentByUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, since time.Time) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
SELECT count(*)::int
FROM feedback.submissions
WHERE user_id = $1 AND created_at >= $2
`, userID, since).Scan(&n)
	return n, err
}

// LookupSubmitter returns name/email for a user id.
func LookupSubmitter(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (SubmitterInfo, error) {
	var info SubmitterInfo
	err := pool.QueryRow(ctx, `
SELECT
    COALESCE(NULLIF(TRIM(display_name), ''), email, ''),
    COALESCE(email, '')
FROM "user".users
WHERE id = $1
`, userID).Scan(&info.Name, &info.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		return SubmitterInfo{}, nil
	}
	return info, err
}

// LookupResolver returns name/email for a resolver user id.
func LookupResolver(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (*ResolverInfo, error) {
	info, err := LookupSubmitter(ctx, pool, userID)
	if err != nil {
		return nil, err
	}
	if info.Email == "" && info.Name == "" {
		return nil, nil
	}
	return &ResolverInfo{Name: info.Name, Email: info.Email}, nil
}

func buildListWhere(f ListFilter) (string, []any) {
	parts := []string{"WHERE 1=1"}
	args := make([]any, 0, 8)
	arg := func(v any) string {
		args = append(args, v)
		return "$" + strconv.Itoa(len(args))
	}
	if s := strings.TrimSpace(f.Status); s != "" {
		parts = append(parts, "AND s.status = "+arg(s))
	}
	if c := strings.TrimSpace(f.Category); c != "" {
		parts = append(parts, "AND s.category = "+arg(c))
	}
	if src := strings.TrimSpace(f.Source); src != "" {
		parts = append(parts, "AND s.source = "+arg(src))
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		parts = append(parts, "AND s.message ILIKE "+arg("%"+q+"%"))
	}
	if f.From != nil {
		parts = append(parts, "AND s.created_at >= "+arg(*f.From))
	}
	if f.To != nil {
		parts = append(parts, "AND s.created_at <= "+arg(*f.To))
	}
	return strings.Join(parts, " "), args
}

// EncodeCursor returns an opaque list cursor for the given offset.
func EncodeCursor(offset int) string {
	if offset <= 0 {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString([]byte("o:" + strconv.Itoa(offset)))
}

// DecodeCursor parses an opaque list cursor into a row offset.
func DecodeCursor(cursor string) (int, error) {
	cursor = strings.TrimSpace(cursor)
	if cursor == "" {
		return 0, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return 0, fmt.Errorf("invalid cursor")
	}
	parts := strings.SplitN(string(raw), ":", 2)
	if len(parts) != 2 || parts[0] != "o" {
		return 0, fmt.Errorf("invalid cursor")
	}
	n, err := strconv.Atoi(parts[1])
	if err != nil || n < 0 {
		return 0, fmt.Errorf("invalid cursor")
	}
	return n, nil
}

func scanSubmission(row pgx.Row) (*Submission, error) {
	var sub Submission
	var cat, src, st string
	var ctxRaw []byte
	var userID, orgID, resolvedBy *uuid.UUID
	err := row.Scan(
		&sub.ID, &userID, &orgID, &sub.Message, &cat, &src, &sub.AppVersion, &ctxRaw,
		&st, &sub.AdminNote, &resolvedBy, &sub.ResolvedAt, &sub.IdempotencyKey,
		&sub.CreatedAt, &sub.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	sub.UserID = userID
	sub.OrgID = orgID
	sub.Category = pfmodel.Category(cat)
	sub.Source = pfmodel.Source(src)
	sub.Status = pfmodel.Status(st)
	sub.ResolvedBy = resolvedBy
	if len(ctxRaw) > 0 {
		_ = json.Unmarshal(ctxRaw, &sub.Context)
	}
	return &sub, nil
}

func isUniqueViolation(err error) bool {
	var pe *pgconn.PgError
	return errors.As(err, &pe) && pe.Code == "23505"
}
