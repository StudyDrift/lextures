package quizgame

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	NicknameMinRunes = 1
	NicknameMaxRunes = 24
)

// ValidateNickname trims and checks length + allowed charset for player nicknames.
// Allowed: letters, numbers, spaces, and common punctuation (_ - . ' !).
func ValidateNickname(raw string) (string, error) {
	nick := strings.TrimSpace(raw)
	n := utf8.RuneCountInString(nick)
	if n < NicknameMinRunes || n > NicknameMaxRunes {
		return "", fmt.Errorf("quizgame: invalid nickname length")
	}
	for _, r := range nick {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' || r == '_' || r == '-' || r == '.' || r == '\'' || r == '!' {
			continue
		}
		return "", fmt.Errorf("quizgame: invalid nickname charset")
	}
	return nick, nil
}
