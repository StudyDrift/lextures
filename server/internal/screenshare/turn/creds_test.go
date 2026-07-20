package turn

import (
	"fmt"
	"testing"
	"time"
)

func TestMintAndValidate(t *testing.T) {
	now := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	c, err := Mint("shared-secret", "user-123", now, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	wantUser := fmt.Sprintf("%d:user-123", now.Add(time.Hour).Unix())
	if c.Username != wantUser {
		t.Fatalf("username got %q want %q", c.Username, wantUser)
	}
	if c.Credential == "" {
		t.Fatal("empty credential")
	}
	if !ValidateExpiry(c.Username, now) {
		t.Fatal("should be valid now")
	}
	if ValidateExpiry(c.Username, now.Add(2*time.Hour)) {
		t.Fatal("should be expired")
	}
}

func TestMintRejectsEmpty(t *testing.T) {
	_, err := Mint("", "u", time.Now(), time.Hour)
	if err == nil {
		t.Fatal("expected error")
	}
	_, err = Mint("s", "", time.Now(), time.Hour)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMintCapsTTL(t *testing.T) {
	c, err := Mint("secret", "u1", time.Now(), 48*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if c.TTLSeconds > int64((12 * time.Hour).Seconds()) {
		t.Fatalf("ttl %d", c.TTLSeconds)
	}
}

func TestValidateExpiryBad(t *testing.T) {
	if ValidateExpiry("not-a-cred", time.Now()) {
		t.Fatal("bad username")
	}
}
