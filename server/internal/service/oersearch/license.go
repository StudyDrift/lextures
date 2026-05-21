package oersearch

import "strings"

// LicenseLabel returns a human-readable CC license name from SPDX.
func LicenseLabel(spdx string) string {
	switch strings.ToUpper(strings.TrimSpace(spdx)) {
	case "CC-BY-4.0", "CC-BY-3.0", "CC-BY-2.0":
		return "CC BY"
	case "CC-BY-SA-4.0", "CC-BY-SA-3.0":
		return "CC BY-SA"
	case "CC-BY-NC-4.0", "CC-BY-NC-3.0":
		return "CC BY-NC"
	case "CC-BY-ND-4.0", "CC-BY-ND-3.0":
		return "CC BY-ND"
	case "CC0-1.0", "CC0":
		return "CC0"
	default:
		if spdx == "" {
			return "Open license"
		}
		return spdx
	}
}

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
