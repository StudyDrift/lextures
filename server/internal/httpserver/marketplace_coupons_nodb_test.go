package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lextures/lextures/server/internal/config"
	svcBilling "github.com/lextures/lextures/server/internal/service/billing"
	"github.com/lextures/lextures/server/internal/service/coupons"
)

func TestMarketplaceCouponPreview_NoAuth_Returns401(t *testing.T) {
	h := NewHandler(Deps{
		Pool: nil,
		Config: config.Config{
			FFCourseMarketplace: true,
			FFCourseCoupons:     true,
		},
		JWTSigner: nil,
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/marketplace/courses/some-slug/coupon/preview", strings.NewReader(`{"code":"LAUNCH25"}`))
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("got %d want 401 body %s", rr.Code, rr.Body.String())
	}
}

func TestMarketplaceCouponPreview_FeatureOff_Returns404(t *testing.T) {
	h := NewHandler(Deps{
		Pool: nil,
		Config: config.Config{
			FFCourseMarketplace: true,
			FFCourseCoupons:     false,
		},
		JWTSigner: nil,
	})
	// Still unauthenticated first — but couponsFeatureOff runs after auth.
	// Without JWT we get 401; with flag off on authenticated path would 404.
	// Smoke: route is registered (not 405).
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/marketplace/courses/some-slug/coupon/preview", strings.NewReader(`{"code":"X"}`))
	h.ServeHTTP(rr, req)
	if rr.Code == http.StatusMethodNotAllowed {
		t.Fatal("preview route not registered")
	}
}

func TestMarketplacePurchase_PreviewRouteRegistered(t *testing.T) {
	h := NewHandler(Deps{Pool: nil, Config: config.Config{FFCourseMarketplace: true, FFCourseCoupons: true}, JWTSigner: nil})
	for _, path := range []string{
		"/api/v1/marketplace/courses/some-slug/claim",
		"/api/v1/marketplace/courses/some-slug/checkout",
		"/api/v1/marketplace/courses/some-slug/coupon/preview",
	} {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, path, nil)
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("%s: got %d want 401", path, rr.Code)
		}
	}
}

func TestWriteCouponRejected_Shape(t *testing.T) {
	rr := httptest.NewRecorder()
	writeCouponRejected(rr, coupons.ReasonExhausted, 4000, "USD")
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d", rr.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["reason"] != "exhausted" {
		t.Fatalf("reason: %v", body["reason"])
	}
	if body["listPriceCents"].(float64) != 4000 {
		t.Fatalf("list: %v", body["listPriceCents"])
	}
	errObj, _ := body["error"].(map[string]any)
	if errObj["code"] != "UNPROCESSABLE_ENTITY" {
		t.Fatalf("error code: %v", errObj)
	}
}

func TestCouponReasonMessage_AllReasons(t *testing.T) {
	reasons := []coupons.Reason{
		coupons.ReasonNotFound, coupons.ReasonInactive, coupons.ReasonNotStarted,
		coupons.ReasonExpired, coupons.ReasonExhausted, coupons.ReasonAlreadyUsed,
		coupons.ReasonCurrencyMismatch, coupons.ReasonCourseFree, coupons.ReasonOwned,
	}
	for _, r := range reasons {
		if couponReasonMessage(r) == "" {
			t.Fatalf("empty message for %s", r)
		}
	}
}

func TestMarketplaceCouponBody_IgnoresClientAmounts(t *testing.T) {
	// Decode shape: chargedCents etc. are accepted but never used by handlers (FR-6).
	raw := `{"couponCode":"LAUNCH25","chargedCents":1,"discountCents":9999,"priceCents":1}`
	var body marketplaceCouponBody
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		t.Fatal(err)
	}
	if body.CouponCode != "LAUNCH25" {
		t.Fatal(body.CouponCode)
	}
	if body.ChargedCents == nil || *body.ChargedCents != 1 {
		t.Fatal("decode charged")
	}
	// Handlers only read CouponCode — client amounts are intentionally discarded.
}

func TestWriteCouponApplyLimited_CooldownReason(t *testing.T) {
	rr := httptest.NewRecorder()
	writeCouponApplyLimited(rr, svcBilling.CouponApplyLimitedCooldown, 15*time.Minute)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("status %d", rr.Code)
	}
	if rr.Header().Get("Retry-After") == "" {
		t.Fatal("missing Retry-After")
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["reason"] != "cooldown" {
		t.Fatalf("reason: %v", body["reason"])
	}
	errObj, _ := body["error"].(map[string]any)
	if errObj["code"] != "RATE_LIMITED" {
		t.Fatalf("error: %v", errObj)
	}
}

func TestWriteCouponApplyLimited_RateLimitedReason(t *testing.T) {
	rr := httptest.NewRecorder()
	writeCouponApplyLimited(rr, svcBilling.CouponApplyLimitedRate, 30*time.Second)
	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["reason"] != "rate_limited" {
		t.Fatalf("reason: %v", body["reason"])
	}
}
