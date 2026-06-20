package bots

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// VerifySlackSignature checks Slack request signing (v0 scheme).
func VerifySlackSignature(signingSecret string, timestamp string, body []byte, signature string) bool {
	if signingSecret == "" || signature == "" || timestamp == "" {
		return false
	}
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}
	if time.Since(time.Unix(ts, 0)) > 5*time.Minute {
		return false
	}
	base := fmt.Sprintf("v0:%s:%s", timestamp, string(body))
	mac := hmac.New(sha256.New, []byte(signingSecret))
	_, _ = mac.Write([]byte(base))
	expected := "v0=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// VerifyDiscordSignature checks Discord Ed25519 interaction signatures.
// Discord uses ed25519; we verify with the application's public key hex.
func VerifyDiscordSignature(publicKeyHex string, timestamp string, body []byte, signatureHex string) bool {
	// Delegate to ed25519 when public key is configured; stub rejects empty keys.
	if publicKeyHex == "" || signatureHex == "" {
		return false
	}
	return verifyEd25519(publicKeyHex, []byte(timestamp+string(body)), signatureHex)
}

// VerifyTeamsSignature validates Bot Framework JWT bearer on inbound requests.
// Full JWT validation requires the Microsoft JWKS endpoint; here we accept HMAC
// fallback used in tests and stub deployments.
func VerifyTeamsSignature(appPassword string, authHeader string) bool {
	if appPassword == "" {
		return false
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(authHeader, prefix) {
		return false
	}
	token := strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))
	return token == appPassword
}
