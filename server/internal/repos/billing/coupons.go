// Course coupon CRUD and seat-count helpers (plan MKTC.1).
package billing

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/service/coupons"
)

// Coupon statuses mirror billing.course_coupons.status.
const (
	CouponStatusActive   = "active"
	CouponStatusDisabled = "disabled"
	CouponStatusArchived = "archived"
)

// Coupon is a row in billing.course_coupons.
type Coupon struct {
	ID                    uuid.UUID
	CourseID              uuid.UUID
	Code                  string
	DiscountType          string // percent | fixed
	PercentOff            *float64
	AmountOffCents        *int
	Currency              *string
	StartsAt              *time.Time
	EndsAt                *time.Time
	MaxRedemptions        *int
	MaxRedemptionsPerUser int
	RedeemedCount         int
	Status                string
	Note                  *string
	CreatedBy             *uuid.UUID
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// ToDomain converts a repo Coupon to the pure coupons.Coupon view.
func (c *Coupon) ToDomain() coupons.Coupon {
	out := coupons.Coupon{
		ID:                    c.ID,
		CourseID:              c.CourseID,
		Code:                  c.Code,
		Kind:                  coupons.Kind(c.DiscountType),
		MaxRedemptionsPerUser: c.MaxRedemptionsPerUser,
		Status:                c.Status,
		StartsAt:              c.StartsAt,
		EndsAt:                c.EndsAt,
		MaxRedemptions:        c.MaxRedemptions,
	}
	if c.PercentOff != nil {
		out.PercentOff = *c.PercentOff
	}
	if c.AmountOffCents != nil {
		out.AmountOffCents = *c.AmountOffCents
	}
	if c.Currency != nil {
		out.Currency = *c.Currency
	}
	return out
}

// CreateCouponInput is the payload for inserting a coupon.
type CreateCouponInput struct {
	CourseID              uuid.UUID
	Code                  string // raw; normalized + validated here
	DiscountType          string
	PercentOff            *float64
	AmountOffCents        *int
	Currency              *string
	StartsAt              *time.Time
	EndsAt                *time.Time
	MaxRedemptions        *int
	MaxRedemptionsPerUser int
	Note                  *string
	CreatedBy             *uuid.UUID
	Status                string // default active
}

// UpdateCouponInput patches mutable coupon fields (not code or course_id).
type UpdateCouponInput struct {
	ID                    uuid.UUID
	PercentOff            *float64
	AmountOffCents        *int
	Currency              *string
	StartsAt              *time.Time
	EndsAt                *time.Time
	MaxRedemptions        *int
	MaxRedemptionsPerUser *int
	Note                  *string
	ClearStartsAt         bool
	ClearEndsAt           bool
	ClearMaxRedemptions   bool
	ClearNote             bool
}

// SeatCount is the consumed seat breakdown for one coupon.
type SeatCount struct {
	CouponID  uuid.UUID
	Reserved  int // non-expired reserved
	Redeemed  int
	Consumed  int // reserved + redeemed (authoritative for caps)
}

const couponSelectCols = `
id, course_id, code, discount_type, percent_off, amount_off_cents, currency,
starts_at, ends_at, max_redemptions, max_redemptions_per_user, redeemed_count,
status, note, created_by, created_at, updated_at
`

func scanCoupon(row pgx.Row) (*Coupon, error) {
	var c Coupon
	err := row.Scan(
		&c.ID, &c.CourseID, &c.Code, &c.DiscountType, &c.PercentOff, &c.AmountOffCents, &c.Currency,
		&c.StartsAt, &c.EndsAt, &c.MaxRedemptions, &c.MaxRedemptionsPerUser, &c.RedeemedCount,
		&c.Status, &c.Note, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// CreateCoupon inserts a new coupon after normalizing and validating the code.
func CreateCoupon(ctx context.Context, pool *pgxpool.Pool, in CreateCouponInput) (*Coupon, error) {
	code := coupons.NormalizeCode(in.Code)
	if err := coupons.ValidateCode(code); err != nil {
		return nil, err
	}
	status := in.Status
	if status == "" {
		status = CouponStatusActive
	}
	switch status {
	case CouponStatusActive, CouponStatusDisabled, CouponStatusArchived:
	default:
		return nil, fmt.Errorf("invalid coupon status %q", status)
	}
	perUser := in.MaxRedemptionsPerUser
	if perUser <= 0 {
		perUser = 1
	}
	dtype := strings.ToLower(strings.TrimSpace(in.DiscountType))
	var curr any
	if in.Currency != nil {
		c := strings.ToLower(strings.TrimSpace(*in.Currency))
		curr = c
	}
	row := pool.QueryRow(ctx, `
INSERT INTO billing.course_coupons (
    course_id, code, discount_type, percent_off, amount_off_cents, currency,
    starts_at, ends_at, max_redemptions, max_redemptions_per_user, status, note, created_by
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
RETURNING `+couponSelectCols, in.CourseID, code, dtype, in.PercentOff, in.AmountOffCents, curr,
		in.StartsAt, in.EndsAt, in.MaxRedemptions, perUser, status, in.Note, in.CreatedBy)
	return scanCoupon(row)
}

// UpdateCoupon patches mutable fields on a non-archived coupon.
func UpdateCoupon(ctx context.Context, pool *pgxpool.Pool, in UpdateCouponInput) (*Coupon, error) {
	// Load current to merge kind-specific fields.
	cur, err := GetCouponByID(ctx, pool, in.ID)
	if err != nil {
		return nil, err
	}
	if cur == nil {
		return nil, pgx.ErrNoRows
	}
	if cur.Status == CouponStatusArchived {
		return nil, errors.New("cannot update archived coupon")
	}

	percent := cur.PercentOff
	amount := cur.AmountOffCents
	curr := cur.Currency
	starts := cur.StartsAt
	ends := cur.EndsAt
	maxR := cur.MaxRedemptions
	perUser := cur.MaxRedemptionsPerUser
	note := cur.Note

	if cur.DiscountType == "percent" && in.PercentOff != nil {
		percent = in.PercentOff
	}
	if cur.DiscountType == "fixed" {
		if in.AmountOffCents != nil {
			amount = in.AmountOffCents
		}
		if in.Currency != nil {
			c := strings.ToLower(strings.TrimSpace(*in.Currency))
			curr = &c
		}
	}
	if in.ClearStartsAt {
		starts = nil
	} else if in.StartsAt != nil {
		starts = in.StartsAt
	}
	if in.ClearEndsAt {
		ends = nil
	} else if in.EndsAt != nil {
		ends = in.EndsAt
	}
	if in.ClearMaxRedemptions {
		maxR = nil
	} else if in.MaxRedemptions != nil {
		maxR = in.MaxRedemptions
	}
	if in.MaxRedemptionsPerUser != nil {
		perUser = *in.MaxRedemptionsPerUser
	}
	if in.ClearNote {
		note = nil
	} else if in.Note != nil {
		note = in.Note
	}

	row := pool.QueryRow(ctx, `
UPDATE billing.course_coupons SET
    percent_off = $2,
    amount_off_cents = $3,
    currency = $4,
    starts_at = $5,
    ends_at = $6,
    max_redemptions = $7,
    max_redemptions_per_user = $8,
    note = $9,
    updated_at = NOW()
WHERE id = $1 AND status <> 'archived'
RETURNING `+couponSelectCols,
		in.ID, percent, amount, curr, starts, ends, maxR, perUser, note)
	return scanCoupon(row)
}

// SetCouponStatus transitions status (active|disabled|archived).
func SetCouponStatus(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, status string) error {
	switch status {
	case CouponStatusActive, CouponStatusDisabled, CouponStatusArchived:
	default:
		return fmt.Errorf("invalid coupon status %q", status)
	}
	tag, err := pool.Exec(ctx, `
UPDATE billing.course_coupons SET status = $2, updated_at = NOW() WHERE id = $1
`, id, status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ListCouponsByCourse returns coupons for a course, newest first.
func ListCouponsByCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, includeArchived bool) ([]Coupon, error) {
	q := `
SELECT ` + couponSelectCols + `
FROM billing.course_coupons
WHERE course_id = $1
`
	if !includeArchived {
		q += ` AND status <> 'archived'`
	}
	q += ` ORDER BY created_at DESC`
	rows, err := pool.Query(ctx, q, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Coupon
	for rows.Next() {
		c, err := scanCoupon(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

// GetCouponByID loads a coupon by primary key.
func GetCouponByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Coupon, error) {
	c, err := scanCoupon(pool.QueryRow(ctx, `
SELECT `+couponSelectCols+` FROM billing.course_coupons WHERE id = $1
`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

// GetCouponByCourseAndID loads a coupon scoped to a course (IDOR-safe; plan MKTC.2).
func GetCouponByCourseAndID(ctx context.Context, pool *pgxpool.Pool, courseID, couponID uuid.UUID) (*Coupon, error) {
	c, err := scanCoupon(pool.QueryRow(ctx, `
SELECT `+couponSelectCols+`
FROM billing.course_coupons
WHERE id = $1 AND course_id = $2
`, couponID, courseID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

// GetCouponByCourseAndCode loads a non-archived coupon by course + normalized code.
func GetCouponByCourseAndCode(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, rawCode string) (*Coupon, error) {
	code := coupons.NormalizeCode(rawCode)
	c, err := scanCoupon(pool.QueryRow(ctx, `
SELECT `+couponSelectCols+`
FROM billing.course_coupons
WHERE course_id = $1 AND code = $2 AND status <> 'archived'
`, courseID, code))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

// CouponSeatCounts returns consumed seat counts for many coupons in one query (FR-17).
// Consumed = non-expired reserved + redeemed.
func CouponSeatCounts(ctx context.Context, pool *pgxpool.Pool, couponIDs []uuid.UUID) (map[uuid.UUID]SeatCount, error) {
	out := make(map[uuid.UUID]SeatCount, len(couponIDs))
	if len(couponIDs) == 0 {
		return out, nil
	}
	for _, id := range couponIDs {
		out[id] = SeatCount{CouponID: id}
	}
	rows, err := pool.Query(ctx, `
SELECT coupon_id,
       COUNT(*) FILTER (
         WHERE status = 'reserved'
           AND (expires_at IS NULL OR expires_at > NOW())
       )::int AS reserved,
       COUNT(*) FILTER (WHERE status = 'redeemed')::int AS redeemed
FROM billing.coupon_redemptions
WHERE coupon_id = ANY($1)
GROUP BY coupon_id
`, couponIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var sc SeatCount
		if err := rows.Scan(&sc.CouponID, &sc.Reserved, &sc.Redeemed); err != nil {
			return nil, err
		}
		sc.Consumed = sc.Reserved + sc.Redeemed
		out[sc.CouponID] = sc
	}
	return out, rows.Err()
}

// UserSeatCount returns non-expired reserved + redeemed seats for one user on one coupon.
func UserSeatCount(ctx context.Context, pool *pgxpool.Pool, couponID, userID uuid.UUID) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
SELECT COUNT(*)::int
FROM billing.coupon_redemptions
WHERE coupon_id = $1 AND user_id = $2
  AND (
    status = 'redeemed'
    OR (status = 'reserved' AND (expires_at IS NULL OR expires_at > NOW()))
  )
`, couponID, userID).Scan(&n)
	return n, err
}

