package transcripts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/models/transcriptorder"
	"github.com/lextures/lextures/server/internal/service/transcriptconsent"
	"github.com/lextures/lextures/server/internal/telemetry"
)

var (
	ErrConsentNotFound          = errors.New("consent not found")
	ErrConsentAlreadySigned     = errors.New("order already has an active consent")
	ErrConsentNotRequired       = errors.New("consent not required for this order")
	ErrConsentRevoked           = errors.New("consent is revoked")
	ErrConsentExpired           = errors.New("consent is expired")
	ErrConsentInvalidSignature  = errors.New("invalid signature")
	ErrConsentNotAgreed         = errors.New("must agree to authorize")
	ErrConsentWrongState        = errors.New("order is not awaiting consent")
	ErrConsentAlreadyDelivered  = errors.New("cannot revoke after delivery")
	ErrConsentStudentIsMinor    = errors.New("minor student cannot self-authorize; guardian required")
	ErrConsentGuardianRequired  = errors.New("guardian authorization required")
	ErrConsentNotGuardian       = errors.New("signer is not a linked guardian for this student")
	ErrConsentGateBlocked       = errors.New("consent gate not satisfied")
)

// SignerRole is who signed the FERPA release.
type SignerRole string

const (
	SignerRoleStudent  SignerRole = "student"
	SignerRoleGuardian SignerRole = "guardian"
)

// SignatureMethod is typed or drawn e-signature capture.
type SignatureMethod string

const (
	SignatureTyped SignatureMethod = "typed"
	SignatureDrawn SignatureMethod = "drawn"
)

// Consent is one append-only FERPA release authorization row.
type Consent struct {
	ID                    uuid.UUID
	OrderID               uuid.UUID
	UserID                uuid.UUID
	SignerID              uuid.UUID
	SignerRole            SignerRole
	GuardianRelationship  *string
	Recipients            json.RawMessage
	Scope                 string
	Purpose               *string
	TextVersion           string
	Locale                string
	SignatureMethod       SignatureMethod
	SignatureData         *string
	SignedIP              *string
	SignedUA              *string
	PayloadHash           string
	SignedAt              time.Time
	RevokedAt             *time.Time
	ExpiresAt             *time.Time
}

// ConsentPreview is the authorization text + scope summary shown before signing.
type ConsentPreview struct {
	OrderID           uuid.UUID
	Status            OrderStatus
	TextVersion       string
	Locale            string
	AuthorizationText string
	Scope             string
	Purpose           string
	Recipients        []transcriptconsent.RecipientSnapshot
	RequiresConsent   bool
	SelfDisclosureOnly bool
	RequiresGuardian  bool
	IsMinor           bool
	ActiveConsent     *Consent
	ConsentRequired   bool
}

// SignConsentInput captures an e-signature submission.
type SignConsentInput struct {
	OrderID           uuid.UUID
	SignerID          uuid.UUID
	SignerRole        SignerRole
	GuardianRel       *string
	Method            SignatureMethod
	SignatureData     string
	Agree             bool
	Locale            string
	Purpose           string
	IP                string
	UserAgent         string
	ExpiresAt         *time.Time
}

// OrderNeedsThirdPartyConsent reports whether any item targets a non-self recipient.
func OrderNeedsThirdPartyConsent(o *Order) bool {
	if o == nil || len(o.Items) == 0 {
		return false
	}
	for _, it := range o.Items {
		if it.Recipient == nil {
			if it.RecipientID != nil && *it.RecipientID == GlobalSelfRecipientID {
				continue
			}
			return true
		}
		if it.Recipient.Type == RecipientSelf || it.Recipient.ID == GlobalSelfRecipientID {
			continue
		}
		return true
	}
	return false
}

// OrderIsSelfDisclosureOnly reports whether every item is self-delivery.
func OrderIsSelfDisclosureOnly(o *Order) bool {
	if o == nil || len(o.Items) == 0 {
		return false
	}
	return !OrderNeedsThirdPartyConsent(o)
}

// UserIsMinor returns the COPPA/is_minor flag for a user.
func UserIsMinor(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (bool, error) {
	var isMinor bool
	err := pool.QueryRow(ctx, `SELECT COALESCE(is_minor, FALSE) FROM "user".users WHERE id = $1`, userID).Scan(&isMinor)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return isMinor, err
}

// ConsentSatisfiedForOrder evaluates whether the consent gate is open for forward progress.
func ConsentSatisfiedForOrder(ctx context.Context, pool *pgxpool.Pool, cfg *Config, o *Order) (bool, error) {
	if o == nil {
		return false, ErrOrderNotFound
	}
	consentRequired := cfg == nil || cfg.ConsentRequired
	if !consentRequired {
		return true, nil
	}
	if !OrderNeedsThirdPartyConsent(o) {
		return true, nil
	}
	isMinor, err := UserIsMinor(ctx, pool, o.UserID)
	if err != nil {
		return false, err
	}
	if isMinor {
		// Minors always need a guardian-signed active consent.
		c, err := GetActiveConsentForOrder(ctx, pool, o.ID)
		if err != nil {
			return false, err
		}
		return c != nil && c.SignerRole == SignerRoleGuardian && consentStillValid(c), nil
	}
	c, err := GetActiveConsentForOrder(ctx, pool, o.ID)
	if err != nil {
		return false, err
	}
	if c == nil {
		return false, nil
	}
	return consentStillValid(c), nil
}

func consentStillValid(c *Consent) bool {
	if c == nil || c.RevokedAt != nil {
		return false
	}
	if c.ExpiresAt != nil && !c.ExpiresAt.After(time.Now().UTC()) {
		return false
	}
	return true
}

// GetActiveConsentForOrder returns the unrevoked consent for an order, if any.
func GetActiveConsentForOrder(ctx context.Context, pool *pgxpool.Pool, orderID uuid.UUID) (*Consent, error) {
	row := pool.QueryRow(ctx, `
SELECT `+consentSelectColumns+`
FROM transcripts.consents
WHERE order_id = $1 AND revoked_at IS NULL
ORDER BY signed_at DESC
LIMIT 1
`, orderID)
	c, err := scanConsent(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

// GetConsentByID loads a consent by id.
func GetConsentByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Consent, error) {
	row := pool.QueryRow(ctx, `
SELECT `+consentSelectColumns+`
FROM transcripts.consents
WHERE id = $1
`, id)
	c, err := scanConsent(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrConsentNotFound
	}
	return c, err
}

const consentSelectColumns = `
id, order_id, user_id, signer_id, signer_role, guardian_relationship, recipients,
scope, purpose, text_version, locale, signature_method, signature_data,
HOST(signed_ip), signed_ua, payload_hash, signed_at, revoked_at, expires_at`

func scanConsent(row pgx.Row) (*Consent, error) {
	var c Consent
	var role, method string
	var recipients []byte
	var signedIP *string
	err := row.Scan(
		&c.ID, &c.OrderID, &c.UserID, &c.SignerID, &role, &c.GuardianRelationship, &recipients,
		&c.Scope, &c.Purpose, &c.TextVersion, &c.Locale, &method, &c.SignatureData,
		&signedIP, &c.SignedUA, &c.PayloadHash, &c.SignedAt, &c.RevokedAt, &c.ExpiresAt,
	)
	if err != nil {
		return nil, err
	}
	c.SignerRole = SignerRole(role)
	c.SignatureMethod = SignatureMethod(method)
	c.Recipients = json.RawMessage(recipients)
	c.SignedIP = signedIP
	return &c, nil
}

func recipientSnapshots(o *Order) []transcriptconsent.RecipientSnapshot {
	out := make([]transcriptconsent.RecipientSnapshot, 0, len(o.Items))
	seen := map[string]struct{}{}
	for _, it := range o.Items {
		var id, typ, name string
		if it.Recipient != nil {
			id = it.Recipient.ID.String()
			typ = string(it.Recipient.Type)
			name = it.Recipient.Name
		} else if it.RecipientID != nil {
			id = it.RecipientID.String()
			typ = "unknown"
			name = id
		} else {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, transcriptconsent.RecipientSnapshot{ID: id, Type: typ, Name: name})
	}
	return out
}

// BuildConsentPreview assembles the authorization text and recipient/scope summary.
func BuildConsentPreview(ctx context.Context, pool *pgxpool.Pool, cfg *Config, o *Order, locale string) (*ConsentPreview, error) {
	if o == nil {
		return nil, ErrOrderNotFound
	}
	loc := strings.TrimSpace(locale)
	if loc == "" {
		loc = "en"
	}
	text, err := transcriptconsent.AuthorizationText(transcriptconsent.CurrentTextVersion, loc)
	if err != nil {
		return nil, err
	}
	isMinor, err := UserIsMinor(ctx, pool, o.UserID)
	if err != nil {
		return nil, err
	}
	consentRequired := cfg == nil || cfg.ConsentRequired
	needsThird := OrderNeedsThirdPartyConsent(o)
	active, err := GetActiveConsentForOrder(ctx, pool, o.ID)
	if err != nil {
		return nil, err
	}
	purpose := transcriptconsent.PurposeTranscriptRelease
	return &ConsentPreview{
		OrderID:            o.ID,
		Status:             o.Status,
		TextVersion:        transcriptconsent.CurrentTextVersion,
		Locale:             loc,
		AuthorizationText:  text,
		Scope:              transcriptconsent.ScopeFullAcademicRecord,
		Purpose:            purpose,
		Recipients:         recipientSnapshots(o),
		RequiresConsent:    consentRequired && needsThird,
		SelfDisclosureOnly: OrderIsSelfDisclosureOnly(o),
		RequiresGuardian:   consentRequired && needsThird && isMinor,
		IsMinor:            isMinor,
		ActiveConsent:      active,
		ConsentRequired:    consentRequired,
	}, nil
}

func validateSignature(method SignatureMethod, data string) error {
	s := strings.TrimSpace(data)
	if s == "" {
		return ErrConsentInvalidSignature
	}
	switch method {
	case SignatureTyped:
		if len([]rune(s)) < 2 || len(s) > 200 {
			return ErrConsentInvalidSignature
		}
		return nil
	case SignatureDrawn:
		if !strings.HasPrefix(s, "data:image/") || !strings.Contains(s, ";base64,") {
			return ErrConsentInvalidSignature
		}
		if len(s) > 512*1024 {
			return ErrConsentInvalidSignature
		}
		return nil
	default:
		return ErrConsentInvalidSignature
	}
}

// SignConsent creates an immutable authorization and advances the order past pending_consent.
func SignConsent(ctx context.Context, pool *pgxpool.Pool, cfg *Config, in SignConsentInput) (*Consent, *Order, error) {
	if !in.Agree {
		return nil, nil, ErrConsentNotAgreed
	}
	if err := validateSignature(in.Method, in.SignatureData); err != nil {
		return nil, nil, err
	}
	o, err := GetOrderByID(ctx, pool, in.OrderID)
	if err != nil {
		return nil, nil, err
	}
	if o.Status != OrderDraft && o.Status != OrderPendingConsent {
		return nil, nil, ErrConsentWrongState
	}
	if len(o.Items) == 0 {
		return nil, nil, ErrOrderEmpty
	}

	preview, err := BuildConsentPreview(ctx, pool, cfg, o, in.Locale)
	if err != nil {
		return nil, nil, err
	}
	if !preview.RequiresConsent {
		return nil, nil, ErrConsentNotRequired
	}
	if preview.ActiveConsent != nil && consentStillValid(preview.ActiveConsent) {
		return nil, nil, ErrConsentAlreadySigned
	}

	switch in.SignerRole {
	case SignerRoleStudent:
		if in.SignerID != o.UserID {
			return nil, nil, ErrOrderNotFound
		}
		if preview.RequiresGuardian {
			return nil, nil, ErrConsentStudentIsMinor
		}
	case SignerRoleGuardian:
		if !preview.IsMinor {
			return nil, nil, ErrConsentGuardianRequired
		}
		if strings.TrimSpace(ptrStr(in.GuardianRel)) == "" {
			rel := "guardian"
			in.GuardianRel = &rel
		}
	default:
		return nil, nil, ErrConsentInvalidSignature
	}

	purpose := strings.TrimSpace(in.Purpose)
	if purpose == "" {
		purpose = transcriptconsent.PurposeTranscriptRelease
	}
	locale := preview.Locale
	if strings.TrimSpace(in.Locale) != "" {
		locale = strings.TrimSpace(in.Locale)
	}
	hash, err := transcriptconsent.HashPayload(transcriptconsent.Payload{
		OrderID:     o.ID.String(),
		UserID:      o.UserID.String(),
		SignerID:    in.SignerID.String(),
		SignerRole:  string(in.SignerRole),
		Recipients:  preview.Recipients,
		Scope:       transcriptconsent.ScopeFullAcademicRecord,
		Purpose:     purpose,
		TextVersion: transcriptconsent.CurrentTextVersion,
		Locale:      locale,
		Agree:       true,
	})
	if err != nil {
		return nil, nil, err
	}
	recipientsJSON, err := json.Marshal(preview.Recipients)
	if err != nil {
		return nil, nil, err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var locked Order
	if err := scanOrder(tx.QueryRow(ctx, `
SELECT `+orderSelectColumns+`
FROM transcripts.orders
WHERE id = $1
FOR UPDATE
`, in.OrderID), &locked); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrOrderNotFound
		}
		return nil, nil, err
	}
	if locked.Status != OrderDraft && locked.Status != OrderPendingConsent {
		return nil, nil, ErrConsentWrongState
	}

	var signedIP *string
	if ip := strings.TrimSpace(in.IP); ip != "" {
		if parsed := net.ParseIP(ip); parsed != nil {
			s := parsed.String()
			signedIP = &s
		}
	}
	var ua *string
	if s := strings.TrimSpace(in.UserAgent); s != "" {
		if len(s) > 512 {
			s = s[:512]
		}
		ua = &s
	}
	sigData := strings.TrimSpace(in.SignatureData)

	var c Consent
	row := tx.QueryRow(ctx, `
INSERT INTO transcripts.consents (
    order_id, user_id, signer_id, signer_role, guardian_relationship, recipients,
    scope, purpose, text_version, locale, signature_method, signature_data,
    signed_ip, signed_ua, payload_hash, expires_at
)
VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7,$8,$9,$10,$11,$12,$13::inet,$14,$15,$16)
RETURNING `+consentSelectColumns+`
`, locked.ID, locked.UserID, in.SignerID, string(in.SignerRole), in.GuardianRel, recipientsJSON,
		transcriptconsent.ScopeFullAcademicRecord, purpose, transcriptconsent.CurrentTextVersion, locale,
		string(in.Method), sigData, signedIP, ua, hash, in.ExpiresAt)
	scanned, err := scanConsent(row)
	if err != nil {
		return nil, nil, err
	}
	c = *scanned

	if _, err := tx.Exec(ctx, `
UPDATE transcripts.orders SET consent_id = $2 WHERE id = $1
`, locked.ID, c.ID); err != nil {
		return nil, nil, err
	}

	// Advance draft → pending_consent first if needed, then past the gate.
	from := locked.Status
	if from == OrderDraft {
		reason := "submitted pending consent advance"
		if _, err := tx.Exec(ctx, `
UPDATE transcripts.orders
SET status = $2, submitted_at = COALESCE(submitted_at, NOW())
WHERE id = $1
`, locked.ID, string(OrderPendingConsent)); err != nil {
			return nil, nil, err
		}
		fromStr := string(OrderDraft)
		toStr := string(OrderPendingConsent)
		if _, err := tx.Exec(ctx, `
INSERT INTO transcripts.order_events (order_id, from_state, to_state, actor_id, reason)
VALUES ($1, $2, $3, $4, $5)
`, locked.ID, fromStr, toStr, in.SignerID, reason); err != nil {
			return nil, nil, err
		}
		from = OrderPendingConsent
	}

	auto := cfg != nil && cfg.AutoApprovalEnabled
	blocked, err := HasBlockingHold(ctx, pool, locked.UserID, locked.OrgID)
	if err != nil {
		return nil, nil, err
	}
	gates := transcriptorder.GateContext{
		ConsentSatisfied: true,
		PaymentSatisfied: true, // T05
		HasBlockingHold:  blocked,
		AutoApproval:     auto,
	}
	target := transcriptorder.ResolveSubmitTarget(gates)
	// Already past consent; ResolveSubmitTarget with ConsentSatisfied true never returns pending_consent
	// unless somehow — but if blocked it returns on_hold. If it returned pending_consent we'd loop.
	if target == transcriptorder.OrderPendingConsent {
		target = transcriptorder.OrderInReview
	}
	to := OrderStatus(target)
	if from != to {
		if err := transcriptorder.ValidateOrderTransition(
			transcriptorder.OrderStatus(from),
			transcriptorder.OrderStatus(to),
		); err != nil {
			return nil, nil, err
		}
		reason := "consent signed"
		if gates.HasBlockingHold {
			reason = "consent signed; blocked by active hold"
		} else if auto && to == OrderProcessing {
			reason = "consent signed; auto-approved"
		}
		if _, err := tx.Exec(ctx, `
UPDATE transcripts.orders SET status = $2 WHERE id = $1
`, locked.ID, string(to)); err != nil {
			return nil, nil, err
		}
		fromStr := string(from)
		if _, err := tx.Exec(ctx, `
INSERT INTO transcripts.order_events (order_id, from_state, to_state, actor_id, reason)
VALUES ($1, $2, $3, $4, $5)
`, locked.ID, fromStr, string(to), in.SignerID, reason); err != nil {
			return nil, nil, err
		}
		if to == OrderProcessing {
			if err := markItemsStatusTx(ctx, tx, locked.ID, &in.SignerID, ItemPending, ItemReady, "ready for delivery"); err != nil {
				return nil, nil, err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}
	telemetry.RecordBusinessEvent("transcript_consent_signed")
	outOrder, err := GetOrderByID(ctx, pool, locked.ID)
	if err != nil {
		return &c, nil, err
	}
	return &c, outOrder, nil
}

// RevokeConsent marks the active authorization revoked and blocks undelivered items.
func RevokeConsent(ctx context.Context, pool *pgxpool.Pool, orderID, actorID uuid.UUID) (*Consent, *Order, error) {
	o, err := GetOrderByID(ctx, pool, orderID)
	if err != nil {
		return nil, nil, err
	}
	if o.UserID != actorID {
		// Only the student (record subject) may revoke; guardian revoke can be added later.
		return nil, nil, ErrOrderNotFound
	}
	c, err := GetActiveConsentForOrder(ctx, pool, orderID)
	if err != nil {
		return nil, nil, err
	}
	if c == nil {
		return nil, nil, ErrConsentNotFound
	}

	// Block revoke once any item is delivered (or delivering).
	for _, it := range o.Items {
		if it.Status == ItemDelivered || it.Status == ItemDelivering {
			return nil, nil, ErrConsentAlreadyDelivered
		}
	}
	if o.Status == OrderCompleted {
		return nil, nil, ErrConsentAlreadyDelivered
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
UPDATE transcripts.consents
SET revoked_at = NOW()
WHERE id = $1 AND revoked_at IS NULL
`, c.ID)
	if err != nil {
		return nil, nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, nil, ErrConsentRevoked
	}
	if _, err := tx.Exec(ctx, `
UPDATE transcripts.orders SET consent_id = NULL WHERE id = $1
`, orderID); err != nil {
		return nil, nil, err
	}

	from := o.Status
	to := OrderPendingConsent
	if from != OrderPendingConsent && from != OrderCanceled && from != OrderRejected && from != OrderFailed {
		if err := transcriptorder.ValidateOrderTransition(
			transcriptorder.OrderStatus(from),
			transcriptorder.OrderStatus(to),
		); err != nil {
			// From processing/in_review/etc. — allow cancel of open items and force pending_consent
			// by adding edges or canceling. Prefer moving via cancel of items + status update when edge missing.
			// pending_consent is reachable from pending_payment/in_review/on_hold? Looking at matrix:
			// pending_payment → in_review/on_hold/canceled/rejected — NOT back to pending_consent
			// in_review → on_hold/processing/rejected/canceled — NOT pending_consent
			// So we need to add edges OR set status directly for revoke. Plan: "block undelivered items".
			// Add legal edges for revoke regression in state.go.
			return nil, nil, err
		}
		reason := "consent revoked"
		if _, err := tx.Exec(ctx, `
UPDATE transcripts.orders SET status = $2 WHERE id = $1
`, orderID, string(to)); err != nil {
			return nil, nil, err
		}
		fromStr := string(from)
		if _, err := tx.Exec(ctx, `
INSERT INTO transcripts.order_events (order_id, from_state, to_state, actor_id, reason)
VALUES ($1, $2, $3, $4, $5)
`, orderID, fromStr, string(to), actorID, reason); err != nil {
			return nil, nil, err
		}
	}

	// Block items that were ready to deliver; leave pending items for a fresh signature.
	if err := cancelReadyItemsTx(ctx, tx, orderID, &actorID, "blocked by consent revocation"); err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}
	telemetry.RecordBusinessEvent("transcript_consent_revoked")
	revoked, err := GetConsentByID(ctx, pool, c.ID)
	if err != nil {
		return nil, nil, err
	}
	outOrder, err := GetOrderByID(ctx, pool, orderID)
	if err != nil {
		return revoked, nil, err
	}
	return revoked, outOrder, nil
}

func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func cancelReadyItemsTx(ctx context.Context, tx pgx.Tx, orderID uuid.UUID, actorID *uuid.UUID, reason string) error {
	rows, err := tx.Query(ctx, `
SELECT id, status FROM transcripts.order_items
WHERE order_id = $1 AND status IN ('ready', 'delivering')
`, orderID)
	if err != nil {
		return err
	}
	defer rows.Close()
	type row struct {
		id uuid.UUID
		st ItemStatus
	}
	var list []row
	for rows.Next() {
		var r row
		var st string
		if err := rows.Scan(&r.id, &st); err != nil {
			return err
		}
		r.st = ItemStatus(st)
		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, it := range list {
		to := ItemCanceled
		if err := transcriptorder.ValidateItemTransition(
			transcriptorder.ItemStatus(it.st),
			transcriptorder.ItemStatus(to),
		); err != nil {
			continue
		}
		if _, err := tx.Exec(ctx, `
UPDATE transcripts.order_items SET status = $2 WHERE id = $1
`, it.id, string(to)); err != nil {
			return err
		}
		fromStr := string(it.st)
		if _, err := tx.Exec(ctx, `
INSERT INTO transcripts.order_events (order_id, item_id, from_state, to_state, actor_id, reason)
VALUES ($1, $2, $3, $4, $5, $6)
`, orderID, it.id, fromStr, string(to), actorID, reason); err != nil {
			return err
		}
	}
	return nil
}

// ExportConsentJSON builds an audit export of the consent record + text signed.
func ExportConsentJSON(ctx context.Context, pool *pgxpool.Pool, orderID, userID uuid.UUID) (map[string]any, error) {
	o, err := GetOrderForUser(ctx, pool, orderID, userID)
	if err != nil {
		return nil, err
	}
	var c *Consent
	if o.ConsentID != nil {
		c, err = GetConsentByID(ctx, pool, *o.ConsentID)
		if err != nil {
			return nil, err
		}
	} else {
		c, err = GetActiveConsentForOrder(ctx, pool, orderID)
		if err != nil {
			return nil, err
		}
		if c == nil {
			// Fall back to most recent including revoked.
			row := pool.QueryRow(ctx, `
SELECT `+consentSelectColumns+`
FROM transcripts.consents
WHERE order_id = $1
ORDER BY signed_at DESC
LIMIT 1
`, orderID)
			c, err = scanConsent(row)
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrConsentNotFound
			}
			if err != nil {
				return nil, err
			}
		}
	}
	text, err := transcriptconsent.AuthorizationText(c.TextVersion, c.Locale)
	if err != nil {
		text = ""
	}
	var recipients any
	_ = json.Unmarshal(c.Recipients, &recipients)
	out := map[string]any{
		"consentId":       c.ID.String(),
		"orderId":         c.OrderID.String(),
		"userId":          c.UserID.String(),
		"signerId":        c.SignerID.String(),
		"signerRole":      string(c.SignerRole),
		"recipients":      recipients,
		"scope":           c.Scope,
		"purpose":         c.Purpose,
		"textVersion":     c.TextVersion,
		"locale":          c.Locale,
		"authorizationText": text,
		"signatureMethod": string(c.SignatureMethod),
		"signatureData":   c.SignatureData,
		"payloadHash":     c.PayloadHash,
		"signedAt":        c.SignedAt.UTC().Format(time.RFC3339),
	}
	if c.GuardianRelationship != nil {
		out["guardianRelationship"] = *c.GuardianRelationship
	}
	if c.SignedIP != nil {
		out["signedIp"] = *c.SignedIP
	}
	if c.SignedUA != nil {
		out["signedUserAgent"] = *c.SignedUA
	}
	if c.RevokedAt != nil {
		out["revokedAt"] = c.RevokedAt.UTC().Format(time.RFC3339)
	}
	if c.ExpiresAt != nil {
		out["expiresAt"] = c.ExpiresAt.UTC().Format(time.RFC3339)
	}
	return out, nil
}

// ExportConsentPDFBytes renders a simple text PDF-like audit document (plain UTF-8 PDF).
func ExportConsentPDFBytes(export map[string]any) []byte {
	text, _ := export["authorizationText"].(string)
	version, _ := export["textVersion"].(string)
	signedAt, _ := export["signedAt"].(string)
	signerRole, _ := export["signerRole"].(string)
	hash, _ := export["payloadHash"].(string)
	orderID, _ := export["orderId"].(string)
	body := fmt.Sprintf(
		"Lextures FERPA Consent Audit Export\n\nOrder: %s\nText version: %s\nSigner role: %s\nSigned at: %s\nPayload hash: %s\n\n%s\n",
		orderID, version, signerRole, signedAt, hash, text,
	)
	return buildMinimalPDF(body)
}

func buildMinimalPDF(text string) []byte {
	// Minimal single-page PDF with Helvetica text (escape parentheses).
	escaped := strings.ReplaceAll(text, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `(`, `\(`)
	escaped = strings.ReplaceAll(escaped, `)`, `\)`)
	escaped = strings.ReplaceAll(escaped, "\r", "")
	lines := strings.Split(escaped, "\n")
	var content strings.Builder
	content.WriteString("BT /F1 10 Tf 40 750 Td 12 TL\n")
	for i, line := range lines {
		if i == 0 {
			content.WriteString(fmt.Sprintf("(%s) Tj\n", line))
		} else {
			content.WriteString(fmt.Sprintf("T* (%s) Tj\n", line))
		}
		if i > 55 {
			break
		}
	}
	content.WriteString("ET")
	stream := content.String()
	objects := []string{
		"1 0 obj<< /Type /Catalog /Pages 2 0 R >>endobj\n",
		"2 0 obj<< /Type /Pages /Kids [3 0 R] /Count 1 >>endobj\n",
		"3 0 obj<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources<< /Font<< /F1 5 0 R >> >> >>endobj\n",
		fmt.Sprintf("4 0 obj<< /Length %d >>stream\n%s\nendstream\nendobj\n", len(stream), stream),
		"5 0 obj<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>endobj\n",
	}
	var buf strings.Builder
	buf.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for i, obj := range objects {
		offsets[i+1] = buf.Len()
		buf.WriteString(obj)
	}
	xrefStart := buf.Len()
	buf.WriteString(fmt.Sprintf("xref\n0 %d\n", len(objects)+1))
	buf.WriteString("0000000000 65535 f \n")
	for i := 1; i <= len(objects); i++ {
		buf.WriteString(fmt.Sprintf("%010d 00000 n \n", offsets[i]))
	}
	buf.WriteString(fmt.Sprintf("trailer<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xrefStart))
	return []byte(buf.String())
}

// LogSelfDisclosureIfNeeded records a FERPA disclosure when submitting a self-only order.
func LogSelfDisclosureIfNeeded(ctx context.Context, pool *pgxpool.Pool, o *Order, actorID uuid.UUID) error {
	if o == nil || !OrderIsSelfDisclosureOnly(o) {
		return nil
	}
	orgID := uuid.Nil
	if o.OrgID != nil {
		orgID = *o.OrgID
	}
	recipient := "self"
	// Best-effort; ignore missing org.
	if orgID == uuid.Nil {
		return nil
	}
	return insertSelfDisclosure(ctx, pool, orgID, actorID, o.UserID, &recipient)
}

func insertSelfDisclosure(ctx context.Context, pool *pgxpool.Pool, orgID, accessorID, studentID uuid.UUID, recipient *string) error {
	_, err := pool.Exec(ctx, `
INSERT INTO compliance.ferpa_disclosure_log (org_id, accessor_id, student_id, data_type, authority_claim, recipient)
VALUES ($1, $2, $3, 'transcript_self_disclosure', 'student', $4)
`, orgID, accessorID, studentID, recipient)
	return err
}
