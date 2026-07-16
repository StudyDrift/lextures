package transcriptconsent

import (
	"strings"
	"testing"
)

func TestAuthorizationText_VersionsAndLocales(t *testing.T) {
	en, err := AuthorizationText(CurrentTextVersion, "en")
	if err != nil || !strings.Contains(en, "FERPA") {
		t.Fatalf("en: %v %q", err, en)
	}
	es, err := AuthorizationText("ferpa-release-v1", "es-MX")
	if err != nil || !strings.Contains(es, "AUTORIZACIÓN") {
		t.Fatalf("es: %v", err)
	}
	fr, err := AuthorizationText(CurrentTextVersion, "fr_FR")
	if err != nil || !strings.Contains(fr, "AUTORISATION") {
		t.Fatalf("fr: %v", err)
	}
	if _, err := AuthorizationText("unknown-v9", "en"); err == nil {
		t.Fatal("expected unknown version error")
	}
}

func TestHashPayload_Stable(t *testing.T) {
	p := Payload{
		OrderID:    "o1",
		UserID:     "u1",
		SignerID:   "s1",
		SignerRole: "student",
		Recipients: []RecipientSnapshot{{ID: "r1", Type: "institution", Name: "State U"}},
		Scope:      ScopeFullAcademicRecord,
		Purpose:    PurposeTranscriptRelease,
		TextVersion: CurrentTextVersion,
		Locale:     "en",
		Agree:      true,
	}
	h1, err := HashPayload(p)
	if err != nil || len(h1) != 64 {
		t.Fatalf("hash1: %v %q", err, h1)
	}
	h2, err := HashPayload(p)
	if err != nil || h1 != h2 {
		t.Fatalf("unstable hash: %s vs %s", h1, h2)
	}
	p.Agree = false
	h3, _ := HashPayload(p)
	if h3 == h1 {
		t.Fatal("expected hash to change when agree flips")
	}
}
