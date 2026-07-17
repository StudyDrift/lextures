package transcriptorder

import (
	"errors"
	"fmt"
	"strings"
)

// OrderStatus is the T03 order lifecycle state.
type OrderStatus string

const (
	OrderDraft           OrderStatus = "draft"
	OrderPendingConsent  OrderStatus = "pending_consent"
	OrderPendingPayment  OrderStatus = "pending_payment"
	OrderInReview        OrderStatus = "in_review"
	OrderOnHold          OrderStatus = "on_hold"
	OrderProcessing      OrderStatus = "processing"
	OrderCompleted       OrderStatus = "completed"
	OrderCanceled        OrderStatus = "canceled"
	OrderRejected        OrderStatus = "rejected"
	OrderFailed          OrderStatus = "failed" // legacy terminal
)

// AllOrderStatuses is the closed set of order statuses.
var AllOrderStatuses = []OrderStatus{
	OrderDraft,
	OrderPendingConsent,
	OrderPendingPayment,
	OrderInReview,
	OrderOnHold,
	OrderProcessing,
	OrderCompleted,
	OrderCanceled,
	OrderRejected,
	OrderFailed,
}

// ItemStatus is per-item fulfillment state.
type ItemStatus string

const (
	ItemPending    ItemStatus = "pending"
	ItemReady      ItemStatus = "ready"
	ItemDelivering ItemStatus = "delivering"
	ItemDelivered  ItemStatus = "delivered"
	ItemFailed     ItemStatus = "failed"
	ItemCanceled   ItemStatus = "canceled"
)

// AllItemStatuses is the closed set of item statuses.
var AllItemStatuses = []ItemStatus{
	ItemPending,
	ItemReady,
	ItemDelivering,
	ItemDelivered,
	ItemFailed,
	ItemCanceled,
}

// Action is a registrar/student/system transition action.
type Action string

const (
	ActionSubmit   Action = "submit"
	ActionApprove  Action = "approve"
	ActionReject   Action = "reject"
	ActionCancel   Action = "cancel"
	ActionComplete Action = "complete"
	ActionHold     Action = "hold"
	ActionRelease  Action = "release"
)

var (
	ErrIllegalTransition = errors.New("illegal order transition")
	ErrReasonRequired    = errors.New("reason is required")
)

// ParseOrderStatus validates and normalizes an order status string.
func ParseOrderStatus(raw string) (OrderStatus, error) {
	s := OrderStatus(strings.ToLower(strings.TrimSpace(raw)))
	switch s {
	case OrderDraft, OrderPendingConsent, OrderPendingPayment, OrderInReview, OrderOnHold,
		OrderProcessing, OrderCompleted, OrderCanceled, OrderRejected, OrderFailed:
		return s, nil
	// Legacy aliases (pre-T03).
	case "submitted":
		return OrderCompleted, nil
	case "queued":
		return OrderInReview, nil
	default:
		return "", fmt.Errorf("invalid order status %q", raw)
	}
}

// ParseItemStatus validates and normalizes an item status string.
func ParseItemStatus(raw string) (ItemStatus, error) {
	s := ItemStatus(strings.ToLower(strings.TrimSpace(raw)))
	switch s {
	case ItemPending, ItemReady, ItemDelivering, ItemDelivered, ItemFailed, ItemCanceled:
		return s, nil
	default:
		return "", fmt.Errorf("invalid item status %q", raw)
	}
}

// ParseAction validates a transition action string.
func ParseAction(raw string) (Action, error) {
	a := Action(strings.ToLower(strings.TrimSpace(raw)))
	switch a {
	case ActionSubmit, ActionApprove, ActionReject, ActionCancel, ActionComplete, ActionHold, ActionRelease:
		return a, nil
	default:
		return "", fmt.Errorf("invalid action %q", raw)
	}
}

// IsTerminal reports whether the order can no longer change.
func (s OrderStatus) IsTerminal() bool {
	switch s {
	case OrderCompleted, OrderCanceled, OrderRejected, OrderFailed:
		return true
	default:
		return false
	}
}

// allowedOrderTransitions is the declarative legal edge set (from → to).
var allowedOrderTransitions = map[OrderStatus]map[OrderStatus]struct{}{
	OrderDraft: {
		OrderPendingConsent: {},
		OrderPendingPayment: {},
		OrderInReview:       {},
		OrderOnHold:         {},
		OrderProcessing:     {},
		OrderCanceled:       {},
	},
	OrderPendingConsent: {
		OrderPendingPayment: {},
		OrderInReview:       {},
		OrderOnHold:         {},
		OrderProcessing:     {}, // consent signed + auto-approval
		OrderCanceled:       {},
		OrderRejected:       {},
	},
	OrderPendingPayment: {
		OrderPendingConsent: {}, // consent revoked / expired (T04)
		OrderInReview:       {},
		OrderOnHold:         {},
		OrderCanceled:       {},
		OrderRejected:       {},
	},
	OrderInReview: {
		OrderPendingConsent: {}, // consent revoked (T04)
		OrderPendingPayment: {}, // payment gate (T05)
		OrderOnHold:         {},
		OrderProcessing:     {},
		OrderRejected:       {},
		OrderCanceled:       {},
	},
	OrderOnHold: {
		OrderPendingConsent: {}, // consent revoked (T04)
		OrderPendingPayment: {}, // payment gate (T05)
		OrderInReview:       {},
		OrderProcessing:     {},
		OrderCanceled:       {},
		OrderRejected:       {},
	},
	OrderProcessing: {
		OrderPendingConsent: {}, // consent revoked before delivery (T04)
		OrderPendingPayment: {}, // payment gate (T05)
		OrderCompleted:      {},
		OrderOnHold:         {},
		OrderCanceled:       {},
	},
}

// CanTransitionOrder reports whether from→to is a legal order edge.
func CanTransitionOrder(from, to OrderStatus) bool {
	if from == to {
		return false
	}
	next, ok := allowedOrderTransitions[from]
	if !ok {
		return false
	}
	_, ok = next[to]
	return ok
}

// ValidateOrderTransition returns ErrIllegalTransition when from→to is not allowed.
func ValidateOrderTransition(from, to OrderStatus) error {
	if CanTransitionOrder(from, to) {
		return nil
	}
	return fmt.Errorf("%w: %s → %s", ErrIllegalTransition, from, to)
}

// allowedItemTransitions is the declarative legal item edge set.
var allowedItemTransitions = map[ItemStatus]map[ItemStatus]struct{}{
	ItemPending: {
		ItemReady:    {},
		ItemCanceled: {},
		ItemFailed:   {},
	},
	ItemReady: {
		ItemDelivering: {},
		ItemCanceled:   {},
		ItemDelivered:  {}, // fallback when delivery is stubbed
	},
	ItemDelivering: {
		ItemDelivered: {},
		ItemFailed:    {},
		ItemCanceled:  {},
	},
	ItemFailed: {
		ItemReady: {}, // T06 resend
	},
}

// CanTransitionItem reports whether from→to is a legal item edge.
func CanTransitionItem(from, to ItemStatus) bool {
	if from == to {
		return false
	}
	next, ok := allowedItemTransitions[from]
	if !ok {
		return false
	}
	_, ok = next[to]
	return ok
}

// ValidateItemTransition returns ErrIllegalTransition when from→to is not allowed.
func ValidateItemTransition(from, to ItemStatus) error {
	if CanTransitionItem(from, to) {
		return nil
	}
	return fmt.Errorf("%w: item %s → %s", ErrIllegalTransition, from, to)
}

// GateContext is evaluated on forward transitions (T04 consent / T05 payment stubs).
type GateContext struct {
	ConsentSatisfied bool
	PaymentSatisfied bool
	HasBlockingHold  bool
	AutoApproval     bool
}

// ResolveSubmitTarget picks the post-submit status from gates (pure).
// Consent/payment default satisfied until T04/T05 wire real checks.
func ResolveSubmitTarget(g GateContext) OrderStatus {
	if g.HasBlockingHold {
		return OrderOnHold
	}
	if !g.ConsentSatisfied {
		return OrderPendingConsent
	}
	if !g.PaymentSatisfied {
		return OrderPendingPayment
	}
	if g.AutoApproval {
		return OrderProcessing
	}
	return OrderInReview
}

// ResolveReleaseTarget picks where an on_hold order goes after holds clear.
func ResolveReleaseTarget(g GateContext) OrderStatus {
	if g.HasBlockingHold {
		return OrderOnHold
	}
	if !g.ConsentSatisfied {
		return OrderPendingConsent
	}
	if !g.PaymentSatisfied {
		return OrderPendingPayment
	}
	if g.AutoApproval {
		return OrderProcessing
	}
	return OrderInReview
}

// TargetForAction maps a registrar/student action to a destination status.
func TargetForAction(from OrderStatus, action Action, reason string) (OrderStatus, error) {
	switch action {
	case ActionApprove:
		if from != OrderInReview {
			return "", fmt.Errorf("%w: approve requires in_review (was %s)", ErrIllegalTransition, from)
		}
		return OrderProcessing, nil
	case ActionReject:
		if strings.TrimSpace(reason) == "" {
			return "", ErrReasonRequired
		}
		switch from {
		case OrderInReview, OrderOnHold, OrderPendingConsent, OrderPendingPayment:
			return OrderRejected, nil
		default:
			return "", fmt.Errorf("%w: reject not allowed from %s", ErrIllegalTransition, from)
		}
	case ActionCancel:
		switch from {
		case OrderDraft, OrderPendingConsent, OrderPendingPayment, OrderInReview, OrderOnHold, OrderProcessing:
			return OrderCanceled, nil
		default:
			return "", fmt.Errorf("%w: cancel not allowed from %s", ErrIllegalTransition, from)
		}
	case ActionComplete:
		if from != OrderProcessing {
			return "", fmt.Errorf("%w: complete requires processing (was %s)", ErrIllegalTransition, from)
		}
		return OrderCompleted, nil
	case ActionHold:
		switch from {
		case OrderInReview, OrderProcessing, OrderPendingConsent, OrderPendingPayment:
			return OrderOnHold, nil
		default:
			return "", fmt.Errorf("%w: hold not allowed from %s", ErrIllegalTransition, from)
		}
	case ActionRelease:
		if from != OrderOnHold {
			return "", fmt.Errorf("%w: release requires on_hold (was %s)", ErrIllegalTransition, from)
		}
		// Caller must compute concrete target via ResolveReleaseTarget.
		return OrderInReview, nil
	case ActionSubmit:
		if from != OrderDraft {
			return "", fmt.Errorf("%w: submit requires draft (was %s)", ErrIllegalTransition, from)
		}
		return OrderInReview, nil // concrete target via ResolveSubmitTarget
	default:
		return "", fmt.Errorf("%w: unknown action %s", ErrIllegalTransition, action)
	}
}
