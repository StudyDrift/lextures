package pdfrender

import (
	"strings"
	"testing"
	"time"
)

func TestBuildCertificate(t *testing.T) {
	pdf, err := BuildCertificate(CertificateInput{
		InstitutionName: "Lextures",
		LearnerName:     "Test Learner",
		AchievementName: "Intro to Testing",
		Description:     "Completed all modules.",
		IssuedAt:        time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC),
		VerificationURL: "http://localhost:5173/verify/00000000-0000-0000-0000-000000000001",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pdf) < 1000 {
		t.Fatalf("pdf too small: %d bytes", len(pdf))
	}
	if !strings.HasPrefix(string(pdf[:4]), "%PDF") {
		t.Fatal("expected PDF header")
	}
}