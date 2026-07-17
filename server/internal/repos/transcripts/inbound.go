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

// Inbound channel / format / status constants (T07).
const (
	InboundChannelAPIPeer = "api_peer"
	InboundChannelSFTP    = "sftp"
	InboundChannelEmail   = "email"

	InboundFormatPESC = "pesc_xml"
	InboundFormatPDF  = "pdf"
	InboundFormatEDI  = "edi"
	InboundFormatOther = "other"

	InboundReceived     = "received"
	InboundQuarantined  = "quarantined"
	InboundParsed       = "parsed"
	InboundMatched      = "matched"
	InboundAccepted     = "accepted"
	InboundRejected     = "rejected"
	InboundUnmatched    = "unmatched"
)

var (
	ErrInboundNotFound      = errors.New("inbound document not found")
	ErrInboundDuplicate     = errors.New("inbound document already received")
	ErrInboundInvalidStatus = errors.New("inbound document status does not allow this action")
)

// InboundDocument is one received transcript artifact.
type InboundDocument struct {
	ID                  uuid.UUID
	OrgID               uuid.UUID
	Channel             string
	SourceName          *string
	ExternalRef         *string
	Format              string
	RawKey              string
	RawBytes            []byte
	ContentHash         string
	ContentType         *string
	ByteSize            int
	Parsed              json.RawMessage
	StudentName         *string
	StudentDOB          *string
	StudentRef          *string
	MatchedUserID       *uuid.UUID
	MatchConfidence     *float64
	MatchDetail         json.RawMessage
	Status              string
	NeedsManualReview   bool
	ReviewerID          *uuid.UUID
	RejectReason        *string
	QuarantineReason    *string
	ReceivedAt          time.Time
	ProcessedAt         *time.Time
	NotifiedReceivedAt  *time.Time
	NotifiedAcceptedAt  *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// InboundEvent is an audit row for inbound lifecycle actions.
type InboundEvent struct {
	ID        uuid.UUID
	InboundID uuid.UUID
	EventType string
	ActorID   *uuid.UUID
	Detail    json.RawMessage
	CreatedAt time.Time
}

// InsertInboundInput creates a received row (pre-processing).
type InsertInboundInput struct {
	OrgID       uuid.UUID
	Channel     string
	SourceName  *string
	ExternalRef *string
	Format      string
	RawKey      string
	RawBytes    []byte
	ContentHash string
	ContentType *string
	StudentName *string
	StudentDOB  *string
	StudentRef  *string
}

// InboundListFilter filters the intake queue.
type InboundListFilter struct {
	OrgID  *uuid.UUID
	Status string
	Query  string
	Limit  int
}

// InsertInboundDocument stores an immutable original and returns the row.
// Duplicate (org, source, external_ref) returns ErrInboundDuplicate.
func InsertInboundDocument(ctx context.Context, pool *pgxpool.Pool, in InsertInboundInput) (*InboundDocument, error) {
	if pool == nil {
		return nil, errors.New("transcripts: nil pool")
	}
	channel := strings.TrimSpace(in.Channel)
	format := strings.TrimSpace(in.Format)
	if channel == "" || format == "" || len(in.RawBytes) == 0 || strings.TrimSpace(in.RawKey) == "" {
		return nil, errors.New("transcripts: incomplete inbound insert")
	}
	doc, err := scanInbound(ctx, pool, `
INSERT INTO transcripts.inbound_documents (
    org_id, channel, source_name, external_ref, format, raw_key, raw_bytes, content_hash,
    content_type, byte_size, student_name, student_dob, student_ref, status
) VALUES (
    $1, $2, NULLIF(BTRIM($3), ''), NULLIF(BTRIM($4), ''), $5, $6, $7, $8,
    $9, $10, NULLIF(BTRIM($11), ''), NULLIF(BTRIM($12), ''), NULLIF(BTRIM($13), ''), 'received'
)
RETURNING `+inboundColumns, inboundArgs(
		in.OrgID, channel, nullableStr(in.SourceName), nullableStr(in.ExternalRef), format,
		in.RawKey, in.RawBytes, in.ContentHash, in.ContentType, len(in.RawBytes),
		nullableStr(in.StudentName), nullableStr(in.StudentDOB), nullableStr(in.StudentRef),
	)...)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrInboundDuplicate
		}
		return nil, err
	}
	_ = AppendInboundEvent(ctx, pool, doc.ID, "received", nil, map[string]any{
		"channel": channel,
		"format":  format,
	})
	return doc, nil
}

// GetInboundDocument returns one inbound row by id.
func GetInboundDocument(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*InboundDocument, error) {
	doc, err := scanInbound(ctx, pool, `
SELECT `+inboundColumns+`
FROM transcripts.inbound_documents
WHERE id = $1
`, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInboundNotFound
	}
	return doc, err
}

// FindInboundByDedupe returns an existing row for (org, source, external_ref).
func FindInboundByDedupe(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, sourceName, externalRef string) (*InboundDocument, error) {
	sourceName = strings.TrimSpace(sourceName)
	externalRef = strings.TrimSpace(externalRef)
	if sourceName == "" || externalRef == "" {
		return nil, ErrInboundNotFound
	}
	doc, err := scanInbound(ctx, pool, `
SELECT `+inboundColumns+`
FROM transcripts.inbound_documents
WHERE org_id = $1 AND source_name = $2 AND external_ref = $3
LIMIT 1
`, orgID, sourceName, externalRef)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInboundNotFound
	}
	return doc, err
}

// ListInboundDocuments returns the registrar intake queue.
func ListInboundDocuments(ctx context.Context, pool *pgxpool.Pool, f InboundListFilter) ([]InboundDocument, error) {
	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	args := []any{}
	where := []string{"TRUE"}
	if f.OrgID != nil {
		args = append(args, *f.OrgID)
		where = append(where, fmt.Sprintf("org_id = $%d", len(args)))
	}
	if status := strings.TrimSpace(f.Status); status != "" {
		args = append(args, status)
		where = append(where, fmt.Sprintf("status = $%d", len(args)))
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		args = append(args, "%"+q+"%")
		n := len(args)
		where = append(where, fmt.Sprintf(
			"(COALESCE(source_name,'') ILIKE $%d OR COALESCE(student_name,'') ILIKE $%d OR COALESCE(external_ref,'') ILIKE $%d OR COALESCE(student_ref,'') ILIKE $%d)",
			n, n, n, n,
		))
	}
	args = append(args, limit)
	q := `
SELECT ` + inboundColumns + `
FROM transcripts.inbound_documents
WHERE ` + strings.Join(where, " AND ") + `
ORDER BY received_at DESC
LIMIT $` + fmt.Sprint(len(args))
	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]InboundDocument, 0)
	for rows.Next() {
		doc, err := scanInboundRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *doc)
	}
	return out, rows.Err()
}

// UpdateInboundAfterProcess stores parse/match results.
func UpdateInboundAfterProcess(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, status string, parsed json.RawMessage, matchedUser *uuid.UUID, confidence *float64, matchDetail json.RawMessage, studentName, studentDOB, studentRef *string, needsManual bool, quarantineReason *string) (*InboundDocument, error) {
	doc, err := scanInbound(ctx, pool, `
UPDATE transcripts.inbound_documents SET
    status = $2,
    parsed = $3,
    matched_user_id = $4,
    match_confidence = $5,
    match_detail = $6,
    student_name = COALESCE(NULLIF(BTRIM($7), ''), student_name),
    student_dob = COALESCE(NULLIF(BTRIM($8), ''), student_dob),
    student_ref = COALESCE(NULLIF(BTRIM($9), ''), student_ref),
    needs_manual_review = $10,
    quarantine_reason = $11,
    processed_at = NOW(),
    updated_at = NOW()
WHERE id = $1
RETURNING `+inboundColumns, id, status, nullJSON(parsed), matchedUser, confidence, nullJSON(matchDetail),
		nullableStr(studentName), nullableStr(studentDOB), nullableStr(studentRef), needsManual, quarantineReason)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInboundNotFound
	}
	return doc, err
}

// MatchInboundDocument assigns (or reassigns) an inbound document to a user.
func MatchInboundDocument(ctx context.Context, pool *pgxpool.Pool, id, userID uuid.UUID, actorID *uuid.UUID, confidence float64, detail map[string]any) (*InboundDocument, error) {
	raw, _ := json.Marshal(detail)
	doc, err := scanInbound(ctx, pool, `
UPDATE transcripts.inbound_documents SET
    matched_user_id = $2,
    match_confidence = $3,
    match_detail = $4,
    status = CASE WHEN status IN ('accepted', 'rejected') THEN status ELSE 'matched' END,
    needs_manual_review = FALSE,
    reviewer_id = COALESCE($5, reviewer_id),
    updated_at = NOW()
WHERE id = $1
RETURNING `+inboundColumns, id, userID, confidence, raw, actorID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInboundNotFound
	}
	if err != nil {
		return nil, err
	}
	_ = AppendInboundEvent(ctx, pool, id, "matched", actorID, map[string]any{
		"userId":     userID.String(),
		"confidence": confidence,
		"detail":     detail,
	})
	return doc, nil
}

// ClearInboundMatch reverses an automatic/manual match (back to unmatched).
func ClearInboundMatch(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, actorID *uuid.UUID, reason string) (*InboundDocument, error) {
	doc, err := GetInboundDocument(ctx, pool, id)
	if err != nil {
		return nil, err
	}
	if doc.Status == InboundAccepted || doc.Status == InboundRejected {
		return nil, ErrInboundInvalidStatus
	}
	updated, err := scanInbound(ctx, pool, `
UPDATE transcripts.inbound_documents SET
    matched_user_id = NULL,
    match_confidence = NULL,
    status = 'unmatched',
    needs_manual_review = TRUE,
    reviewer_id = COALESCE($2, reviewer_id),
    updated_at = NOW()
WHERE id = $1
RETURNING `+inboundColumns, id, actorID)
	if err != nil {
		return nil, err
	}
	_ = AppendInboundEvent(ctx, pool, id, "match_cleared", actorID, map[string]any{"reason": reason})
	return updated, nil
}

// AcceptInboundDocument attaches the document to the matched applicant.
func AcceptInboundDocument(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, actorID *uuid.UUID) (*InboundDocument, error) {
	doc, err := GetInboundDocument(ctx, pool, id)
	if err != nil {
		return nil, err
	}
	if doc.MatchedUserID == nil {
		return nil, ErrInboundInvalidStatus
	}
	if doc.Status == InboundRejected || doc.Status == InboundQuarantined {
		return nil, ErrInboundInvalidStatus
	}
	updated, err := scanInbound(ctx, pool, `
UPDATE transcripts.inbound_documents SET
    status = 'accepted',
    reviewer_id = COALESCE($2, reviewer_id),
    reject_reason = NULL,
    updated_at = NOW()
WHERE id = $1
RETURNING `+inboundColumns, id, actorID)
	if err != nil {
		return nil, err
	}
	_ = AppendInboundEvent(ctx, pool, id, "accepted", actorID, map[string]any{
		"userId": doc.MatchedUserID.String(),
	})
	return updated, nil
}

// RejectInboundDocument rejects with a reason.
func RejectInboundDocument(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, actorID *uuid.UUID, reason string) (*InboundDocument, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, errors.New("transcripts: reject reason required")
	}
	doc, err := scanInbound(ctx, pool, `
UPDATE transcripts.inbound_documents SET
    status = 'rejected',
    reject_reason = $2,
    reviewer_id = COALESCE($3, reviewer_id),
    updated_at = NOW()
WHERE id = $1 AND status NOT IN ('accepted')
RETURNING `+inboundColumns, id, reason, actorID)
	if errors.Is(err, pgx.ErrNoRows) {
		existing, gerr := GetInboundDocument(ctx, pool, id)
		if gerr != nil {
			return nil, gerr
		}
		if existing.Status == InboundAccepted {
			return nil, ErrInboundInvalidStatus
		}
		return nil, ErrInboundNotFound
	}
	if err != nil {
		return nil, err
	}
	_ = AppendInboundEvent(ctx, pool, id, "rejected", actorID, map[string]any{"reason": reason})
	return doc, nil
}

// MarkInboundNotifiedReceived sets the received-notification timestamp once.
func MarkInboundNotifiedReceived(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) error {
	_, err := pool.Exec(ctx, `
UPDATE transcripts.inbound_documents
SET notified_received_at = COALESCE(notified_received_at, NOW()), updated_at = NOW()
WHERE id = $1
`, id)
	return err
}

// MarkInboundNotifiedAccepted sets the accepted-notification timestamp once.
func MarkInboundNotifiedAccepted(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) error {
	_, err := pool.Exec(ctx, `
UPDATE transcripts.inbound_documents
SET notified_accepted_at = COALESCE(notified_accepted_at, NOW()), updated_at = NOW()
WHERE id = $1
`, id)
	return err
}

// AppendInboundEvent writes an audit event.
func AppendInboundEvent(ctx context.Context, pool *pgxpool.Pool, inboundID uuid.UUID, eventType string, actorID *uuid.UUID, detail map[string]any) error {
	var raw []byte
	if detail != nil {
		raw, _ = json.Marshal(detail)
	}
	_, err := pool.Exec(ctx, `
INSERT INTO transcripts.inbound_events (inbound_id, event_type, actor_id, detail)
VALUES ($1, $2, $3, $4)
`, inboundID, eventType, actorID, nullJSON(raw))
	return err
}

// ListInboundEvents returns audit events for a document.
func ListInboundEvents(ctx context.Context, pool *pgxpool.Pool, inboundID uuid.UUID) ([]InboundEvent, error) {
	rows, err := pool.Query(ctx, `
SELECT id, inbound_id, event_type, actor_id, detail, created_at
FROM transcripts.inbound_events
WHERE inbound_id = $1
ORDER BY created_at ASC
`, inboundID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]InboundEvent, 0)
	for rows.Next() {
		var e InboundEvent
		if err := rows.Scan(&e.ID, &e.InboundID, &e.EventType, &e.ActorID, &e.Detail, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// MatchCandidate is a user row considered for inbound matching.
type MatchCandidate struct {
	UserID      uuid.UUID
	Email       string
	DisplayName *string
	FirstName   *string
	LastName    *string
	SID         *string
}

// ListInboundForUser returns inbound documents matched to a learner.
func ListInboundForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, limit int) ([]InboundDocument, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := pool.Query(ctx, `
SELECT `+inboundColumns+`
FROM transcripts.inbound_documents
WHERE matched_user_id = $1
ORDER BY received_at DESC
LIMIT $2
`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]InboundDocument, 0)
	for rows.Next() {
		doc, err := scanInboundRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *doc)
	}
	return out, rows.Err()
}

// ListMatchCandidates returns org users for name/SID matching.
func ListMatchCandidates(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, nameHint, sidHint string, limit int) ([]MatchCandidate, error) {
	if limit <= 0 || limit > 100 {
		limit = 40
	}
	nameHint = strings.TrimSpace(nameHint)
	sidHint = strings.TrimSpace(sidHint)
	rows, err := pool.Query(ctx, `
SELECT id, email, display_name, first_name, last_name, sid
FROM "user".users
WHERE org_id = $1
  AND (
    ($2 <> '' AND (
      COALESCE(display_name, '') ILIKE '%' || $2 || '%'
      OR COALESCE(first_name, '') ILIKE '%' || $2 || '%'
      OR COALESCE(last_name, '') ILIKE '%' || $2 || '%'
      OR LOWER(COALESCE(first_name,'') || ' ' || COALESCE(last_name,'')) = LOWER($2)
    ))
    OR ($3 <> '' AND COALESCE(sid, '') ILIKE $3)
  )
ORDER BY email
LIMIT $4
`, orgID, nameHint, sidHint, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]MatchCandidate, 0)
	for rows.Next() {
		var c MatchCandidate
		if err := rows.Scan(&c.UserID, &c.Email, &c.DisplayName, &c.FirstName, &c.LastName, &c.SID); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

const inboundColumns = `
id, org_id, channel, source_name, external_ref, format, raw_key, raw_bytes, content_hash,
content_type, byte_size, parsed, student_name, student_dob, student_ref, matched_user_id,
match_confidence, match_detail, status, needs_manual_review, reviewer_id, reject_reason,
quarantine_reason, received_at, processed_at, notified_received_at, notified_accepted_at,
created_at, updated_at
`

func inboundArgs(v ...any) []any { return v }

func scanInbound(ctx context.Context, pool *pgxpool.Pool, q string, args ...any) (*InboundDocument, error) {
	row := pool.QueryRow(ctx, q, args...)
	return scanInboundRow(row)
}

type scannable interface {
	Scan(dest ...any) error
}

func scanInboundRow(row scannable) (*InboundDocument, error) {
	var d InboundDocument
	err := row.Scan(
		&d.ID, &d.OrgID, &d.Channel, &d.SourceName, &d.ExternalRef, &d.Format, &d.RawKey, &d.RawBytes, &d.ContentHash,
		&d.ContentType, &d.ByteSize, &d.Parsed, &d.StudentName, &d.StudentDOB, &d.StudentRef, &d.MatchedUserID,
		&d.MatchConfidence, &d.MatchDetail, &d.Status, &d.NeedsManualReview, &d.ReviewerID, &d.RejectReason,
		&d.QuarantineReason, &d.ReceivedAt, &d.ProcessedAt, &d.NotifiedReceivedAt, &d.NotifiedAcceptedAt,
		&d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func nullableStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func nullJSON(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	return raw
}

