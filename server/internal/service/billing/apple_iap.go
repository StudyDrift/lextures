// Package billing — Apple In-App Purchase verification and entitlement grant (App Store 3.1.1 Path A).
package billing

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/config"
	repoBilling "github.com/lextures/lextures/server/internal/repos/billing"
	"github.com/lextures/lextures/server/internal/telemetry"
)

// Apple IAP product kinds exposed to clients.
const (
	AppleProductCoursePurchase      = "course_purchase"
	AppleProductSubscriptionMonthly = "subscription_monthly"
	AppleProductSubscriptionAnnual  = "subscription_annual"
)

// AppleIAPConfig is process configuration for StoreKit verification.
type AppleIAPConfig struct {
	BundleID           string
	MonthlyProductID   string
	AnnualProductID    string
	// SkipSignatureVerify decodes JWS without cryptographic verification.
	// Allowed only when AppEnv is local (enforced by callers).
	SkipSignatureVerify bool
	// RootCAPEM is an optional PEM bundle of trusted Apple roots for x5c chain checks.
	// When empty, only the leaf certificate signature of the JWS is verified.
	RootCAPEM string
}

// AppleTransactionClaims are the relevant StoreKit 2 JWS transaction fields.
// See https://developer.apple.com/documentation/appstoreserverapi/jwstransactiondecodedpayload
type AppleTransactionClaims struct {
	TransactionID         string `json:"transactionId"`
	OriginalTransactionID string `json:"originalTransactionId"`
	ProductID             string `json:"productId"`
	BundleID              string `json:"bundleId"`
	Type                  string `json:"type"`
	Environment           string `json:"environment"`
	AppAccountToken       string `json:"appAccountToken"`
	PurchaseDate          int64  `json:"purchaseDate"`
	ExpiresDate           int64  `json:"expiresDate"`
	RevocationDate        int64  `json:"revocationDate"`
	Price                 int64  `json:"price"`    // milliunits in StoreKit 2
	Currency              string `json:"currency"` // ISO 4217
	Quantity              int    `json:"quantity"`
}

// AppleProductInfo is a client-facing product mapping.
type AppleProductInfo struct {
	ProductID       string  `json:"productId"`
	Kind            string  `json:"kind"`
	CourseID        *string `json:"courseId,omitempty"`
	DisplayHint     string  `json:"displayHint,omitempty"`
}

// AppleVerifyResult is returned after a successful verify+grant.
type AppleVerifyResult struct {
	EntitlementID   string  `json:"entitlementId"`
	EntitlementType string  `json:"entitlementType"`
	CourseID        *string `json:"courseId,omitempty"`
	TransactionID   string  `json:"transactionId"`
	ProductID       string  `json:"productId"`
	Created         bool    `json:"created"`
	AlreadyOwned    bool    `json:"alreadyOwned,omitempty"`
}

// DecodeAppleTransactionJWS verifies (unless skipped) and decodes a StoreKit 2 signed transaction.
func DecodeAppleTransactionJWS(signed string, cfg AppleIAPConfig) (*AppleTransactionClaims, error) {
	signed = strings.TrimSpace(signed)
	if signed == "" {
		return nil, errors.New("signedTransaction required")
	}
	parts := strings.Split(signed, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid JWS format")
	}

	if cfg.SkipSignatureVerify {
		payload, err := base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			return nil, fmt.Errorf("decode payload: %w", err)
		}
		var claims AppleTransactionClaims
		if err := json.Unmarshal(payload, &claims); err != nil {
			return nil, fmt.Errorf("parse claims: %w", err)
		}
		return &claims, nil
	}

	keyFunc := func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodES256.Alg() {
			return nil, fmt.Errorf("unexpected alg %s", token.Method.Alg())
		}
		x5c, ok := token.Header["x5c"].([]any)
		if !ok || len(x5c) == 0 {
			return nil, errors.New("missing x5c certificate chain")
		}
		leafDER, err := base64.StdEncoding.DecodeString(fmt.Sprint(x5c[0]))
		if err != nil {
			return nil, fmt.Errorf("decode leaf cert: %w", err)
		}
		leaf, err := x509.ParseCertificate(leafDER)
		if err != nil {
			return nil, fmt.Errorf("parse leaf cert: %w", err)
		}
		if cfg.RootCAPEM != "" {
			if err := verifyAppleCertChain(leaf, x5c, cfg.RootCAPEM); err != nil {
				return nil, err
			}
		}
		pub, ok := leaf.PublicKey.(*ecdsa.PublicKey)
		if !ok {
			return nil, errors.New("leaf cert is not ECDSA")
		}
		return pub, nil
	}

	tok, err := jwt.Parse(signed, keyFunc, jwt.WithValidMethods([]string{jwt.SigningMethodES256.Alg()}))
	if err != nil {
		return nil, fmt.Errorf("verify JWS: %w", err)
	}
	mapClaims, ok := tok.Claims.(jwt.MapClaims)
	if !ok || !tok.Valid {
		return nil, errors.New("invalid token claims")
	}
	raw, err := json.Marshal(mapClaims)
	if err != nil {
		return nil, err
	}
	var claims AppleTransactionClaims
	if err := json.Unmarshal(raw, &claims); err != nil {
		return nil, fmt.Errorf("parse claims: %w", err)
	}
	return &claims, nil
}

func verifyAppleCertChain(leaf *x509.Certificate, x5c []any, rootsPEM string) error {
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM([]byte(rootsPEM)) {
		return errors.New("invalid APPLE_IAP_ROOT_CA_PEM")
	}
	intermediates := x509.NewCertPool()
	for i := 1; i < len(x5c); i++ {
		der, err := base64.StdEncoding.DecodeString(fmt.Sprint(x5c[i]))
		if err != nil {
			continue
		}
		cert, err := x509.ParseCertificate(der)
		if err != nil {
			continue
		}
		intermediates.AddCert(cert)
	}
	_, err := leaf.Verify(x509.VerifyOptions{
		Roots:         roots,
		Intermediates: intermediates,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	})
	return err
}

// ResolveAppleProduct maps a StoreKit product id to entitlement kind + optional course.
func ResolveAppleProduct(ctx context.Context, pool *pgxpool.Pool, cfg AppleIAPConfig, productID string) (kind string, courseID *uuid.UUID, err error) {
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return "", nil, errors.New("productId required")
	}
	if cfg.MonthlyProductID != "" && productID == cfg.MonthlyProductID {
		return AppleProductSubscriptionMonthly, nil, nil
	}
	if cfg.AnnualProductID != "" && productID == cfg.AnnualProductID {
		return AppleProductSubscriptionAnnual, nil, nil
	}
	if pool != nil {
		cid, err := repoBilling.CourseAppleProduct(ctx, pool, productID)
		if err != nil {
			return "", nil, err
		}
		if cid != nil {
			return AppleProductCoursePurchase, cid, nil
		}
	}
	return "", nil, fmt.Errorf("unknown Apple product id %q", productID)
}

// ListAppleProductsForCourse returns IAP product ids the client should load for a course purchase.
// pool may be nil when only env-configured subscription products are needed.
func ListAppleProductsForCourse(ctx context.Context, pool *pgxpool.Pool, cfg AppleIAPConfig, courseID *uuid.UUID) ([]AppleProductInfo, error) {
	out := make([]AppleProductInfo, 0, 3)
	if cfg.MonthlyProductID != "" {
		out = append(out, AppleProductInfo{
			ProductID:   cfg.MonthlyProductID,
			Kind:        AppleProductSubscriptionMonthly,
			DisplayHint: "subscription_monthly",
		})
	}
	if cfg.AnnualProductID != "" {
		out = append(out, AppleProductInfo{
			ProductID:   cfg.AnnualProductID,
			Kind:        AppleProductSubscriptionAnnual,
			DisplayHint: "subscription_annual",
		})
	}
	if courseID != nil && pool != nil {
		pid, err := repoBilling.CourseAppleProductID(ctx, pool, *courseID)
		if err != nil {
			return nil, err
		}
		if pid != "" {
			s := courseID.String()
			out = append(out, AppleProductInfo{
				ProductID:   pid,
				Kind:        AppleProductCoursePurchase,
				CourseID:    &s,
				DisplayHint: "course_purchase",
			})
		}
	}
	return out, nil
}

// AppleIAPConfigured reports whether the server can verify StoreKit purchases.
func AppleIAPConfigured(cfg AppleIAPConfig) bool {
	return strings.TrimSpace(cfg.BundleID) != ""
}

// AppleIAPConfigFrom maps process config into AppleIAPConfig.
func AppleIAPConfigFrom(cfg config.Config) AppleIAPConfig {
	skip := cfg.AppleIAPSkipSignatureVerify
	// Never skip cryptographic verification outside local/dev.
	if skip && !strings.EqualFold(strings.TrimSpace(cfg.AppEnv), "local") {
		skip = false
	}
	return AppleIAPConfig{
		BundleID:            strings.TrimSpace(cfg.AppleIAPBundleID),
		MonthlyProductID:    strings.TrimSpace(cfg.AppleIAPMonthlyProductID),
		AnnualProductID:     strings.TrimSpace(cfg.AppleIAPAnnualProductID),
		SkipSignatureVerify: skip,
		RootCAPEM:           cfg.AppleIAPRootCAPEM,
	}
}

// VerifyAndGrantAppleIAP validates a signed transaction and creates the matching entitlement.
func VerifyAndGrantAppleIAP(
	ctx context.Context,
	pool *pgxpool.Pool,
	cfg AppleIAPConfig,
	userID uuid.UUID,
	signedTransaction string,
	expectedCourseID *uuid.UUID,
) (*AppleVerifyResult, error) {
	claims, err := DecodeAppleTransactionJWS(signedTransaction, cfg)
	if err != nil {
		return nil, err
	}
	if claims.TransactionID == "" || claims.ProductID == "" {
		return nil, errors.New("transaction missing id or productId")
	}
	if claims.RevocationDate > 0 {
		return nil, errors.New("transaction was revoked")
	}
	if cfg.BundleID != "" && claims.BundleID != "" && claims.BundleID != cfg.BundleID {
		return nil, fmt.Errorf("bundleId mismatch: got %q want %q", claims.BundleID, cfg.BundleID)
	}
	// Optional binding: client sets appAccountToken to the Lextures user UUID at purchase time.
	if tok := strings.TrimSpace(claims.AppAccountToken); tok != "" {
		if appUser, err := uuid.Parse(tok); err == nil && appUser != userID {
			return nil, errors.New("appAccountToken does not match signed-in user")
		}
	}

	kind, mappedCourseID, err := ResolveAppleProduct(ctx, pool, cfg, claims.ProductID)
	if err != nil {
		return nil, err
	}
	if kind == AppleProductCoursePurchase {
		if mappedCourseID == nil {
			return nil, errors.New("course product not mapped")
		}
		if expectedCourseID != nil && *expectedCourseID != *mappedCourseID {
			return nil, errors.New("courseId does not match product")
		}
	}

	amountCents := milliunitsToCents(claims.Price)
	currency := strings.ToLower(strings.TrimSpace(claims.Currency))
	if currency == "" {
		currency = "usd"
	}

	var purchasedAt *time.Time
	if claims.PurchaseDate > 0 {
		t := time.UnixMilli(claims.PurchaseDate).UTC()
		purchasedAt = &t
	}
	var expiresAt *time.Time
	if claims.ExpiresDate > 0 {
		t := time.UnixMilli(claims.ExpiresDate).UTC()
		expiresAt = &t
	} else if kind == AppleProductSubscriptionMonthly {
		t := time.Now().UTC().AddDate(0, 1, 0)
		expiresAt = &t
	} else if kind == AppleProductSubscriptionAnnual {
		t := time.Now().UTC().AddDate(1, 0, 0)
		expiresAt = &t
	}

	idempotencyKey := "apple:" + claims.TransactionID
	var ent *repoBilling.Entitlement
	var created bool

	// Use CreateIdempotent for all Apple grants so retries key on apple:<transactionId>.
	var courseArg *uuid.UUID
	if kind == AppleProductCoursePurchase {
		courseArg = mappedCourseID
	}
	ent, created, err = repoBilling.CreateIdempotent(ctx, pool, repoBilling.CreateInput{
		UserID:            userID,
		EntitlementType:   kind,
		CourseID:          courseArg,
		StripeEventID:     idempotencyKey,
		AmountPaidCents:   amountCents,
		Currency:          currency,
		ValidUntil:        expiresAt,
		AcquisitionSource: repoBilling.AcquisitionApple,
	})
	if err != nil {
		return nil, err
	}
	if created {
		RecordPayment(amountCents, kind)
		if kind == AppleProductCoursePurchase && mappedCourseID != nil {
			if err := enrollCoursePurchase(ctx, pool, userID, *mappedCourseID); err != nil {
				return nil, err
			}
			if amountCents > 0 {
				telemetry.RecordMarketplacePurchaseCompleted()
			}
		}
	}

	claimsJSON, _ := json.Marshal(claims)
	sum := sha256.Sum256([]byte(signedTransaction))
	var entID *uuid.UUID
	if ent != nil {
		entID = &ent.ID
	}
	_, _, _ = repoBilling.InsertAppleIAPTransactionIdempotent(ctx, pool, repoBilling.AppleIAPTransactionInput{
		UserID:                userID,
		TransactionID:         claims.TransactionID,
		OriginalTransactionID: firstNonEmpty(claims.OriginalTransactionID, claims.TransactionID),
		ProductID:             claims.ProductID,
		BundleID:              firstNonEmpty(claims.BundleID, cfg.BundleID),
		Environment:           firstNonEmpty(claims.Environment, "Production"),
		EntitlementID:         entID,
		SignedPayloadSHA256:   hex.EncodeToString(sum[:]),
		PurchasedAt:           purchasedAt,
		ExpiresAt:             expiresAt,
		RawClaims:             claimsJSON,
	})

	result := &AppleVerifyResult{
		TransactionID:   claims.TransactionID,
		ProductID:       claims.ProductID,
		EntitlementType: kind,
		Created:         created,
	}
	if ent != nil {
		result.EntitlementID = ent.ID.String()
		if !created {
			result.AlreadyOwned = true
		}
	}
	if mappedCourseID != nil {
		s := mappedCourseID.String()
		result.CourseID = &s
	}
	return result, nil
}

func milliunitsToCents(milli int64) int {
	if milli <= 0 {
		return 0
	}
	// StoreKit 2 price is in milliunits (1/1000 of currency unit); cents = milli/10 for 2-decimal currencies.
	return int(math.Round(float64(milli) / 10.0))
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

