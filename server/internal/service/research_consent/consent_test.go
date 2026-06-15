package research_consent

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSignRecord_DeterministicAndVerifiable(t *testing.T) {
	secret := "01234567890123456789012345678901"
	study := uuid.New()
	user := uuid.New()
	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)

	sig := SignRecord(secret, study, user, "granted", now)
	if sig == "" {
		t.Fatal("expected non-empty signature")
	}
	if again := SignRecord(secret, study, user, "granted", now); again != sig {
		t.Fatalf("signature not deterministic: %s != %s", sig, again)
	}
	if !VerifyRecord(secret, sig, study, user, "granted", now) {
		t.Fatal("VerifyRecord should accept a matching signature")
	}
}

func TestSignRecord_TamperEvidence(t *testing.T) {
	secret := "01234567890123456789012345678901"
	study := uuid.New()
	user := uuid.New()
	now := time.Now().UTC()
	sig := SignRecord(secret, study, user, "granted", now)

	// Changing the decision must invalidate the signature.
	if VerifyRecord(secret, sig, study, user, "withdrawn", now) {
		t.Fatal("tampered decision should fail verification")
	}
	// A different user must invalidate the signature.
	if VerifyRecord(secret, sig, study, uuid.New(), "granted", now) {
		t.Fatal("tampered user should fail verification")
	}
}

func TestSignRecord_EmptySecretDisablesSigning(t *testing.T) {
	if got := SignRecord("", uuid.New(), uuid.New(), "granted", time.Now()); got != "" {
		t.Fatalf("empty secret should yield empty signature, got %q", got)
	}
	if VerifyRecord("", "anything", uuid.New(), uuid.New(), "granted", time.Now()) {
		t.Fatal("verification with empty secret should fail")
	}
}

func TestValidDecision(t *testing.T) {
	for _, d := range []string{"granted", "declined", "withdrawn"} {
		if !ValidDecision(d) {
			t.Errorf("%q should be valid", d)
		}
	}
	for _, d := range []string{"", "GRANTED", "yes", "revoked"} {
		if ValidDecision(d) {
			t.Errorf("%q should be invalid", d)
		}
	}
}
