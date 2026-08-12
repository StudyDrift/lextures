package marketingcontent

import (
	"strings"
	"time"
	"unicode"
)

const DefaultLocale = "en"

// PublicPath is the canonical public URL for an article. English stays unprefixed;
// every other locale is path-prefixed (/{locale}/blog/{slug} or /{locale}/docs/{cat}/{slug}).
func PublicPath(kind, locale, categorySlug, slug string) string {
	locale = NormalizeLocaleCode(locale)
	prefix := ""
	if locale != "" && locale != DefaultLocale {
		prefix = "/" + locale
	}
	if kind == "blog" {
		return prefix + "/blog/" + slug
	}
	return prefix + "/docs/" + categorySlug + "/" + slug
}

// NormalizeLocaleCode lowercases a BCP-47 tag and rejects path-unsafe input.
func NormalizeLocaleCode(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return DefaultLocale
	}
	var b strings.Builder
	for i, r := range s {
		if i == 0 {
			if !unicode.IsLetter(r) {
				return ""
			}
		} else if r != '-' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return ""
		}
		b.WriteRune(unicode.ToLower(r))
	}
	out := b.String()
	if strings.Contains(out, "..") || strings.ContainsAny(out, `/\`) {
		return ""
	}
	if len(out) < 2 || len(out) > 12 {
		return ""
	}
	return out
}

// TranslationIsStale is true when the source article was edited after the
// translation was last marked in sync.
func TranslationIsStale(sourceUpdated, syncedAt *time.Time) bool {
	if sourceUpdated == nil || syncedAt == nil {
		return false
	}
	return sourceUpdated.After(*syncedAt)
}
