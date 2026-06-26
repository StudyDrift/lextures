package l10n

import (
	"errors"
	"regexp"
	"strings"
)

var localeTagRe = regexp.MustCompile(`^[a-zA-Z]{2,3}(-[a-zA-Z0-9]{2,8})*$`)

// NormalizeLocale canonicalizes a BCP 47 language tag for storage.
func NormalizeLocale(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", errors.New("locale is required")
	}
	if len(s) > 35 {
		return "", errors.New("locale tag is too long")
	}
	parts := strings.Split(s, "-")
	if len(parts) == 0 || len(parts[0]) < 2 || len(parts[0]) > 3 {
		return "", errors.New("invalid locale language subtag")
	}
	parts[0] = strings.ToLower(parts[0])
	for i := 1; i < len(parts); i++ {
		p := parts[i]
		if len(p) == 2 {
			parts[i] = strings.ToUpper(p)
		} else if len(p) == 4 {
			parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
		} else {
			parts[i] = strings.ToLower(p)
		}
	}
	out := strings.Join(parts, "-")
	if !localeTagRe.MatchString(out) {
		return "", errors.New("invalid BCP 47 locale tag")
	}
	return out, nil
}
