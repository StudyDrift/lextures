package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/config"
	repoBilling "github.com/lextures/lextures/server/internal/repos/billing"
	"github.com/lextures/lextures/server/internal/repos/course"
)

func TestCouponsFeatureOff_RequiresBothFlags(t *testing.T) {
	cases := []struct {
		name string
		cfg  config.Config
		off  bool
	}{
		{"both off", config.Config{FFCourseMarketplace: false, FFCourseCoupons: false}, true},
		{"marketplace only", config.Config{FFCourseMarketplace: true, FFCourseCoupons: false}, true},
		{"coupons only", config.Config{FFCourseMarketplace: false, FFCourseCoupons: true}, true},
		{"both on", config.Config{FFCourseMarketplace: true, FFCourseCoupons: true}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := Deps{Config: tc.cfg}
			rr := httptest.NewRecorder()
			got := d.couponsFeatureOff(rr)
			if got != tc.off {
				t.Fatalf("couponsFeatureOff=%v want %v (body %s)", got, tc.off, rr.Body.String())
			}
			if tc.off && rr.Code != http.StatusNotFound {
				t.Fatalf("status: got %d want 404", rr.Code)
			}
		})
	}
}

func TestCouponShareURLs_SlugFallbackAndPublic(t *testing.T) {
	share, pub := couponShareURLs("https://app.example.edu", "https://lextures.com", "CS101", "intro-cs", true, "LAUNCH25")
	if !strings.HasSuffix(share, "/marketplace/intro-cs?coupon=LAUNCH25") {
		t.Fatalf("shareUrl: %s", share)
	}
	if !strings.HasPrefix(share, "https://app.example.edu") {
		t.Fatalf("share origin: %s", share)
	}
	if pub == nil || *pub != "https://lextures.com/courses/intro-cs?coupon=LAUNCH25" {
		t.Fatalf("publicShareUrl: %v", pub)
	}

	// Fallback to courseCode when slug empty.
	share2, pub2 := couponShareURLs("https://app.example.edu/", "https://lextures.com/", "CS101", "", false, "SAVE10")
	if !strings.Contains(share2, "/marketplace/CS101?coupon=SAVE10") {
		t.Fatalf("share fallback: %s", share2)
	}
	if pub2 != nil {
		t.Fatalf("expected null publicShareUrl when not public, got %v", *pub2)
	}
}

func TestCouponShareURLs_URLEncodesCode(t *testing.T) {
	// Codes are A–Z0–9_- so encoding is identity for normal codes; still use QueryEscape path.
	share, _ := couponShareURLs("https://app.test", "https://www.test", "c1", "slug", false, "A_B-1")
	if !strings.Contains(share, "coupon=A_B-1") {
		t.Fatalf("share: %s", share)
	}
}

func TestCouponToJSON_SeatsRemainingNullWhenUnlimited(t *testing.T) {
	id := uuid.New()
	courseID := uuid.New()
	c := &repoBilling.Coupon{
		ID:                    id,
		CourseID:              courseID,
		Code:                  "FREE50",
		DiscountType:          "percent",
		PercentOff:            floatPtr(50),
		MaxRedemptions:        nil,
		MaxRedemptionsPerUser: 1,
		Status:                "active",
		CreatedAt:             time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		UpdatedAt:             time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}
	dto := couponToJSON(c, repoBilling.SeatCount{Consumed: 3, Reserved: 1, Redeemed: 2}, "https://x/marketplace/s?coupon=FREE50", nil, nil)
	if dto.Seats.Remaining != nil {
		t.Fatalf("remaining want null, got %v", *dto.Seats.Remaining)
	}
	if dto.Seats.Consumed != 3 || dto.Seats.Reserved != 1 || dto.Seats.Redeemed != 2 {
		t.Fatalf("seats: %+v", dto.Seats)
	}
	raw, _ := json.Marshal(dto)
	if !strings.Contains(string(raw), `"remaining":null`) {
		t.Fatalf("json should include remaining:null: %s", raw)
	}
}

func TestCouponToJSON_RemainingCappedAtZero(t *testing.T) {
	max := 5
	c := &repoBilling.Coupon{
		ID:                    uuid.New(),
		CourseID:              uuid.New(),
		Code:                  "CAP5",
		DiscountType:          "percent",
		PercentOff:            floatPtr(10),
		MaxRedemptions:        &max,
		MaxRedemptionsPerUser: 1,
		Status:                "active",
		CreatedAt:             time.Now().UTC(),
		UpdatedAt:             time.Now().UTC(),
	}
	dto := couponToJSON(c, repoBilling.SeatCount{Consumed: 8}, "https://x/?coupon=CAP5", nil, []string{"clamps_to_free"})
	if dto.Seats.Remaining == nil || *dto.Seats.Remaining != 0 {
		t.Fatalf("remaining want 0, got %v", dto.Seats.Remaining)
	}
	if len(dto.Warnings) != 1 || dto.Warnings[0] != "clamps_to_free" {
		t.Fatalf("warnings: %v", dto.Warnings)
	}
}

func TestParseOptionalRFC3339(t *testing.T) {
	raw := "2026-06-01T12:00:00Z"
	got, err := parseOptionalRFC3339("startsAt", &raw)
	if err != nil || got == nil {
		t.Fatalf("parse: %v %v", got, err)
	}
	if !got.Equal(time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("got %v", got)
	}
	bad := "not-a-date"
	if _, err := parseOptionalRFC3339("endsAt", &bad); err == nil {
		t.Fatal("expected error for bad date")
	}
}

func TestCouponClampsToFree(t *testing.T) {
	listing := &course.CatalogListing{PriceCents: 100, PriceCurrency: "usd"}
	// 99% of $1.00 → residual 1 cent, below Stripe min → clamps to free.
	c := &repoBilling.Coupon{
		DiscountType: "percent",
		PercentOff:   floatPtr(99),
		Status:       "active",
	}
	if !couponClampsToFree(listing, c) {
		t.Fatal("expected clamps_to_free for near-100% on low price")
	}
	c2 := &repoBilling.Coupon{DiscountType: "percent", PercentOff: floatPtr(10)}
	if couponClampsToFree(listing, c2) {
		t.Fatal("10% of $1 should not clamp to free")
	}
}

func TestCourseCoupons_Unauthenticated_Returns401(t *testing.T) {
	h := NewHandler(Deps{
		Pool:      nil,
		Config:    config.Config{FFCourseMarketplace: true, FFCourseCoupons: true},
		JWTSigner: nil,
	})
	cid := uuid.NewString()
	cases := []struct {
		method, path string
	}{
		{http.MethodGet, "/api/v1/courses/CS101/coupons"},
		{http.MethodPost, "/api/v1/courses/CS101/coupons"},
		{http.MethodGet, "/api/v1/courses/CS101/coupons/summary"},
		{http.MethodPatch, "/api/v1/courses/CS101/coupons/" + cid},
		{http.MethodDelete, "/api/v1/courses/CS101/coupons/" + cid},
		{http.MethodGet, "/api/v1/courses/CS101/coupons/" + cid + "/redemptions"},
		{http.MethodGet, "/api/v1/courses/CS101/coupons/" + cid + "/redemptions.csv"},
	}
	for _, tc := range cases {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, nil)
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s: got %d want 401 (body %s)", tc.method, tc.path, rr.Code, rr.Body.String())
		}
	}
}

func TestFormatPercentCeiling(t *testing.T) {
	if got := formatPercentCeiling(50); got != "50" {
		t.Fatalf("got %s", got)
	}
	if got := formatPercentCeiling(33.5); got != "33.50" {
		t.Fatalf("got %s", got)
	}
}

func floatPtr(f float64) *float64 { return &f }
