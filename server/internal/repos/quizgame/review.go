package quizgame

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

const (
	ReviewKindCatalogSubmission = "catalog_submission"
	ReviewKindReportedContent   = "reported_content"

	ReviewStatusPending  = "pending"
	ReviewStatusApproved = "approved"
	ReviewStatusRejected = "rejected"
	ReviewStatusActioned = "actioned"
)

// ModerationQueueItem is one moderation / catalog review queue row.
type ModerationQueueItem struct {
	ID         string          `json:"id"`
	Kind       string          `json:"kind"`
	KitID      *string         `json:"kitId,omitempty"`
	SessionID  *string         `json:"sessionId,omitempty"`
	Detail     json.RawMessage `json:"detail"`
	Status     string          `json:"status"`
	ReviewerID *string         `json:"reviewerId,omitempty"`
	Reason     *string         `json:"reason,omitempty"`
	CreatedAt  time.Time       `json:"createdAt"`
	ReviewedAt *time.Time      `json:"reviewedAt,omitempty"`
	KitTitle   string          `json:"kitTitle,omitempty"`
	Submitter  *string         `json:"submitterId,omitempty"`
}

// EnqueueCatalogSubmission adds a pending catalog review item for a kit.
func EnqueueCatalogSubmission(ctx context.Context, pool *pgxpool.Pool, kitID string, submitter uuid.UUID, detail map[string]any) (*ModerationQueueItem, error) {
	kid, err := uuid.Parse(kitID)
	if err != nil {
		return nil, fmt.Errorf("quizgame: invalid kit id")
	}
	if detail == nil {
		detail = map[string]any{}
	}
	detail["submitterId"] = submitter.String()
	raw, _ := json.Marshal(detail)
	row := pool.QueryRow(ctx, `
		INSERT INTO quizgame.review_queue (kind, kit_id, detail, status)
		VALUES ($1, $2, $3::jsonb, $4)
		RETURNING id, kind, kit_id, session_id, detail, status, reviewer_id, reason, created_at, reviewed_at
	`, ReviewKindCatalogSubmission, kid, raw, ReviewStatusPending)
	item, err := scanModerationQueueItem(row)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// EnqueueReportedContent adds a reported-content review item.
func EnqueueReportedContent(ctx context.Context, pool *pgxpool.Pool, kitID, sessionID *string, detail map[string]any) (*ModerationQueueItem, error) {
	var kid, sid any
	if kitID != nil {
		id, err := uuid.Parse(*kitID)
		if err != nil {
			return nil, fmt.Errorf("quizgame: invalid kit id")
		}
		kid = id
	}
	if sessionID != nil {
		id, err := uuid.Parse(*sessionID)
		if err != nil {
			return nil, fmt.Errorf("quizgame: invalid session id")
		}
		sid = id
	}
	if kid == nil && sid == nil {
		return nil, fmt.Errorf("quizgame: kit or session required")
	}
	if detail == nil {
		detail = map[string]any{}
	}
	raw, _ := json.Marshal(detail)
	row := pool.QueryRow(ctx, `
		INSERT INTO quizgame.review_queue (kind, kit_id, session_id, detail, status)
		VALUES ($1, $2, $3, $4::jsonb, $5)
		RETURNING id, kind, kit_id, session_id, detail, status, reviewer_id, reason, created_at, reviewed_at
	`, ReviewKindReportedContent, kid, sid, raw, ReviewStatusPending)
	item, err := scanModerationQueueItem(row)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// ListReviewQueue returns pending (or all) review items newest-first.
func ListReviewQueue(ctx context.Context, pool *pgxpool.Pool, status string, limit int) ([]ModerationQueueItem, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	status = strings.TrimSpace(strings.ToLower(status))
	var rows pgx.Rows
	var err error
	if status == "" || status == "pending" {
		rows, err = pool.Query(ctx, `
			SELECT q.id, q.kind, q.kit_id, q.session_id, q.detail, q.status, q.reviewer_id, q.reason,
			       q.created_at, q.reviewed_at, COALESCE(k.title, ''), k.created_by
			FROM quizgame.review_queue q
			LEFT JOIN quizgame.kits k ON k.id = q.kit_id
			WHERE q.status = 'pending'
			ORDER BY q.created_at ASC
			LIMIT $1
		`, limit)
	} else {
		rows, err = pool.Query(ctx, `
			SELECT q.id, q.kind, q.kit_id, q.session_id, q.detail, q.status, q.reviewer_id, q.reason,
			       q.created_at, q.reviewed_at, COALESCE(k.title, ''), k.created_by
			FROM quizgame.review_queue q
			LEFT JOIN quizgame.kits k ON k.id = q.kit_id
			WHERE q.status = $1
			ORDER BY q.created_at DESC
			LIMIT $2
		`, status, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ModerationQueueItem, 0)
	for rows.Next() {
		item, err := scanModerationQueueItemJoined(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// CountPendingReviews returns the moderation queue depth.
func CountPendingReviews(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*)::int FROM quizgame.review_queue WHERE status = 'pending'
	`).Scan(&n)
	return n, err
}

// GetModerationQueueItem loads one queue row.
func GetModerationQueueItem(ctx context.Context, pool *pgxpool.Pool, id string) (*ModerationQueueItem, error) {
	rid, err := uuid.Parse(id)
	if err != nil {
		return nil, nil
	}
	row := pool.QueryRow(ctx, `
		SELECT q.id, q.kind, q.kit_id, q.session_id, q.detail, q.status, q.reviewer_id, q.reason,
		       q.created_at, q.reviewed_at, COALESCE(k.title, ''), k.created_by
		FROM quizgame.review_queue q
		LEFT JOIN quizgame.kits k ON k.id = q.kit_id
		WHERE q.id = $1
	`, rid)
	item, err := scanModerationQueueItemJoined(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

// ApproveReview marks the item approved and lists the kit in the public catalog when applicable.
func ApproveReview(ctx context.Context, pool *pgxpool.Pool, id string, reviewer uuid.UUID) (*ModerationQueueItem, error) {
	item, err := GetModerationQueueItem(ctx, pool, id)
	if err != nil || item == nil {
		return item, err
	}
	if item.Status != ReviewStatusPending {
		return nil, fmt.Errorf("quizgame: review item is not pending")
	}
	if item.Kind == ReviewKindCatalogSubmission && item.KitID != nil {
		if _, err := SetCatalogStatus(ctx, pool, *item.KitID, "listed"); err != nil {
			return nil, err
		}
	}
	return finalizeReview(ctx, pool, id, reviewer, ReviewStatusApproved, "")
}

// RejectReview marks the item rejected, unlists the kit, and stores the reason.
func RejectReview(ctx context.Context, pool *pgxpool.Pool, id string, reviewer uuid.UUID, reason string) (*ModerationQueueItem, error) {
	item, err := GetModerationQueueItem(ctx, pool, id)
	if err != nil || item == nil {
		return item, err
	}
	if item.Status != ReviewStatusPending {
		return nil, fmt.Errorf("quizgame: review item is not pending")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, fmt.Errorf("quizgame: rejection reason is required")
	}
	if item.Kind == ReviewKindCatalogSubmission && item.KitID != nil {
		if _, err := SetCatalogStatus(ctx, pool, *item.KitID, "rejected"); err != nil {
			return nil, err
		}
	}
	return finalizeReview(ctx, pool, id, reviewer, ReviewStatusRejected, reason)
}

// ActionReview marks reported content as actioned (takedown already applied by caller or here).
func ActionReview(ctx context.Context, pool *pgxpool.Pool, id string, reviewer uuid.UUID, reason string) (*ModerationQueueItem, error) {
	item, err := GetModerationQueueItem(ctx, pool, id)
	if err != nil || item == nil {
		return item, err
	}
	if item.Status != ReviewStatusPending {
		return nil, fmt.Errorf("quizgame: review item is not pending")
	}
	if item.KitID != nil {
		_, _ = SetCatalogStatus(ctx, pool, *item.KitID, "unlisted")
	}
	return finalizeReview(ctx, pool, id, reviewer, ReviewStatusActioned, reason)
}

func finalizeReview(ctx context.Context, pool *pgxpool.Pool, id string, reviewer uuid.UUID, status, reason string) (*ModerationQueueItem, error) {
	rid, err := uuid.Parse(id)
	if err != nil {
		return nil, nil
	}
	var reasonAny any
	if strings.TrimSpace(reason) != "" {
		reasonAny = strings.TrimSpace(reason)
	}
	_, err = pool.Exec(ctx, `
		UPDATE quizgame.review_queue
		SET status = $2, reviewer_id = $3, reason = $4, reviewed_at = NOW()
		WHERE id = $1
	`, rid, status, reviewer, reasonAny)
	if err != nil {
		return nil, err
	}
	return GetModerationQueueItem(ctx, pool, id)
}

func scanModerationQueueItem(row pgx.Row) (ModerationQueueItem, error) {
	var item ModerationQueueItem
	var id uuid.UUID
	var kitID, sessionID, reviewerID *uuid.UUID
	var reason *string
	var reviewedAt *time.Time
	var detail []byte
	if err := row.Scan(&id, &item.Kind, &kitID, &sessionID, &detail, &item.Status, &reviewerID, &reason, &item.CreatedAt, &reviewedAt); err != nil {
		return item, err
	}
	item.ID = id.String()
	if kitID != nil {
		s := kitID.String()
		item.KitID = &s
	}
	if sessionID != nil {
		s := sessionID.String()
		item.SessionID = &s
	}
	if reviewerID != nil {
		s := reviewerID.String()
		item.ReviewerID = &s
	}
	item.Reason = reason
	item.ReviewedAt = reviewedAt
	if len(detail) == 0 {
		item.Detail = json.RawMessage(`{}`)
	} else {
		item.Detail = detail
	}
	return item, nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanModerationQueueItemJoined(row scannable) (ModerationQueueItem, error) {
	var item ModerationQueueItem
	var id uuid.UUID
	var kitID, sessionID, reviewerID, createdBy *uuid.UUID
	var reason *string
	var reviewedAt *time.Time
	var detail []byte
	var title string
	if err := row.Scan(&id, &item.Kind, &kitID, &sessionID, &detail, &item.Status, &reviewerID, &reason,
		&item.CreatedAt, &reviewedAt, &title, &createdBy); err != nil {
		return item, err
	}
	item.ID = id.String()
	item.KitTitle = title
	if kitID != nil {
		s := kitID.String()
		item.KitID = &s
	}
	if sessionID != nil {
		s := sessionID.String()
		item.SessionID = &s
	}
	if reviewerID != nil {
		s := reviewerID.String()
		item.ReviewerID = &s
	}
	if createdBy != nil {
		s := createdBy.String()
		item.Submitter = &s
	}
	item.Reason = reason
	item.ReviewedAt = reviewedAt
	if len(detail) == 0 {
		item.Detail = json.RawMessage(`{}`)
	} else {
		item.Detail = detail
		var m map[string]any
		if json.Unmarshal(detail, &m) == nil {
			if sid, ok := m["submitterId"].(string); ok && sid != "" && item.Submitter == nil {
				item.Submitter = &sid
			}
		}
	}
	return item, nil
}
