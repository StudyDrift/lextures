package demographics

import "strings"

// LunchFlags holds parsed free/reduced lunch eligibility from a SIS lunch code.
type LunchFlags struct {
	FreeLunch    *bool
	ReducedLunch *bool
}

// MapLunchCode normalizes SIS lunch codes to free/reduced flags.
// Known codes: F/Free → free lunch; R/Reduced → reduced lunch; N/None/Paid → neither.
func MapLunchCode(code string) LunchFlags {
	c := strings.ToUpper(strings.TrimSpace(code))
	switch c {
	case "F", "FREE", "FL":
		return LunchFlags{FreeLunch: boolPtr(true), ReducedLunch: boolPtr(false)}
	case "R", "REDUCED", "RL":
		return LunchFlags{FreeLunch: boolPtr(false), ReducedLunch: boolPtr(true)}
	case "N", "NONE", "PAID", "P", "":
		f, r := false, false
		return LunchFlags{FreeLunch: &f, ReducedLunch: &r}
	default:
		return LunchFlags{}
	}
}

func boolPtr(v bool) *bool { return &v }
