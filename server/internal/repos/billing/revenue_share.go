// Package billing persists creator earnings and affiliate codes (plan 15.8).
package billing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	EntrySale      = "sale"
	EntryAffiliate = "affiliate"
	EntryRefund    = "refund"
	EntryPayout    = "payout"

	EarningsPending = "pending"
	EarningsPaid    = "paid"
	EarningsHeld    = "held"

	defaultPlatformFeePct  = 0.30
	defaultAffiliateFeePct = 0.10
)

// RevenueConfig holds per-creator fee overrides.
type RevenueConfig struct {
	UserID          uuid.UUID
	PlatformFeePct  float64
	AffiliateFeePct float64
}

// LedgerEntry is a row in billing.earnings_ledger.
type LedgerEntry struct {
	ID               uuid.UUID
	PayeeID          uuid.UUID
	EntryType        string
	AmountCents      int
	Currency         string
	StripeEventID    *string
	StripeTransferID *string
	CourseID         *uuid.UUID
	AffiliateCode    *string
	Status           string
	CreatedAt        time.Time
}

// AffiliateCode is a referral code owned by a user.
type AffiliateCode struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	Code       string
	CourseID   *uuid.UUID
	ClickCount int
	CreatedAt  time.Time
}

// EarningsSummary aggregates pending and paid balances.
type EarningsSummary struct {
	PendingCents int
	PaidCents    int
	Currency     string
}

// GetRevenueConfig returns fee percentages for a creator, using defaults when unset.
func GetRevenueConfig(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (RevenueConfig, error) {
	cfg := RevenueConfig{
		UserID:          userID,
		PlatformFeePct:  defaultPlatformFeePct,
		AffiliateFeePct: defaultAffiliateFeePct,
	}
	err := pool.QueryRow(ctx, `
SELECT platform_fee_pct, affiliate_fee_pct
FROM billing.creator_revenue_configs WHERE user_id = $1
`, userID).Scan(&cfg.PlatformFeePct, &cfg.AffiliateFeePct)
	if errors.Is(err, pgx.ErrNoRows) {
		return cfg, nil
	}
	return cfg, err
}

// UpsertRevenueConfig stores admin overrides for a creator.
func UpsertRevenueConfig(ctx context.Context, pool *pgxpool.Pool, cfg RevenueConfig) error {
	_, err := pool.Exec(ctx, `
INSERT INTO billing.creator_revenue_configs (user_id, platform_fee_pct, affiliate_fee_pct, updated_at)
VALUES ($1, $2, $3, NOW())
ON CONFLICT (user_id) DO UPDATE SET
    platform_fee_pct = EXCLUDED.platform_fee_pct,
    affiliate_fee_pct = EXCLUDED.affiliate_fee_pct,
    updated_at = NOW()
`, cfg.UserID, cfg.PlatformFeePct, cfg.AffiliateFeePct)
	return err
}

// CourseCreatorID returns the user who created the course.
func CourseCreatorID(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
SELECT created_by_user_id FROM course.courses WHERE id = $1
`, courseID).Scan(&id)
	return id, err
}

// CreateLedgerEntryInput is the payload for idempotent ledger writes.
type CreateLedgerEntryInput struct {
	PayeeID          uuid.UUID
	EntryType        string
	AmountCents      int
	Currency         string
	StripeEventID    string
	StripeTransferID *string
	CourseID         *uuid.UUID
	AffiliateCode    *string
	Status           string
}

// CreateLedgerEntryIdempotent inserts or returns existing row for stripe_event_id.
func CreateLedgerEntryIdempotent(ctx context.Context, pool *pgxpool.Pool, in CreateLedgerEntryInput) (*LedgerEntry, bool, error) {
	if in.StripeEventID == "" {
		return nil, false, errors.New("stripe_event_id required")
	}
	status := in.Status
	if status == "" {
		status = EarningsPending
	}
	currency := in.Currency
	if currency == "" {
		currency = "usd"
	}
	e, err := scanLedgerEntry(ctx, pool, `
INSERT INTO billing.earnings_ledger (
    payee_id, entry_type, amount_cents, currency, stripe_event_id,
    stripe_transfer_id, course_id, affiliate_code, status
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (stripe_event_id) DO NOTHING
RETURNING id, payee_id, entry_type, amount_cents, currency, stripe_event_id,
          stripe_transfer_id, course_id, affiliate_code, status, created_at
`, in.PayeeID, in.EntryType, in.AmountCents, currency, in.StripeEventID,
		in.StripeTransferID, in.CourseID, in.AffiliateCode, status)
	if err == nil {
		return e, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, err
	}
	e, err = scanLedgerEntry(ctx, pool, `
SELECT id, payee_id, entry_type, amount_cents, currency, stripe_event_id,
       stripe_transfer_id, course_id, affiliate_code, status, created_at
FROM billing.earnings_ledger WHERE stripe_event_id = $1
`, in.StripeEventID)
	if err != nil {
		return nil, false, err
	}
	return e, false, nil
}

func scanLedgerEntry(ctx context.Context, pool *pgxpool.Pool, query string, args ...any) (*LedgerEntry, error) {
	var e LedgerEntry
	var courseID *uuid.UUID
	var affiliateCode *string
	var stripeEventID *string
	var stripeTransferID *string
	err := pool.QueryRow(ctx, query, args...).Scan(
		&e.ID, &e.PayeeID, &e.EntryType, &e.AmountCents, &e.Currency, &stripeEventID,
		&stripeTransferID, &courseID, &affiliateCode, &e.Status, &e.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	e.StripeEventID = stripeEventID
	e.StripeTransferID = stripeTransferID
	e.CourseID = courseID
	e.AffiliateCode = affiliateCode
	return &e, nil
}

// EarningsSummaryForUser aggregates ledger balances.
func EarningsSummaryForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (EarningsSummary, error) {
	var s EarningsSummary
	s.Currency = "usd"
	err := pool.QueryRow(ctx, `
SELECT
    COALESCE(SUM(CASE WHEN status = 'pending' AND entry_type != 'payout' THEN amount_cents ELSE 0 END), 0),
    COALESCE(SUM(CASE WHEN status = 'paid' AND entry_type = 'payout' THEN -amount_cents ELSE 0 END), 0)
FROM billing.earnings_ledger
WHERE payee_id = $1
`, userID).Scan(&s.PendingCents, &s.PaidCents)
	return s, err
}

// ListLedgerForUser returns paginated ledger entries for a payee.
func ListLedgerForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, limit int, before *time.Time) ([]LedgerEntry, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	args := []any{userID, limit}
	query := `
SELECT id, payee_id, entry_type, amount_cents, currency, stripe_event_id,
       stripe_transfer_id, course_id, affiliate_code, status, created_at
FROM billing.earnings_ledger
WHERE payee_id = $1
`
	if before != nil {
		query += ` AND created_at < $3`
		args = append(args, *before)
	}
	query += ` ORDER BY created_at DESC LIMIT $2`

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LedgerEntry
	for rows.Next() {
		var e LedgerEntry
		var courseID *uuid.UUID
		var affiliateCode *string
		var stripeEventID *string
		var stripeTransferID *string
		if err := rows.Scan(
			&e.ID, &e.PayeeID, &e.EntryType, &e.AmountCents, &e.Currency, &stripeEventID,
			&stripeTransferID, &courseID, &affiliateCode, &e.Status, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		e.StripeEventID = stripeEventID
		e.StripeTransferID = stripeTransferID
		e.CourseID = courseID
		e.AffiliateCode = affiliateCode
		out = append(out, e)
	}
	return out, rows.Err()
}

// CreateAffiliateCode inserts a new referral code.
func CreateAffiliateCode(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, courseID *uuid.UUID) (*AffiliateCode, error) {
	code := uuid.New().String()[:8]
	for i := 0; i < 5; i++ {
		ac, err := insertAffiliateCode(ctx, pool, userID, code, courseID)
		if err == nil {
			return ac, nil
		}
		if !isUniqueViolation(err) {
			return nil, err
		}
		code = uuid.New().String()[:8]
	}
	return nil, fmt.Errorf("could not generate unique affiliate code")
}

func insertAffiliateCode(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, code string, courseID *uuid.UUID) (*AffiliateCode, error) {
	var ac AffiliateCode
	var cid *uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO billing.affiliate_codes (user_id, code, course_id)
VALUES ($1, $2, $3)
RETURNING id, user_id, code, course_id, click_count, created_at
`, userID, code, courseID).Scan(&ac.ID, &ac.UserID, &ac.Code, &cid, &ac.ClickCount, &ac.CreatedAt)
	if err != nil {
		return nil, err
	}
	ac.CourseID = cid
	return &ac, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// ListAffiliateCodesForUser returns codes owned by a user with conversion counts.
func ListAffiliateCodesForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]AffiliateCode, map[string]int, error) {
	rows, err := pool.Query(ctx, `
SELECT id, user_id, code, course_id, click_count, created_at
FROM billing.affiliate_codes
WHERE user_id = $1
ORDER BY created_at DESC
`, userID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var codes []AffiliateCode
	codeStrs := []string{}
	for rows.Next() {
		var ac AffiliateCode
		var cid *uuid.UUID
		if err := rows.Scan(&ac.ID, &ac.UserID, &ac.Code, &cid, &ac.ClickCount, &ac.CreatedAt); err != nil {
			return nil, nil, err
		}
		ac.CourseID = cid
		codes = append(codes, ac)
		codeStrs = append(codeStrs, ac.Code)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	conversions := map[string]int{}
	if len(codeStrs) == 0 {
		return codes, conversions, nil
	}
	convRows, err := pool.Query(ctx, `
SELECT affiliate_code, COUNT(*)::int
FROM billing.earnings_ledger
WHERE entry_type = 'affiliate' AND affiliate_code = ANY($1)
GROUP BY affiliate_code
`, codeStrs)
	if err != nil {
		return nil, nil, err
	}
	defer convRows.Close()
	for convRows.Next() {
		var code string
		var count int
		if err := convRows.Scan(&code, &count); err != nil {
			return nil, nil, err
		}
		conversions[code] = count
	}
	return codes, conversions, convRows.Err()
}

// LookupAffiliateCode resolves a referral code to its owner.
func LookupAffiliateCode(ctx context.Context, pool *pgxpool.Pool, code string) (*AffiliateCode, error) {
	var ac AffiliateCode
	var cid *uuid.UUID
	err := pool.QueryRow(ctx, `
SELECT id, user_id, code, course_id, click_count, created_at
FROM billing.affiliate_codes WHERE code = $1
`, code).Scan(&ac.ID, &ac.UserID, &ac.Code, &cid, &ac.ClickCount, &ac.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	ac.CourseID = cid
	return &ac, nil
}

// IncrementAffiliateClickCount records a referral link click.
func IncrementAffiliateClickCount(ctx context.Context, pool *pgxpool.Pool, code string) error {
	_, err := pool.Exec(ctx, `
UPDATE billing.affiliate_codes SET click_count = click_count + 1 WHERE code = $1
`, code)
	return err
}

// StripeConnectID returns the stored Connect account id.
func StripeConnectID(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (string, error) {
	var id *string
	err := pool.QueryRow(ctx, `
SELECT stripe_connect_id FROM "user".users WHERE id = $1
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

// SetStripeConnectID stores the Connect account id.
func SetStripeConnectID(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, connectID string) error {
	_, err := pool.Exec(ctx, `
UPDATE "user".users SET stripe_connect_id = $2 WHERE id = $1
`, userID, connectID)
	return err
}

// PendingPayoutsByUser sums pending earnings per payee with Connect accounts.
type PendingPayout struct {
	UserID       uuid.UUID
	AmountCents  int
	Currency     string
	ConnectID    string
	LedgerIDs    []uuid.UUID
}

// ListPendingPayouts returns users with pending earnings above minCents.
func ListPendingPayouts(ctx context.Context, pool *pgxpool.Pool, minCents int) ([]PendingPayout, error) {
	rows, err := pool.Query(ctx, `
SELECT u.id, u.stripe_connect_id,
       COALESCE(SUM(e.amount_cents), 0)::int,
       COALESCE(MAX(e.currency), 'usd'),
       ARRAY_AGG(e.id ORDER BY e.created_at)
FROM "user".users u
JOIN billing.earnings_ledger e ON e.payee_id = u.id
WHERE e.status = 'pending'
  AND e.entry_type IN ('sale', 'affiliate', 'refund')
  AND u.stripe_connect_id IS NOT NULL
GROUP BY u.id, u.stripe_connect_id
HAVING COALESCE(SUM(e.amount_cents), 0) >= $1
`, minCents)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PendingPayout
	for rows.Next() {
		var p PendingPayout
		var connectID *string
		if err := rows.Scan(&p.UserID, &connectID, &p.AmountCents, &p.Currency, &p.LedgerIDs); err != nil {
			return nil, err
		}
		if connectID == nil || *connectID == "" || p.AmountCents <= 0 {
			continue
		}
		p.ConnectID = *connectID
		out = append(out, p)
	}
	return out, rows.Err()
}

// MarkLedgerPaid marks pending entries as paid and records a payout entry.
func MarkLedgerPaid(ctx context.Context, pool *pgxpool.Pool, payeeID uuid.UUID, ledgerIDs []uuid.UUID, transferID string, amountCents int, currency string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
UPDATE billing.earnings_ledger SET status = 'paid'
WHERE id = ANY($1) AND payee_id = $2 AND status = 'pending'
`, ledgerIDs, payeeID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("no pending entries to mark paid")
	}

	payoutEventID := fmt.Sprintf("payout_%s", transferID)
	_, err = tx.Exec(ctx, `
INSERT INTO billing.earnings_ledger (
    payee_id, entry_type, amount_cents, currency, stripe_event_id, stripe_transfer_id, status
) VALUES ($1, 'payout', $2, $3, $4, $5, 'paid')
ON CONFLICT (stripe_event_id) DO NOTHING
`, payeeID, -amountCents, currency, payoutEventID, transferID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// PlatformRevenueSummary aggregates platform-side revenue metrics.
type PlatformRevenueSummary struct {
	TotalSalesCents     int
	TotalAffiliateCents int
	TotalCreatorCents   int
	PendingPayoutCents  int
}

// PlatformRevenueOverview returns admin revenue metrics.
func PlatformRevenueOverview(ctx context.Context, pool *pgxpool.Pool) (PlatformRevenueSummary, error) {
	var s PlatformRevenueSummary
	err := pool.QueryRow(ctx, `
SELECT
    COALESCE(SUM(CASE WHEN entry_type = 'sale' THEN amount_cents ELSE 0 END), 0),
    COALESCE(SUM(CASE WHEN entry_type = 'affiliate' THEN amount_cents ELSE 0 END), 0),
    COALESCE(SUM(CASE WHEN entry_type IN ('sale', 'affiliate') AND status = 'pending' THEN amount_cents ELSE 0 END), 0)
FROM billing.earnings_ledger
`).Scan(&s.TotalCreatorCents, &s.TotalAffiliateCents, &s.PendingPayoutCents)
	if err != nil {
		return s, err
	}
	// Total sales = creator + affiliate + platform fee approximated from sale entries
	err = pool.QueryRow(ctx, `
SELECT COALESCE(SUM(amount_paid_cents), 0)
FROM billing.user_entitlements
WHERE status = 'active' AND entitlement_type = 'course_purchase'
`).Scan(&s.TotalSalesCents)
	return s, err
}
