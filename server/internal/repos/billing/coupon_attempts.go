// Coupon attempt audit trail (plan MKTC.7 FR-3).
package billing

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CouponAttemptRetention is how long failed apply rows are kept (30 days).
const CouponAttemptRetention = 30 * 24 * time.Hour

// InsertCouponAttemptInput is a single failed apply log row.
type InsertCouponAttemptInput struct {
	UserID   uuid.UUID
	CourseID uuid.UUID
	// CodeHash is a salted hash of the code (required). Never store the raw
	// unknown code — callers must hash not_found codes before insert.
	CodeHash string
	Reason   string
	IPPrefix string
}

// InsertCouponAttempt appends one failed apply to billing.coupon_attempts.
func InsertCouponAttempt(ctx context.Context, pool *pgxpool.Pool, in InsertCouponAttemptInput) error {
	if pool == nil {
		return nil
	}
	if in.CodeHash == "" {
		in.CodeHash = "unknown"
	}
	if in.Reason == "" {
		in.Reason = "not_found"
	}
	var userID any
	if in.UserID != uuid.Nil {
		userID = in.UserID
	}
	var courseID any
	if in.CourseID != uuid.Nil {
		courseID = in.CourseID
	}
	var ip any
	if in.IPPrefix != "" {
		ip = in.IPPrefix
	}
	_, err := pool.Exec(ctx, `
INSERT INTO billing.coupon_attempts (user_id, course_id, code_hash, reason, ip_prefix)
VALUES ($1, $2, $3, $4, $5)
`, userID, courseID, in.CodeHash, in.Reason, ip)
	return err
}

// DeleteExpiredCouponAttempts removes rows older than retention (default 30 days).
// Returns the number of deleted rows.
func DeleteExpiredCouponAttempts(ctx context.Context, pool *pgxpool.Pool, now time.Time) (int64, error) {
	if pool == nil {
		return 0, nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	cutoff := now.Add(-CouponAttemptRetention)
	tag, err := pool.Exec(ctx, `
DELETE FROM billing.coupon_attempts
WHERE created_at < $1
`, cutoff)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
