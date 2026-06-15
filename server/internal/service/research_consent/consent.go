// Package research_consent provides HMAC tamper-evidence signing and helpers for
// the research / IRB consent module (plan 14.15).
package research_consent

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

// SignRecord returns a hex HMAC-SHA256 over the immutable fields of a consent
// decision, providing tamper evidence for audit (FR-7). An empty secret yields
// an empty signature (signing disabled).
func SignRecord(secret string, studyID, userID uuid.UUID, decision string, createdAt time.Time) string {
	if secret == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(secret))
	payload := studyID.String() + "|" + userID.String() + "|" + decision + "|" +
		createdAt.UTC().Format(time.RFC3339Nano)
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyRecord reports whether sig matches the HMAC of the given fields.
func VerifyRecord(secret, sig string, studyID, userID uuid.UUID, decision string, createdAt time.Time) bool {
	if secret == "" {
		return false
	}
	want := SignRecord(secret, studyID, userID, decision, createdAt)
	return hmac.Equal([]byte(want), []byte(sig))
}

// ValidDecision reports whether a decision string is one of the accepted values.
func ValidDecision(d string) bool {
	switch d {
	case "granted", "declined", "withdrawn":
		return true
	default:
		return false
	}
}
