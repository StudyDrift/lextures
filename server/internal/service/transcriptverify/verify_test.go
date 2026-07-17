package transcriptverify

import (
	"testing"
	"time"

	vcsigning "github.com/lextures/lextures/server/internal/service/vc_signing"
)

func TestBuildTranscriptSubject(t *testing.T) {
	s := BuildTranscriptSubject("tok-1", "hash-1", "official")
	if s["id"] != "urn:lextures:transcript:tok-1" {
		t.Fatalf("id=%v", s["id"])
	}
	if s["contentHash"] != "hash-1" {
		t.Fatalf("hash=%v", s["contentHash"])
	}
}

func TestVerificationURL(t *testing.T) {
	got := VerificationURL("https://app.example.com/", "abc")
	if got != "https://app.example.com/verify/abc" {
		t.Fatalf("got %q", got)
	}
	if VerificationURL("", "abc") != "" {
		t.Fatal("empty origin should yield empty URL")
	}
}

func TestPDFHashDeterministic(t *testing.T) {
	a := PDFHash([]byte("%PDF-1.4 test"))
	b := PDFHash([]byte("%PDF-1.4 test"))
	if a != b || a == "" {
		t.Fatalf("hash mismatch or empty: %q %q", a, b)
	}
	if PDFHash([]byte("%PDF-1.4 other")) == a {
		t.Fatal("different bytes must produce different hash")
	}
}

func TestSignRoundTripMatchesVerifyCredential(t *testing.T) {
	key, err := vcsigning.GenerateKey("http://localhost:5173")
	if err != nil {
		t.Fatal(err)
	}
	subject := BuildTranscriptSubject("token", "deadbeef", "official")
	vc, err := vcsigning.SignTranscriptCredential(subject, "U", key, time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	ok, err := vcsigning.VerifyCredential(vc, key.PublicKey)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}
