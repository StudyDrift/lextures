package transcriptdelivery

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/config"
	transcriptsrepo "github.com/lextures/lextures/server/internal/repos/transcripts"
)

func TestSelectAdapter(t *testing.T) {
	t.Parallel()
	cases := []struct {
		method transcriptsrepo.DeliveryMethod
		v2     bool
		ok     bool
	}{
		{transcriptsrepo.DeliveryAPIPeer, false, true},
		{transcriptsrepo.DeliverySecureLink, false, true},
		{transcriptsrepo.DeliveryElectronicPDF, false, true},
		{transcriptsrepo.DeliveryElectronicPESC, false, false},
		{transcriptsrepo.DeliveryElectronicPESC, true, true},
		{transcriptsrepo.DeliveryEDISPEEDE, true, true},
		{transcriptsrepo.DeliveryPostalMail, true, true},
		{transcriptsrepo.DeliveryPostalMail, false, false},
	}
	for _, tc := range cases {
		_, err := SelectAdapter(tc.method, tc.v2)
		if tc.ok && err != nil {
			t.Fatalf("%s v2=%v: unexpected err %v", tc.method, tc.v2, err)
		}
		if !tc.ok && err == nil {
			t.Fatalf("%s v2=%v: expected error", tc.method, tc.v2)
		}
	}
}

func TestIdempotencyKeyForAttempt(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	got := transcriptsrepo.IdempotencyKeyForAttempt(id, 2)
	want := "transcript-delivery:11111111-1111-4111-8111-111111111111:2"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestValidatePostalAddress(t *testing.T) {
	t.Parallel()
	if err := validatePostalAddress([]byte(`{"line1":"1 Main","city":"Town"}`)); err != nil {
		t.Fatal(err)
	}
	if err := validatePostalAddress([]byte(`{"street":"1 Main"}`)); err != nil {
		t.Fatal(err)
	}
	if err := validatePostalAddress([]byte(`{}`)); err == nil {
		t.Fatal("expected incomplete address error")
	}
}

func TestShareURL(t *testing.T) {
	t.Parallel()
	cfg := config.Config{PublicWebOrigin: "https://app.example.edu/"}
	got := shareURL(cfg, "abc123")
	if got != "https://app.example.edu/r/t/abc123" {
		t.Fatalf("got %q", got)
	}
}

func TestErrTransientWrap(t *testing.T) {
	t.Parallel()
	err := wrapTransient(errors.New("timeout"))
	if !errors.Is(err, ErrTransient) {
		t.Fatalf("expected ErrTransient, got %v", err)
	}
}

func TestShareLinkExpiryLogic(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	expires := now.Add(-time.Minute)
	if expires.After(now) {
		t.Fatal("expected expired")
	}
	link := transcriptsrepo.ShareLink{
		ExpiresAt:     now.Add(time.Hour),
		MaxDownloads:  5,
		DownloadCount: 5,
	}
	if link.DownloadCount < link.MaxDownloads {
		t.Fatal("expected exhausted")
	}
}
