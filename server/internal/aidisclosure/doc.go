// Package aidisclosure holds static AI disclosure documents and hashing helpers (plan 10.17).
package aidisclosure

import _ "embed"

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/google/uuid"
)

//go:embed disclosure_models.json
var PublicDisclosureJSON []byte

// UserIDHash is HMAC-SHA256 of the user UUID for audit logs.
func UserIDHash(secret string, userID uuid.UUID) string {
	key := []byte(strings.TrimSpace(secret))
	if len(key) == 0 {
		key = []byte("lextures-ai-log-default")
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(userID.String()))
	return hex.EncodeToString(mac.Sum(nil))
}

// ContentHash is SHA-256 hex of content (post-redaction prompt).
func ContentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}
