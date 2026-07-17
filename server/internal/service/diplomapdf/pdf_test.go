package diplomapdf

import (
	"bytes"
	"testing"
	"time"
)

func TestBuildDiplomaPDF(t *testing.T) {
	pdf, err := Build(Input{
		Kind:            "diploma",
		InstitutionName: "Test University",
		LearnerName:     "Ada Lovelace",
		CredentialTitle: "Bachelor of Science",
		Program:         "Computer Science",
		Honors:          "summa cum laude",
		ConferredAt:     time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC),
		VerificationURL: "https://app.example.com/verify/abc",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pdf) < 100 {
		t.Fatalf("pdf too small: %d", len(pdf))
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF")) {
		t.Fatalf("missing PDF header")
	}
}

func TestBuildCertificatePDF(t *testing.T) {
	pdf, err := Build(Input{
		Kind:            "certificate",
		InstitutionName: "Test University",
		LearnerName:     "Grace Hopper",
		CredentialTitle: "Program Completion Certificate",
		ConferredAt:     time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF")) {
		t.Fatalf("missing PDF header")
	}
}
