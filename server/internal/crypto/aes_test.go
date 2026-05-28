package crypto

import (
	"strings"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	ciphertext, err := EncryptString("parent@example.com")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if !strings.HasPrefix(ciphertext, "enc:v1:") {
		t.Fatalf("ciphertext prefix missing: %q", ciphertext)
	}
	plaintext, err := DecryptString(ciphertext)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if plaintext != "parent@example.com" {
		t.Fatalf("roundtrip mismatch: got %q", plaintext)
	}
}

func TestEncryptProducesDifferentCiphertext(t *testing.T) {
	a, err := EncryptString("same")
	if err != nil {
		t.Fatal(err)
	}
	b, err := EncryptString("same")
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatalf("ciphertexts should differ due to random nonce")
	}
}
