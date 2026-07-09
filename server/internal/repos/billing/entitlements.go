// Package billing persists Stripe-derived learner entitlements (plan 15.3).
package billing

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	TypeCoursePurchase      = "course_purchase"
	TypeSubscriptionMonthly = "subscription_monthly"
	TypeSubscriptionAnnual  = "subscription_annual"

	StatusActive   = "active"
	StatusExpired  = "expired"
	StatusRefunded = "refunded"

	// Acquisition sources for course_purchase entitlements (plan MKT1).
	AcquisitionStripe = "stripe"
	AcquisitionFree   = "free"
	AcquisitionComp   = "comp"
)

// Entitlement is a row in billing.user_entitlements.
type Entitlement struct {
	ID                uuid.UUID
	UserID            uuid.UUID
	EntitlementType   string
	CourseID          *uuid.UUID
	StripeEventID     string
	StripeInvoiceID   *string
	AmountPaidCents   int
	SubtotalCents     int
	TaxAmountCents    int
	TaxType           string
	TaxJurisdiction   string
	ReverseCharge     bool
	InvoiceID         *uuid.UUID
	Currency          string
	ValidFrom         time.Time
	ValidUntil        *time.Time
	Status            string
	AcquisitionSource string
	CreatedAt         time.Time
}

// CreateInput is the payload for idempotent entitlement creation.
type CreateInput struct {
	UserID          uuid.UUID
	EntitlementType string
	CourseID        *uuid.UUID
	StripeEventID   string
	StripeInvoiceID *string
	AmountPaidCents int
	Currency        string
	ValidUntil      *time.Time
}

// CourseGrantInput is the payload for free/comp/stripe course_purchase grants (plan MKT1).
// Free claims may omit StripeEventID; grants are idempotent per (user_id, course_id).
type CourseGrantInput struct {
	UserID            uuid.UUID
	CourseID          uuid.UUID
	AcquisitionSource string // stripe | free | comp
	AmountPaidCents   int
	Currency          string
	StripeEventID     *string
	StripeInvoiceID   *string
	ValidUntil        *time.Time
}

// CreateIdempotent inserts an entitlement or returns the existing row for stripe_event_id.
func CreateIdempotent(ctx context.Context, pool *pgxpool.Pool, in CreateInput) (*Entitlement, bool, error) {
	if in.StripeEventID == "" {
		return nil, false, errors.New("stripe_event_id required")
	}
	currency := in.Currency
	if currency == "" {
		currency = "usd"
	}
	e, err := scanEntitlement(ctx, pool, `
INSERT INTO billing.user_entitlements (
    user_id, entitlement_type, course_id, stripe_event_id, stripe_invoice_id,
    amount_paid_cents, currency, valid_until, status, acquisition_source
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'active', 'stripe')
ON CONFLICT (stripe_event_id) DO NOTHING
RETURNING id, user_id, entitlement_type, course_id, COALESCE(stripe_event_id, ''), stripe_invoice_id,
          amount_paid_cents, currency, valid_from, valid_until, status,
          COALESCE(acquisition_source, 'stripe'), created_at
`, in.UserID, in.EntitlementType, in.CourseID, in.StripeEventID, in.StripeInvoiceID,
		in.AmountPaidCents, currency, in.ValidUntil)
	if err == nil {
		return e, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, err
	}
	e, err = scanEntitlement(ctx, pool, `
SELECT id, user_id, entitlement_type, course_id, COALESCE(stripe_event_id, ''), stripe_invoice_id,
       amount_paid_cents, currency, valid_from, valid_until, status,
       COALESCE(acquisition_source, 'stripe'), created_at
FROM billing.user_entitlements WHERE stripe_event_id = $1
`, in.StripeEventID)
	if err != nil {
		return nil, false, err
	}
	return e, false, nil
}

// CreateCourseGrantIdempotent creates a course_purchase entitlement without requiring a Stripe
// event (free claims / comps). Idempotent under the partial unique index on
// (user_id, course_id) WHERE entitlement_type = 'course_purchase' AND status = 'active'.
func CreateCourseGrantIdempotent(ctx context.Context, pool *pgxpool.Pool, in CourseGrantInput) (*Entitlement, bool, error) {
	src := in.AcquisitionSource
	if src == "" {
		if in.StripeEventID != nil && *in.StripeEventID != "" {
			src = AcquisitionStripe
		} else {
			src = AcquisitionFree
		}
	}
	switch src {
	case AcquisitionStripe, AcquisitionFree, AcquisitionComp:
	default:
		return nil, false, errors.New("invalid acquisition_source")
	}
	currency := in.Currency
	if currency == "" {
		currency = "usd"
	}
	var stripeEvent any
	if in.StripeEventID != nil && *in.StripeEventID != "" {
		stripeEvent = *in.StripeEventID
	}
	e, err := scanEntitlement(ctx, pool, `
INSERT INTO billing.user_entitlements (
    user_id, entitlement_type, course_id, stripe_event_id, stripe_invoice_id,
    amount_paid_cents, currency, valid_until, status, acquisition_source
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'active', $9)
ON CONFLICT (user_id, course_id) WHERE entitlement_type = 'course_purchase' AND status = 'active'
DO NOTHING
RETURNING id, user_id, entitlement_type, course_id, COALESCE(stripe_event_id, ''), stripe_invoice_id,
          amount_paid_cents, currency, valid_from, valid_until, status,
          COALESCE(acquisition_source, 'stripe'), created_at
`, in.UserID, TypeCoursePurchase, in.CourseID, stripeEvent, in.StripeInvoiceID,
		in.AmountPaidCents, currency, in.ValidUntil, src)
	if err == nil {
		return e, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, err
	}
	e, err = scanEntitlement(ctx, pool, `
SELECT id, user_id, entitlement_type, course_id, COALESCE(stripe_event_id, ''), stripe_invoice_id,
       amount_paid_cents, currency, valid_from, valid_until, status,
       COALESCE(acquisition_source, 'stripe'), created_at
FROM billing.user_entitlements
WHERE user_id = $1 AND course_id = $2
  AND entitlement_type = 'course_purchase' AND status = 'active'
ORDER BY created_at DESC
LIMIT 1
`, in.UserID, in.CourseID)
	if err != nil {
		return nil, false, err
	}
	return e, false, nil
}

func scanEntitlement(ctx context.Context, pool *pgxpool.Pool, query string, args ...any) (*Entitlement, error) {
	var e Entitlement
	var courseID *uuid.UUID
	err := pool.QueryRow(ctx, query, args...).Scan(
		&e.ID, &e.UserID, &e.EntitlementType, &courseID, &e.StripeEventID, &e.StripeInvoiceID,
		&e.AmountPaidCents, &e.Currency, &e.ValidFrom, &e.ValidUntil, &e.Status,
		&e.AcquisitionSource, &e.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	e.CourseID = courseID
	return &e, nil
}

// ListActiveByUser returns active entitlements for a user.
func ListActiveByUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]Entitlement, error) {
	rows, err := pool.Query(ctx, `
SELECT id, user_id, entitlement_type, course_id, COALESCE(stripe_event_id, ''), stripe_invoice_id,
       amount_paid_cents, subtotal_cents, tax_amount_cents, tax_type,
       COALESCE(tax_jurisdiction, ''), reverse_charge, invoice_id,
       currency, valid_from, valid_until, status, COALESCE(acquisition_source, 'stripe'), created_at
FROM billing.user_entitlements
WHERE user_id = $1
  AND status = 'active'
  AND (valid_until IS NULL OR valid_until > NOW())
ORDER BY created_at DESC
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Entitlement
	for rows.Next() {
		var e Entitlement
		var courseID *uuid.UUID
		if err := rows.Scan(
			&e.ID, &e.UserID, &e.EntitlementType, &courseID, &e.StripeEventID, &e.StripeInvoiceID,
			&e.AmountPaidCents, &e.SubtotalCents, &e.TaxAmountCents, &e.TaxType, &e.TaxJurisdiction,
			&e.ReverseCharge, &e.InvoiceID, &e.Currency, &e.ValidFrom, &e.ValidUntil, &e.Status,
			&e.AcquisitionSource, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		e.CourseID = courseID
		out = append(out, e)
	}
	return out, rows.Err()
}

// HasCourseAccess reports whether the user may access a paid course.
func HasCourseAccess(ctx context.Context, pool *pgxpool.Pool, userID, courseID uuid.UUID) (bool, error) {
	var priceCents int
	err := pool.QueryRow(ctx, `SELECT price_cents FROM course.courses WHERE id = $1`, courseID).Scan(&priceCents)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if priceCents <= 0 {
		return true, nil
	}
	return MarketplaceAccess(ctx, pool, userID, courseID)
}

// MarketplaceAccess reports whether the user owns/has claimed a course via an active
// course_purchase entitlement or subscription (plan MKT1 FR-7). Unlike HasCourseAccess,
// free list price alone does not grant access — a free claim must be recorded.
func MarketplaceAccess(ctx context.Context, pool *pgxpool.Pool, userID, courseID uuid.UUID) (bool, error) {
	var ok bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM billing.user_entitlements e
  WHERE e.user_id = $1
    AND e.status = 'active'
    AND (e.valid_until IS NULL OR e.valid_until > NOW())
    AND (
      (e.entitlement_type = 'course_purchase' AND e.course_id = $2)
      OR e.entitlement_type LIKE 'subscription%'
    )
)
`, userID, courseID).Scan(&ok)
	return ok, err
}

// PurchasedCourseMap returns acquisition_source for each courseID the user acquired via an
// active course_purchase entitlement (plan MKT5). Refunded/expired rows are excluded.
// Subscription-only access does not count as a marketplace purchase indicator.
// On empty input returns an empty map (not nil) so callers can range safely.
func PurchasedCourseMap(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, courseIDs []uuid.UUID) (map[uuid.UUID]string, error) {
	out := make(map[uuid.UUID]string, len(courseIDs))
	if len(courseIDs) == 0 {
		return out, nil
	}
	rows, err := pool.Query(ctx, `
SELECT DISTINCT ON (e.course_id) e.course_id, COALESCE(e.acquisition_source, 'stripe')
FROM billing.user_entitlements e
WHERE e.user_id = $1
  AND e.status = 'active'
  AND (e.valid_until IS NULL OR e.valid_until > NOW())
  AND e.entitlement_type = 'course_purchase'
  AND e.course_id = ANY($2::uuid[])
ORDER BY e.course_id, e.created_at DESC
`, userID, courseIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var src string
		if err := rows.Scan(&id, &src); err != nil {
			return nil, err
		}
		out[id] = src
	}
	return out, rows.Err()
}

// CoursePurchase is a marketplace acquisition with course metadata for "My purchases" (plan MKT5).
type CoursePurchase struct {
	EntitlementID     uuid.UUID
	CourseID          uuid.UUID
	CourseCode        string
	Title             string
	AmountPaidCents   int
	Currency          string
	AcquisitionSource string
	AcquiredAt        time.Time
	HasReceipt        bool // true when amount > 0 (paid) — client links to /me/billing
}

// ListMyPurchases returns active course_purchase entitlements for the user, newest first.
func ListMyPurchases(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]CoursePurchase, error) {
	rows, err := pool.Query(ctx, `
SELECT e.id, e.course_id, c.course_code, c.title,
       e.amount_paid_cents, e.currency, COALESCE(e.acquisition_source, 'stripe'), e.created_at
FROM billing.user_entitlements e
JOIN course.courses c ON c.id = e.course_id
WHERE e.user_id = $1
  AND e.status = 'active'
  AND (e.valid_until IS NULL OR e.valid_until > NOW())
  AND e.entitlement_type = 'course_purchase'
  AND e.course_id IS NOT NULL
ORDER BY e.created_at DESC
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CoursePurchase
	for rows.Next() {
		var p CoursePurchase
		if err := rows.Scan(
			&p.EntitlementID, &p.CourseID, &p.CourseCode, &p.Title,
			&p.AmountPaidCents, &p.Currency, &p.AcquisitionSource, &p.AcquiredAt,
		); err != nil {
			return nil, err
		}
		p.HasReceipt = p.AmountPaidCents > 0 || p.AcquisitionSource == AcquisitionStripe
		out = append(out, p)
	}
	if out == nil {
		out = []CoursePurchase{}
	}
	return out, rows.Err()
}

// OwnedCourseIDs returns the subset of courseIDs the user owns via marketplace access
// (active course_purchase for that course, or any active subscription). Used by MKT3
// to overlay `owned` on a cached storefront page in one round-trip.
// On empty input returns an empty set (not nil) so callers can range safely.
func OwnedCourseIDs(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, courseIDs []uuid.UUID) (map[uuid.UUID]struct{}, error) {
	out := make(map[uuid.UUID]struct{}, len(courseIDs))
	if len(courseIDs) == 0 {
		return out, nil
	}

	var hasSub bool
	if err := pool.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM billing.user_entitlements e
  WHERE e.user_id = $1
    AND e.status = 'active'
    AND (e.valid_until IS NULL OR e.valid_until > NOW())
    AND e.entitlement_type LIKE 'subscription%'
)
`, userID).Scan(&hasSub); err != nil {
		return nil, err
	}
	if hasSub {
		for _, id := range courseIDs {
			out[id] = struct{}{}
		}
		return out, nil
	}

	rows, err := pool.Query(ctx, `
SELECT e.course_id
FROM billing.user_entitlements e
WHERE e.user_id = $1
  AND e.status = 'active'
  AND (e.valid_until IS NULL OR e.valid_until > NOW())
  AND e.entitlement_type = 'course_purchase'
  AND e.course_id = ANY($2::uuid[])
`, userID, courseIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = struct{}{}
	}
	return out, rows.Err()
}

// RefundCourseEntitlement marks the user's active course_purchase for a course as refunded
// (plan MKT4 FR-8). Returns true when a row was updated. Does not unenroll — access is
// revoked via status='refunded' while the enrollment row is retained by default.
func RefundCourseEntitlement(ctx context.Context, pool *pgxpool.Pool, userID, courseID uuid.UUID) (bool, error) {
	tag, err := pool.Exec(ctx, `
UPDATE billing.user_entitlements
SET status = 'refunded'
WHERE user_id = $1
  AND course_id = $2
  AND entitlement_type = 'course_purchase'
  AND status = 'active'
`, userID, courseID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// ActiveCoursePurchase returns the user's active course_purchase entitlement for a course, if any.
func ActiveCoursePurchase(ctx context.Context, pool *pgxpool.Pool, userID, courseID uuid.UUID) (*Entitlement, error) {
	e, err := scanEntitlement(ctx, pool, `
SELECT id, user_id, entitlement_type, course_id, COALESCE(stripe_event_id, ''), stripe_invoice_id,
       amount_paid_cents, currency, valid_from, valid_until, status,
       COALESCE(acquisition_source, 'stripe'), created_at
FROM billing.user_entitlements
WHERE user_id = $1 AND course_id = $2
  AND entitlement_type = 'course_purchase' AND status = 'active'
  AND (valid_until IS NULL OR valid_until > NOW())
ORDER BY created_at DESC
LIMIT 1
`, userID, courseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return e, err
}

// ExpireActiveSubscriptions marks subscription entitlements expired for a user.
func ExpireActiveSubscriptions(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (int64, error) {
	tag, err := pool.Exec(ctx, `
UPDATE billing.user_entitlements
SET status = 'expired'
WHERE user_id = $1
  AND entitlement_type LIKE 'subscription%'
  AND status = 'active'
`, userID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// StripeCustomerID returns the stored Stripe customer id, if any.
func StripeCustomerID(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (string, error) {
	var id *string
	err := pool.QueryRow(ctx, `
SELECT stripe_customer_id FROM "user".users WHERE id = $1
`, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if id == nil {
		return "", nil
	}
	return *id, nil
}

// SetStripeCustomerID stores the Stripe customer id for a user.
func SetStripeCustomerID(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, customerID string) error {
	_, err := pool.Exec(ctx, `
UPDATE "user".users SET stripe_customer_id = $2 WHERE id = $1
`, userID, customerID)
	return err
}

// CoursePrice loads list price for checkout.
type CoursePrice struct {
	ID         uuid.UUID
	Title      string
	PriceCents int
	Currency   string
	OrgID      uuid.UUID
}

// CoursePriceByID returns pricing metadata for a course.
func CoursePriceByID(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (*CoursePrice, error) {
	var p CoursePrice
	err := pool.QueryRow(ctx, `
SELECT id, title, price_cents, price_currency, org_id
FROM course.courses WHERE id = $1
`, courseID).Scan(&p.ID, &p.Title, &p.PriceCents, &p.Currency, &p.OrgID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}
