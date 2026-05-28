package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const ciphertextPrefix = "enc:v1:"

var errInvalidCiphertext = errors.New("crypto: invalid ciphertext")

func deriveKey() []byte {
	if raw := strings.TrimSpace(os.Getenv("COLUMN_ENCRYPTION_KEY")); raw != "" {
		if decoded, err := base64.StdEncoding.DecodeString(raw); err == nil && len(decoded) == 32 {
			return decoded
		}
	}
	sum := sha256.Sum256([]byte("lextures-dev-column-encryption-key"))
	return sum[:]
}

func EncryptString(plaintext string) (string, error) {
	key := deriveKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("crypto: new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("crypto: new gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("crypto: nonce: %w", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return ciphertextPrefix + base64.StdEncoding.EncodeToString(ciphertext), nil
}

func DecryptString(ciphertext string) (string, error) {
	if !strings.HasPrefix(ciphertext, ciphertextPrefix) {
		return "", errInvalidCiphertext
	}
	encoded := strings.TrimPrefix(ciphertext, ciphertextPrefix)
	payload, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("crypto: decode: %w", err)
	}
	key := deriveKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("crypto: new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("crypto: new gcm: %w", err)
	}
	if len(payload) < gcm.NonceSize() {
		return "", errInvalidCiphertext
	}
	nonce, sealed := payload[:gcm.NonceSize()], payload[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return "", fmt.Errorf("crypto: open: %w", err)
	}
	return string(plaintext), nil
}

func MaybeDecryptString(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	if !strings.HasPrefix(*value, ciphertextPrefix) {
		return value, nil
	}
	plaintext, err := DecryptString(*value)
	if err != nil {
		return nil, err
	}
	return &plaintext, nil
}
