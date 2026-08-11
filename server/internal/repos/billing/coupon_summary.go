// Coupon performance summary aggregation (plan MKTC.7 FR-7, FR-8).
package billing

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CouponSummaryRow is one coupon's performance figures for the creator table.
type CouponSummaryRow struct {
	CouponID        uuid.UUID
	Code            string
	RedeemedCount   int
	RefundedCount   int
	GrossListCents  int
	DiscountCents   int
	NetChargedCents int
	Currency        string
	FirstRedeemedAt *time.Time
	LastRedeemedAt  *time.Time
}

// CouponSummaryByCourse returns performance figures for every coupon on a course
// in a single aggregate query (FR-7). Redeemed rows feed net revenue; rows that
// were redeemed then released (refund path) appear as refundedCount only (FR-8).
func CouponSummaryByCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]CouponSummaryRow, error) {
	if pool == nil {
		return nil, nil
	}
	rows, err := pool.Query(ctx, `
SELECT
    c.id,
    c.code,
    COALESCE(COUNT(*) FILTER (WHERE r.status = 'redeemed'), 0)::int AS redeemed_count,
    COALESCE(COUNT(*) FILTER (
        WHERE r.status = 'released' AND r.redeemed_at IS NOT NULL
    ), 0)::int AS refunded_count,
    COALESCE(SUM(r.list_price_cents) FILTER (WHERE r.status = 'redeemed'), 0)::int AS gross_list_cents,
    COALESCE(SUM(r.discount_cents) FILTER (WHERE r.status = 'redeemed'), 0)::int AS discount_cents,
    COALESCE(SUM(r.charged_cents) FILTER (WHERE r.status = 'redeemed'), 0)::int AS net_charged_cents,
    COALESCE(
        MAX(r.currency) FILTER (WHERE r.status = 'redeemed'),
        MAX(r.currency) FILTER (WHERE r.status = 'released' AND r.redeemed_at IS NOT NULL),
        ''
    ) AS currency,
    MIN(r.redeemed_at) FILTER (WHERE r.status = 'redeemed') AS first_redeemed_at,
    MAX(r.redeemed_at) FILTER (WHERE r.status = 'redeemed') AS last_redeemed_at
FROM billing.course_coupons c
LEFT JOIN billing.coupon_redemptions r ON r.coupon_id = c.id
WHERE c.course_id = $1
GROUP BY c.id, c.code
ORDER BY c.created_at DESC
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CouponSummaryRow
	for rows.Next() {
		var row CouponSummaryRow
		if err := rows.Scan(
			&row.CouponID,
			&row.Code,
			&row.RedeemedCount,
			&row.RefundedCount,
			&row.GrossListCents,
			&row.DiscountCents,
			&row.NetChargedCents,
			&row.Currency,
			&row.FirstRedeemedAt,
			&row.LastRedeemedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// StreamCouponRedemptionsForExport yields all redemptions for a coupon ordered
// by reserved_at ascending for CSV export (FR-9). Caller owns iteration.
func StreamCouponRedemptionsForExport(ctx context.Context, pool *pgxpool.Pool, couponID uuid.UUID) ([]Redemption, error) {
	if pool == nil {
		return nil, nil
	}
	rows, err := pool.Query(ctx, `
SELECT `+redemptionSelectCols+`
FROM billing.coupon_redemptions
WHERE coupon_id = $1
ORDER BY reserved_at ASC, id ASC
`, couponID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Redemption
	for rows.Next() {
		r, err := scanRedemption(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}
