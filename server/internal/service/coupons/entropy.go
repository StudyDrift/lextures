// Low-entropy code warnings for creator create responses (plan MKTC.7 FR-4).
package coupons

import "strings"

// WarningLowEntropy is returned on create when a code is short or dictionary-like.
const WarningLowEntropy = "low_entropy"

// commonCouponDictionary is a small list of obvious guessable words. Not exhaustive;
// the goal is a non-blocking creator hint, not a perfect denylist.
var commonCouponDictionary = map[string]struct{}{
	"FREE": {}, "FREEBIE": {}, "DISCOUNT": {}, "SAVE": {}, "SALE": {},
	"LAUNCH": {}, "WELCOME": {}, "VIP": {}, "TEST": {}, "DEMO": {},
	"COUPON": {}, "PROMO": {}, "DEAL": {}, "OFFER": {}, "SUMMER": {},
	"WINTER": {}, "SPRING": {}, "FALL": {}, "AUTUMN": {}, "HOLIDAY": {},
	"XMAS": {}, "CHRISTMAS": {}, "NEWYEAR": {}, "STUDENT": {}, "TEACHER": {},
	"ADMIN": {}, "PASSWORD": {}, "SECRET": {}, "OPEN": {}, "PUBLIC": {},
	"HOMESCHOOL": {}, "LEXTURES": {}, "STUDY": {}, "LEARN": {}, "CLASS": {},
}

// LowEntropyWarnings returns ["low_entropy"] when the (already normalized) code is
// under 6 characters or appears on the simple dictionary list. Empty otherwise.
func LowEntropyWarnings(code string) []string {
	c := strings.ToUpper(strings.TrimSpace(code))
	if c == "" {
		return nil
	}
	if len(c) < 6 {
		return []string{WarningLowEntropy}
	}
	if _, ok := commonCouponDictionary[c]; ok {
		return []string{WarningLowEntropy}
	}
	return nil
}
