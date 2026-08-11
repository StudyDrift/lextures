package billing

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func hashCouponCode(code, salt string) string {
	normalized := strings.ToUpper(strings.TrimSpace(code))
	sum := sha256.Sum256([]byte(salt + ":" + normalized))
	return hex.EncodeToString(sum[:])
}
