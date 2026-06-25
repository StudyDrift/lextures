// Package payments persists payment transactions and subscriptions (plan 16.8).
package payments

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	ProviderStripe = "stripe"
	ProviderPayPal = "paypal"

	StatusPending   = "pending"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
	StatusRefunded  = "refunded"

	SubStatusActive   = "active"
	SubStatusPastDue  = "past_due"
	SubStatusCanceled = "canceled"
)

// Transaction is a row in payments.transactions.
type Transaction struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	CourseID        *uuid.UUID
	Provider        string
	ProviderTxnID   string
	IdempotencyKey  string
	AmountCents     int
	Currency        string
	Status          string
	SubscriptionID  *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// CreateTransactionInput inserts or returns an existing transaction by idempotency key.
type CreateTransactionInput struct {
	UserID         uuid.UUID
	CourseID       *uuid.UUID
	Provider       string
	ProviderTxnID  string
	IdempotencyKey string
	AmountCents    int
	Currency       string
	Status         string
	SubscriptionID *string
}

// CreateIdempotent inserts a transaction or returns the existing row.
func CreateIdempotent(ctx context.Context, pool *pgxpool.Pool, in CreateTransactionInput) (*Transaction, bool, error) {
	if in.IdempotencyKey == "" {
		return nil, false, errors.New("idempotency_key required")
	}
	status := in.Status
	if status == "" {
		status = StatusPending
	}
	currency := in.Currency
	if currency == "" {
		currency = "usd"
	}
	tx, err := scanTransaction(ctx, pool, `
INSERT INTO payments.transactions (
    user_id, course_id, provider, provider_txn_id, idempotency_key,
    amount_cents, currency, status, subscription_id
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (idempotency_key) DO NOTHING
RETURNING id, user_id, course_id, provider, provider_txn_id, idempotency_key,
          amount_cents, currency, status, subscription_id, created_at, updated_at
`, in.UserID, in.CourseID, in.Provider, in.ProviderTxnID, in.IdempotencyKey,
		in.AmountCents, currency, status, in.SubscriptionID)
	if err == nil {
		return tx, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, err
	}
	tx, err = GetByIdempotencyKey(ctx, pool, in.IdempotencyKey)
	if err != nil {
		return nil, false, err
	}
	return tx, false, nil
}

// GetByID loads a transaction by primary key.
func GetByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Transaction, error) {
	return scanTransaction(ctx, pool, `
SELECT id, user_id, course_id, provider, provider_txn_id, idempotency_key,
       amount_cents, currency, status, subscription_id, created_at, updated_at
FROM payments.transactions WHERE id = $1
`, id)
}

// GetByIdempotencyKey loads a transaction by idempotency key.
func GetByIdempotencyKey(ctx context.Context, pool *pgxpool.Pool, key string) (*Transaction, error) {
	return scanTransaction(ctx, pool, `
SELECT id, user_id, course_id, provider, provider_txn_id, idempotency_key,
       amount_cents, currency, status, subscription_id, created_at, updated_at
FROM payments.transactions WHERE idempotency_key = $1
`, key)
}

// ListByUser returns transactions for purchase history, newest first.
func ListByUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, limit int) ([]Transaction, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := pool.Query(ctx, `
SELECT id, user_id, course_id, provider, provider_txn_id, idempotency_key,
       amount_cents, currency, status, subscription_id, created_at, updated_at
FROM payments.transactions
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2
`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Transaction
	for rows.Next() {
		tx, err := scanTransactionRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *tx)
	}
	return out, rows.Err()
}

// UpdateStatus sets transaction status and updated_at.
func UpdateStatus(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, status string) error {
	_, err := pool.Exec(ctx, `
UPDATE payments.transactions SET status = $2, updated_at = NOW() WHERE id = $1
`, id, status)
	return err
}

// UpdateStatusByProviderTxn sets status for a provider transaction id.
func UpdateStatusByProviderTxn(ctx context.Context, pool *pgxpool.Pool, provider, providerTxnID, status string) error {
	_, err := pool.Exec(ctx, `
UPDATE payments.transactions SET status = $3, updated_at = NOW()
WHERE provider = $1 AND provider_txn_id = $2
`, provider, providerTxnID, status)
	return err
}

func scanTransaction(ctx context.Context, pool *pgxpool.Pool, query string, args ...any) (*Transaction, error) {
	row := pool.QueryRow(ctx, query, args...)
	return scanTransactionRow(row)
}

func scanTransactionRow(row pgx.Row) (*Transaction, error) {
	var tx Transaction
	var courseID *uuid.UUID
	err := row.Scan(
		&tx.ID, &tx.UserID, &courseID, &tx.Provider, &tx.ProviderTxnID, &tx.IdempotencyKey,
		&tx.AmountCents, &tx.Currency, &tx.Status, &tx.SubscriptionID, &tx.CreatedAt, &tx.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	tx.CourseID = courseID
	return &tx, nil
}
