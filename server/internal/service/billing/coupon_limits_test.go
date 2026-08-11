package billing

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// resetCouponLimitsForTest clears all in-memory coupon limit state (tests only).
func resetCouponLimitsForTest() {
	couponLimitMu.Lock()
	defer couponLimitMu.Unlock()
	couponUCBuckets = map[userCourseKey]windowBucket{}
	couponUserHour = map[uuid.UUID]hourBucket{}
	couponIPHour = map[string]hourBucket{}
	couponFailState = map[userCourseKey]failState{}
	couponExportHour = map[uuid.UUID]hourBucket{}
}

func TestCheckCouponApplyLimit_PerUserCourseMinute(t *testing.T) {
	resetCouponLimitsForTest()
	user := uuid.New()
	course := uuid.New()
	now := time.Now()

	for i := 0; i < CouponApplyUserCoursePerMinute; i++ {
		r := CheckCouponApplyLimit(user, course, "203.0.113.10", now)
		if !r.Allowed {
			t.Fatalf("attempt %d should allow: %+v", i+1, r)
		}
	}
	r := CheckCouponApplyLimit(user, course, "203.0.113.10", now)
	if r.Allowed || r.Reason != CouponApplyLimitedRate {
		t.Fatalf("16th should rate-limit: %+v", r)
	}
	if r.RetryAfter <= 0 {
		t.Fatal("expected RetryAfter > 0")
	}
}

func TestCheckCouponApplyLimit_PerUserHourAcrossCourses(t *testing.T) {
	resetCouponLimitsForTest()
	user := uuid.New()
	now := time.Now()
	// Use different courses so per-(user,course) hour (60) is not the bottleneck.
	// 100/hour per user: 50 courses × 2 applies each.
	for i := 0; i < CouponApplyUserPerHour; i++ {
		course := uuid.New()
		r := CheckCouponApplyLimit(user, course, "203.0.113.20", now)
		if !r.Allowed {
			t.Fatalf("attempt %d should allow: %+v", i+1, r)
		}
	}
	r := CheckCouponApplyLimit(user, uuid.New(), "203.0.113.20", now)
	if r.Allowed || r.Reason != CouponApplyLimitedRate {
		t.Fatalf("101st should rate-limit: %+v", r)
	}
}

func TestCheckCouponApplyLimit_PerIPHour(t *testing.T) {
	resetCouponLimitsForTest()
	now := time.Now()
	ip := "198.51.100.50"
	for i := 0; i < CouponApplyIPPerHour; i++ {
		r := CheckCouponApplyLimit(uuid.New(), uuid.New(), ip, now)
		if !r.Allowed {
			t.Fatalf("attempt %d should allow: %+v", i+1, r)
		}
	}
	r := CheckCouponApplyLimit(uuid.New(), uuid.New(), ip, now)
	if r.Allowed || r.Reason != CouponApplyLimitedRate {
		t.Fatalf("201st from same IP should rate-limit: %+v", r)
	}
}

func TestRecordCouponApplyOutcome_Cooldown(t *testing.T) {
	resetCouponLimitsForTest()
	user := uuid.New()
	course := uuid.New()
	now := time.Now()

	for i := 0; i < CouponApplyConsecutiveFailLimit; i++ {
		// Consume a rate slot then record failure.
		if r := CheckCouponApplyLimit(user, course, "203.0.113.1", now); !r.Allowed {
			t.Fatalf("setup apply %d blocked: %+v", i+1, r)
		}
		RecordCouponApplyOutcome(user, course, false, now)
	}

	// 11th attempt — even before another fail record — must cool down.
	r := CheckCouponApplyLimit(user, course, "203.0.113.1", now)
	if r.Allowed || r.Reason != CouponApplyLimitedCooldown {
		t.Fatalf("expected cooldown: %+v", r)
	}
	if r.RetryAfter < 14*time.Minute {
		t.Fatalf("cooldown retry too short: %v", r.RetryAfter)
	}

	// Success before threshold resets (fresh keys so minute bucket is not exhausted).
	resetCouponLimitsForTest()
	user2 := uuid.New()
	course2 := uuid.New()
	for i := 0; i < CouponApplyConsecutiveFailLimit-1; i++ {
		if res := CheckCouponApplyLimit(user2, course2, "203.0.113.2", now); !res.Allowed {
			t.Fatalf("setup fail %d blocked: %+v", i+1, res)
		}
		RecordCouponApplyOutcome(user2, course2, false, now)
	}
	if res := CheckCouponApplyLimit(user2, course2, "203.0.113.2", now); !res.Allowed {
		t.Fatalf("success path apply blocked: %+v", res)
	}
	RecordCouponApplyOutcome(user2, course2, true, now)
	// After success, more fails should not immediately cool down.
	// Use a new minute window so the 15/min tier is not the bottleneck.
	later := now.Add(2 * time.Minute)
	for i := 0; i < CouponApplyConsecutiveFailLimit-1; i++ {
		if res := CheckCouponApplyLimit(user2, course2, "203.0.113.2", later); !res.Allowed {
			t.Fatalf("post-success fail %d blocked: %+v", i+1, res)
		}
		RecordCouponApplyOutcome(user2, course2, false, later)
	}
	r = CheckCouponApplyLimit(user2, course2, "203.0.113.2", later)
	if !r.Allowed {
		t.Fatalf("after success reset should still allow before 10 fails: %+v", r)
	}
}

func TestHashCouponAttemptCode_StableSalted(t *testing.T) {
	a := HashCouponAttemptCode("launch25")
	b := HashCouponAttemptCode("LAUNCH25")
	if a != b {
		t.Fatalf("normalize: %s vs %s", a, b)
	}
	if a == "" || len(a) != 64 {
		t.Fatalf("expected sha256 hex, got %q", a)
	}
	if strings.Contains(strings.ToLower(a), "launch") {
		t.Fatal("hash must not contain raw code")
	}
	// Different salt path: known code must not equal unsalted form.
	if a == hashCouponCode("LAUNCH25", "") {
		t.Fatal("salt must change digest")
	}
}

func TestIPPrefix(t *testing.T) {
	if got := IPPrefix("203.0.113.45"); got != "203.0.113.0/24" {
		t.Fatalf("ipv4: %s", got)
	}
	if got := IPPrefix("203.0.113.45:54321"); got != "203.0.113.0/24" {
		t.Fatalf("ipv4 with port: %s", got)
	}
	// 2001:db8:abcd:0012:0000:0000:0000:0001 → /48
	got := IPPrefix("2001:db8:abcd:12::1")
	if !strings.HasSuffix(got, "/48") {
		t.Fatalf("ipv6 suffix: %s", got)
	}
	if !strings.HasPrefix(got, "2001:db8:abcd:") && !strings.HasPrefix(got, "2001:db8:abcd::") {
		// Accept either compressed form of the network.
		if !strings.Contains(got, "2001:db8:abcd") {
			t.Fatalf("ipv6 prefix: %s", got)
		}
	}
	if IPPrefix("") != "" || IPPrefix("not-an-ip") != "" {
		t.Fatal("empty/unparseable should return empty")
	}
}

func TestCheckCouponExportRateLimit(t *testing.T) {
	resetCouponLimitsForTest()
	user := uuid.New()
	now := time.Now()
	for i := 0; i < CouponExportPerHour; i++ {
		ok, _ := CheckCouponExportRateLimit(user, now)
		if !ok {
			t.Fatalf("export %d should allow", i+1)
		}
	}
	ok, retry := CheckCouponExportRateLimit(user, now)
	if ok || retry <= 0 {
		t.Fatalf("6th export should limit: ok=%v retry=%v", ok, retry)
	}
}
