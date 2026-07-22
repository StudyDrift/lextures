package oidcauth

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestHashNonceSHA256(t *testing.T) {
	raw := "test-nonce-value"
	sum := sha256.Sum256([]byte(raw))
	want := hex.EncodeToString(sum[:])
	if got := HashNonceSHA256(raw); got != want {
		t.Fatalf("HashNonceSHA256: got %q want %q", got, want)
	}
	if HashNonceSHA256("") == HashNonceSHA256("x") {
		t.Fatal("empty and non-empty nonces should differ")
	}
}

func TestAudienceAllowed(t *testing.T) {
	allowed := []string{"com.lextures.ios", "com.lextures.ios.tests"}
	if !audienceAllowed("com.lextures.ios", allowed) {
		t.Fatal("expected bundle id allowed")
	}
	if audienceAllowed("com.apple.servicesid", allowed) {
		t.Fatal("web Services ID must not be accepted as native audience")
	}
	if audienceAllowed("", allowed) {
		t.Fatal("empty aud rejected")
	}
	if audienceAllowed("com.lextures.ios", nil) {
		t.Fatal("empty allow-list rejects")
	}
}

func TestConfigAppleNativeAudiences(t *testing.T) {
	c := config.Config{OIDCAppleNativeAudience: " com.lextures.ios , com.lextures.ios.tests "}
	got := c.OIDCAppleNativeAudiences()
	if len(got) != 2 || got[0] != "com.lextures.ios" || got[1] != "com.lextures.ios.tests" {
		t.Fatalf("audiences: %#v", got)
	}
	c2 := config.Config{}
	def := c2.OIDCAppleNativeAudiences()
	if len(def) != 1 || def[0] != "com.lextures.ios" {
		t.Fatalf("default audiences: %#v", def)
	}
}

func TestConfigGoogleNativeAudienceResolved(t *testing.T) {
	c := config.Config{OIDCGoogleClientID: "web-client", OIDCGoogleNativeAudience: "native-client"}
	if c.OIDCGoogleNativeAudienceResolved() != "native-client" {
		t.Fatal("prefer explicit native audience")
	}
	c2 := config.Config{OIDCGoogleClientID: "web-only"}
	if c2.OIDCGoogleNativeAudienceResolved() != "web-only" {
		t.Fatal("fallback to web client id")
	}
}

func TestNativeAvailableGates(t *testing.T) {
	on := config.Config{
		OIDCAppleNativeAudience: "com.lextures.ios",
		OIDCGoogleClientID:      "g-client",
	}
	if !on.OIDCAppleNativeAvailable() || !on.OIDCGoogleNativeAvailable() {
		t.Fatal("audiences configured → native available")
	}
	// Empty audience list only if OIDCAppleNativeAudience is explicitly empty after resolution —
	// defaults still yield com.lextures.ios.
	if !(config.Config{}).OIDCAppleNativeAvailable() {
		t.Fatal("default apple audience available")
	}
	noGoogle := config.Config{OIDCAppleNativeAudience: "com.lextures.ios"}
	if !noGoogle.OIDCAppleNativeAvailable() {
		t.Fatal("apple still available")
	}
	if noGoogle.OIDCGoogleNativeAvailable() {
		t.Fatal("google needs audience")
	}
}

func TestPrivateRelayEmailShape(t *testing.T) {
	// Acceptance: relay addresses are valid emails (server stores them as-is after normalize).
	email := userNormalizeEmail("  alice@privaterelay.appleid.com  ")
	if email != "alice@privaterelay.appleid.com" {
		t.Fatalf("got %q", email)
	}
	if !containsAt(email) {
		t.Fatal("relay must pass @ check")
	}
}

func containsAt(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '@' {
			return true
		}
	}
	return false
}
