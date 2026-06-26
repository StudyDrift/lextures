package oersearch

import "strings"

// MatchesLicenseFilter returns true when result license satisfies the filter.
// filter "CC-BY" excludes NC, ND, and SA-only licenses.
func MatchesLicenseFilter(spdx, filter string) bool {
	f := strings.ToUpper(strings.TrimSpace(filter))
	if f == "" {
		return true
	}
	s := strings.ToUpper(strings.TrimSpace(spdx))
	if f == "CC-BY" {
		if strings.Contains(s, "NC") || strings.Contains(s, "ND") || strings.Contains(s, "SA") {
			return false
		}
		return strings.Contains(s, "CC-BY") || s == "CC0-1.0" || s == "CC0"
	}
	return strings.Contains(s, f)
}

// AllowsImportCopy reports whether license permits server-side copy import.
func AllowsImportCopy(spdx string) bool {
	s := strings.ToUpper(strings.TrimSpace(spdx))
	if strings.Contains(s, "NC") || strings.Contains(s, "ND") {
		return false
	}
	return strings.Contains(s, "CC-BY") || s == "CC0-1.0" || s == "CC0"
}
