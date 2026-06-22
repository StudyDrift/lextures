package lti

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func testKeyPair(t *testing.T) (*rsa.PrivateKey, *RsaKeyPair) {
	t.Helper()
	pk, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(pk)
	if err != nil {
		t.Fatal(err)
	}
	pemData := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
	pair, err := FromPKCS8PEM(pemData, "test-kid")
	if err != nil {
		t.Fatal(err)
	}
	return pk, pair
}

func TestSignAndVerifyConsumerLoginHint(t *testing.T) {
	_, pair := testKeyPair(t)
	hint, err := pair.SignConsumerLoginHint("https://platform.example", "https://tool.example", "user-1", "tool-id", "course-1", "item-1", "mod-1", "https://return", true)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := VerifyConsumerLoginHint(hint, pair.PublicKey(), "https://platform.example", "https://tool.example")
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "user-1" || !claims.DeepLink || claims.CourseID != "course-1" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestSignDeepLinkingMessageHint(t *testing.T) {
	_, pair := testKeyPair(t)
	tok, err := pair.SignDeepLinkingMessageHint("https://platform.example", "client-1", "user-1", "https://platform.example/api/v1/lti/deep-link", `{"courseId":"c1"}`)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := DecodeJWTPayloadJSON(tok)
	if err != nil {
		t.Fatal(err)
	}
	if payload[ClaimMessageType] != MsgDeepLinkingRequest {
		t.Fatalf("message type: %v", payload[ClaimMessageType])
	}
}

func TestPlatformLaunchIDTokenClaims(t *testing.T) {
	claims := PlatformLaunchIDTokenClaims(
		"https://platform.example/", "client-1", "user-1", "nonce-1", "https://target",
		MsgResourceLinkRequest, "course-1", "item-1", "rl-1", "Title", nil, nil, "",
	)
	if claims["iss"] != "https://platform.example" {
		t.Fatalf("iss: %v", claims["iss"])
	}
	if claims[ClaimMessageType] != MsgResourceLinkRequest {
		t.Fatal("message type")
	}
}

func TestParseDeepLinkingContentItems(t *testing.T) {
	items, err := ParseDeepLinkingContentItems(map[string]any{
		ClaimMessageType: MsgDeepLinkingResponse,
		ClaimDLContentItems: []any{
			map[string]any{"type": "ltiResourceLink", "title": "A"},
		},
	})
	if err != nil || len(items) != 1 {
		t.Fatalf("items=%v err=%v", items, err)
	}
	if _, err := ParseDeepLinkingContentItems(map[string]any{ClaimMessageType: MsgResourceLinkRequest}); err == nil {
		t.Fatal("expected error for wrong message type")
	}
}

func TestVerifyToolMessageJWT(t *testing.T) {
	pk, pair := testKeyPair(t)
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   "https://tool.example",
		"aud":   "https://platform.example",
		"iat":   now.Unix(),
		"exp":   now.Add(5 * time.Minute).Unix(),
		ClaimMessageType: MsgDeepLinkingResponse,
		ClaimDLContentItems: []any{map[string]any{"type": "link", "url": "https://x"}},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := tok.SignedString(pk)
	if err != nil {
		t.Fatal(err)
	}
	_ = pair
	got, err := VerifyToolMessageJWT(signed, &pk.PublicKey, "https://platform.example", "client-1")
	if err != nil {
		t.Fatal(err)
	}
	if got[ClaimMessageType] != MsgDeepLinkingResponse {
		t.Fatal("message type mismatch")
	}
}