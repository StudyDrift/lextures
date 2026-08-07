package billing

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const AcquisitionApple = "apple"

// AppleIAPTransaction is a verified StoreKit transaction audit row.
type AppleIAPTransaction struct {
	ID                    uuid.UUID
	UserID                uuid.UUID
	TransactionID         string
	OriginalTransactionID string
	ProductID             string
	BundleID              string
	Environment           string
	EntitlementID         *uuid.UUID
	PurchasedAt           *time.Time
	ExpiresAt             *time.Time
	CreatedAt             time.Time
}

// AppleIAPTransactionInput inserts a verified Apple IAP transaction.
type AppleIAPTransactionInput struct {
	UserID                uuid.UUID
	TransactionID         string
	OriginalTransactionID string
	ProductID             string
	BundleID              string
	Environment           string
	EntitlementID         *uuid.UUID
	SignedPayloadSHA256   string
	PurchasedAt           *time.Time
	ExpiresAt             *time.Time
	RawClaims             json.RawMessage
}

// InsertAppleIAPTransactionIdempotent records a verified transaction once per transaction_id.
func InsertAppleIAPTransactionIdempotent(ctx context.Context, pool *pgxpool.Pool, in AppleIAPTransactionInput) (*AppleIAPTransaction, bool, error) {
	if in.TransactionID == "" {
		return nil, false, errors.New("transaction_id required")
	}
	env := in.Environment
	if env == "" {
		env = "Production"
	}
	var raw any
	if len(in.RawClaims) > 0 {
		raw = in.RawClaims
	}
	var row AppleIAPTransaction
	err := pool.QueryRow(ctx, `
INSERT INTO billing.apple_iap_transactions (
    user_id, transaction_id, original_transaction_id, product_id, bundle_id,
    environment, entitlement_id, signed_payload_sha256, purchased_at, expires_at, raw_claims
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
ON CONFLICT (transaction_id) DO NOTHING
RETURNING id, user_id, transaction_id, original_transaction_id, product_id, bundle_id,
          environment, entitlement_id, purchased_at, expires_at, created_at
`, in.UserID, in.TransactionID, in.OriginalTransactionID, in.ProductID, in.BundleID,
		env, in.EntitlementID, nullIfEmpty(in.SignedPayloadSHA256), in.PurchasedAt, in.ExpiresAt, raw,
	).Scan(
		&row.ID, &row.UserID, &row.TransactionID, &row.OriginalTransactionID, &row.ProductID, &row.BundleID,
		&row.Environment, &row.EntitlementID, &row.PurchasedAt, &row.ExpiresAt, &row.CreatedAt,
	)
	if err == nil {
		return &row, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, err
	}
	err = pool.QueryRow(ctx, `
SELECT id, user_id, transaction_id, original_transaction_id, product_id, bundle_id,
       environment, entitlement_id, purchased_at, expires_at, created_at
FROM billing.apple_iap_transactions WHERE transaction_id = $1
`, in.TransactionID).Scan(
		&row.ID, &row.UserID, &row.TransactionID, &row.OriginalTransactionID, &row.ProductID, &row.BundleID,
		&row.Environment, &row.EntitlementID, &row.PurchasedAt, &row.ExpiresAt, &row.CreatedAt,
	)
	if err != nil {
		return nil, false, err
	}
	return &row, false, nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// CourseAppleProduct returns course id for an App Store product id when mapped.
func CourseAppleProduct(ctx context.Context, pool *pgxpool.Pool, productID string) (*uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
SELECT id FROM course.courses
WHERE apple_product_id = $1
LIMIT 1
`, productID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// CourseAppleProductID returns the App Store product id configured on a course.
func CourseAppleProductID(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (string, error) {
	var productID *string
	err := pool.QueryRow(ctx, `
SELECT NULLIF(btrim(apple_product_id), '') FROM course.courses WHERE id = $1
`, courseID).Scan(&productID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if productID == nil {
		return "", nil
	}
	return *productID, nil
}
