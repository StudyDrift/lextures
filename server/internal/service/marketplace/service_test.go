package marketplace_test

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"

	"github.com/google/uuid"

	svcMarket "github.com/lextures/lextures/server/internal/service/marketplace"
)

func TestPKCE_Valid(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])

	if !svcMarket.VerifyPKCE(challenge, verifier) {
		t.Fatal("PKCE should be valid")
	}
}

func TestPKCE_Invalid(t *testing.T) {
	if svcMarket.VerifyPKCE("challenge", "wrong-verifier") {
		t.Fatal("PKCE should be invalid")
	}
}

func TestPKCE_EmptyInputs(t *testing.T) {
	if svcMarket.VerifyPKCE("", "verifier") {
		t.Fatal("empty challenge should be invalid")
	}
	if svcMarket.VerifyPKCE("challenge", "") {
		t.Fatal("empty verifier should be invalid")
	}
}

func TestStateSignRoundtrip(t *testing.T) {
	secret := []byte("test-secret-32-bytes-long-enough!")
	svc := svcMarket.New(secret)

	orgID := uuid.New()
	userID := uuid.New()
	clientID := "my-client-id"
	redirectURI := "https://app.example.com/callback"
	scopes := []string{"courses:read", "grades:read"}
	challenge := "abc123"

	state, err := svc.BuildConsentURL(orgID, userID, clientID, redirectURI, scopes, challenge)
	if err != nil {
		t.Fatalf("BuildConsentURL error: %v", err)
	}

	claims, err := svc.VerifyConsentState(state)
	if err != nil {
		t.Fatalf("VerifyConsentState error: %v", err)
	}

	if claims.OrgID != orgID {
		t.Errorf("org_id mismatch: got %v want %v", claims.OrgID, orgID)
	}
	if claims.UserID != userID {
		t.Errorf("user_id mismatch: got %v want %v", claims.UserID, userID)
	}
	if claims.ClientID != clientID {
		t.Errorf("client_id mismatch: got %v want %v", claims.ClientID, clientID)
	}
	if claims.CodeChallenge != challenge {
		t.Errorf("code_challenge mismatch: got %v want %v", claims.CodeChallenge, challenge)
	}
}

func TestStateVerify_TamperedSignature(t *testing.T) {
	svc := svcMarket.New([]byte("secret"))
	state, _ := svc.BuildConsentURL(uuid.New(), uuid.New(), "c", "https://x.com", nil, "cc")

	tampered := state[:len(state)-3] + "xxx"
	_, err := svc.VerifyConsentState(tampered)
	if err == nil {
		t.Fatal("expected error for tampered state")
	}
}

func TestStateVerify_WrongSecret(t *testing.T) {
	svc1 := svcMarket.New([]byte("secret-1"))
	svc2 := svcMarket.New([]byte("secret-2"))

	state, _ := svc1.BuildConsentURL(uuid.New(), uuid.New(), "c", "https://x.com", nil, "cc")
	_, err := svc2.VerifyConsentState(state)
	if err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

func TestScopeLabel_GradesWrite(t *testing.T) {
	label := svcMarket.ScopeLabel("grades:write")
	if label == "grades:write" {
		t.Error("grades:write should have a human-readable label, not the scope string itself")
	}
}

func TestScopeIsWrite(t *testing.T) {
	if !svcMarket.ScopeIsWrite("grades:write") {
		t.Error("grades:write should be identified as a write scope")
	}
	if svcMarket.ScopeIsWrite("grades:read") {
		t.Error("grades:read should not be identified as a write scope")
	}
}

func TestValidateRedirectURI(t *testing.T) {
	registered := []string{"https://app.example.com/cb", "https://dev.example.com/cb"}
	if !svcMarket.ValidateRedirectURI(registered, "https://app.example.com/cb") {
		t.Error("registered URI should be valid")
	}
	if svcMarket.ValidateRedirectURI(registered, "https://evil.example.com/cb") {
		t.Error("unregistered URI should be invalid")
	}
	if svcMarket.ValidateRedirectURI(registered, "") {
		t.Error("empty URI should be invalid")
	}
}
