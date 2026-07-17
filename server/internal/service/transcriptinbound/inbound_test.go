package transcriptinbound

import (
	"testing"

	transcriptsrepo "github.com/lextures/lextures/server/internal/repos/transcripts"
)

func TestNormalizeName(t *testing.T) {
	if got := normalizeName("  Ada, Lovelace "); got != "ada lovelace" {
		t.Fatalf("got %q", got)
	}
}

func TestScoreCandidate_SIDAndName(t *testing.T) {
	sid := "S123"
	display := "Ada Lovelace"
	c := transcriptsrepo.MatchCandidate{
		Email:       "ada@example.com",
		DisplayName: &display,
		SID:         &sid,
	}
	score, reasons := scoreCandidate(c, "ada lovelace", "S123", "2000-01-01", "Prior College")
	if score < AutoMatchMin {
		t.Fatalf("expected auto-match score, got %v reasons=%v", score, reasons)
	}
}

func TestScoreCandidate_LowConfidence(t *testing.T) {
	last := "Smith"
	c := transcriptsrepo.MatchCandidate{
		Email:    "x@example.com",
		LastName: &last,
	}
	score, _ := scoreCandidate(c, "ada lovelace", "", "", "")
	if score >= AutoMatchMin {
		t.Fatalf("expected low confidence, got %v", score)
	}
}

func TestInferFormat(t *testing.T) {
	if got := inferFormat("application/pdf", []byte("%PDF-1.4")); got != transcriptsrepo.InboundFormatPDF {
		t.Fatalf("pdf: %s", got)
	}
	if got := inferFormat("application/xml", []byte(`<?xml version="1.0"?><CollegeTranscript`)); got != transcriptsrepo.InboundFormatPESC {
		t.Fatalf("xml: %s", got)
	}
}

func TestDetectUnsafe_XXE(t *testing.T) {
	raw := []byte(`<?xml version="1.0"?><!DOCTYPE x [<!ENTITY y SYSTEM "file:///etc/passwd">]><a>&y;</a>`)
	unsafe, reason := detectUnsafe(raw, transcriptsrepo.InboundFormatPESC)
	if !unsafe {
		t.Fatalf("expected unsafe, reason=%s", reason)
	}
}

func TestDetectUnsafe_CleanPESC(t *testing.T) {
	raw := []byte(`<?xml version="1.0"?><CollegeTranscript><Student/></CollegeTranscript>`)
	unsafe, reason := detectUnsafe(raw, transcriptsrepo.InboundFormatPESC)
	if unsafe {
		t.Fatalf("unexpected quarantine: %s", reason)
	}
}
