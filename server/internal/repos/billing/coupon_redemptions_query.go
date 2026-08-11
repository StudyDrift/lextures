// Coupon redemption ledger list/get helpers (plan MKTC.1).
package billing

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ListCouponRedemptions returns redemptions for a coupon, newest first, with a
// created_at-style cursor (reserved_at + id) for pagination.
func ListCouponRedemptions(ctx context.Context, pool *pgxpool.Pool, couponID uuid.UUID, cursor string, limit int) ([]Redemption, string, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var rows pgx.Rows
	var err error
	if cursor == "" {
		rows, err = pool.Query(ctx, `
SELECT `+redemptionSelectCols+`
FROM billing.coupon_redemptions
WHERE coupon_id = $1
ORDER BY reserved_at DESC, id DESC
LIMIT $2
`, couponID, limit)
	} else {
		// Cursor format: RFC3339Nano|uuid
		parts := splitCursor(cursor)
		if len(parts) != 2 {
			return nil, "", errors.New("invalid cursor")
		}
		cursorTime, parseErr := time.Parse(time.RFC3339Nano, parts[0])
		if parseErr != nil {
			return nil, "", errors.New("invalid cursor time")
		}
		cursorID, parseErr := uuid.Parse(parts[1])
		if parseErr != nil {
			return nil, "", errors.New("invalid cursor id")
		}
		rows, err = pool.Query(ctx, `
SELECT `+redemptionSelectCols+`
FROM billing.coupon_redemptions
WHERE coupon_id = $1
  AND (reserved_at, id) < ($2, $3)
ORDER BY reserved_at DESC, id DESC
LIMIT $4
`, couponID, cursorTime, cursorID, limit)
	}
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var out []Redemption
	for rows.Next() {
		r, err := scanRedemption(rows)
		if err != nil {
			return nil, "", err
		}
		out = append(out, *r)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	next := ""
	if len(out) == limit {
		last := out[len(out)-1]
		next = last.ReservedAt.UTC().Format(time.RFC3339Nano) + "|" + last.ID.String()
	}
	return out, next, nil
}

// ListCouponRedemptionsForUser returns all redemptions for a user (DSAR export).
func ListCouponRedemptionsForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]Redemption, error) {
	rows, err := pool.Query(ctx, `
SELECT `+redemptionSelectCols+`
FROM billing.coupon_redemptions
WHERE user_id = $1
ORDER BY reserved_at DESC
`, userID)
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

func splitCursor(c string) []string {
	i := -1
	for j := 0; j < len(c); j++ {
		if c[j] == '|' {
			i = j
		}
	}
	if i < 0 {
		return nil
	}
	return []string{c[:i], c[i+1:]}
}

// SetRedemptionCheckoutSession attaches a provider session id to a reserved row (MKTC.3 FR-4e).
func SetRedemptionCheckoutSession(ctx context.Context, pool *pgxpool.Pool, redemptionID uuid.UUID, sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return errors.New("checkout_session_id required")
	}
	tag, err := pool.Exec(ctx, `
UPDATE billing.coupon_redemptions
SET checkout_session_id = $2
WHERE id = $1 AND status = 'reserved'
`, redemptionID, sessionID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("checkout session already linked to another reservation")
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// GetRedemptionByCheckoutSession loads a redemption by Stripe/PayPal session id.
func GetRedemptionByCheckoutSession(ctx context.Context, pool *pgxpool.Pool, sessionID string) (*Redemption, error) {
	if sessionID == "" {
		return nil, nil
	}
	r, err := scanRedemption(pool.QueryRow(ctx, `
SELECT `+redemptionSelectCols+`
FROM billing.coupon_redemptions WHERE checkout_session_id = $1
`, sessionID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return r, err
}

// GetActiveReservationForUser returns a non-expired reserved row for user+coupon, if any.
func GetActiveReservationForUser(ctx context.Context, pool *pgxpool.Pool, couponID, userID uuid.UUID, now time.Time) (*Redemption, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	r, err := scanRedemption(pool.QueryRow(ctx, `
SELECT `+redemptionSelectCols+`
FROM billing.coupon_redemptions
WHERE coupon_id = $1 AND user_id = $2 AND status = 'reserved'
  AND (expires_at IS NULL OR expires_at > $3)
ORDER BY reserved_at DESC
LIMIT 1
`, couponID, userID, now.UTC()))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return r, err
}

// GetRedeemedByUserAndCourse returns the most recent redeemed redemption for a user on a course.
func GetRedeemedByUserAndCourse(ctx context.Context, pool *pgxpool.Pool, userID, courseID uuid.UUID) (*Redemption, error) {
	r, err := scanRedemption(pool.QueryRow(ctx, `
SELECT `+redemptionSelectCols+`
FROM billing.coupon_redemptions
WHERE user_id = $1 AND course_id = $2 AND status = 'redeemed'
ORDER BY redeemed_at DESC NULLS LAST
LIMIT 1
`, userID, courseID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return r, err
}

// ReleaseCouponReservationBySession releases a reserved row linked to a checkout session (MKTC.3 FR-17).
func ReleaseCouponReservationBySession(ctx context.Context, pool *pgxpool.Pool, sessionID, reason string) error {
	if sessionID == "" {
		return nil
	}
	r, err := GetRedemptionByCheckoutSession(ctx, pool, sessionID)
	if err != nil || r == nil {
		return err
	}
	if r.Status != RedemptionReserved {
		return nil
	}
	return ReleaseCouponReservation(ctx, pool, r.ID, reason)
}
