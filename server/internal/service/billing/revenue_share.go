package billing

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/account"
	"github.com/stripe/stripe-go/v81/accountlink"
	"github.com/stripe/stripe-go/v81/transfer"

	repoBilling "github.com/lextures/lextures/server/internal/repos/billing"
)

const minPayoutCents = 2500 // $25 minimum payout threshold

// SaleEarningsInput is passed when a course purchase is confirmed.
type SaleEarningsInput struct {
	BuyerUserID   uuid.UUID
	CourseID      uuid.UUID
	AmountCents   int
	Currency      string
	StripeEventID string
	AffiliateCode string
}

// ComputeCreatorShare returns creator earnings in cents after platform fee.
func ComputeCreatorShare(amountCents int, platformFeePct float64) int {
	if amountCents <= 0 {
		return 0
	}
	share := float64(amountCents) * (1 - platformFeePct)
	return int(math.Round(share))
}

// ComputeAffiliateCommission returns affiliate commission in cents.
func ComputeAffiliateCommission(amountCents int, affiliateFeePct float64) int {
	if amountCents <= 0 {
		return 0
	}
	commission := float64(amountCents) * affiliateFeePct
	return int(math.Round(commission))
}

// IsSelfReferral reports whether affiliate commission should be blocked.
func IsSelfReferral(buyerID, affiliateOwnerID, courseCreatorID uuid.UUID) bool {
	if buyerID == affiliateOwnerID {
		return true
	}
	if buyerID == courseCreatorID && affiliateOwnerID == courseCreatorID {
		return true
	}
	return false
}

// RecordSaleEarnings creates creator and optional affiliate ledger entries.
func RecordSaleEarnings(ctx context.Context, pool *pgxpool.Pool, in SaleEarningsInput) error {
	if pool == nil || in.CourseID == uuid.Nil || in.StripeEventID == "" {
		return nil
	}
	creatorID, err := repoBilling.CourseCreatorID(ctx, pool, in.CourseID)
	if err != nil {
		return err
	}
	if creatorID == uuid.Nil {
		return nil
	}
	cfg, err := repoBilling.GetRevenueConfig(ctx, pool, creatorID)
	if err != nil {
		return err
	}
	creatorCents := ComputeCreatorShare(in.AmountCents, cfg.PlatformFeePct)
	if creatorCents > 0 {
		_, _, err = repoBilling.CreateLedgerEntryIdempotent(ctx, pool, repoBilling.CreateLedgerEntryInput{
			PayeeID:       creatorID,
			EntryType:     repoBilling.EntrySale,
			AmountCents:   creatorCents,
			Currency:      in.Currency,
			StripeEventID: in.StripeEventID + ":sale",
			CourseID:      &in.CourseID,
		})
		if err != nil {
			return err
		}
		RecordCreatorEarnings(creatorCents)
	}

	code := strings.TrimSpace(in.AffiliateCode)
	if code == "" {
		return nil
	}
	ac, err := repoBilling.LookupAffiliateCode(ctx, pool, code)
	if err != nil || ac == nil {
		return err
	}
	if ac.CourseID != nil && *ac.CourseID != in.CourseID {
		return nil
	}
	if IsSelfReferral(in.BuyerUserID, ac.UserID, creatorID) {
		return nil
	}
	affCfg, err := repoBilling.GetRevenueConfig(ctx, pool, ac.UserID)
	if err != nil {
		return err
	}
	affCents := ComputeAffiliateCommission(in.AmountCents, affCfg.AffiliateFeePct)
	if affCents <= 0 {
		return nil
	}
	affCode := code
	_, _, err = repoBilling.CreateLedgerEntryIdempotent(ctx, pool, repoBilling.CreateLedgerEntryInput{
		PayeeID:       ac.UserID,
		EntryType:     repoBilling.EntryAffiliate,
		AmountCents:   affCents,
		Currency:      in.Currency,
		StripeEventID: in.StripeEventID + ":affiliate",
		CourseID:      &in.CourseID,
		AffiliateCode: &affCode,
	})
	if err != nil {
		return err
	}
	RecordAffiliateEarnings(affCents)
	return nil
}

// RefundEarningsInput offsets prior creator/affiliate earnings on refund.
type RefundEarningsInput struct {
	CourseID      uuid.UUID
	AmountCents   int
	Currency      string
	StripeEventID string
	AffiliateCode string
	OriginalEvent string
}

// RecordRefundEarnings creates negative ledger entries for a refund.
func RecordRefundEarnings(ctx context.Context, pool *pgxpool.Pool, in RefundEarningsInput) error {
	if pool == nil || in.CourseID == uuid.Nil || in.StripeEventID == "" {
		return nil
	}
	creatorID, err := repoBilling.CourseCreatorID(ctx, pool, in.CourseID)
	if err != nil {
		return err
	}
	if creatorID == uuid.Nil {
		return nil
	}
	cfg, err := repoBilling.GetRevenueConfig(ctx, pool, creatorID)
	if err != nil {
		return err
	}
	creatorCents := -ComputeCreatorShare(in.AmountCents, cfg.PlatformFeePct)
	if creatorCents != 0 {
		_, _, err = repoBilling.CreateLedgerEntryIdempotent(ctx, pool, repoBilling.CreateLedgerEntryInput{
			PayeeID:       creatorID,
			EntryType:     repoBilling.EntryRefund,
			AmountCents:   creatorCents,
			Currency:      in.Currency,
			StripeEventID: in.StripeEventID + ":refund:sale",
			CourseID:      &in.CourseID,
		})
		if err != nil {
			return err
		}
	}

	code := strings.TrimSpace(in.AffiliateCode)
	if code == "" {
		return nil
	}
	ac, err := repoBilling.LookupAffiliateCode(ctx, pool, code)
	if err != nil || ac == nil {
		return err
	}
	affCfg, err := repoBilling.GetRevenueConfig(ctx, pool, ac.UserID)
	if err != nil {
		return err
	}
	affCents := -ComputeAffiliateCommission(in.AmountCents, affCfg.AffiliateFeePct)
	if affCents == 0 {
		return nil
	}
	affCode := code
	_, _, err = repoBilling.CreateLedgerEntryIdempotent(ctx, pool, repoBilling.CreateLedgerEntryInput{
		PayeeID:       ac.UserID,
		EntryType:     repoBilling.EntryRefund,
		AmountCents:   affCents,
		Currency:      in.Currency,
		StripeEventID: in.StripeEventID + ":refund:affiliate",
		CourseID:      &in.CourseID,
		AffiliateCode: &affCode,
	})
	return err
}

// ConnectConfig holds Stripe Connect settings.
type ConnectConfig struct {
	SecretKey       string
	PublicWebOrigin string
}

func (c ConnectConfig) IsConfigured() bool {
	return strings.TrimSpace(c.SecretKey) != ""
}

// EnsureConnectAccount creates or returns a Stripe Connect Express account.
func EnsureConnectAccount(ctx context.Context, pool *pgxpool.Pool, cfg ConnectConfig, userID uuid.UUID, email string) (string, error) {
	existing, err := repoBilling.StripeConnectID(ctx, pool, userID)
	if err != nil {
		return "", err
	}
	if existing != "" {
		return existing, nil
	}
	if !cfg.IsConfigured() {
		return "", errors.New("stripe not configured")
	}
	stripe.Key = cfg.SecretKey
	acct, err := account.New(&stripe.AccountParams{
		Type:  stripe.String(string(stripe.AccountTypeExpress)),
		Email: stripe.String(email),
		Metadata: map[string]string{
			"user_id": userID.String(),
		},
	})
	if err != nil {
		return "", err
	}
	if err := repoBilling.SetStripeConnectID(ctx, pool, userID, acct.ID); err != nil {
		return "", err
	}
	return acct.ID, nil
}

// CreateConnectOnboardingLink returns a Stripe-hosted onboarding URL.
func CreateConnectOnboardingLink(ctx context.Context, pool *pgxpool.Pool, cfg ConnectConfig, userID uuid.UUID, email string) (string, error) {
	acctID, err := EnsureConnectAccount(ctx, pool, cfg, userID, email)
	if err != nil {
		return "", err
	}
	stripe.Key = cfg.SecretKey
	origin := strings.TrimRight(cfg.PublicWebOrigin, "/")
	link, err := accountlink.New(&stripe.AccountLinkParams{
		Account:    stripe.String(acctID),
		RefreshURL: stripe.String(origin + "/me/creator/earnings?connect=refresh"),
		ReturnURL:  stripe.String(origin + "/me/creator/earnings?connect=done"),
		Type:       stripe.String("account_onboarding"),
	})
	if err != nil {
		return "", err
	}
	return link.URL, nil
}

// PayoutResult summarizes a payout batch run.
type PayoutResult struct {
	Processed int
	Failed    int
	Errors    []string
}

// RunMonthlyPayouts transfers pending earnings to connected accounts.
func RunMonthlyPayouts(ctx context.Context, pool *pgxpool.Pool, cfg ConnectConfig) (*PayoutResult, error) {
	if !cfg.IsConfigured() {
		return nil, errors.New("stripe not configured")
	}
	pending, err := repoBilling.ListPendingPayouts(ctx, pool, minPayoutCents)
	if err != nil {
		return nil, err
	}
	result := &PayoutResult{}
	stripe.Key = cfg.SecretKey
	for _, p := range pending {
		idempotencyKey := fmt.Sprintf("payout_%s_%d", p.UserID.String(), p.AmountCents)
		tr, err := transfer.New(&stripe.TransferParams{
			Amount:      stripe.Int64(int64(p.AmountCents)),
			Currency:    stripe.String(p.Currency),
			Destination: stripe.String(p.ConnectID),
			Metadata: map[string]string{
				"user_id": p.UserID.String(),
			},
			Params: stripe.Params{
				IdempotencyKey: stripe.String(idempotencyKey),
			},
		})
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, err.Error())
			RecordPayoutFailure()
			continue
		}
		if err := repoBilling.MarkLedgerPaid(ctx, pool, p.UserID, p.LedgerIDs, tr.ID, p.AmountCents, p.Currency); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, err.Error())
			RecordPayoutFailure()
			continue
		}
		result.Processed++
		RecordPayoutSuccess(p.AmountCents)
	}
	return result, nil
}
