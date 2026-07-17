// Package transcriptnotify sends learner/guardian/registrar notifications for transcript orders (T10).
package transcriptnotify

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/logging"
	"github.com/lextures/lextures/server/internal/notificationevents"
	"github.com/lextures/lextures/server/internal/repos/emailjobs"
	"github.com/lextures/lextures/server/internal/repos/parentlinks"
	transcriptsrepo "github.com/lextures/lextures/server/internal/repos/transcripts"
	"github.com/lextures/lextures/server/internal/service/notifications"
	"github.com/lextures/lextures/server/internal/telemetry"
)

// LifecycleEvent is a normalized T10 notification event key.
type LifecycleEvent string

const (
	EventSubmitted      LifecycleEvent = "submitted"
	EventOnHold         LifecycleEvent = "on_hold"
	EventConsentNeeded  LifecycleEvent = "consent_needed"
	EventPaymentNeeded  LifecycleEvent = "payment_needed"
	EventApproved       LifecycleEvent = "approved"
	EventRejected       LifecycleEvent = "rejected"
	EventCanceled       LifecycleEvent = "canceled"
	EventSent           LifecycleEvent = "sent"
	EventDelivered      LifecycleEvent = "delivered"
	EventOpened         LifecycleEvent = "opened"
	EventFailed         LifecycleEvent = "failed"
	EventExceptionHold  LifecycleEvent = "exception_hold"
	EventExceptionFail  LifecycleEvent = "exception_failed"
)

// mapping describes one event → channels / copy.
type mapping struct {
	EventType     string
	Title         string
	Body          string
	Transactional bool // email bypasses preference opt-out
	Learner       bool
	Guardian      bool
	Registrar     bool
}

var eventMap = map[LifecycleEvent]mapping{
	EventSubmitted: {
		EventType: notificationevents.TranscriptOrderSubmitted,
		Title:     "Transcript order submitted",
		Body:      "Your transcript order was submitted and is being reviewed.",
		Learner:   true, Guardian: true,
	},
	EventOnHold: {
		EventType: notificationevents.TranscriptOrderOnHold,
		Title:     "Transcript order on hold",
		Body:      "Your transcript order is on hold. Open the order for details.",
		Learner:   true, Guardian: true,
	},
	EventConsentNeeded: {
		EventType:     notificationevents.TranscriptOrderConsent,
		Title:         "Consent needed for transcript order",
		Body:          "Authorization is required before we can release your transcript.",
		Transactional: true,
		Learner:       true, Guardian: true,
	},
	EventPaymentNeeded: {
		EventType:     notificationevents.TranscriptOrderPayment,
		Title:         "Payment needed for transcript order",
		Body:          "Payment is required to continue processing your transcript order.",
		Transactional: true,
		Learner:       true, Guardian: true,
	},
	EventApproved: {
		EventType: notificationevents.TranscriptOrderApproved,
		Title:     "Transcript order approved",
		Body:      "Your transcript order was approved and is being prepared for delivery.",
		Learner:   true, Guardian: true,
	},
	EventRejected: {
		EventType: notificationevents.TranscriptOrderRejected,
		Title:     "Transcript order rejected",
		Body:      "Your transcript order was rejected. Open the order for the reason.",
		Learner:   true, Guardian: true,
	},
	EventCanceled: {
		EventType: notificationevents.TranscriptOrderCanceled,
		Title:     "Transcript order canceled",
		Body:      "Your transcript order was canceled.",
		Learner:   true, Guardian: true,
	},
	EventSent: {
		EventType: notificationevents.TranscriptOrderSent,
		Title:     "Transcript sent",
		Body:      "Your transcript was sent to a recipient. Track delivery in your order.",
		Learner:   true, Guardian: true,
	},
	EventDelivered: {
		EventType: notificationevents.TranscriptOrderDelivered,
		Title:     "Transcript delivered",
		Body:      "A recipient received your transcript delivery.",
		Learner:   true, Guardian: true,
	},
	EventOpened: {
		EventType: notificationevents.TranscriptOrderOpened,
		Title:     "Transcript opened",
		Body:      "A recipient opened your secure transcript link.",
		Learner:   true, Guardian: true,
	},
	EventFailed: {
		EventType: notificationevents.TranscriptOrderFailed,
		Title:     "Transcript delivery failed",
		Body:      "Delivery failed for a recipient. You may resend from your order.",
		Learner:   true, Guardian: true,
	},
	EventExceptionHold: {
		EventType: notificationevents.TranscriptOrderException,
		Title:     "Transcript order needs attention",
		Body:      "A transcript order is on hold and needs registrar action.",
		Registrar: true,
	},
	EventExceptionFail: {
		EventType: notificationevents.TranscriptOrderException,
		Title:     "Transcript delivery dead-letter",
		Body:      "A transcript delivery failed permanently and needs registrar action.",
		Registrar: true,
	},
}

// Service fans out email + push for transcript order events.
type Service struct {
	Pool   *pgxpool.Pool
	Config config.Config
	Email  *notifications.Service
	Push   *notifications.PushService
}

// NotifyOrderStatus maps an order status (after transition/submit) to learner notifications.
func (s *Service) NotifyOrderStatus(ctx context.Context, order *transcriptsrepo.Order) {
	if s == nil || s.Pool == nil || order == nil {
		return
	}
	ev, ok := EventForOrderStatus(order.Status)
	if !ok {
		return
	}
	s.Notify(ctx, order, nil, ev)
	if order.Status == transcriptsrepo.OrderOnHold {
		s.Notify(ctx, order, nil, EventExceptionHold)
	}
}

// NotifyDeliveryStatus notifies on T06 attempt status changes.
func (s *Service) NotifyDeliveryStatus(
	ctx context.Context,
	order *transcriptsrepo.Order,
	itemID uuid.UUID,
	status transcriptsrepo.DeliveryAttemptStatus,
) {
	if s == nil || order == nil {
		return
	}
	var ev LifecycleEvent
	switch status {
	case transcriptsrepo.AttemptSent:
		ev = EventSent
	case transcriptsrepo.AttemptDelivered:
		ev = EventDelivered
	case transcriptsrepo.AttemptOpened:
		ev = EventOpened
	case transcriptsrepo.AttemptFailed:
		ev = EventFailed
	default:
		return
	}
	id := itemID
	s.Notify(ctx, order, &id, ev)
	if status == transcriptsrepo.AttemptFailed {
		s.Notify(ctx, order, &id, EventExceptionFail)
	}
}

// EventForOrderStatus maps order status to a T10 lifecycle event.
func EventForOrderStatus(status transcriptsrepo.OrderStatus) (LifecycleEvent, bool) {
	switch status {
	case transcriptsrepo.OrderInReview:
		return EventSubmitted, true
	case transcriptsrepo.OrderOnHold:
		return EventOnHold, true
	case transcriptsrepo.OrderPendingConsent:
		return EventConsentNeeded, true
	case transcriptsrepo.OrderPendingPayment:
		return EventPaymentNeeded, true
	case transcriptsrepo.OrderProcessing:
		return EventApproved, true
	case transcriptsrepo.OrderRejected:
		return EventRejected, true
	case transcriptsrepo.OrderCanceled:
		return EventCanceled, true
	default:
		return "", false
	}
}

// Notify sends mapped channels for one lifecycle event (idempotent per claim).
func (s *Service) Notify(
	ctx context.Context,
	order *transcriptsrepo.Order,
	itemID *uuid.UUID,
	event LifecycleEvent,
) {
	if s == nil || s.Pool == nil || order == nil {
		return
	}
	m, ok := eventMap[event]
	if !ok {
		return
	}
	link := s.orderLink(order.ID)
	adminLink := s.adminLink(order.ID)

	if m.Learner {
		s.sendToUser(ctx, order, itemID, event, m, order.UserID, link, false)
	}
	if m.Guardian {
		s.sendToGuardians(ctx, order, itemID, event, m, link)
	}
	if m.Registrar {
		s.sendToRegistrars(ctx, order, itemID, event, m, adminLink)
	}
	telemetry.RecordBusinessEvent("transcript.notification.sent")
}

func (s *Service) sendToGuardians(
	ctx context.Context,
	order *transcriptsrepo.Order,
	itemID *uuid.UUID,
	event LifecycleEvent,
	m mapping,
	link string,
) {
	if order.OrgID == nil {
		return
	}
	links, err := parentlinks.ListParentsForStudent(ctx, s.Pool, order.UserID, *order.OrgID)
	if err != nil || len(links) == 0 {
		return
	}
	for _, l := range links {
		if l.Status != "active" && l.Status != "pending" {
			continue
		}
		s.sendToUser(ctx, order, itemID, event, m, l.ParentUserID, link, false)
	}
}

func (s *Service) sendToRegistrars(
	ctx context.Context,
	order *transcriptsrepo.Order,
	itemID *uuid.UUID,
	event LifecycleEvent,
	m mapping,
	link string,
) {
	if order.OrgID == nil {
		return
	}
	ids, err := transcriptsrepo.ListOrgAdminUserIDs(ctx, s.Pool, *order.OrgID, 20)
	if err != nil || len(ids) == 0 {
		return
	}
	for _, id := range ids {
		s.sendToUser(ctx, order, itemID, event, m, id, link, true)
	}
}

func (s *Service) sendToUser(
	ctx context.Context,
	order *transcriptsrepo.Order,
	itemID *uuid.UUID,
	event LifecycleEvent,
	m mapping,
	userID uuid.UUID,
	link string,
	registrar bool,
) {
	recipient := userID.String()
	eventKey := string(event)
	if registrar {
		eventKey = string(event) + ":registrar:" + recipient
	} else if userID != order.UserID {
		eventKey = string(event) + ":guardian:" + recipient
	}

	// Email
	claimedEmail, err := transcriptsrepo.TryClaimNotification(
		ctx, s.Pool, order.ID, itemID, eventKey, string(transcriptsrepo.NotifyChannelEmail), recipient,
	)
	if err != nil {
		slog.Warn("transcriptnotify.claim_email", "err", err, "order_id", order.ID.String())
	} else if claimedEmail {
		s.enqueueEmail(ctx, userID, m, link, registrar)
		logging.GlobalTranscriptNotificationMetrics.Inc("email")
	}

	// Push + in-app (PushService inserts inbox; claim push once)
	claimedPush, err := transcriptsrepo.TryClaimNotification(
		ctx, s.Pool, order.ID, itemID, eventKey, string(transcriptsrepo.NotifyChannelPush), recipient,
	)
	if err != nil {
		slog.Warn("transcriptnotify.claim_push", "err", err, "order_id", order.ID.String())
	} else if claimedPush && s.Push != nil {
		title := m.Title
		body := m.Body
		if err := s.Push.Enqueue(ctx, userID, m.EventType, title, body, link); err != nil {
			slog.Warn("transcriptnotify.push", "err", err, "user_id", userID.String())
		} else {
			logging.GlobalTranscriptNotificationMetrics.Inc("push")
			logging.GlobalTranscriptNotificationMetrics.Inc("in_app")
			_, _ = transcriptsrepo.TryClaimNotification(
				ctx, s.Pool, order.ID, itemID, eventKey, string(transcriptsrepo.NotifyChannelInApp), recipient,
			)
		}
	}
}

func (s *Service) enqueueEmail(ctx context.Context, userID uuid.UUID, m mapping, link string, registrar bool) {
	if !s.Config.EmailNotificationsEnabled || s.Pool == nil {
		return
	}
	template := "transcript_order_update"
	if registrar {
		template = "transcript_order_exception"
	}
	vars := map[string]string{
		"subject": m.Title,
		"title":   m.Title,
		"message": m.Body,
		"link":    link,
	}
	if m.Transactional {
		// Consent/payment are transactional — bypass preference opt-out (FR-3).
		if s.Email != nil {
			vars["unsubscribeUrl"] = s.Email.UnsubscribeURL(userID, m.EventType)
		}
		if _, err := emailjobs.Enqueue(ctx, s.Pool, userID, m.EventType, m.Title, template, vars); err != nil {
			slog.Warn("transcriptnotify.email_tx", "err", err, "user_id", userID.String())
		}
		return
	}
	if s.Email == nil {
		return
	}
	if err := s.Email.EnqueueEmail(ctx, userID, m.EventType, template, vars, nil); err != nil {
		slog.Warn("transcriptnotify.email", "err", err, "user_id", userID.String())
	}
}

func (s *Service) orderLink(orderID uuid.UUID) string {
	origin := strings.TrimRight(strings.TrimSpace(s.Config.PublicWebOrigin), "/")
	if origin == "" {
		origin = "http://localhost:5173"
	}
	return fmt.Sprintf("%s/transcripts?orderId=%s", origin, orderID.String())
}

func (s *Service) adminLink(orderID uuid.UUID) string {
	origin := strings.TrimRight(strings.TrimSpace(s.Config.PublicWebOrigin), "/")
	if origin == "" {
		origin = "http://localhost:5173"
	}
	return fmt.Sprintf("%s/admin/transcripts?orderId=%s", origin, orderID.String())
}

// MappingForTest exposes event mapping for unit tests.
func MappingForTest(event LifecycleEvent) (eventType, title string, transactional, learner, guardian, registrar bool, ok bool) {
	m, ok := eventMap[event]
	if !ok {
		return "", "", false, false, false, false, false
	}
	return m.EventType, m.Title, m.Transactional, m.Learner, m.Guardian, m.Registrar, true
}
