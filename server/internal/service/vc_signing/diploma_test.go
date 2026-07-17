package vcsigning

import (
	"testing"
	"time"
)

func TestSignAndVerifyDiplomaCredential(t *testing.T) {
	key, err := GenerateKey("http://localhost:5173")
	if err != nil {
		t.Fatal(err)
	}
	subject := map[string]any{
		"id":              "urn:lextures:diploma:test-token",
		"type":            "diploma",
		"credentialTitle": "Bachelor of Science",
		"contentHash":     "abc123",
		"verifyToken":     "test-token",
		"version":         1,
	}
	vc, err := SignDiplomaCredential(subject, "Test University", key, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	types, _ := vc["type"].([]string)
	if len(types) != 2 || types[1] != "DiplomaCredential" {
		t.Fatalf("unexpected types: %#v", vc["type"])
	}
	valid, err := VerifyCredential(vc, key.PublicKey)
	if err != nil || !valid {
		t.Fatalf("valid=%v err=%v", valid, err)
	}
}
