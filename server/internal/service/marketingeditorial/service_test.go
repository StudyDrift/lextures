package marketingeditorial

import (
	"testing"
	"time"
)

func TestPublicURLBlocksPrivateDestinations(t *testing.T) {
	for _, raw := range []string{"http://127.0.0.1/admin", "http://[::1]/", "http://169.254.169.254/latest/meta-data"} {
		if _, err := publicURL(raw); err == nil {
			t.Errorf("publicURL(%q) accepted a private destination", raw)
		}
	}
}

func TestParseRange(t *testing.T) {
	from, to, err := ParseRange("2026-08-01", "2026-09-01")
	if err != nil {
		t.Fatal(err)
	}
	if got := to.Sub(from); got != 32*24*time.Hour {
		t.Fatalf("inclusive query range = %v", got)
	}
	if _, _, err = ParseRange("2026-09-01", "2026-08-01"); err == nil {
		t.Fatal("expected reversed range to fail")
	}
}
