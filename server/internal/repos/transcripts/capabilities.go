package transcripts

import (
	"strings"

	"github.com/google/uuid"
)

// DeliveryMethod is how an order item is delivered to a recipient.
type DeliveryMethod string

const (
	DeliveryElectronicPESC DeliveryMethod = "electronic_pesc"
	DeliveryEDISPEEDE      DeliveryMethod = "edi_speede"
	DeliveryElectronicPDF  DeliveryMethod = "electronic_pdf"
	DeliverySecureLink     DeliveryMethod = "secure_link_email"
	DeliveryPostalMail     DeliveryMethod = "postal_mail"
	DeliveryAPIPeer        DeliveryMethod = "api_peer"
)

// AllDeliveryMethods is the closed set of delivery methods (shared with T06).
var AllDeliveryMethods = []DeliveryMethod{
	DeliveryElectronicPESC,
	DeliveryEDISPEEDE,
	DeliveryElectronicPDF,
	DeliverySecureLink,
	DeliveryPostalMail,
	DeliveryAPIPeer,
}

// RecipientType classifies a directory entry.
type RecipientType string

const (
	RecipientInstitution        RecipientType = "institution"
	RecipientApplicationService RecipientType = "application_service"
	RecipientEmployer           RecipientType = "employer"
	RecipientSelf               RecipientType = "self"
	RecipientOther              RecipientType = "other"
)

// OrderUrgency is standard vs rush (pricing refined in T05).
type OrderUrgency string

const (
	UrgencyStandard OrderUrgency = "standard"
	UrgencyRush     OrderUrgency = "rush"
)

// OrderStatus is the T03 order lifecycle state.
type OrderStatus string

const (
	OrderDraft          OrderStatus = "draft"
	OrderPendingConsent OrderStatus = "pending_consent"
	OrderPendingPayment OrderStatus = "pending_payment"
	OrderInReview       OrderStatus = "in_review"
	OrderOnHold         OrderStatus = "on_hold"
	OrderProcessing     OrderStatus = "processing"
	OrderCompleted      OrderStatus = "completed"
	OrderCanceled       OrderStatus = "canceled"
	OrderRejected       OrderStatus = "rejected"
	OrderFailed         OrderStatus = "failed" // legacy terminal
	// Legacy aliases retained for callers that still reference pre-T03 names.
	OrderQueued    OrderStatus = "in_review"
	OrderSubmitted OrderStatus = "completed"
)

// ItemStatus is per-item fulfillment state (T03/T06).
type ItemStatus string

const (
	ItemPending    ItemStatus = "pending"
	ItemReady      ItemStatus = "ready"
	ItemDelivering ItemStatus = "delivering"
	ItemDelivered  ItemStatus = "delivered"
	ItemFailed     ItemStatus = "failed"
	ItemCanceled   ItemStatus = "canceled"
)

// GlobalSelfRecipientID is the seeded "Myself" directory row.
var GlobalSelfRecipientID = uuid.MustParse("a0000000-0000-4000-8000-000000000001")

// ParseDeliveryMethod validates and normalizes a delivery method string.
func ParseDeliveryMethod(s string) (DeliveryMethod, bool) {
	m := DeliveryMethod(strings.TrimSpace(strings.ToLower(s)))
	switch m {
	case DeliveryElectronicPESC, DeliveryEDISPEEDE, DeliveryElectronicPDF, DeliverySecureLink, DeliveryPostalMail, DeliveryAPIPeer:
		return m, true
	default:
		return "", false
	}
}

// ParseRecipientType validates a recipient type string.
func ParseRecipientType(s string) (RecipientType, bool) {
	t := RecipientType(strings.TrimSpace(strings.ToLower(s)))
	switch t {
	case RecipientInstitution, RecipientApplicationService, RecipientEmployer, RecipientSelf, RecipientOther:
		return t, true
	default:
		return "", false
	}
}

// ParseOrderUrgency validates urgency; empty defaults to standard.
func ParseOrderUrgency(s string) (OrderUrgency, bool) {
	u := OrderUrgency(strings.TrimSpace(strings.ToLower(s)))
	if u == "" {
		return UrgencyStandard, true
	}
	switch u {
	case UrgencyStandard, UrgencyRush:
		return u, true
	default:
		return "", false
	}
}

// MethodAllowedByCapabilities reports whether method is in the recipient's capability set.
func MethodAllowedByCapabilities(method DeliveryMethod, capabilities []string) bool {
	for _, c := range capabilities {
		if DeliveryMethod(strings.TrimSpace(strings.ToLower(c))) == method {
			return true
		}
	}
	return false
}

// NormalizeCapabilities trims/lowercases and drops unknown values.
func NormalizeCapabilities(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}
	seen := make(map[DeliveryMethod]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, raw := range in {
		m, ok := ParseDeliveryMethod(raw)
		if !ok {
			continue
		}
		if _, dup := seen[m]; dup {
			continue
		}
		seen[m] = struct{}{}
		out = append(out, string(m))
	}
	return out
}

// NormalizeCanonicalKey lowercases and trims; empty becomes "".
func NormalizeCanonicalKey(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// CanonicalKeyFromName builds a normalized-name key when no CEEB/domain is supplied.
func CanonicalKeyFromName(name string) string {
	n := NormalizeCanonicalKey(name)
	if n == "" {
		return ""
	}
	return "name:" + n
}

// OrgEnabledDeliveryMethods returns delivery methods the institution currently supports.
// Pickup maps onto secure_link_email for the order model; webhook presence enables electronic paths.
// When delivery_v2 is on, PESC/EDI/PDF/postal adapters are available without requiring a webhook.
func OrgEnabledDeliveryMethods(cfg *Config) map[DeliveryMethod]bool {
	out := map[DeliveryMethod]bool{
		DeliverySecureLink:    true,
		DeliveryElectronicPDF: true,
		DeliveryPostalMail:    true,
	}
	hasWebhook := cfg != nil && cfg.WebhookURL != nil && strings.TrimSpace(*cfg.WebhookURL) != ""
	if hasWebhook {
		out[DeliveryAPIPeer] = true
	}
	if cfg != nil && cfg.DeliveryV2 {
		out[DeliveryElectronicPESC] = true
		out[DeliveryEDISPEEDE] = true
		out[DeliveryAPIPeer] = true
	} else if hasWebhook {
		// Pre-T06: webhook presence unlocked PESC + peer for directory orders.
		out[DeliveryElectronicPESC] = true
	}
	return out
}
