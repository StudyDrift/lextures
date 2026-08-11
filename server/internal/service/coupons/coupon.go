// Package coupons is the pure discount engine for course coupon codes (plan MKTC.1).
// It has no database dependency so callers and unit tests can evaluate eligibility
// and amounts without a pool.
package coupons

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

// Kind is the discount arithmetic mode.
type Kind string

const (
	KindPercent Kind = "percent"
	KindFixed   Kind = "fixed"
)

// Coupon statuses (billing.course_coupons.status).
const (
	StatusActive   = "active"
	StatusDisabled = "disabled"
	StatusArchived = "archived"
)

// codeShape is the post-normalization code pattern (FR-2).
var codeShape = regexp.MustCompile(`^[A-Z0-9][A-Z0-9_-]{3,31}$`)

// Coupon is the pure-domain view of a course coupon (no SQL types).
type Coupon struct {
	ID                    uuid.UUID
	CourseID              uuid.UUID
	Code                  string
	Kind                  Kind
	PercentOff            float64 // 0 when fixed
	AmountOffCents        int     // 0 when percent
	Currency              string  // set when fixed
	StartsAt              *time.Time
	EndsAt                *time.Time
	MaxRedemptions        *int
	MaxRedemptionsPerUser int
	Status                string
}

// Quote is the result of applying a coupon to a list price.
type Quote struct {
	ListCents     int
	DiscountCents int
	ChargedCents  int
	Currency      string
	ClampedToFree bool
}

// Reason is a stable machine token for eligibility outcomes (FR-13).
type Reason string

const (
	ReasonOK               Reason = "ok"
	ReasonNotFound         Reason = "not_found"
	ReasonInactive         Reason = "inactive"
	ReasonNotStarted       Reason = "not_started"
	ReasonExpired          Reason = "expired"
	ReasonExhausted        Reason = "exhausted"
	ReasonAlreadyUsed      Reason = "already_used"
	ReasonCurrencyMismatch Reason = "currency_mismatch"
	ReasonCourseFree       Reason = "course_free"
	ReasonOwned            Reason = "owned"
)

// EvalContext carries live course/user state for Evaluate (no DB handle).
type EvalContext struct {
	Now            time.Time
	CoursePrice    int
	CourseCurrency string
	ConsumedSeats  int
	UserSeats      int
	AlreadyOwned   bool
}

// ErrInvalidCode is returned by ValidateCode when the shape is wrong.
var ErrInvalidCode = errors.New("coupons: invalid code shape")

// NormalizeCode trims, upper-cases, and removes internal whitespace (FR-2).
// It does not validate shape — call ValidateCode after normalizing for inserts.
func NormalizeCode(raw string) string {
	if raw == "" {
		return ""
	}
	// Cap pathological inputs early (fuzz / abuse).
	if len(raw) > 10_000 {
		raw = raw[:10_000]
	}
	var b strings.Builder
	b.Grow(len(raw))
	for _, r := range raw {
		if unicode.IsSpace(r) {
			continue
		}
		b.WriteRune(unicode.ToUpper(r))
	}
	return b.String()
}

// ValidateCode checks a *normalized* code against the CHECK shape.
func ValidateCode(code string) error {
	if !codeShape.MatchString(code) {
		return fmt.Errorf("%w: must be 4–32 chars matching [A-Z0-9][A-Z0-9_-]*", ErrInvalidCode)
	}
	return nil
}
