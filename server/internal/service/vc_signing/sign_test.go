package vcsigning

import (
	"encoding/base64"
	"testing"
	"time"
)

func TestSignAndVerifyCredential(t *testing.T) {
	key, err := GenerateKey("http://localhost:8080")
	if err != nil {
		t.Fatal(err)
	}
	subject := map[string]any{
		"id":   "urn:uuid:clr:test",
		"type": "ClrCredential",
		"assertions": []map[string]any{
			{
				"id":   "urn:uuid:assertion:1",
				"type": "Assertion",
				"achievement": map[string]any{
					"name": "Leadership Role",
				},
				"issuedOn": time.Now().UTC().Format(time.RFC3339),
			},
		},
	}
	vc, err := SignCredential(subject, "Test University", key, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	valid, err := VerifyCredential(vc, key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	if !valid {
		t.Fatal("expected valid credential")
	}
}

func TestKeyFromPrivateSeedDeterministic(t *testing.T) {
	seed := base64.StdEncoding.EncodeToString(make([]byte, 32))
	k1, err := KeyFromPrivateSeed(seed, "http://localhost:8080")
	if err != nil {
		t.Fatal(err)
	}
	k2, err := KeyFromPrivateSeed(seed, "http://localhost:8080")
	if err != nil {
		t.Fatal(err)
	}
	if k1.DID != k2.DID {
		t.Fatalf("DID mismatch: %s vs %s", k1.DID, k2.DID)
	}
}

func TestSignAndVerifyAchievementCredential(t *testing.T) {
	key, err := GenerateKey("http://localhost:8080")
	if err != nil {
		t.Fatal(err)
	}
	subject := map[string]any{
		"type": []string{"AchievementSubject"},
		"id":   "urn:uuid:user:test",
		"name": "Learner",
		"achievement": map[string]any{
			"type": []string{"Achievement"},
			"name": "Course Complete",
		},
	}
	vc, err := SignAchievementCredential(subject, "Lextures", key, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	valid, err := VerifyCredential(vc, key.PublicKey)
	if err != nil || !valid {
		t.Fatalf("valid=%v err=%v", valid, err)
	}
}

func TestVerifyCredentialRejectsTamperedProof(t *testing.T) {
	key, err := GenerateKey("http://localhost:8080")
	if err != nil {
		t.Fatal(err)
	}
	vc, err := SignCredential(map[string]any{"id": "x", "type": "ClrCredential"}, "Inst", key, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	proof := vc["proof"].(map[string]any)
	proof["proofValue"] = base64.StdEncoding.EncodeToString([]byte("tampered"))
	valid, err := VerifyCredential(vc, key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	if valid {
		t.Fatal("expected invalid credential after tampering")
	}
}
