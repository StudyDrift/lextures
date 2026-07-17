package vcsigning

import (
	"testing"
	"time"
)

func TestSignAndVerifyTranscriptCredential(t *testing.T) {
	key, err := GenerateKey("http://localhost:5173")
	if err != nil {
		t.Fatal(err)
	}
	subject := map[string]any{
		"id":          "urn:lextures:transcript:test-token",
		"type":        "OfficialTranscript",
		"contentHash": "abc123",
		"variant":     "official",
	}
	vc, err := SignTranscriptCredential(subject, "Test University", key, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	types, _ := vc["type"].([]string)
	if len(types) != 2 || types[1] != "OfficialTranscriptCredential" {
		t.Fatalf("unexpected types: %#v", vc["type"])
	}
	valid, err := VerifyCredential(vc, key.PublicKey)
	if err != nil || !valid {
		t.Fatalf("valid=%v err=%v", valid, err)
	}
}
