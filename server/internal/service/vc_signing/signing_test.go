package vc_signing

import (
	"encoding/json"
	"testing"
)

func TestDIDWebFromOrigin(t *testing.T) {
	got, err := DIDWebFromOrigin("https://app.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got != "did:web:app.example.com" {
		t.Fatalf("got %q", got)
	}
}

func TestSignAndVerifyCLR(t *testing.T) {
	km, err := GenerateKeyMaterial("https://app.example.com")
	if err != nil {
		t.Fatal(err)
	}
	clr := map[string]any{
		"@context": clrContextURL,
		"type":     "Clr",
		"name":     "Test CLR",
		"assertions": []map[string]any{
			{
				"id":   "urn:uuid:11111111-1111-4111-8111-111111111111",
				"type": "Assertion",
				"achievement": map[string]any{
					"id":   "urn:uuid:22222222-2222-4222-8222-222222222222",
					"type": "Achievement",
					"name": "Completed Course",
				},
				"issuedOn": "2026-01-01T00:00:00Z",
			},
		},
	}
	signed, err := SignCLR(km, clr, "did:example:student")
	if err != nil {
		t.Fatal(err)
	}
	if signed.JWT == "" {
		t.Fatal("expected jwt")
	}
	pub, err := PublicKeyFromJWK(km.PublicKeyJWK)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := VerifyJWT(pub, signed.JWT)
	if err != nil || !ok {
		t.Fatalf("verify: ok=%v err=%v", ok, err)
	}
	doc, err := BuildDIDDocument(km)
	if err != nil {
		t.Fatal(err)
	}
	if doc["id"] != km.IssuerDID {
		t.Fatalf("did doc id %v", doc["id"])
	}
	_, _ = json.Marshal(signed.Credential)
}
