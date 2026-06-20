package webhooks

import (
	"testing"
)

func TestSignAndVerifyPayload(t *testing.T) {
	key := []byte("01234567890123456789012345678901")
	body := []byte(`{"event_id":"abc","event_type":"grade.posted"}`)
	sig := SignPayload(body, key)
	ok, err := VerifySignature(body, key, sig)
	if err != nil || !ok {
		t.Fatalf("verify failed: ok=%v err=%v sig=%s", ok, err, sig)
	}
	tampered := append(body, 'x')
	ok, _ = VerifySignature(tampered, key, sig)
	if ok {
		t.Fatal("expected tampered payload to fail verification")
	}
}

func TestEncryptDecryptSigningKey(t *testing.T) {
	secretsKey := []byte("01234567890123456789012345678901")
	plain, err := GenerateSigningKey()
	if err != nil {
		t.Fatal(err)
	}
	enc, err := EncryptSigningKey(plain, secretsKey)
	if err != nil {
		t.Fatal(err)
	}
	out, err := DecryptSigningKey(enc, secretsKey)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string(plain) {
		t.Fatalf("round trip mismatch")
	}
}

func TestPayloadHashStable(t *testing.T) {
	body := []byte(`{"a":1}`)
	h1 := PayloadHash(body)
	h2 := PayloadHash(body)
	if h1 != h2 || h1 == "" {
		t.Fatalf("hash=%q", h1)
	}
}
