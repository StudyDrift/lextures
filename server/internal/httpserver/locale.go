package httpserver

import (
	"strings"

	"github.com/lextures/lextures/server/internal/repos/user"
)

// v1Locales supported for locale switching (plan 11.1/11.2).
var v1Locales = map[string]struct{}{
	"en": {},
	"es": {},
	"fr": {},
	"ar": {},
	"he": {},
	"fa": {},
	"ur": {},
	"ps": {},
}

func normalizeUserLocale(raw *string) (string, error) {
	if raw == nil {
		return "", apierrError("Locale is required.")
	}
	primary := user.NormalizeLocalePrimary(*raw)
	if primary == "" {
		return "", apierrError("Locale must be a valid BCP 47 language tag.")
	}
	if _, ok := v1Locales[primary]; !ok {
		return "", apierrError("Locale is not supported.")
	}
	return primary, nil
}

func localeOrDefault(s string) string {
	primary := user.NormalizeLocalePrimary(s)
	if primary == "" {
		return user.DefaultLocale
	}
	if _, ok := v1Locales[primary]; !ok {
		return user.DefaultLocale
	}
	return primary
}

func detectBrowserLocale(acceptLanguage string) string {
	if strings.TrimSpace(acceptLanguage) == "" {
		return user.DefaultLocale
	}
	for _, part := range strings.Split(acceptLanguage, ",") {
		tag := strings.TrimSpace(strings.Split(part, ";")[0])
		if tag == "" || tag == "*" {
			continue
		}
		if loc := localeOrDefault(tag); loc != user.DefaultLocale || strings.HasPrefix(strings.ToLower(tag), "en") {
			return loc
		}
		primary := user.NormalizeLocalePrimary(tag)
		if primary != "" {
			if _, ok := v1Locales[primary]; ok {
				return primary
			}
		}
	}
	return user.DefaultLocale
}
