package credentialwallet

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/config"
	walletrepo "github.com/lextures/lextures/server/internal/repos/wallet"
)

func TestNormalizeDisclosure(t *testing.T) {
	if NormalizeDisclosure("") != walletrepo.DisclosureValidity {
		t.Fatalf("empty should default to validity")
	}
	if NormalizeDisclosure("full") != walletrepo.DisclosureFull {
		t.Fatalf("full")
	}
	if NormalizeDisclosure("nope") != walletrepo.DisclosureValidity {
		t.Fatalf("invalid should default to validity")
	}
}

func TestFilterItem_DisclosureLevels(t *testing.T) {
	tok := "abc123"
	issuer := "State U"
	issued := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	it := walletrepo.Item{
		ID:          uuid.New(),
		Kind:        walletrepo.KindTranscript,
		Title:       "Official transcript (v1)",
		Issuer:      &issuer,
		IssuedAt:    &issued,
		VerifyToken: &tok,
		Revoked:     false,
	}

	validity := FilterItem(it, walletrepo.DisclosureValidity, "http://localhost:5173")
	if validity.Title != nil || validity.Issuer != nil || validity.VerifyURL != nil {
		t.Fatalf("validity should be minimal: %+v", validity)
	}
	if !validity.Valid || validity.Kind != "transcript" {
		t.Fatalf("validity basics: %+v", validity)
	}

	summary := FilterItem(it, walletrepo.DisclosureSummary, "http://localhost:5173")
	if summary.Title == nil || *summary.Title != it.Title {
		t.Fatalf("summary title: %+v", summary)
	}
	if summary.VerifyURL != nil {
		t.Fatalf("summary must not include verify URL")
	}
	if summary.VerifyStatus == nil || *summary.VerifyStatus != "verified" {
		t.Fatalf("summary status: %+v", summary)
	}

	full := FilterItem(it, walletrepo.DisclosureFull, "http://localhost:5173")
	if full.VerifyURL == nil || *full.VerifyURL != "http://localhost:5173/verify/abc123" {
		t.Fatalf("full verify URL: %+v", full)
	}

	it.Revoked = true
	revoked := FilterItem(it, walletrepo.DisclosureSummary, "http://localhost:5173")
	if revoked.Valid || !revoked.Revoked {
		t.Fatalf("revoked projection: %+v", revoked)
	}
	if revoked.VerifyStatus == nil || *revoked.VerifyStatus != "revoked" {
		t.Fatalf("revoked status: %+v", revoked)
	}
}

func TestVerifyStatus(t *testing.T) {
	it := walletrepo.Item{Kind: walletrepo.KindCERecord}
	if VerifyStatus(it) != "unavailable" {
		t.Fatalf("ce_record")
	}
	tok := "x"
	it = walletrepo.Item{Kind: walletrepo.KindBadge, VerifyToken: &tok}
	if VerifyStatus(it) != "verified" {
		t.Fatalf("badge with token")
	}
}

func TestEnabled(t *testing.T) {
	if Enabled(config.Config{}) {
		t.Fatal("all off")
	}
	if !Enabled(config.Config{FFTranscripts: true}) {
		t.Fatal("transcripts on")
	}
	if !Enabled(config.Config{FFCompetencyBadges: true}) {
		t.Fatal("badges on")
	}
}

func TestSanitizePath(t *testing.T) {
	if sanitizePath("Official Transcript!") != "official-transcript" {
		t.Fatalf("got %q", sanitizePath("Official Transcript!"))
	}
	if sanitizePath("@@@") != "item" {
		t.Fatalf("empty fallback")
	}
}
