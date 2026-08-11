// Coupon apply rate limits and cool-down (plan MKTC.7 FR-1, FR-2).
//
// Limits live here with named constants (not scattered magic numbers). Checks use
// in-memory buckets matching the billing checkout limiter shape. When Redis is
// unavailable the limiter fails open — only guessing capacity increases; coupon
// caps still live in Postgres.
package billing

import (
	"net"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Named limits (MKTC.7 FR-1 / FR-2).
const (
	// CouponApplyUserCoursePerMinute is the per-(user, course) minute cap.
	CouponApplyUserCoursePerMinute = 15
	// CouponApplyUserCoursePerHour is the per-(user, course) hour cap.
	CouponApplyUserCoursePerHour = 60
	// CouponApplyUserPerHour is the per-user cap across all courses.
	CouponApplyUserPerHour = 100
	// CouponApplyIPPerHour is the per-IP hour cap.
	CouponApplyIPPerHour = 200
	// CouponApplyConsecutiveFailLimit triggers a cool-down after this many
	// consecutive failed applies on the same (user, course) within an hour.
	CouponApplyConsecutiveFailLimit = 10
	// CouponApplyCooldownDuration is how long cool-down lasts after the fail threshold.
	CouponApplyCooldownDuration = 15 * time.Minute
	// CouponExportPerHour is the redemptions CSV export cap per user (FR-9).
	CouponExportPerHour = 5
)

// CouponApplyLimitReason distinguishes rate-limit vs cool-down for client copy.
type CouponApplyLimitReason string

const (
	// CouponApplyLimitedRate is a tiered rate-limit breach.
	CouponApplyLimitedRate CouponApplyLimitReason = "rate_limited"
	// CouponApplyLimitedCooldown is the consecutive-failure cool-down.
	CouponApplyLimitedCooldown CouponApplyLimitReason = "cooldown"
)

// CouponApplyLimitResult is the outcome of CheckCouponApplyLimit.
type CouponApplyLimitResult struct {
	Allowed    bool
	Reason     CouponApplyLimitReason
	RetryAfter time.Duration // zero when Allowed
}

type userCourseKey struct {
	userID   uuid.UUID
	courseID uuid.UUID
}

type windowBucket struct {
	minuteStart time.Time
	minuteCount int
	hourStart   time.Time
	hourCount   int
}

type failState struct {
	hourStart     time.Time
	consecutive   int
	cooldownUntil time.Time
}

type hourBucket struct {
	hourStart time.Time
	count     int
}

var (
	couponLimitMu sync.Mutex

	couponUCBuckets  = map[userCourseKey]windowBucket{}
	couponUserHour   = map[uuid.UUID]hourBucket{}
	couponIPHour     = map[string]hourBucket{}
	couponFailState  = map[userCourseKey]failState{}
	couponExportHour = map[uuid.UUID]hourBucket{}
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

// CheckCouponApplyLimit enforces layered apply limits and cool-down (FR-1, FR-2).
// On allow it increments the rate buckets. Cool-down is checked first and does not
// consume a rate slot when already cooling down.
func CheckCouponApplyLimit(userID, courseID uuid.UUID, ip string, now time.Time) CouponApplyLimitResult {
	if now.IsZero() {
		now = time.Now()
	}
	couponLimitMu.Lock()
	defer couponLimitMu.Unlock()

	key := userCourseKey{userID: userID, courseID: courseID}

	// Cool-down first (FR-2).
	if fs, ok := couponFailState[key]; ok {
		if !fs.cooldownUntil.IsZero() && now.Before(fs.cooldownUntil) {
			return CouponApplyLimitResult{
				Allowed:    false,
				Reason:     CouponApplyLimitedCooldown,
				RetryAfter: fs.cooldownUntil.Sub(now),
			}
		}
	}

	// Per (user, course) 15/min + 60/hour.
	uc := couponUCBuckets[key]
	if uc.minuteStart.IsZero() || now.Sub(uc.minuteStart) >= time.Minute {
		uc.minuteStart = now
		uc.minuteCount = 0
	}
	if uc.hourStart.IsZero() || now.Sub(uc.hourStart) >= time.Hour {
		uc.hourStart = now
		uc.hourCount = 0
	}
	if uc.minuteCount >= CouponApplyUserCoursePerMinute || uc.hourCount >= CouponApplyUserCoursePerHour {
		retry := remainingInWindow(now, uc.minuteStart, time.Minute)
		if uc.hourCount >= CouponApplyUserCoursePerHour {
			retry = remainingInWindow(now, uc.hourStart, time.Hour)
		}
		return CouponApplyLimitResult{Allowed: false, Reason: CouponApplyLimitedRate, RetryAfter: retry}
	}

	// Per user 100/hour across courses.
	uh := couponUserHour[userID]
	if uh.hourStart.IsZero() || now.Sub(uh.hourStart) >= time.Hour {
		uh.hourStart = now
		uh.count = 0
	}
	if uh.count >= CouponApplyUserPerHour {
		return CouponApplyLimitResult{
			Allowed:    false,
			Reason:     CouponApplyLimitedRate,
			RetryAfter: remainingInWindow(now, uh.hourStart, time.Hour),
		}
	}

	// Per IP 200/hour.
	ip = strings.TrimSpace(ip)
	if ip != "" {
		ih := couponIPHour[ip]
		if ih.hourStart.IsZero() || now.Sub(ih.hourStart) >= time.Hour {
			ih.hourStart = now
			ih.count = 0
		}
		if ih.count >= CouponApplyIPPerHour {
			return CouponApplyLimitResult{
				Allowed:    false,
				Reason:     CouponApplyLimitedRate,
				RetryAfter: remainingInWindow(now, ih.hourStart, time.Hour),
			}
		}
		ih.count++
		couponIPHour[ip] = ih
	}

	uc.minuteCount++
	uc.hourCount++
	couponUCBuckets[key] = uc
	uh.count++
	couponUserHour[userID] = uh

	return CouponApplyLimitResult{Allowed: true}
}

// RecordCouponApplyOutcome updates consecutive-failure / cool-down state (FR-2).
// success resets the counter; failure increments and may start a cool-down.
func RecordCouponApplyOutcome(userID, courseID uuid.UUID, success bool, now time.Time) {
	if now.IsZero() {
		now = time.Now()
	}
	couponLimitMu.Lock()
	defer couponLimitMu.Unlock()

	key := userCourseKey{userID: userID, courseID: courseID}
	fs := couponFailState[key]
	if fs.hourStart.IsZero() || now.Sub(fs.hourStart) >= time.Hour {
		fs.hourStart = now
		fs.consecutive = 0
		// Do not clear an active cool-down solely because the hour window rolled;
		// cool-down has its own absolute expiry.
		if !fs.cooldownUntil.IsZero() && !now.Before(fs.cooldownUntil) {
			fs.cooldownUntil = time.Time{}
		}
	}
	if success {
		fs.consecutive = 0
		fs.cooldownUntil = time.Time{}
		couponFailState[key] = fs
		return
	}
	fs.consecutive++
	if fs.consecutive >= CouponApplyConsecutiveFailLimit {
		fs.cooldownUntil = now.Add(CouponApplyCooldownDuration)
		fs.consecutive = 0
	}
	couponFailState[key] = fs
}

// CheckCouponExportRateLimit enforces 5 CSV exports per hour per user (FR-9).
func CheckCouponExportRateLimit(userID uuid.UUID, now time.Time) (allowed bool, retryAfter time.Duration) {
	if now.IsZero() {
		now = time.Now()
	}
	couponLimitMu.Lock()
	defer couponLimitMu.Unlock()
	b := couponExportHour[userID]
	if b.hourStart.IsZero() || now.Sub(b.hourStart) >= time.Hour {
		b.hourStart = now
		b.count = 0
	}
	if b.count >= CouponExportPerHour {
		return false, remainingInWindow(now, b.hourStart, time.Hour)
	}
	b.count++
	couponExportHour[userID] = b
	return true, 0
}

func remainingInWindow(now, windowStart time.Time, window time.Duration) time.Duration {
	if windowStart.IsZero() {
		return window
	}
	end := windowStart.Add(window)
	if !now.Before(end) {
		return time.Second
	}
	d := end.Sub(now)
	if d < time.Second {
		return time.Second
	}
	return d
}

// CouponAttemptCodeSalt is mixed into not_found code hashes so the attempt log
// cannot be mined for near-miss guesses (FR-3).
const CouponAttemptCodeSalt = "lextures-coupon-attempt-v1"

// HashCouponAttemptCode returns a stable salted hex hash of a normalized code.
// The raw code must never be stored for not_found results.
func HashCouponAttemptCode(code string) string {
	// Imported via crypto in the same package file — keep pure hashing here.
	return hashCouponCode(code, CouponAttemptCodeSalt)
}

// IPPrefix returns the privacy-preserving network prefix for attempt logs:
// /24 for IPv4, /48 for IPv6 (FR-3). Empty or unparseable input returns "".
func IPPrefix(ip string) string {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return ""
	}
	// Strip port if present (clientIP sometimes returns host:port).
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ""
	}
	if v4 := parsed.To4(); v4 != nil {
		mask := net.CIDRMask(24, 32)
		return v4.Mask(mask).String() + "/24"
	}
	mask := net.CIDRMask(48, 128)
	return parsed.Mask(mask).String() + "/48"
}
