package engine

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// JoinCodeDigits is the default join-code length (FR-9).
const JoinCodeDigits = 6

// GenerateJoinCode returns a cryptographically random numeric code of n digits
// (leading zeros allowed so the space is full 10^n).
func GenerateJoinCode(n int) (string, error) {
	if n <= 0 {
		n = JoinCodeDigits
	}
	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n)), nil)
	v, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", fmt.Errorf("join code: %w", err)
	}
	return fmt.Sprintf("%0*d", n, v.Int64()), nil
}
