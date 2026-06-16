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
)

// Entitlement is a row in billing.user_entitlements.
type Entitlement struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	EntitlementType string
	CourseID        *uuid.UUID
	StripeEventID   string
	StripeInvoiceID *string
	AmountPaidCents int
	Currency        string
	ValidFrom       time.Time
	ValidUntil      *time.Time
	Status          string
	CreatedAt       time.Time
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
    amount_paid_cents, currency, valid_until, status
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'active')
ON CONFLICT (stripe_event_id) DO NOTHING
RETURNING id, user_id, entitlement_type, course_id, stripe_event_id, stripe_invoice_id,
          amount_paid_cents, currency, valid_from, valid_until, status, created_at
`, in.UserID, in.EntitlementType, in.CourseID, in.StripeEventID, in.StripeInvoiceID,
		in.AmountPaidCents, currency, in.ValidUntil)
	if err == nil {
		return e, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, err
	}
	e, err = scanEntitlement(ctx, pool, `
SELECT id, user_id, entitlement_type, course_id, stripe_event_id, stripe_invoice_id,
       amount_paid_cents, currency, valid_from, valid_until, status, created_at
FROM billing.user_entitlements WHERE stripe_event_id = $1
`, in.StripeEventID)
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
		&e.AmountPaidCents, &e.Currency, &e.ValidFrom, &e.ValidUntil, &e.Status, &e.CreatedAt,
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
SELECT id, user_id, entitlement_type, course_id, stripe_event_id, stripe_invoice_id,
       amount_paid_cents, currency, valid_from, valid_until, status, created_at
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
			&e.AmountPaidCents, &e.Currency, &e.ValidFrom, &e.ValidUntil, &e.Status, &e.CreatedAt,
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
	var ok bool
	err = pool.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM billing.user_entitlements e
  WHERE e.user_id = $1
    AND e.status = 'active'
    AND (e.valid_until IS NULL OR e.valid_until > NOW())
    AND (
      e.course_id = $2
      OR e.entitlement_type LIKE 'subscription%'
    )
)
`, userID, courseID).Scan(&ok)
	return ok, err
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
	ID           uuid.UUID
	Title        string
	PriceCents   int
	Currency     string
	OrgID        uuid.UUID
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
