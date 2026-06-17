package organization

import (
	"errors"
	"regexp"
	"strings"
	"unicode"
)

var slugPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)

var reservedSlugs = map[string]struct{}{
	"default":     {},
	"www":         {},
	"app":         {},
	"api":         {},
	"admin":       {},
	"login":       {},
	"signup":      {},
	"mfa":         {},
	"magic-link":  {},
}

// NormalizeSlug lowercases and trims a tenant slug input.
func NormalizeSlug(raw string) string {
	return strings.TrimSpace(strings.ToLower(raw))
}

// SuggestSlugFromName derives a URL-safe slug from a display name.
func SuggestSlugFromName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	var b strings.Builder
	lastHyphen := false
	for _, r := range strings.ToLower(name) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastHyphen = false
			continue
		}
		if !lastHyphen && b.Len() > 0 {
			b.WriteByte('-')
			lastHyphen = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 32 {
		out = strings.Trim(out[:32], "-")
	}
	return out
}

// ValidateSlug returns nil when slug is acceptable for a new or updated organization.
func ValidateSlug(slug string) error {
	slug = NormalizeSlug(slug)
	if slug == "" {
		return errors.New("organization: slug is required")
	}
	if len(slug) < 2 {
		return errors.New("organization: slug must be at least 2 characters")
	}
	if len(slug) > 32 {
		return errors.New("organization: slug must be at most 32 characters")
	}
	if !slugPattern.MatchString(slug) {
		return errors.New("organization: slug may only contain lowercase letters, numbers, and hyphens")
	}
	if _, reserved := reservedSlugs[slug]; reserved {
		return errors.New("organization: that slug is reserved")
	}
	return nil
}