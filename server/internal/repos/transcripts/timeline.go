package transcripts

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/models/transcriptorder"
)

// TimelineKind classifies a merged timeline entry.
type TimelineKind string

const (
	TimelineKindOrder    TimelineKind = "order"
	TimelineKindDelivery TimelineKind = "delivery"
)

// TimelineEntry is one step in the learner-facing order tracking timeline (T10).
type TimelineEntry struct {
	ID        string
	Kind      TimelineKind
	At        time.Time
	Status    string
	Label     string
	ItemID    *uuid.UUID
	Adapter   *string
	AttemptNo *int
	Detail    *string
	Reason    *string
}

// OrderTimeline is the merged tracking view for one order.
type OrderTimeline struct {
	OrderID        uuid.UUID
	Status         OrderStatus
	CanCancel      bool
	CanResendItems []uuid.UUID
	Entries        []TimelineEntry
	Items          []OrderItem
}

// LearnerCancelAllowed reports whether the student may cancel (pre-delivery).
func LearnerCancelAllowed(o *Order) bool {
	if o == nil {
		return false
	}
	switch o.Status {
	case OrderDraft, OrderPendingConsent, OrderPendingPayment, OrderInReview, OrderOnHold:
		return true
	case OrderProcessing:
		for _, it := range o.Items {
			switch it.Status {
			case ItemDelivered, ItemDelivering:
				return false
			}
		}
		return true
	default:
		return false
	}
}

// BuildOrderTimeline merges T03 order_events with T06 delivery attempts.
func BuildOrderTimeline(ctx context.Context, pool *pgxpool.Pool, orderID, userID uuid.UUID) (*OrderTimeline, error) {
	o, err := GetOrderForUser(ctx, pool, orderID, userID)
	if err != nil {
		return nil, err
	}
	events, err := ListOrderEvents(ctx, pool, orderID)
	if err != nil {
		return nil, err
	}
	var entries []TimelineEntry
	for _, e := range events {
		label := timelineLabelForOrderState(e.ToState)
		entries = append(entries, TimelineEntry{
			ID:     e.ID.String(),
			Kind:   TimelineKindOrder,
			At:     e.CreatedAt,
			Status: e.ToState,
			Label:  label,
			ItemID: e.ItemID,
			Reason: e.Reason,
		})
	}
	var resendable []uuid.UUID
	for _, it := range o.Items {
		if it.Status == ItemFailed || it.Status == ItemDelivered {
			resendable = append(resendable, it.ID)
		}
		attempts, aerr := ListDeliveryAttemptsForItem(ctx, pool, it.ID)
		if aerr != nil {
			return nil, aerr
		}
		itemID := it.ID
		for _, a := range attempts {
			adapter := string(a.Adapter)
			attemptNo := a.AttemptNo
			status := string(a.Status)
			at := a.UpdatedAt
			if at.IsZero() {
				at = a.CreatedAt
			}
			entries = append(entries, TimelineEntry{
				ID:        a.ID.String(),
				Kind:      TimelineKindDelivery,
				At:        at,
				Status:    status,
				Label:     timelineLabelForDelivery(status),
				ItemID:    &itemID,
				Adapter:   &adapter,
				AttemptNo: &attemptNo,
				Detail:    a.Detail,
			})
		}
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].At.Equal(entries[j].At) {
			return entries[i].ID < entries[j].ID
		}
		return entries[i].At.Before(entries[j].At)
	})
	return &OrderTimeline{
		OrderID:        o.ID,
		Status:         o.Status,
		CanCancel:      LearnerCancelAllowed(o),
		CanResendItems: resendable,
		Entries:        entries,
		Items:          o.Items,
	}, nil
}

func timelineLabelForOrderState(state string) string {
	switch transcriptorder.OrderStatus(state) {
	case transcriptorder.OrderDraft:
		return "Draft"
	case transcriptorder.OrderPendingConsent:
		return "Consent needed"
	case transcriptorder.OrderPendingPayment:
		return "Payment needed"
	case transcriptorder.OrderInReview:
		return "Submitted"
	case transcriptorder.OrderOnHold:
		return "On hold"
	case transcriptorder.OrderProcessing:
		return "Processing"
	case transcriptorder.OrderCompleted:
		return "Completed"
	case transcriptorder.OrderCanceled:
		return "Canceled"
	case transcriptorder.OrderRejected:
		return "Rejected"
	case transcriptorder.OrderFailed:
		return "Failed"
	default:
		if state == "ready" || state == "delivering" || state == "delivered" || state == "failed" {
			return timelineLabelForDelivery(state)
		}
		return state
	}
}

func timelineLabelForDelivery(status string) string {
	switch status {
	case "queued":
		return "Queued"
	case "sent":
		return "Sent"
	case "delivered":
		return "Delivered"
	case "opened":
		return "Opened"
	case "failed":
		return "Failed"
	default:
		return status
	}
}
