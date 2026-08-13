package billing

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestDecodeAppleTransactionJWS_SkipVerify(t *testing.T) {
	claims := AppleTransactionClaims{
		TransactionID:         "1000000123456789",
		OriginalTransactionID: "1000000123456789",
		ProductID:             "com.lextures.ios.course.demo",
		BundleID:              "com.lextures.ios",
		Environment:           "Sandbox",
		AppAccountToken:       "11111111-1111-1111-1111-111111111111",
		PurchaseDate:          1_700_000_000_000,
		Price:                 2990, // $2.99 in milliunits? actually 2990 milli = $2.99 → 299 cents
		Currency:              "USD",
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	// Minimal unsigned JWS (header.payload.sig) for skip-verify path.
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"ES256","typ":"JWT"}`))
	body := base64.RawURLEncoding.EncodeToString(payload)
	signed := header + "." + body + ".fakesig"

	decoded, err := DecodeAppleTransactionJWS(signed, AppleIAPConfig{SkipSignatureVerify: true})
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.TransactionID != claims.TransactionID {
		t.Fatalf("transactionId: got %q", decoded.TransactionID)
	}
	if decoded.ProductID != claims.ProductID {
		t.Fatalf("productId: got %q", decoded.ProductID)
	}
	if decoded.BundleID != claims.BundleID {
		t.Fatalf("bundleId: got %q", decoded.BundleID)
	}
}

func TestDecodeAppleTransactionJWS_InvalidFormat(t *testing.T) {
	_, err := DecodeAppleTransactionJWS("not-a-jws", AppleIAPConfig{SkipSignatureVerify: true})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMilliunitsToCents(t *testing.T) {
	if got := milliunitsToCents(2990); got != 299 {
		t.Fatalf("2990 milli -> cents: got %d want 299", got)
	}
	if got := milliunitsToCents(0); got != 0 {
		t.Fatalf("0: got %d", got)
	}
}

func TestResolveAppleProduct_Subscriptions(t *testing.T) {
	cfg := AppleIAPConfig{
		MonthlyProductID: "com.lextures.ios.sub.monthly",
		AnnualProductID:  "com.lextures.ios.sub.annual",
	}
	kind, course, err := ResolveAppleProduct(t.Context(), nil, cfg, "com.lextures.ios.sub.monthly")
	if err != nil || kind != AppleProductSubscriptionMonthly || course != nil {
		t.Fatalf("monthly: kind=%q course=%v err=%v", kind, course, err)
	}
	kind, course, err = ResolveAppleProduct(t.Context(), nil, cfg, "com.lextures.ios.sub.annual")
	if err != nil || kind != AppleProductSubscriptionAnnual || course != nil {
		t.Fatalf("annual: kind=%q course=%v err=%v", kind, course, err)
	}
	_, _, err = ResolveAppleProduct(t.Context(), nil, cfg, "unknown.product")
	if err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("unknown product: err=%v", err)
	}
}

func TestListAppleProductsForCourse_EnvOnly(t *testing.T) {
	cfg := AppleIAPConfig{
		MonthlyProductID: "sub.m",
		AnnualProductID:  "sub.a",
	}
	list, err := ListAppleProductsForCourse(t.Context(), nil, cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("len=%d want 2", len(list))
	}
}

func TestAppleIAPConfigured(t *testing.T) {
	if AppleIAPConfigured(AppleIAPConfig{}) {
		t.Fatal("empty should be false")
	}
	if !AppleIAPConfigured(AppleIAPConfig{BundleID: "com.lextures.ios"}) {
		t.Fatal("bundle only should be true")
	}
}

func TestListAppleProductsForCourse_EmptyEnvUsesDefaultsWhenMapped(t *testing.T) {
	cfg := AppleIAPConfigFrom(config.Config{})
	if cfg.MonthlyProductID != DefaultAppleMonthlyProductID {
		t.Fatalf("monthly default: %q", cfg.MonthlyProductID)
	}
	if cfg.AnnualProductID != DefaultAppleAnnualProductID {
		t.Fatalf("annual default: %q", cfg.AnnualProductID)
	}
	list, err := ListAppleProductsForCourse(t.Context(), nil, cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("len=%d want 2", len(list))
	}
}
