package transcripts

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AfterOrderStatusChange is invoked after submit/transition/payment advances (T10 notify hook).
var AfterOrderStatusChange func(ctx context.Context, pool *pgxpool.Pool, order *Order)

// AfterDeliveryReceipt is invoked after a delivery attempt reaches sent/delivered/opened/failed (T10).
var AfterDeliveryReceipt func(ctx context.Context, pool *pgxpool.Pool, order *Order, itemID uuid.UUID, status DeliveryAttemptStatus)

// NotifyOrderStatusChange calls AfterOrderStatusChange when configured.
func NotifyOrderStatusChange(ctx context.Context, pool *pgxpool.Pool, order *Order) {
	if AfterOrderStatusChange != nil && pool != nil && order != nil {
		AfterOrderStatusChange(ctx, pool, order)
	}
}

// NotifyDeliveryReceipt calls AfterDeliveryReceipt when configured.
func NotifyDeliveryReceipt(ctx context.Context, pool *pgxpool.Pool, order *Order, itemID uuid.UUID, status DeliveryAttemptStatus) {
	if AfterDeliveryReceipt != nil && pool != nil && order != nil && itemID != uuid.Nil {
		AfterDeliveryReceipt(ctx, pool, order, itemID, status)
	}
}
