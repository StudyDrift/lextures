package webhooks

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/lextures/lextures/server/internal/crypto/appsecrets"
)

const signatureHeader = "X-Lextures-Signature"

// GenerateSigningKey returns a random 32-byte signing key.
func GenerateSigningKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return key, nil
}

// EncryptSigningKey stores a signing key encrypted at rest.
func EncryptSigningKey(plaintext, secretsKey []byte) (string, error) {
	if len(secretsKey) != 32 {
		return "", appsecrets.ErrInvalidKey
	}
	blob, err := appsecrets.Encrypt(plaintext, secretsKey)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(blob), nil
}

// DecryptSigningKey reverses EncryptSigningKey.
func DecryptSigningKey(encoded string, secretsKey []byte) ([]byte, error) {
	if len(secretsKey) != 32 {
		return nil, appsecrets.ErrInvalidKey
	}
	blob, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	return appsecrets.Decrypt(blob, secretsKey)
}

// SignPayload returns the RFC 8959-style signature value for a payload body.
func SignPayload(body, signingKey []byte) string {
	mac := hmac.New(sha256.New, signingKey)
	_, _ = mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// SignatureHeaderName is the outbound request header carrying the HMAC.
func SignatureHeaderName() string {
	return signatureHeader
}

// PayloadHash returns a SHA-256 hex digest of the payload for dedup logging.
func PayloadHash(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// VerifySignature checks an RFC 8959 sha256= signature (for tests and docs).
func VerifySignature(body []byte, signingKey []byte, headerValue string) (bool, error) {
	const prefix = "sha256="
	if !strings.HasPrefix(headerValue, prefix) {
		return false, fmt.Errorf("invalid signature format")
	}
	expected := SignPayload(body, signingKey)
	return hmac.Equal([]byte(expected), []byte(headerValue)), nil
}
